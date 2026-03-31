package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/liigoo/pi-fm-rds-go/internal/audio"
	"github.com/liigoo/pi-fm-rds-go/internal/playlist"
	"github.com/liigoo/pi-fm-rds-go/internal/process"
	"github.com/liigoo/pi-fm-rds-go/internal/storage"
	ws "github.com/liigoo/pi-fm-rds-go/internal/websocket"
)

// Handler API 处理器
type Handler struct {
	processManager  process.Manager
	storageManager  storage.Manager
	playlistManager playlist.Manager
	audioManager    *audio.Manager
	transcoder      *audio.Transcoder
	wsHub           *ws.Hub
	upgrader        websocket.Upgrader

	controlMu sync.Mutex
	paused    bool
	stopped   bool

	currentFrequency float64
}

// NewHandler 创建新的 API 处理器
func NewHandler(
	pm process.Manager,
	sm storage.Manager,
	plm playlist.Manager,
	am *audio.Manager,
	hub *ws.Hub,
	defaultFrequency float64,
) *Handler {
	h := &Handler{
		processManager:  pm,
		storageManager:  sm,
		playlistManager: plm,
		audioManager:    am,
		transcoder:      audio.NewTranscoder("/tmp/pi-fm-rds-cache"),
		wsHub:           hub,
		paused:          false,
		stopped:         true,
		currentFrequency: defaultFrequency,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	go h.autoAdvanceLoop()
	return h
}

// SetFrequency 设置 FM 频率 POST /api/frequency
func (h *Handler) SetFrequency(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Frequency float64 `json:"frequency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}
	if req.Frequency < 87.5 || req.Frequency > 108.0 {
		respondError(w, http.StatusBadRequest, "频率必须在 87.5-108.0 MHz 之间")
		return
	}

	h.controlMu.Lock()
	h.currentFrequency = req.Frequency
	running := h.processManager.IsRunning()
	h.controlMu.Unlock()

	if running {
		if err := h.processManager.Restart(req.Frequency); err != nil {
			respondError(w, http.StatusInternalServerError, "设置频率失败")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"frequency": req.Frequency,
		"message":   "频率设置成功",
	})
	h.broadcastStatus()
}

// StartBroadcast 开始广播 POST /api/broadcast/start
func (h *Handler) StartBroadcast(w http.ResponseWriter, r *http.Request) {
	h.controlMu.Lock()
	if h.processManager.IsRunning() {
		h.controlMu.Unlock()
		respondError(w, http.StatusBadRequest, "广播已在运行中")
		return
	}

	audioStream := h.audioManager.GetAudioStream()
	if audioStream != nil {
		freq := h.currentFrequencyLocked()
		if err := h.processManager.Start(freq, audioStream); err != nil {
			h.controlMu.Unlock()
			respondError(w, http.StatusInternalServerError, "启动广播失败: "+err.Error())
			return
		}

		h.paused = false
		h.stopped = false
		h.controlMu.Unlock()

		respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "广播已启动"})
		h.broadcastStatus()
		return
	}
	h.controlMu.Unlock()

	h.Play(w, r)
}

// StopBroadcast 停止广播 POST /api/broadcast/stop
func (h *Handler) StopBroadcast(w http.ResponseWriter, r *http.Request) {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	if !h.processManager.IsRunning() {
		respondError(w, http.StatusBadRequest, "广播未在运行")
		return
	}

	if err := h.processManager.Stop(); err != nil {
		respondError(w, http.StatusInternalServerError, "停止广播失败")
		return
	}
	_ = h.audioManager.Stop()

	h.paused = false
	h.stopped = true
	h.playlistManager.ResetCurrent()

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "广播已停止"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// Play 播放/恢复播放 POST /api/playback/play
func (h *Handler) Play(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Index  *int   `json:"index"`
		FileID string `json:"file_id"`
	}

	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			respondError(w, http.StatusBadRequest, "无效的请求格式")
			return
		}
	}

	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	fileID, err := h.resolveTargetFileLocked(req.Index, req.FileID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.startPlaybackForFileLocked(fileID); err != nil {
		respondError(w, http.StatusInternalServerError, "播放失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "开始播放"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// Pause 暂停播放 POST /api/playback/pause
func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	if !h.processManager.IsRunning() {
		respondError(w, http.StatusBadRequest, "当前未在播放")
		return
	}

	if err := h.processManager.Stop(); err != nil {
		respondError(w, http.StatusInternalServerError, "暂停失败")
		return
	}
	_ = h.audioManager.Stop()

	h.paused = true
	h.stopped = false

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "已暂停"})
	h.broadcastStatus()
}

// StopPlayback 停止播放 POST /api/playback/stop
func (h *Handler) StopPlayback(w http.ResponseWriter, r *http.Request) {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	if h.processManager.IsRunning() {
		if err := h.processManager.Stop(); err != nil {
			respondError(w, http.StatusInternalServerError, "停止广播失败")
			return
		}
	}
	_ = h.audioManager.Stop()

	h.paused = false
	h.stopped = true
	h.playlistManager.ResetCurrent()

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "广播已停止"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// Next 播放下一曲 POST /api/playback/next
func (h *Handler) Next(w http.ResponseWriter, r *http.Request) {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	if h.processManager.IsRunning() {
		if err := h.processManager.Stop(); err != nil {
			respondError(w, http.StatusInternalServerError, "切换下一曲失败")
			return
		}
		_ = h.audioManager.Stop()
	}

	fileID, err := h.playlistManager.Next()
	if err != nil {
		respondError(w, http.StatusBadRequest, "下一曲不可用: "+err.Error())
		return
	}

	if err := h.startPlaybackForFileLocked(fileID); err != nil {
		respondError(w, http.StatusInternalServerError, "切换下一曲失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "已切到下一曲"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// Prev 播放上一曲 POST /api/playback/prev
func (h *Handler) Prev(w http.ResponseWriter, r *http.Request) {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	if h.processManager.IsRunning() {
		if err := h.processManager.Stop(); err != nil {
			respondError(w, http.StatusInternalServerError, "切换上一曲失败")
			return
		}
		_ = h.audioManager.Stop()
	}

	fileID, err := h.playlistManager.Prev()
	if err != nil {
		respondError(w, http.StatusBadRequest, "上一曲不可用: "+err.Error())
		return
	}

	if err := h.startPlaybackForFileLocked(fileID); err != nil {
		respondError(w, http.StatusInternalServerError, "切换上一曲失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "已切到上一曲"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

func (h *Handler) resolveTargetFileLocked(index *int, fileID string) (string, error) {
	if fileID != "" {
		idx, err := h.ensureFileInPlaylistLocked(fileID)
		if err != nil {
			return "", fmt.Errorf("无法播放所选文件")
		}
		return h.playlistManager.SetCurrent(idx)
	}

	if index != nil {
		return h.playlistManager.SetCurrent(*index)
	}

	cur := h.playlistManager.GetCurrent()
	if cur != nil && cur.FileID != "" && h.canPlayFile(cur.FileID) {
		return cur.FileID, nil
	}

	items := h.playlistManager.GetAll()
	for _, item := range items {
		if item.FileID == "" || !h.canPlayFile(item.FileID) {
			continue
		}
		if _, err := h.playlistManager.SetCurrent(item.Index); err == nil {
			return item.FileID, nil
		}
	}

	return "", fmt.Errorf("无可用音频源，请先上传并加入播放列表")
}

func (h *Handler) ensureFileInPlaylistLocked(fileID string) (int, error) {
	if idx := h.playlistManager.IndexOf(fileID); idx >= 0 {
		return idx, nil
	}

	info, err := h.storageManager.GetFile(fileID)
	if err != nil || info == nil {
		return -1, fmt.Errorf("file %s not found", fileID)
	}

	if err := h.playlistManager.Add(fileID, info.Filename, info.Duration); err != nil && !strings.Contains(err.Error(), "already exists") {
		return -1, err
	}

	idx := h.playlistManager.IndexOf(fileID)
	if idx < 0 {
		return -1, fmt.Errorf("failed to locate file in playlist")
	}
	return idx, nil
}

func (h *Handler) startPlaybackForFileLocked(fileID string) error {
	if fileID == "" {
		return fmt.Errorf("empty file id")
	}

	if err := h.playByFileID(fileID); err != nil {
		return err
	}

	audioStream := h.audioManager.GetAudioStream()
	if audioStream == nil {
		return fmt.Errorf("audio source unavailable")
	}

	if h.processManager.IsRunning() {
		if err := h.processManager.Stop(); err != nil {
			return err
		}
	}

	freq := h.currentFrequencyLocked()

	if err := h.processManager.Start(freq, audioStream); err != nil {
		return err
	}

	h.paused = false
	h.stopped = false
	return nil
}

func (h *Handler) canPlayFile(fileID string) bool {
	if fileID == "" {
		return false
	}
	filePath, err := h.storageManager.GetFilePath(fileID)
	if err != nil {
		return false
	}
	if _, err := os.Stat(filePath); err != nil {
		return false
	}
	return true
}

func (h *Handler) autoAdvanceLoop() {
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		h.controlMu.Lock()
		if h.processManager.IsRunning() || h.paused || h.stopped {
			h.controlMu.Unlock()
			continue
		}

		nextFileID, err := h.playlistManager.Next()
		if err != nil {
			h.stopped = true
			h.controlMu.Unlock()
			continue
		}

		if err := h.startPlaybackForFileLocked(nextFileID); err != nil {
			h.stopped = true
		}
		h.controlMu.Unlock()

		h.broadcastPlaylist()
		h.broadcastStatus()
	}
}

func (h *Handler) playByFileID(fileID string) error {
	filePath, err := h.storageManager.GetFilePath(fileID)
	if err != nil {
		return err
	}

	playPath := filePath
	if h.transcoder != nil {
		if format, detectErr := h.transcoder.DetectFormat(filePath); detectErr == nil && format != audio.FormatWAV {
			transcodedPath, transcodeErr := h.transcoder.Transcode(filePath)
			if transcodeErr != nil {
				return transcodeErr
			}
			playPath = transcodedPath
		}
	}

	return h.audioManager.PlayFile(playPath)
}

// UploadFile 上传音频文件 POST /api/files/upload
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "解析表单失败")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "获取文件失败: "+err.Error())
		return
	}
	defer file.Close()

	fileID, err := h.storageManager.Upload(file, header)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "上传文件失败: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"file_id": fileID,
		"message": "文件上传成功",
	})
	h.broadcastStatus()
}

// ListFiles 获取文件列表 GET /api/files
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.storageManager.ListFiles()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "获取文件列表失败")
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Filename) < strings.ToLower(files[j].Filename)
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "files": files})
}

// DeleteFile 删除文件 DELETE /api/files/{id}
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := path.Base(r.URL.Path)
	if fileID == "" || fileID == "." {
		respondError(w, http.StatusBadRequest, "文件 ID 不能为空")
		return
	}

	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	current := h.playlistManager.GetCurrent()
	isCurrent := current != nil && current.FileID == fileID

	if err := h.storageManager.Delete(fileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "文件未找到")
			return
		}
		respondError(w, http.StatusInternalServerError, "删除文件失败")
		return
	}

	_ = h.playlistManager.Remove(fileID)

	if isCurrent {
		if h.processManager.IsRunning() {
			_ = h.processManager.Stop()
		}
		_ = h.audioManager.Stop()
		h.paused = false
		h.stopped = true
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "文件删除成功"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// AddToPlaylist 添加到播放列表 POST /api/playlist/add
func (h *Handler) AddToPlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileID   string `json:"file_id"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}
	if req.FileID == "" {
		respondError(w, http.StatusBadRequest, "文件 ID 不能为空")
		return
	}
	if err := h.playlistManager.Add(req.FileID, req.Filename, 0); err != nil {
		respondError(w, http.StatusBadRequest, "添加到播放列表失败: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "已添加到播放列表"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// RemoveFromPlaylist 从播放列表移除 DELETE /api/playlist/{id}
func (h *Handler) RemoveFromPlaylist(w http.ResponseWriter, r *http.Request) {
	fileID := path.Base(r.URL.Path)
	if fileID == "" || fileID == "." {
		respondError(w, http.StatusBadRequest, "文件 ID 不能为空")
		return
	}
	if err := h.playlistManager.Remove(fileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "播放列表中未找到该文件")
			return
		}
		respondError(w, http.StatusInternalServerError, "从播放列表移除失败")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "已从播放列表移除"})
	h.broadcastPlaylist()
	h.broadcastStatus()
}

