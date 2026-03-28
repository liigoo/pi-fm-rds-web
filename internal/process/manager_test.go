package process

import (
	"bytes"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestProcessStart 测试进程启动
func TestProcessStart(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	tests := []struct {
		name        string
		binaryPath  string
		frequency   float64
		audioSource io.Reader
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid frequency and audio source",
			binaryPath:  mockBinary,
			frequency:   88.0,
			audioSource: bytes.NewReader([]byte("test audio data")),
			wantErr:     false,
		},
		{
			name:        "frequency too low",
			binaryPath:  mockBinary,
			frequency:   80.0,
			audioSource: bytes.NewReader([]byte("test audio data")),
			wantErr:     true,
			errContains: "frequency",
		},
		{
			name:        "frequency too high",
			binaryPath:  mockBinary,
			frequency:   110.0,
			audioSource: bytes.NewReader([]byte("test audio data")),
			wantErr:     true,
			errContains: "frequency",
		},
		{
			name:        "nil audio source",
			binaryPath:  mockBinary,
			frequency:   88.0,
			audioSource: nil,
			wantErr:     true,
			errContains: "audio source",
		},
		{
			name:        "invalid binary path",
			binaryPath:  "/nonexistent/pi_fm_rds",
			frequency:   88.0,
			audioSource: bytes.NewReader([]byte("test audio data")),
			wantErr:     true,
			errContains: "binary not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(tt.binaryPath)
			err := mgr.Start(tt.frequency, tt.audioSource)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Start() expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Start() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Start() unexpected error = %v", err)
				}
				// Cleanup
				if mgr.IsRunning() {
					_ = mgr.Stop()
				}
			}
		})
	}
}

// TestProcessStop 测试进程停止
func TestProcessStop(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Test stopping when not running
	err := mgr.Stop()
	if err == nil {
		t.Error("Stop() should return error when process not running")
	}

	// Start a process
	audioSource := bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Cannot start process for testing: %v", err)
	}

	// Stop the process
	err = mgr.Stop()
	if err != nil {
		t.Errorf("Stop() unexpected error = %v", err)
	}

	// Verify process is stopped
	if mgr.IsRunning() {
		t.Error("Process should not be running after Stop()")
	}
}

// TestProcessRestart 测试进程重启
func TestProcessRestart(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Start initial process
	audioSource := bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Cannot start process for testing: %v", err)
	}
	defer mgr.Stop()

	oldPID := mgr.GetStatus().PID

	// Restart with new frequency
	err := mgr.Restart(90.0)
	if err != nil {
		t.Errorf("Restart() unexpected error = %v", err)
	}

	// Verify new process is running
	if !mgr.IsRunning() {
		t.Error("Process should be running after Restart()")
	}

	newPID := mgr.GetStatus().PID
	if oldPID == newPID {
		t.Error("Restart() should create a new process with different PID")
	}

	status := mgr.GetStatus()
	if status.Frequency != 90.0 {
		t.Errorf("Restart() frequency = %.1f, want 90.0", status.Frequency)
	}
}

// TestProcessStatus 测试状态查询
func TestProcessStatus(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Status when not running
	status := mgr.GetStatus()
	if status.Running {
		t.Error("GetStatus() Running should be false when not started")
	}
	if status.PID != 0 {
		t.Errorf("GetStatus() PID = %d, want 0 when not running", status.PID)
	}

	// Start process
	audioSource := bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(88.5, audioSource); err != nil {
		t.Fatalf("Cannot start process for testing: %v", err)
	}
	defer mgr.Stop()

	// Status when running
	status = mgr.GetStatus()
	if !status.Running {
		t.Error("GetStatus() Running should be true after Start()")
	}
	if status.PID <= 0 {
		t.Errorf("GetStatus() PID = %d, want > 0 when running", status.PID)
	}
	if status.Frequency != 88.5 {
		t.Errorf("GetStatus() Frequency = %.1f, want 88.5", status.Frequency)
	}
	if status.StartTime.IsZero() {
		t.Error("GetStatus() StartTime should not be zero when running")
	}
}

