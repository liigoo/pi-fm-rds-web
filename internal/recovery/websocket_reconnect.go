package recovery

import (
	"sync"
	"time"
)

// WebSocketReconnectManager WebSocket 重连管理器
type WebSocketReconnectManager struct {
	reconnectTimeout   time.Duration
	disconnected       bool
	reconnectCount     int
	state              string
	lastDisconnectTime time.Time
	mu                 sync.RWMutex
}

// NewWebSocketReconnectManager 创建 WebSocket 重连管理器
func NewWebSocketReconnectManager(reconnectTimeout time.Duration) *WebSocketReconnectManager {
	return &WebSocketReconnectManager{
		reconnectTimeout: reconnectTimeout,
		disconnected:     false,
		reconnectCount:   0,
		state:            "",
	}
}

// OnDisconnect 处理断开事件
func (m *WebSocketReconnectManager) OnDisconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnected = true
	m.lastDisconnectTime = time.Now()
}

// OnReconnect 处理重连成功事件
func (m *WebSocketReconnectManager) OnReconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnected = false
	m.reconnectCount = 0
}

// IsDisconnected 检查是否断开
func (m *WebSocketReconnectManager) IsDisconnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disconnected
}

// ShouldReconnect 是否应该重连
func (m *WebSocketReconnectManager) ShouldReconnect() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disconnected
}

// GetReconnectTimeout 获取重连超时时间
func (m *WebSocketReconnectManager) GetReconnectTimeout() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reconnectTimeout
}

// IncrementReconnectCount 增加重连次数
func (m *WebSocketReconnectManager) IncrementReconnectCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconnectCount++
}

// GetReconnectCount 获取重连次数
func (m *WebSocketReconnectManager) GetReconnectCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reconnectCount
}

// SetState 设置状态
func (m *WebSocketReconnectManager) SetState(state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

// GetState 获取状态
func (m *WebSocketReconnectManager) GetState() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// GetLastDisconnectTime 获取最后断开时间
func (m *WebSocketReconnectManager) GetLastDisconnectTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastDisconnectTime
}
