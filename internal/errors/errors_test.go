package errors

import (
	"errors"
	"testing"
)

// TestNewAppError 测试创建错误
func TestNewAppError(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		message     string
		wantCode    int
		wantMessage string
	}{
		{
			name:        "创建配置加载错误",
			code:        ErrConfigLoad,
			message:     "配置文件不存在",
			wantCode:    1001,
			wantMessage: "配置文件不存在",
		},
		{
			name:        "创建进程启动错误",
			code:        ErrProcessStart,
			message:     "进程启动失败",
			wantCode:    2001,
			wantMessage: "进程启动失败",
		},
		{
			name:        "创建音频文件错误",
			code:        ErrAudioFileNotFound,
			message:     "音频文件未找到",
			wantCode:    3001,
			wantMessage: "音频文件未找到",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.message)
			if err == nil {
				t.Fatal("期望得到错误，但得到 nil")
			}

			appErr, ok := err.(*AppError)
			if !ok {
				t.Fatal("期望得到 *AppError 类型")
			}

			if appErr.Code != tt.wantCode {
				t.Errorf("错误代码 = %d, 期望 %d", appErr.Code, tt.wantCode)
			}

			if appErr.Message != tt.wantMessage {
				t.Errorf("错误消息 = %s, 期望 %s", appErr.Message, tt.wantMessage)
			}
		})
	}
}

// TestErrorWrapping 测试错误包装
func TestErrorWrapping(t *testing.T) {
	originalErr := errors.New("原始错误")

	wrappedErr := Wrap(ErrConfigLoad, "配置加载失败", originalErr)

	if wrappedErr == nil {
		t.Fatal("期望得到错误，但得到 nil")
	}

	appErr, ok := wrappedErr.(*AppError)
	if !ok {
		t.Fatal("期望得到 *AppError 类型")
	}

	if appErr.Code != ErrConfigLoad {
		t.Errorf("错误代码 = %d, 期望 %d", appErr.Code, ErrConfigLoad)
	}

	if appErr.Err != originalErr {
		t.Error("包装的错误不匹配")
	}
}

// TestErrorUnwrapping 测试错误解包
func TestErrorUnwrapping(t *testing.T) {
	originalErr := errors.New("底层错误")
	wrappedErr := Wrap(ErrProcessStart, "进程启动失败", originalErr)

	unwrapped := errors.Unwrap(wrappedErr)
	if unwrapped != originalErr {
		t.Error("解包后的错误不匹配原始错误")
	}

	// 测试多层包装
	doubleWrapped := Wrap(ErrProcessCrash, "进程崩溃", wrappedErr)
	unwrapped2 := errors.Unwrap(doubleWrapped)
	if unwrapped2 != wrappedErr {
		t.Error("多层包装解包失败")
	}
}

// TestChineseMessages 测试中文错误消息
func TestChineseMessages(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		wantMsg  string
	}{
		{"配置加载错误", ErrConfigLoad, "配置加载失败"},
		{"配置验证错误", ErrConfigValidation, "配置验证失败"},
		{"依赖缺失错误", ErrDependencyMissing, "依赖项缺失"},
		{"进程启动错误", ErrProcessStart, "进程启动失败"},
		{"进程停止错误", ErrProcessStop, "进程停止失败"},
		{"进程崩溃错误", ErrProcessCrash, "进程意外崩溃"},
		{"GPIO占用错误", ErrGPIOBusy, "GPIO引脚被占用"},
		{"音频文件未找到", ErrAudioFileNotFound, "音频文件未找到"},
		{"音频格式无效", ErrAudioFormatInvalid, "音频格式无效"},
		{"音频转码失败", ErrAudioTranscodeFailed, "音频转码失败"},
		{"麦克风未找到", ErrMicrophoneNotFound, "麦克风设备未找到"},
		{"麦克风断开", ErrMicrophoneDisconnected, "麦克风设备已断开"},
		{"存储配额超限", ErrStorageQuotaExceeded, "存储配额已超限"},
		{"文件大小超限", ErrFileSizeExceeded, "文件大小超过限制"},
		{"文件上传失败", ErrFileUploadFailed, "文件上传失败"},
		{"WebSocket客户端数超限", ErrWebSocketMaxClients, "WebSocket客户端数量已达上限"},
		{"WebSocket断开", ErrWebSocketDisconnected, "WebSocket连接已断开"},
		{"播放列表已满", ErrPlaylistFull, "播放列表已满"},
		{"播放列表为空", ErrPlaylistEmpty, "播放列表为空"},
		{"播放列表未找到", ErrPlaylistNotFound, "播放列表未找到"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetMessage(tt.code)
			if msg != tt.wantMsg {
				t.Errorf("GetMessage(%d) = %s, 期望 %s", tt.code, msg, tt.wantMsg)
			}
		})
	}

	// 测试未知错误代码
	t.Run("未知错误代码", func(t *testing.T) {
		msg := GetMessage(9999)
		if msg != "未知错误" {
			t.Errorf("GetMessage(9999) = %s, 期望 '未知错误'", msg)
		}
	})
}

// TestErrorError 测试 Error() 方法
func TestErrorError(t *testing.T) {
	err := New(ErrConfigLoad, "配置文件不存在")
	expected := "[1001] 配置文件不存在"
	if err.Error() != expected {
		t.Errorf("Error() = %s, 期望 %s", err.Error(), expected)
	}

	// 测试包装错误的 Error() 输出
	originalErr := errors.New("底层错误")
	wrappedErr := Wrap(ErrProcessStart, "进程启动失败", originalErr)
	expectedWrapped := "[2001] 进程启动失败: 底层错误"
	if wrappedErr.Error() != expectedWrapped {
		t.Errorf("Error() = %s, 期望 %s", wrappedErr.Error(), expectedWrapped)
	}
}

// TestErrorCodes 测试所有错误代码常量
func TestErrorCodes(t *testing.T) {
	codes := map[string]int{
		"ErrConfigLoad":              1001,
		"ErrConfigValidation":        1002,
		"ErrDependencyMissing":       1003,
		"ErrProcessStart":            2001,
		"ErrProcessStop":             2002,
		"ErrProcessCrash":            2003,
		"ErrGPIOBusy":                2004,
		"ErrAudioFileNotFound":       3001,
		"ErrAudioFormatInvalid":      3002,
		"ErrAudioTranscodeFailed":    3003,
		"ErrMicrophoneNotFound":      3004,
		"ErrMicrophoneDisconnected":  3005,
		"ErrStorageQuotaExceeded":    4001,
		"ErrFileSizeExceeded":        4002,
		"ErrFileUploadFailed":        4003,
		"ErrWebSocketMaxClients":     5001,
		"ErrWebSocketDisconnected":   5002,
		"ErrPlaylistFull":            6001,
		"ErrPlaylistEmpty":           6002,
		"ErrPlaylistNotFound":        6003,
	}

	// 验证错误代码数量
	if len(codes) < 20 {
		t.Errorf("错误代码数量 = %d, 期望至少 20 个", len(codes))
	}
}