// TestOrphanCleanup 测试孤儿进程清理
func TestOrphanCleanup(t *testing.T) {
	mgr := NewManager("/usr/bin/pi_fm_rds")

	// Test cleanup when no orphans exist
	err := mgr.CleanupOrphans()
	if err != nil {
		t.Errorf("CleanupOrphans() unexpected error = %v", err)
	}

	// Create a mock orphan process (this is hard to test without actually creating orphans)
	// In real scenarios, orphans would be detected via ps command
	// For now, we just verify the method doesn't crash
}

// TestGPIOValidation 测试 GPIO 验证
func TestGPIOValidation(t *testing.T) {
	mgr := NewManager("/usr/bin/pi_fm_rds")

	// Test GPIO validation
	err := mgr.ValidateGPIO()

	// On non-Raspberry Pi systems, this should return an error or skip
	// On Raspberry Pi, it should check GPIO 4 availability
	if err != nil {
		t.Logf("GPIO validation failed (expected on non-Pi systems): %v", err)
	}
}

// TestStopTimeout 测试停止超时处理
func TestStopTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Start a process
	audioSource := bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Cannot start process for testing: %v", err)
	}

	// Stop should complete within reasonable time (< 6 seconds for 5s timeout + buffer)
	done := make(chan error, 1)
	go func() {
		done <- mgr.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Stop() unexpected error = %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Error("Stop() took too long (should timeout and force kill after 5s)")
	}
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Start a process
	audioSource := bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Cannot start process for testing: %v", err)
	}
	defer mgr.Stop()

	// Try to start again (should fail)
	err := mgr.Start(90.0, audioSource)
	if err == nil {
		t.Error("Start() should fail when process already running")
	}

	// Multiple status checks should work
	for i := 0; i < 10; i++ {
		status := mgr.GetStatus()
		if !status.Running {
			t.Error("GetStatus() should show running")
		}
	}
}

// TestInvalidBinaryPath 测试无效的二进制路径
func TestInvalidBinaryPath(t *testing.T) {
	mgr := NewManager("/nonexistent/pi_fm_rds")

	audioSource := bytes.NewReader([]byte("test audio"))
	err := mgr.Start(88.0, audioSource)

	if err == nil {
		t.Error("Start() should fail with invalid binary path")
	}
}

// Mock helper for testing without actual pi_fm_rds binary
func createMockPiFmRds(t *testing.T) string {
	// Create a simple mock script that simulates pi_fm_rds
	mockScript := `#!/bin/bash
# Mock pi_fm_rds for testing
while true; do
	sleep 1
done
`
	tmpFile, err := os.CreateTemp("", "mock_pi_fm_rds_*.sh")
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	if _, err := tmpFile.WriteString(mockScript); err != nil {
		t.Fatalf("Failed to write mock script: %v", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatalf("Failed to chmod mock script: %v", err)
	}

	return tmpFile.Name()
}

// TestWithMockBinary 使用模拟二进制测试完整流程
func TestWithMockBinary(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Test full lifecycle
	audioSource := bytes.NewReader([]byte("test audio"))

	// Start
	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify running
	if !mgr.IsRunning() {
		t.Error("Process should be running")
	}

	status := mgr.GetStatus()
	if status.PID <= 0 {
		t.Error("PID should be positive")
	}

	// Stop
	if err := mgr.Stop(); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	// Verify stopped
	if mgr.IsRunning() {
		t.Error("Process should not be running after Stop()")
	}
}

// Helper to check if process exists
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// TestProcessMonitoring 测试进程监控
func TestProcessMonitoring(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)
	audioSource := bytes.NewReader([]byte("test audio"))

	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer mgr.Stop()

	pid := mgr.GetStatus().PID

	// Kill the process externally
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("Failed to kill process: %v", err)
	}

	// Wait a bit for monitoring to detect
	time.Sleep(100 * time.Millisecond)

	// Manager should detect process is no longer running
	if mgr.IsRunning() {
		t.Error("Manager should detect process has died")
	}
}

