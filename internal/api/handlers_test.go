package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/liigoo/pi-fm-rds-go/internal/audio"
	"github.com/liigoo/pi-fm-rds-go/internal/playlist"
	"github.com/liigoo/pi-fm-rds-go/internal/process"
	"github.com/liigoo/pi-fm-rds-go/internal/storage"
	ws "github.com/liigoo/pi-fm-rds-go/internal/websocket"
)

type mockProcessManager struct {
	running   bool
	frequency float64
}

func (m *mockProcessManager) Start(freq float64, src io.Reader) error {
	m.running = true
	m.frequency = freq
	return nil
}
func (m *mockProcessManager) Stop() error             { m.running = false; return nil }
func (m *mockProcessManager) Restart(f float64) error { m.frequency = f; return nil }
func (m *mockProcessManager) IsRunning() bool         { return m.running }
func (m *mockProcessManager) GetStatus() process.ProcessStatus {
	return process.ProcessStatus{Running: m.running, PID: 12345, Frequency: m.frequency, StartTime: time.Now()}
}
func (m *mockProcessManager) CleanupOrphans() error { return nil }
func (m *mockProcessManager) ValidateGPIO() error   { return nil }

type mockStorageManager struct {
	files map[string]*storage.FileInfo
	dir   string
}

func newMockStorage() *mockStorageManager {
	dir, _ := os.MkdirTemp("", "mock-storage-*")
	return &mockStorageManager{files: make(map[string]*storage.FileInfo), dir: dir}
}

func (m *mockStorageManager) Upload(file io.Reader, h *multipart.FileHeader) (string, error) {
	id := "test-file-id"
	m.files[id] = &storage.FileInfo{ID: id, Filename: h.Filename, Size: h.Size, Format: "audio/mpeg"}
	return id, nil
}
func (m *mockStorageManager) Delete(id string) error {
	if _, ok := m.files[id]; !ok {
		return io.ErrUnexpectedEOF // not found
	}
	delete(m.files, id)
	return nil
}
func (m *mockStorageManager) GetFile(id string) (*storage.FileInfo, error) { return m.files[id], nil }
func (m *mockStorageManager) GetFilePath(id string) (string, error) {
	return filepath.Join(m.dir, id), nil
}
func (m *mockStorageManager) ListFiles() ([]*storage.FileInfo, error) {
	files := make([]*storage.FileInfo, 0)
	for _, f := range m.files {
		files = append(files, f)
	}
	return files, nil
}
func (m *mockStorageManager) GetQuotaInfo() storage.QuotaInfo {
	return storage.QuotaInfo{Used: 1024, Total: 10240, Available: 9216}
}

func newTestHandler() (*Handler, *mockProcessManager, *mockStorageManager) {
	pm := &mockProcessManager{}
	sm := newMockStorage()
	plm := playlist.NewManager()
	am := audio.NewManager(&audio.Config{SampleRate: 44100, Channels: 2})
	hub := ws.NewHub(10)
	return NewHandler(pm, sm, plm, am, hub), pm, sm
}

