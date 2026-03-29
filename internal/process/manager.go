package process

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
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

	audioArg := "-"
	audioInput := audioSource
	if fileSource, ok := audioSource.(interface{ Path() string }); ok {
		if p := fileSource.Path(); p != "" {
			audioArg = p
			audioInput = nil
		}
	}

	// 创建并启动命令（优先 sudo；受限环境回退直启）
	cmd, err := m.startCommand(frequency, audioArg, audioInput)
	if err != nil {
		return err
	}
	m.cmd = cmd
	m.audioSource = audioSource

	// 更新状态
	m.status = ProcessStatus{
		Running:   true,
		PID:       m.cmd.Process.Pid,
		Frequency: frequency,
		StartTime: time.Now(),
	}

	// 启动监控 goroutine（绑定当前命令，避免旧进程退出覆盖新状态）
	go m.monitor(cmd)

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
	if err := m.signalManagedProcess(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	pid := m.cmd.Process.Pid
	if !waitForProcessExit(pid, 5*time.Second) {
		// 超时，强制终止
		if err := m.signalManagedProcess(syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		if !waitForProcessExit(pid, 2*time.Second) {
			return fmt.Errorf("process did not exit after SIGKILL")
		}
	}

	// 更新状态
	m.status.Running = false
	m.status.PID = 0
	m.cmd = nil

	return nil
}

func (m *manager) signalManagedProcess(sig syscall.Signal) error {
	if m.cmd == nil || m.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	if err := m.cmd.Process.Signal(sig); err == nil || errors.Is(err, os.ErrProcessDone) {
		return nil
	} else if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
		return err
	}

	// 受权限限制时，尝试通过 sudo 发送信号（Raspberry Pi 生产环境）
	pidNum := m.cmd.Process.Pid
	pid := strconv.Itoa(pidNum)
	sigName := signalName(sig)
	_ = exec.Command("sudo", "-n", "pkill", "-"+sigName, "-P", pid).Run()
	if err := exec.Command("sudo", "-n", "kill", "-"+sigName, pid).Run(); err != nil {
		// 进程已在前一步退出时，kill 可能返回非零；视为成功
		if !processAlive(pidNum) {
			return nil
		}
		return err
	}
	return nil
}

func signalName(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGKILL:
		return "KILL"
	default:
		return "TERM"
	}
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !processAlive(pid) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}

	// EPERM 表示进程存在但当前用户无权限发送信号
	if errors.Is(err, syscall.EPERM) {
		return true
	}

	return !errors.Is(err, syscall.ESRCH)
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
		// 受限沙箱/容器环境中可能禁止执行 ps，清理操作降级为 no-op
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			return nil
		}
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

func (m *manager) startCommand(frequency float64, audioArg string, audioSource io.Reader) (*exec.Cmd, error) {
	args := []string{"-freq", fmt.Sprintf("%.1f", frequency), "-audio", audioArg}

	sudoCmd := exec.Command("sudo", append([]string{m.binaryPath}, args...)...)
	if audioSource != nil {
		sudoCmd.Stdin = audioSource
	}
	if err := sudoCmd.Start(); err == nil {
		return sudoCmd, nil
	} else if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) && !errors.Is(err, exec.ErrNotFound) {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// 回退：在无 sudo 或被策略拦截时，直接执行目标二进制（便于测试/受限环境运行）
	directCmd := exec.Command(m.binaryPath, args...)
	if audioSource != nil {
		directCmd.Stdin = audioSource
	}
	if err := directCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}
	return directCmd, nil
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
func (m *manager) monitor(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}

	// 等待进程退出
	_ = cmd.Wait()

	// 仅在当前管理对象仍指向该命令时更新状态，防止旧进程覆盖新进程状态
	m.mu.Lock()
	if m.cmd == cmd {
		m.status.Running = false
		m.status.PID = 0
		m.cmd = nil
	}
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
