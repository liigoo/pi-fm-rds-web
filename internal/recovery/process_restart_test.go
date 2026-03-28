package recovery

import (
	"testing"
	"time"
)

// TestProcessRestartManager_CrashDetection 测试进程崩溃检测
func TestProcessRestartManager_CrashDetection(t *testing.T) {
	manager := NewProcessRestartManager(3, time.Second)

	// 模拟进程崩溃
	crashed := manager.OnProcessCrash()
	if !crashed {
		t.Error("Expected crash detection to return true")
	}

	// 验证重启次数
	if manager.GetRestartCount() != 1 {
		t.Errorf("Expected restart count 1, got %d", manager.GetRestartCount())
	}
}

// TestProcessRestartManager_ExponentialBackoff 测试指数退避
func TestProcessRestartManager_ExponentialBackoff(t *testing.T) {
	manager := NewProcessRestartManager(3, time.Second)

	// 第一次重启：1秒
	delay1 := manager.GetNextRestartDelay()
	if delay1 != time.Second {
		t.Errorf("Expected 1s delay, got %v", delay1)
	}

	manager.OnProcessCrash()

	// 第二次重启：2秒
	delay2 := manager.GetNextRestartDelay()
	if delay2 != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", delay2)
	}

	manager.OnProcessCrash()

	// 第三次重启：4秒
	delay3 := manager.GetNextRestartDelay()
	if delay3 != 4*time.Second {
		t.Errorf("Expected 4s delay, got %v", delay3)
	}
}

// TestProcessRestartManager_MaxRetries 测试最大重试次数
func TestProcessRestartManager_MaxRetries(t *testing.T) {
	manager := NewProcessRestartManager(3, time.Second)

	// 模拟3次崩溃
	for i := 0; i < 3; i++ {
		if !manager.OnProcessCrash() {
			t.Errorf("Crash %d should be allowed", i+1)
		}
	}

	// 第4次应该被拒绝
	if manager.OnProcessCrash() {
		t.Error("Expected 4th crash to be rejected")
	}

	if !manager.ShouldGiveUp() {
		t.Error("Expected manager to give up after max retries")
	}
}

// TestProcessRestartManager_Reset 测试重置
func TestProcessRestartManager_Reset(t *testing.T) {
	manager := NewProcessRestartManager(3, time.Second)

	manager.OnProcessCrash()
	manager.OnProcessCrash()

	if manager.GetRestartCount() != 2 {
		t.Errorf("Expected restart count 2, got %d", manager.GetRestartCount())
	}

	manager.Reset()

	if manager.GetRestartCount() != 0 {
		t.Errorf("Expected restart count 0 after reset, got %d", manager.GetRestartCount())
	}
}
