package recovery

import (
	"testing"
	"time"
)

// TestMicrophoneReconnectManager_DisconnectDetection 测试断开检测
func TestMicrophoneReconnectManager_DisconnectDetection(t *testing.T) {
	manager := NewMicrophoneReconnectManager(5 * time.Second)

	// 模拟断开
	manager.OnDisconnect()

	if !manager.IsDisconnected() {
		t.Error("Expected microphone to be disconnected")
	}
}

// TestMicrophoneReconnectManager_RetryInterval 测试重试间隔
func TestMicrophoneReconnectManager_RetryInterval(t *testing.T) {
	manager := NewMicrophoneReconnectManager(5 * time.Second)

	interval := manager.GetRetryInterval()
	if interval != 5*time.Second {
		t.Errorf("Expected 5s retry interval, got %v", interval)
	}
}

// TestMicrophoneReconnectManager_ShouldRetry 测试是否应该重试
func TestMicrophoneReconnectManager_ShouldRetry(t *testing.T) {
	manager := NewMicrophoneReconnectManager(5 * time.Second)

	// 初始状态不应该重试
	if manager.ShouldRetry() {
		t.Error("Should not retry when not disconnected")
	}

	// 断开后应该重试
	manager.OnDisconnect()
	if !manager.ShouldRetry() {
		t.Error("Should retry after disconnect")
	}
}

// TestMicrophoneReconnectManager_OnReconnect 测试重连成功
func TestMicrophoneReconnectManager_OnReconnect(t *testing.T) {
	manager := NewMicrophoneReconnectManager(5 * time.Second)

	manager.OnDisconnect()
	manager.OnReconnect()

	if manager.IsDisconnected() {
		t.Error("Expected microphone to be connected after reconnect")
	}

	if manager.ShouldRetry() {
		t.Error("Should not retry after successful reconnect")
	}
}

// TestMicrophoneReconnectManager_GetRetryCount 测试重试次数
func TestMicrophoneReconnectManager_GetRetryCount(t *testing.T) {
	manager := NewMicrophoneReconnectManager(5 * time.Second)

	manager.OnDisconnect()
	manager.IncrementRetryCount()
	manager.IncrementRetryCount()

	if manager.GetRetryCount() != 2 {
		t.Errorf("Expected retry count 2, got %d", manager.GetRetryCount())
	}

	manager.OnReconnect()
	if manager.GetRetryCount() != 0 {
		t.Errorf("Expected retry count 0 after reconnect, got %d", manager.GetRetryCount())
	}
}
