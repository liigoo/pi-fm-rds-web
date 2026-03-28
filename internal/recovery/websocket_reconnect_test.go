package recovery

import (
	"testing"
	"time"
)

// TestWebSocketReconnectManager_DisconnectDetection 测试断开检测
func TestWebSocketReconnectManager_DisconnectDetection(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	// 模拟断开
	manager.OnDisconnect()

	if !manager.IsDisconnected() {
		t.Error("Expected WebSocket to be disconnected")
	}
}

// TestWebSocketReconnectManager_ReconnectTimeout 测试重连超时
func TestWebSocketReconnectManager_ReconnectTimeout(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	timeout := manager.GetReconnectTimeout()
	if timeout != 5*time.Second {
		t.Errorf("Expected 5s reconnect timeout, got %v", timeout)
	}
}

// TestWebSocketReconnectManager_ShouldReconnect 测试是否应该重连
func TestWebSocketReconnectManager_ShouldReconnect(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	// 初始状态不应该重连
	if manager.ShouldReconnect() {
		t.Error("Should not reconnect when not disconnected")
	}

	// 断开后应该重连
	manager.OnDisconnect()
	if !manager.ShouldReconnect() {
		t.Error("Should reconnect after disconnect")
	}
}

// TestWebSocketReconnectManager_OnReconnect 测试重连成功
func TestWebSocketReconnectManager_OnReconnect(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	manager.OnDisconnect()
	manager.OnReconnect()

	if manager.IsDisconnected() {
		t.Error("Expected WebSocket to be connected after reconnect")
	}

	if manager.ShouldReconnect() {
		t.Error("Should not reconnect after successful reconnect")
	}
}

// TestWebSocketReconnectManager_StateSync 测试状态同步
func TestWebSocketReconnectManager_StateSync(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	// 设置状态
	manager.SetState("playing")
	state := manager.GetState()
	if state != "playing" {
		t.Errorf("Expected state 'playing', got '%s'", state)
	}

	// 断开后状态应该保留
	manager.OnDisconnect()
	state = manager.GetState()
	if state != "playing" {
		t.Errorf("Expected state 'playing' after disconnect, got '%s'", state)
	}

	// 重连后状态应该同步
	manager.OnReconnect()
	state = manager.GetState()
	if state != "playing" {
		t.Errorf("Expected state 'playing' after reconnect, got '%s'", state)
	}
}

// TestWebSocketReconnectManager_GetReconnectCount 测试重连次数
func TestWebSocketReconnectManager_GetReconnectCount(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	manager.OnDisconnect()
	manager.IncrementReconnectCount()
	manager.IncrementReconnectCount()

	if manager.GetReconnectCount() != 2 {
		t.Errorf("Expected reconnect count 2, got %d", manager.GetReconnectCount())
	}

	manager.OnReconnect()
	if manager.GetReconnectCount() != 0 {
		t.Errorf("Expected reconnect count 0 after reconnect, got %d", manager.GetReconnectCount())
	}
}

// TestWebSocketReconnectManager_LastDisconnectTime 测试最后断开时间
func TestWebSocketReconnectManager_LastDisconnectTime(t *testing.T) {
	manager := NewWebSocketReconnectManager(5 * time.Second)

	before := time.Now()
	manager.OnDisconnect()
	after := time.Now()

	lastDisconnect := manager.GetLastDisconnectTime()
	if lastDisconnect.Before(before) || lastDisconnect.After(after) {
		t.Error("Last disconnect time should be between before and after")
	}
}
