package audio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// SourceType 音频源类型
type SourceType int

const (
	SourceTypeFile SourceType = iota
	SourceTypeMicrophone
)

// Config 音频配置
type Config struct {
	SampleRate int
	Channels   int
}

// AudioSource 音频源接口
type AudioSource interface {
	Read(p []byte) (n int, err error)
	Close() error
}

// FileSource 文件音频源
type FileSource struct {
	file *os.File
	path string
	loop bool
}

func NewFileSource(path string) (*FileSource, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	return &FileSource{file: file, path: path, loop: true}, nil
}

func (f *FileSource) Read(p []byte) (n int, err error) {
	n, err = f.file.Read(p)
	if err == io.EOF && f.loop {
		if _, seekErr := f.file.Seek(0, 0); seekErr != nil {
			return n, err
		}
		if n > 0 {
			return n, nil
		}
		return f.file.Read(p)
	}
	return n, err
}

func (f *FileSource) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// Path 返回音频文件路径
func (f *FileSource) Path() string {
	return f.path
}

// MicrophoneSource 麦克风音频源（模拟实现）
type MicrophoneSource struct {
	deviceID string
	buffer   *bytes.Buffer
	mu       sync.Mutex
}

func NewMicrophoneSource(deviceID string) (*MicrophoneSource, error) {
	// 模拟麦克风数据
	buffer := bytes.NewBuffer(make([]byte, 4096))
	return &MicrophoneSource{
		deviceID: deviceID,
		buffer:   buffer,
	}, nil
}

func (m *MicrophoneSource) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buffer.Read(p)
}

func (m *MicrophoneSource) Close() error {
	return nil
}

// Manager 音频管理器
type Manager struct {
	cfg           *Config
	mu            sync.Mutex
	currentSource AudioSource
	sourceType    SourceType
	audioStream   io.Reader
	spectrumChan  chan []int16
	stopChan      chan struct{}
	running       bool
}

// NewManager 创建音频管理器
func NewManager(cfg *Config) *Manager {
	return &Manager{
		cfg:          cfg,
		spectrumChan: make(chan []int16, 10),
		stopChan:     make(chan struct{}),
	}
}

// PlayFile 播放音频文件
func (m *Manager) PlayFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭当前音频源
	if m.currentSource != nil {
		m.currentSource.Close()
	}

	// 创建文件音频源
	source, err := NewFileSource(path)
	if err != nil {
		return err
	}

	m.currentSource = source
	m.sourceType = SourceTypeFile
	m.audioStream = source
	m.running = true

	// 启动频谱数据流
	go m.streamSpectrum()

	return nil
}

// PlayMicrophone 播放麦克风
func (m *Manager) PlayMicrophone(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭当前音频源
	if m.currentSource != nil {
		m.currentSource.Close()
	}

	// 创建麦克风音频源
	source, err := NewMicrophoneSource(deviceID)
	if err != nil {
		return err
	}

	m.currentSource = source
	m.sourceType = SourceTypeMicrophone
	m.audioStream = source
	m.running = true

	// 启动频谱数据流
	go m.streamSpectrum()

	return nil
}

// Stop 停止播放
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentSource != nil {
		m.currentSource.Close()
		m.currentSource = nil
	}

	m.audioStream = nil
	m.running = false

	// 发送停止信号
	select {
	case m.stopChan <- struct{}{}:
	default:
	}

	return nil
}

// SwitchSource 切换音频源
func (m *Manager) SwitchSource(newSource SourceType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if newSource == m.sourceType {
		return nil
	}

	// 关闭当前音频源
	if m.currentSource != nil {
		m.currentSource.Close()
	}

	// 根据新源类型创建音频源
	var source AudioSource
	var err error

	switch newSource {
	case SourceTypeFile:
		// 这里需要保存文件路径，简化实现使用模拟数据
		source = &FileSource{}
	case SourceTypeMicrophone:
		source, err = NewMicrophoneSource("default")
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown source type: %d", newSource)
	}

	m.currentSource = source
	m.sourceType = newSource
	m.audioStream = source

	return nil
}

// GetAudioStream 获取音频流
func (m *Manager) GetAudioStream() io.Reader {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.audioStream
}

// GetSpectrumStream 获取频谱数据流
func (m *Manager) GetSpectrumStream() <-chan []int16 {
	return m.spectrumChan
}

// streamSpectrum 频谱数据流（goroutine）
func (m *Manager) streamSpectrum() {
	buffer := make([]byte, 1024)
	spectrumData := make([]int16, 512)

	for {
		select {
		case <-m.stopChan:
			return
		default:
			// 读取音频数据（在锁内检查 running 状态）
			m.mu.Lock()
			if !m.running || m.currentSource == nil {
				m.mu.Unlock()
				return
			}
			n, err := m.currentSource.Read(buffer)
			m.mu.Unlock()

			if err != nil && err != io.EOF {
				return
			}

			if n > 0 {
				// 模拟频谱数据（简化实现）
				for i := range spectrumData {
					if i < n/2 {
						spectrumData[i] = int16(buffer[i*2]) | int16(buffer[i*2+1])<<8
					} else {
						spectrumData[i] = 0
					}
				}

				// 非阻塞发送频谱数据
				select {
				case m.spectrumChan <- spectrumData:
				default:
					// 通道满时丢弃数据
				}
			}
		}
	}
}
