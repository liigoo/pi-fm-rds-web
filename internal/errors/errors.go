package errors

import "fmt"

// 错误代码常量

// 系统错误 (1xxx)
const (
	ErrConfigLoad       = 1001
	ErrConfigValidation = 1002
	ErrDependencyMissing = 1003
)

// 进程管理错误 (2xxx)
const (
	ErrProcessStart = 2001
	ErrProcessStop  = 2002
	ErrProcessCrash = 2003
	ErrGPIOBusy     = 2004
)

// 音频错误 (3xxx)
const (
	ErrAudioFileNotFound      = 3001
	ErrAudioFormatInvalid     = 3002
	ErrAudioTranscodeFailed   = 3003
	ErrMicrophoneNotFound     = 3004
	ErrMicrophoneDisconnected = 3005
)

// 存储错误 (4xxx)
const (
	ErrStorageQuotaExceeded = 4001
	ErrFileSizeExceeded     = 4002
	ErrFileUploadFailed     = 4003
)

// WebSocket 错误 (5xxx)
const (
	ErrWebSocketMaxClients    = 5001
	ErrWebSocketDisconnected  = 5002
)

// 播放列表错误 (6xxx)
const (
	ErrPlaylistFull     = 6001
	ErrPlaylistEmpty    = 6002
	ErrPlaylistNotFound = 6003
)

// AppError 应用错误结构体
type AppError struct {
	Code    int
	Message string
	Err     error
}

// 中文错误消息映射
var chineseMessages = map[int]string{
	// 系统错误
	ErrConfigLoad:             "配置加载失败",
	ErrConfigValidation:       "配置验证失败",
	ErrDependencyMissing:      "依赖项缺失",

	// 进程管理错误
	ErrProcessStart:           "进程启动失败",
	ErrProcessStop:            "进程停止失败",
	ErrProcessCrash:           "进程意外崩溃",
	ErrGPIOBusy:               "GPIO引脚被占用",

	// 音频错误
	ErrAudioFileNotFound:      "音频文件未找到",
	ErrAudioFormatInvalid:     "音频格式无效",
	ErrAudioTranscodeFailed:   "音频转码失败",
	ErrMicrophoneNotFound:     "麦克风设备未找到",
	ErrMicrophoneDisconnected: "麦克风设备已断开",

	// 存储错误
	ErrStorageQuotaExceeded:   "存储配额已超限",
	ErrFileSizeExceeded:        "文件大小超过限制",
	ErrFileUploadFailed:        "文件上传失败",

	// WebSocket 错误
	ErrWebSocketMaxClients:     "WebSocket客户端数量已达上限",
	ErrWebSocketDisconnected:   "WebSocket连接已断开",

	// 播放列表错误
	ErrPlaylistFull:            "播放列表已满",
	ErrPlaylistEmpty:           "播放列表为空",
	ErrPlaylistNotFound:        "播放列表未找到",
}

// New 创建新的应用错误
func New(code int, message string) error {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     nil,
	}
}

// Wrap 包装错误，添加错误代码和消息
func Wrap(code int, message string, err error) error {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口，支持错误链
func (e *AppError) Unwrap() error {
	return e.Err
}

// GetMessage 根据错误代码获取中文错误消息
func GetMessage(code int) string {
	if msg, ok := chineseMessages[code]; ok {
		return msg
	}
	return "未知错误"
}
