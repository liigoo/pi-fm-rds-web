package recovery

import (
	"fmt"
	"os"
	"sync"
)

// TranscodeError 转码错误信息
type TranscodeError struct {
	FileName string
	Message  string
}

// TranscodeErrorHandler 转码错误处理器
type TranscodeErrorHandler struct {
	errors    []TranscodeError
	tempFiles []string
	mu        sync.RWMutex
}

// NewTranscodeErrorHandler 创建转码错误处理器
func NewTranscodeErrorHandler() *TranscodeErrorHandler {
	return &TranscodeErrorHandler{
		errors:    make([]TranscodeError, 0),
		tempFiles: make([]string, 0),
	}
}

// CaptureError 捕获转码错误
func (h *TranscodeErrorHandler) CaptureError(fileName, message string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	err := TranscodeError{
		FileName: fileName,
		Message:  message,
	}
	h.errors = append(h.errors, err)

	return fmt.Errorf("transcode error for %s: %s", fileName, message)
}

// HasError 检查是否有错误
func (h *TranscodeErrorHandler) HasError() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.errors) > 0
}

// GetUserFriendlyMessage 获取用户友好的错误提示
func (h *TranscodeErrorHandler) GetUserFriendlyMessage() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.errors) == 0 {
		return ""
	}

	lastError := h.errors[len(h.errors)-1]
	return fmt.Sprintf("无法处理音频文件 '%s': %s。请检查文件格式是否正确。",
		lastError.FileName, lastError.Message)
}

// RegisterTempFile 注册临时文件
func (h *TranscodeErrorHandler) RegisterTempFile(filePath string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tempFiles = append(h.tempFiles, filePath)
}

// CleanupTempFiles 清理临时文件
func (h *TranscodeErrorHandler) CleanupTempFiles() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, filePath := range h.tempFiles {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove temp file %s: %w", filePath, err)
		}
	}

	h.tempFiles = make([]string, 0)
	return nil
}

// Reset 重置错误处理器
func (h *TranscodeErrorHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = make([]TranscodeError, 0)
	h.tempFiles = make([]string, 0)
}

// GetTempFiles 获取临时文件列表
func (h *TranscodeErrorHandler) GetTempFiles() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]string{}, h.tempFiles...)
}

// GetAllErrors 获取所有错误
func (h *TranscodeErrorHandler) GetAllErrors() []TranscodeError {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]TranscodeError{}, h.errors...)
}
