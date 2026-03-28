package recovery

import (
	"sync"
	"time"
)

// MicrophoneReconnectManager 麦克风重连管理器
type MicrophoneReconnectManager struct {
	retryInterval time.Duration
	disconnected  bool
	retryCount    int
	mu            sync.RWMutex
}

// NewMicrophoneReconnectManager 创建麦克风重连管理器
func NewMicrophoneReconnectManager(retryInterval time.Duration) *MicrophoneReconnectManager {
	return &MicrophoneReconnectManager{
		retryInterval: retryInterval,
		disconnected:  false,
		retryCount:    0,
	}
}

// OnDisconnect 处理断开事件
func (m *MicrophoneReconnectManager) OnDisconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnected = true
}

// OnReconnect 处理重连成功事件
func (m *MicrophoneReconnectManager) OnReconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnected = false
	m.retryCount = 0
}

// IsDisconnected 检查是否断开
func (m *MicrophoneReconnectManager) IsDisconnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disconnected
}

// ShouldRetry 是否应该重试
func (m *MicrophoneReconnectManager) ShouldRetry() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disconnected
}

// GetRetryInterval 获取重试间隔
func (m *MicrophoneReconnectManager) GetRetryInterval() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.retryInterval
}

// IncrementRetryCount 增加重试次数
func (m *MicrophoneReconnectManager) IncrementRetryCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retryCount++
}

// GetRetryCount 获取重试次数
func (m *MicrophoneReconnectManager) GetRetryCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.retryCount
}