func TestFrequencyHandler(t *testing.T) {
	handler, _, _ := newTestHandler()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid frequency", `{"frequency": 100.5}`, http.StatusOK},
		{"too low", `{"frequency": 80.0}`, http.StatusBadRequest},
		{"too high", `{"frequency": 110.0}`, http.StatusBadRequest},
		{"invalid json", `{invalid}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/frequency", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.SetFrequency(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestBroadcastHandlers(t *testing.T) {
	t.Run("start broadcast", func(t *testing.T) {
		handler, pm, _ := newTestHandler()
		if err := handler.audioManager.PlayMicrophone("default"); err != nil {
			t.Fatalf("failed to prepare microphone source: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/broadcast/start", nil)
		w := httptest.NewRecorder()
		handler.StartBroadcast(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
		if !pm.IsRunning() {
			t.Error("process should be running")
		}
	})

	t.Run("start broadcast - no audio source", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/api/broadcast/start", nil)
		w := httptest.NewRecorder()
		handler.StartBroadcast(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("start broadcast - use playlist file as source", func(t *testing.T) {
		handler, pm, sm := newTestHandler()
		if err := handler.playlistManager.Add("missing-file", "missing.wav", 0); err != nil {
			t.Fatalf("failed to add first playlist item: %v", err)
		}

		fileID := "playlist-file-1"
		filePath := filepath.Join(sm.dir, fileID)
		if err := os.WriteFile(filePath, []byte("audio-bytes"), 0644); err != nil {
			t.Fatalf("failed to write mock audio file: %v", err)
		}
		sm.files[fileID] = &storage.FileInfo{ID: fileID, Filename: "test.wav", Size: 10, Format: "audio/wav"}
		if err := handler.playlistManager.Add(fileID, "test.wav", 0); err != nil {
			t.Fatalf("failed to add playlist item: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/broadcast/start", nil)
		w := httptest.NewRecorder()
		handler.StartBroadcast(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
		if !pm.IsRunning() {
			t.Error("process should be running after start")
		}
	})

	t.Run("stop broadcast", func(t *testing.T) {
		handler, pm, _ := newTestHandler()
		pm.running = true
		req := httptest.NewRequest(http.MethodPost, "/api/broadcast/stop", nil)
		w := httptest.NewRecorder()
		handler.StopBroadcast(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
		if pm.IsRunning() {
			t.Error("process should not be running")
		}
	})

	t.Run("stop when not running", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/api/broadcast/stop", nil)
		w := httptest.NewRecorder()
		handler.StopBroadcast(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})
}

func TestFileHandlers(t *testing.T) {
	t.Run("upload file", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.mp3")
		part.Write([]byte("fake audio"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		handler.UploadFile(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("list files", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
		w := httptest.NewRecorder()
		handler.ListFiles(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("delete file", func(t *testing.T) {
		handler, _, sm := newTestHandler()
		sm.files["test-id"] = &storage.FileInfo{ID: "test-id"}

		req := httptest.NewRequest(http.MethodDelete, "/api/files/test-id", nil)
		w := httptest.NewRecorder()
		handler.DeleteFile(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestPlaylistHandlers(t *testing.T) {
	t.Run("add to playlist", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		body := `{"file_id": "test-id", "filename": "test.mp3"}`
		req := httptest.NewRequest(http.MethodPost, "/api/playlist/add", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.AddToPlaylist(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("remove from playlist", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		handler.playlistManager.Add("test-id", "test.mp3", 0)

		req := httptest.NewRequest(http.MethodDelete, "/api/playlist/test-id", nil)
		w := httptest.NewRecorder()
		handler.RemoveFromPlaylist(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("reorder playlist", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		handler.playlistManager.Add("id1", "f1.mp3", 0)
		handler.playlistManager.Add("id2", "f2.mp3", 0)

		body := `{"from_index": 0, "to_index": 1}`
		req := httptest.NewRequest(http.MethodPost, "/api/playlist/reorder", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ReorderPlaylist(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestStatusHandler(t *testing.T) {
	handler, pm, _ := newTestHandler()
	pm.running = true
	pm.frequency = 100.5

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["running"] != true {
		t.Error("should return running=true")
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	// 在受限环境中，本地监听端口可能被禁止；此用例跳过以避免 panic。
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local listen not permitted in this environment: %v", err)
	}
	_ = l.Close()

	handler, _, _ := newTestHandler()
	go handler.wsHub.Run()
	defer handler.wsHub.Stop()

	server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	conn.Close()
}

func TestFileHandlersErrorCases(t *testing.T) {
	t.Run("upload - no file field", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("other", "value")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		handler.UploadFile(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("delete - file not found", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodDelete, "/api/files/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.DeleteFile(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("list files - returns success", func(t *testing.T) {
		handler, _, sm := newTestHandler()
		sm.files["id1"] = &storage.FileInfo{ID: "id1", Filename: "test.mp3"}
		req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
		w := httptest.NewRecorder()
		handler.ListFiles(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", w.Code, http.StatusOK)
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["success"] != true {
			t.Error("should return success=true")
		}
	})
}

func TestPlaylistHandlersErrorCases(t *testing.T) {
	t.Run("add - missing file_id", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		body := `{"filename": "test.mp3"}`
		req := httptest.NewRequest(http.MethodPost, "/api/playlist/add", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.AddToPlaylist(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("add - invalid json", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/api/playlist/add", strings.NewReader("{bad}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.AddToPlaylist(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("remove - not in playlist", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodDelete, "/api/playlist/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.RemoveFromPlaylist(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})

	t.Run("reorder - invalid json", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/api/playlist/reorder", strings.NewReader("{bad}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ReorderPlaylist(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("reorder - invalid index", func(t *testing.T) {
		handler, _, _ := newTestHandler()
		handler.playlistManager.Add("id1", "f1.mp3", 0)
		body := `{"from_index": 0, "to_index": 99}`
		req := httptest.NewRequest(http.MethodPost, "/api/playlist/reorder", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ReorderPlaylist(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})
}

func TestBroadcastAlreadyRunning(t *testing.T) {
	handler, pm, _ := newTestHandler()
	pm.running = true
	req := httptest.NewRequest(http.MethodPost, "/api/broadcast/start", nil)
	w := httptest.NewRecorder()
	handler.StartBroadcast(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}
