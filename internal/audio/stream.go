package audio

import (
	"fmt"
	"io"
	"sync"
)

// StreamState 流状态
type StreamState int

const (
	StreamStateStopped StreamState = iota
	StreamStatePlaying
	StreamStatePaused
)

// RingBuffer 环形缓冲区
type RingBuffer struct {
	buffer []byte
	size   int
	head   int
	tail   int
	count  int
	mu     sync.Mutex
}

// NewRingBuffer 创建环形缓冲区
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]byte, size),
		size:   size,
	}
}

// Write 写入数据
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for i := 0; i < len(p) && rb.count < rb.size; i++ {
		rb.buffer[rb.tail] = p[i]
		rb.tail = (rb.tail + 1) % rb.size
		rb.count++
		n++
	}

	return n, nil
}

// Read 读取数据
func (rb *RingBuffer) Read(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return 0, io.EOF
	}

	for i := 0; i < len(p) && rb.count > 0; i++ {
		p[i] = rb.buffer[rb.head]
		rb.head = (rb.head + 1) % rb.size
		rb.count--
		n++
	}

	return n, nil
}

// Available 返回可读字节数
func (rb *RingBuffer) Available() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

// StreamManager 音频流管理器
type StreamManager struct {
	mu            sync.RWMutex
	currentSource io.Reader
	buffer        *RingBuffer
	state         StreamState
	stopChan      chan struct{}
	running       bool
}

// NewStreamManager 创建流管理器
func NewStreamManager() *StreamManager {
	return &StreamManager{
		buffer:   NewRingBuffer(8192),
		state:    StreamStateStopped,
		stopChan: make(chan struct{}),
	}
}

// SwitchSource 切换音频源
func (sm *StreamManager) SwitchSource(source io.Reader) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if source == nil {
		return fmt.Errorf("source cannot be nil")
	}

	// Stop current streaming
	if sm.running {
		select {
		case sm.stopChan <- struct{}{}:
		default:
		}
		sm.running = false
	}

	sm.currentSource = source
	return nil
}

// GetCurrentSource 获取当前音频源
func (sm *StreamManager) GetCurrentSource() io.Reader {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentSource
}

// Start 开始流传输
func (sm *StreamManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.currentSource == nil {
		return fmt.Errorf("no audio source set")
	}

	if sm.running {
		return nil
	}

	sm.state = StreamStatePlaying
	sm.running = true
	sm.stopChan = make(chan struct{})

	// Start streaming goroutine
	go sm.streamLoop()

	return nil
}

// Stop 停止流传输
func (sm *StreamManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	select {
	case sm.stopChan <- struct{}{}:
	default:
	}

	sm.running = false
	sm.state = StreamStateStopped

	return nil
}

// Read 从流中读取数据
func (sm *StreamManager) Read(p []byte) (n int, err error) {
	return sm.buffer.Read(p)
}

// GetState 获取流状态
func (sm *StreamManager) GetState() StreamState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// streamLoop 流传输循环
func (sm *StreamManager) streamLoop() {
	buf := make([]byte, 1024)

	for {
		select {
		case <-sm.stopChan:
			return
		default:
			sm.mu.RLock()
			source := sm.currentSource
			sm.mu.RUnlock()

			if source == nil {
				return
			}

			n, err := source.Read(buf)
			if err != nil {
				if err == io.EOF {
					sm.mu.Lock()
					sm.state = StreamStateStopped
					sm.running = false
					sm.mu.Unlock()
				}
				return
			}

			if n > 0 {
				sm.buffer.Write(buf[:n])
			}
		}
	}
}