// TestRestartWithoutAudioSource 测试没有音频源的重启
func TestRestartWithoutAudioSource(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	// Try to restart without starting first
	err := mgr.Restart(88.0)
	if err == nil {
		t.Error("Restart() should fail when no audio source available")
	}
	if !strings.Contains(err.Error(), "audio source") {
		t.Errorf("Restart() error = %v, want error containing 'audio source'", err)
	}
}

// TestHelperFunctions 测试辅助函数
func TestHelperFunctions(t *testing.T) {
	// Test splitLines
	lines := splitLines("line1\nline2\nline3")
	if len(lines) != 3 {
		t.Errorf("splitLines() returned %d lines, want 3", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("splitLines() = %v, want [line1 line2 line3]", lines)
	}

	// Test splitFields
	fields := splitFields("field1  field2\tfield3")
	if len(fields) != 3 {
		t.Errorf("splitFields() returned %d fields, want 3", len(fields))
	}

	// Test contains
	if !contains("hello world", "world") {
		t.Error("contains() should return true for 'hello world' contains 'world'")
	}
	if contains("hello", "world") {
		t.Error("contains() should return false for 'hello' contains 'world'")
	}

	// Test findSubstring
	if findSubstring("hello world", "world") != 6 {
		t.Error("findSubstring() should return 6 for 'hello world' find 'world'")
	}
	if findSubstring("hello", "world") != -1 {
		t.Error("findSubstring() should return -1 for 'hello' find 'world'")
	}
}

// TestCleanupOrphansWithRunningProcess 测试清理孤儿进程（有运行进程）
func TestCleanupOrphansWithRunningProcess(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)
	audioSource := bytes.NewReader([]byte("test audio"))

	// Start a process
	if err := mgr.Start(88.0, audioSource); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer mgr.Stop()

	// Cleanup should not kill the managed process
	if err := mgr.CleanupOrphans(); err != nil {
		t.Errorf("CleanupOrphans() unexpected error = %v", err)
	}

	// Managed process should still be running
	if !mgr.IsRunning() {
		t.Error("Managed process should still be running after CleanupOrphans()")
	}
}

// TestMultipleStartStop 测试多次启动停止
func TestMultipleStartStop(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)

	for i := 0; i < 3; i++ {
		audioSource := bytes.NewReader([]byte("test audio"))
		if err := mgr.Start(88.0, audioSource); err != nil {
			t.Fatalf("Start() iteration %d failed: %v", i, err)
		}

		if !mgr.IsRunning() {
			t.Errorf("Process should be running after Start() iteration %d", i)
		}

		if err := mgr.Stop(); err != nil {
			t.Fatalf("Stop() iteration %d failed: %v", i, err)
		}

		if mgr.IsRunning() {
			t.Errorf("Process should not be running after Stop() iteration %d", i)
		}
	}
}

// TestFrequencyBoundaries 测试频率边界值
func TestFrequencyBoundaries(t *testing.T) {
	mockBinary := createMockPiFmRds(t)
	defer os.Remove(mockBinary)

	mgr := NewManager(mockBinary)
	audioSource := bytes.NewReader([]byte("test audio"))

	// Test minimum valid frequency
	if err := mgr.Start(87.5, audioSource); err != nil {
		t.Errorf("Start() with frequency 87.5 should succeed: %v", err)
	}
	mgr.Stop()

	// Test maximum valid frequency
	audioSource = bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(108.0, audioSource); err != nil {
		t.Errorf("Start() with frequency 108.0 should succeed: %v", err)
	}
	mgr.Stop()

	// Test just below minimum
	audioSource = bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(87.4, audioSource); err == nil {
		t.Error("Start() with frequency 87.4 should fail")
		mgr.Stop()
	}

	// Test just above maximum
	audioSource = bytes.NewReader([]byte("test audio"))
	if err := mgr.Start(108.1, audioSource); err == nil {
		t.Error("Start() with frequency 108.1 should fail")
		mgr.Stop()
	}
}
