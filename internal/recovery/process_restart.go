package recovery

import (
	"sync"
	"time"
)

// ProcessRestartManager 进程自动重启管理器
type ProcessRestartManager struct {
	maxRetries   int
	baseDelay    time.Duration
	restartCount int
	mu           sync.RWMutex
}

// NewProcessRestartManager 创建进程重启管理器
func NewProcessRestartManager(maxRetries int, baseDelay time.Duration) *ProcessRestartManager {
	return &ProcessRestartManager{
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// OnProcessCrash 处理进程崩溃，返回是否应该重启
func (m *ProcessRestartManager) OnProcessCrash() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.restartCount >= m.maxRetries {
		return false
	}

	m.restartCount++
	return true
}

// GetNextRestartDelay 获取下次重启延迟（指数退避）
func (m *ProcessRestartManager) GetNextRestartDelay() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 指数退避: 1s, 2s, 4s
	multiplier := 1 << m.restartCount
	return m.baseDelay * time.Duration(multiplier)
}

// GetRestartCount 获取重启次数
func (m *ProcessRestartManager) GetRestartCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.restartCount
}

// ShouldGiveUp 是否应该放弃重启
func (m *ProcessRestartManager) ShouldGiveUp() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.restartCount >= m.maxRetries
}

// Reset 重置重启计数器
func (m *ProcessRestartManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartCount = 0
}
