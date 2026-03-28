package recovery

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTranscodeErrorHandler_CaptureError 测试错误捕获
func TestTranscodeErrorHandler_CaptureError(t *testing.T) {
	handler := NewTranscodeErrorHandler()

	err := handler.CaptureError("test.mp3", "invalid format")
	if err == nil {
		t.Error("Expected error to be captured")
	}

	if !handler.HasError() {
		t.Error("Expected handler to have error")
	}
}

// TestTranscodeErrorHandler_GetUserFriendlyMessage 测试用户友好错误提示
func TestTranscodeErrorHandler_GetUserFriendlyMessage(t *testing.T) {
	handler := NewTranscodeErrorHandler()

	handler.CaptureError("test.mp3", "invalid format")
	msg := handler.GetUserFriendlyMessage()

	if msg == "" {
		t.Error("Expected non-empty user friendly message")
	}

	if len(msg) < 10 {
		t.Error("Expected detailed user friendly message")
	}
}

// TestTranscodeErrorHandler_CleanupTempFiles 测试临时文件清理
func TestTranscodeErrorHandler_CleanupTempFiles(t *testing.T) {
	handler := NewTranscodeErrorHandler()

	// 创建临时文件
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "temp_audio.wav")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// 注册临时文件
	handler.RegisterTempFile(tempFile)

	// 清理临时文件
	if err := handler.CleanupTempFiles(); err != nil {
		t.Errorf("Failed to cleanup temp files: %v", err)
	}

	// 验证文件已删除
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Expected temp file to be deleted")
	}
}

// TestTranscodeErrorHandler_Reset 测试重置
func TestTranscodeErrorHandler_Reset(t *testing.T) {
	handler := NewTranscodeErrorHandler()

	handler.CaptureError("test.mp3", "invalid format")
	handler.RegisterTempFile("/tmp/test.wav")

	handler.Reset()

	if handler.HasError() {
		t.Error("Expected no error after reset")
	}

	if len(handler.GetTempFiles()) != 0 {
		t.Error("Expected no temp files after reset")
	}
}

// TestTranscodeErrorHandler_MultipleErrors 测试多个错误
func TestTranscodeErrorHandler_MultipleErrors(t *testing.T) {
	handler := NewTranscodeErrorHandler()

	handler.CaptureError("test1.mp3", "error 1")
	handler.CaptureError("test2.mp3", "error 2")

	errors := handler.GetAllErrors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
}