// ReorderPlaylist 调整播放顺序 POST /api/playlist/reorder
func (h *Handler) ReorderPlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromIndex int `json:"from_index"`
		ToIndex   int `json:"to_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}
	if err := h.playlistManager.Reorder(req.FromIndex, req.ToIndex); err != nil {
		respondError(w, http.StatusBadRequest, "调整播放顺序失败: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "播放顺序调整成功"})
	h.broadcastPlaylist()
}

// GetPlaylist 获取播放列表 GET /api/playlist
func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	items := h.playlistManager.GetAll()
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "items": items})
}

// GetStatus 获取系统状态 GET /api/status
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ps := h.processManager.GetStatus()
	qi := h.storageManager.GetQuotaInfo()
	items := h.playlistManager.GetAll()
	cur := h.playlistManager.GetCurrent()

	var curFileID string
	curIndex := -1
	if cur != nil {
		curFileID = cur.FileID
		curIndex = cur.Index
	}

	h.controlMu.Lock()
	paused := h.paused
	stopped := h.stopped
	frequency := h.currentFrequency
	h.controlMu.Unlock()

	if ps.Frequency >= 87.5 && ps.Frequency <= 108.0 {
		frequency = ps.Frequency
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"running":           ps.Running,
		"paused":            paused,
		"stopped":           stopped,
		"pid":               ps.PID,
		"frequency":         frequency,
		"start_time":        ps.StartTime,
		"playlist_count":    len(items),
		"current_file":      curFileID,
		"current_index":     curIndex,
		"storage_used":      qi.Used,
		"storage_total":     qi.Total,
		"storage_available": qi.Available,
		"ws_clients":        h.wsHub.GetClientCount(),
	})
}

func (h *Handler) broadcastPlaylist() {
	h.broadcast("playlist", h.playlistManager.GetAll())
}

func (h *Handler) broadcastStatus() {
	ps := h.processManager.GetStatus()
	frequency := h.currentFrequency
	if frequency < 87.5 || frequency > 108.0 {
		frequency = 100.0
	}
	if ps.Frequency >= 87.5 && ps.Frequency <= 108.0 {
		frequency = ps.Frequency
	}

	h.broadcast("status", map[string]interface{}{
		"running":   ps.Running,
		"paused":    h.paused,
		"frequency": frequency,
		"pid":       ps.PID,
	})
}

func (h *Handler) currentFrequencyLocked() float64 {
	if h.currentFrequency >= 87.5 && h.currentFrequency <= 108.0 {
		return h.currentFrequency
	}
	return 100.0
}

func (h *Handler) broadcast(messageType string, data interface{}) {
	payload, err := json.Marshal(map[string]interface{}{
		"type": messageType,
		"data": data,
	})
	if err != nil {
		return
	}
	_ = h.wsHub.Broadcast(payload)
}

// HandleWebSocket WebSocket 升级处理 GET /ws
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "WebSocket 升级失败")
		return
	}
	client := ws.NewClient("", conn, h.wsHub)
	h.wsHub.Register(client)
	go client.WritePump()
	go client.ReadPump()
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{"success": false, "error": message})
}
