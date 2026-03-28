package api

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

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
	wsHub           *ws.Hub
	upgrader        websocket.Upgrader
}

// NewHandler 创建新的 API 处理器
func NewHandler(
	pm process.Manager,
	sm storage.Manager,
	plm playlist.Manager,
	am *audio.Manager,
	hub *ws.Hub,
) *Handler {
	return &Handler{
		processManager:  pm,
		storageManager:  sm,
		playlistManager: plm,
		audioManager:    am,
		wsHub:           hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
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
	if h.processManager.IsRunning() {
		if err := h.processManager.Restart(req.Frequency); err != nil {
			respondError(w, http.StatusInternalServerError, "设置频率失败")
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true, "frequency": req.Frequency, "message": "频率设置成功",
	})
}

// StartBroadcast 开始广播 POST /api/broadcast/start
func (h *Handler) StartBroadcast(w http.ResponseWriter, r *http.Request) {
	if h.processManager.IsRunning() {
		respondError(w, http.StatusBadRequest, "广播已在运行中")
		return
	}
	audioStream := h.audioManager.GetAudioStream()
	if audioStream == nil {
		audioStream = strings.NewReader("")
	}
	status := h.processManager.GetStatus()
	freq := status.Frequency
	if freq == 0 {
		freq = 100.0
	}
	if err := h.processManager.Start(freq, audioStream); err != nil {
		respondError(w, http.StatusInternalServerError, "启动广播失败: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "广播已启动"})
}

// StopBroadcast 停止广播 POST /api/broadcast/stop
func (h *Handler) StopBroadcast(w http.ResponseWriter, r *http.Request) {
	if !h.processManager.IsRunning() {
		respondError(w, http.StatusBadRequest, "广播未在运行")
		return
	}
	if err := h.processManager.Stop(); err != nil {
		respondError(w, http.StatusInternalServerError, "停止广播失败")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "广播已停止"})
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
		"success": true, "file_id": fileID, "message": "文件上传成功",
	})
}

// ListFiles 获取文件列表 GET /api/files
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.storageManager.ListFiles()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "获取文件列表失败")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "files": files})
}

// DeleteFile 删除文件 DELETE /api/files/{id}
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := path.Base(r.URL.Path)
	if fileID == "" || fileID == "." {
		respondError(w, http.StatusBadRequest, "文件 ID 不能为空")
		return
	}
	if err := h.storageManager.Delete(fileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "文件未找到")
			return
		}
		respondError(w, http.StatusInternalServerError, "删除文件失败")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "文件删除成功"})
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
}

// GetStatus 获取系统状态 GET /api/status
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ps := h.processManager.GetStatus()
	qi := h.storageManager.GetQuotaInfo()
	items := h.playlistManager.GetAll()
	cur := h.playlistManager.GetCurrent()
	var curFileID string
	if cur != nil {
		curFileID = cur.FileID
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"running": ps.Running, "pid": ps.PID, "frequency": ps.Frequency,
		"start_time": ps.StartTime, "playlist_count": len(items),
		"current_file": curFileID, "storage_used": qi.Used,
		"storage_total": qi.Total, "storage_available": qi.Available,
		"ws_clients": h.wsHub.GetClientCount(),
	})
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
