package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Manager PiFmRds 进程管理器接口
type Manager interface {
	Start(frequency float64, audioSource io.Reader) error
	Stop() error
	Restart(frequency float64) error
	IsRunning() bool
	GetStatus() ProcessStatus
	CleanupOrphans() error
	ValidateGPIO() error
}

// ProcessStatus 进程状态
type ProcessStatus struct {
	Running   bool
	PID       int
	Frequency float64
	StartTime time.Time
}

// manager 进程管理器实现
type manager struct {
	binaryPath  string
	cmd         *exec.Cmd
	audioSource io.Reader
	status      ProcessStatus
	mu          sync.RWMutex
}

// NewManager 创建新的进程管理器
func NewManager(binaryPath string) Manager {
	return &manager{
		binaryPath: binaryPath,
		status: ProcessStatus{
			Running: false,
		},
	}
}

// Start 启动 PiFmRds 进程
func (m *manager) Start(frequency float64, audioSource io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证参数
	if frequency < 87.5 || frequency > 108.0 {
		return fmt.Errorf("invalid frequency: %.1f (must be 87.5-108.0 MHz)", frequency)
	}

	if audioSource == nil {
		return fmt.Errorf("audio source cannot be nil")
	}

	// 检查是否已经在运行
	if m.status.Running {
		return fmt.Errorf("process already running")
	}

	// 验证二进制文件存在
	if _, err := os.Stat(m.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("pi_fm_rds binary not found at %s", m.binaryPath)
	}

	// 创建命令
	m.cmd = exec.Command("sudo", m.binaryPath, "-freq", fmt.Sprintf("%.1f", frequency), "-audio", "-")
	m.cmd.Stdin = audioSource
	m.audioSource = audioSource

	// 启动进程
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// 更新状态
	m.status = ProcessStatus{
		Running:   true,
		PID:       m.cmd.Process.Pid,
		Frequency: frequency,
		StartTime: time.Now(),
	}

	// 启动监控 goroutine
	go m.monitor()

	return nil
}

// Stop 停止 PiFmRds 进程
func (m *manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.status.Running || m.cmd == nil || m.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	// 发送 SIGTERM 信号优雅停止
	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// 等待进程退出，最多 5 秒
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case <-done:
		// 进程已退出
	case <-time.After(5 * time.Second):
		// 超时，强制终止
		if err := m.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		<-done // 等待 Wait() 完成
	}

	// 更新状态
	m.status.Running = false
	m.status.PID = 0
	m.cmd = nil

	return nil
}

// Restart 重启 PiFmRds 进程
func (m *manager) Restart(frequency float64) error {
	// 停止当前进程
	if m.IsRunning() {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	// 启动新进程（使用原来的 audioSource）
	if m.audioSource == nil {
		return fmt.Errorf("no audio source available for restart")
	}

	return m.Start(frequency, m.audioSource)
}

// IsRunning 检查进程是否在运行
func (m *manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.Running
}

// GetStatus 获取进程状态
func (m *manager) GetStatus() ProcessStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// CleanupOrphans 清理孤儿进程
func (m *manager) CleanupOrphans() error {
	// 查找所有 pi_fm_rds 进程
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}

	// 解析输出，查找 pi_fm_rds 进程
	lines := string(output)
	var orphanPIDs []int

	for _, line := range splitLines(lines) {
		if contains(line, "pi_fm_rds") && !contains(line, "grep") {
			// 提取 PID（第二列）
			fields := splitFields(line)
			if len(fields) >= 2 {
				var pid int
				if _, err := fmt.Sscanf(fields[1], "%d", &pid); err == nil {
					// 检查是否是当前管理的进程
					m.mu.RLock()
					currentPID := m.status.PID
					m.mu.RUnlock()

					if pid != currentPID {
						orphanPIDs = append(orphanPIDs, pid)
					}
				}
			}
		}
	}

	// 终止孤儿进程
	for _, pid := range orphanPIDs {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			// 如果 SIGTERM 失败，尝试 SIGKILL
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}

	return nil
}

// ValidateGPIO 验证 GPIO 4 可用性
func (m *manager) ValidateGPIO() error {
	// 检查 GPIO 4 是否可用
	gpioPath := "/sys/class/gpio/gpio4"
	if _, err := os.Stat(gpioPath); os.IsNotExist(err) {
		// 尝试导出 GPIO 4
		exportPath := "/sys/class/gpio/export"
		if err := os.WriteFile(exportPath, []byte("4"), 0644); err != nil {
			return fmt.Errorf("GPIO 4 not available and cannot be exported: %w", err)
		}
	}

	// 检查 GPIO 4 方向
	directionPath := gpioPath + "/direction"
	direction, err := os.ReadFile(directionPath)
	if err != nil {
		return fmt.Errorf("failed to read GPIO 4 direction: %w", err)
	}

	if string(direction) != "out\n" {
		return fmt.Errorf("GPIO 4 is not configured as output")
	}

	return nil
}

// monitor 监控进程状态
func (m *manager) monitor() {
	if m.cmd == nil {
		return
	}

	// 等待进程退出
	_ = m.cmd.Wait()

	// 更新状态
	m.mu.Lock()
	m.status.Running = false
	m.status.PID = 0
	m.mu.Unlock()
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitFields(s string) []string {
	var fields []string
	var current string
	inSpace := true

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			if !inSpace && current != "" {
				fields = append(fields, current)
				current = ""
			}
			inSpace = true
		} else {
			current += string(s[i])
			inSpace = false
		}
	}

	if current != "" {
		fields = append(fields, current)
	}

	return fields
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
