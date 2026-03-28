package audio

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestRingBuffer_WriteRead(t *testing.T) {
	rb := NewRingBuffer(1024)

	// Test write and read
	data := []byte("Hello, World!")
	n, err := rb.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(data))
	}

	// Read back
	buf := make([]byte, len(data))
	n, err = rb.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Read() read %d bytes, want %d", n, len(data))
	}
	if !bytes.Equal(buf, data) {
		t.Errorf("Read() = %v, want %v", buf, data)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(10)

	// Write more than capacity
	data := make([]byte, 20)
	for i := range data {
		data[i] = byte(i)
	}

	n, err := rb.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Should only write up to capacity
	if n > 10 {
		t.Errorf("Write() wrote %d bytes, expected <= 10", n)
	}
}

func TestRingBuffer_Available(t *testing.T) {
	rb := NewRingBuffer(100)

	if rb.Available() != 0 {
		t.Errorf("Available() = %d, want 0", rb.Available())
	}

	data := []byte("test")
	rb.Write(data)

	if rb.Available() != len(data) {
		t.Errorf("Available() = %d, want %d", rb.Available(), len(data))
	}
}

func TestStreamManager_SwitchSource(t *testing.T) {
	sm := NewStreamManager()

	// Create test sources
	source1 := bytes.NewReader([]byte("source1"))
	source2 := bytes.NewReader([]byte("source2"))

	// Switch to source1
	err := sm.SwitchSource(source1)
	if err != nil {
		t.Fatalf("SwitchSource() error = %v", err)
	}

	// Verify current source
	if sm.GetCurrentSource() != source1 {
		t.Error("Current source mismatch")
	}

	// Switch to source2
	err = sm.SwitchSource(source2)
	if err != nil {
		t.Fatalf("SwitchSource() error = %v", err)
	}

	if sm.GetCurrentSource() != source2 {
		t.Error("Current source mismatch after switch")
	}
}

func TestStreamManager_GetState(t *testing.T) {
	sm := NewStreamManager()

	// Initial state
	state := sm.GetState()
	if state != StreamStateStopped {
		t.Errorf("Initial state = %v, want %v", state, StreamStateStopped)
	}

	// Start streaming
	source := bytes.NewReader([]byte("test"))
	sm.SwitchSource(source)
	sm.Start()

	state = sm.GetState()
	if state != StreamStatePlaying {
		t.Errorf("State after Start() = %v, want %v", state, StreamStatePlaying)
	}

	// Stop streaming
	sm.Stop()
	state = sm.GetState()
	if state != StreamStateStopped {
		t.Errorf("State after Stop() = %v, want %v", state, StreamStateStopped)
	}
}

func TestStreamManager_BufferManagement(t *testing.T) {
	sm := NewStreamManager()

	// Create a large data source
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	source := bytes.NewReader(data)

	err := sm.SwitchSource(source)
	if err != nil {
		t.Fatalf("SwitchSource() error = %v", err)
	}

	sm.Start()
	defer sm.Stop()

	// Read from stream
	buf := make([]byte, 1024)
	totalRead := 0

	for i := 0; i < 5; i++ {
		n, err := sm.Read(buf)
		if err != nil && err != io.EOF {
			t.Fatalf("Read() error = %v", err)
		}
		totalRead += n
		time.Sleep(10 * time.Millisecond)
	}

	if totalRead == 0 {
		t.Error("No data read from stream")
	}
}

func TestStreamManager_ConcurrentAccess(t *testing.T) {
	sm := NewStreamManager()

	source := bytes.NewReader(make([]byte, 10000))
	sm.SwitchSource(source)
	sm.Start()
	defer sm.Stop()

	// Concurrent reads
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			buf := make([]byte, 100)
			for j := 0; j < 10; j++ {
				sm.Read(buf)
				time.Sleep(5 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
