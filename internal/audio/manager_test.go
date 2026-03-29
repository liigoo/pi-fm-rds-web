package audio

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func createTestWAVFile(t *testing.T, dir string) string {
	testFile := filepath.Join(dir, "test.wav")
	wavData := make([]byte, 1024)
	copy(wavData[0:4], []byte("RIFF"))
	copy(wavData[8:12], []byte("WAVE"))
	if err := os.WriteFile(testFile, wavData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return testFile
}

func TestPlayFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createTestWAVFile(t, tmpDir)
	cfg := &Config{SampleRate: 48000, Channels: 2}
	mgr := NewManager(cfg)
	defer mgr.Stop()

	if err := mgr.PlayFile(testFile); err != nil {
		t.Errorf("PlayFile() error = %v", err)
	}
	if stream := mgr.GetAudioStream(); stream == nil {
		t.Error("GetAudioStream() returned nil")
	}
}

func TestPlayMicrophone(t *testing.T) {
	cfg := &Config{SampleRate: 48000, Channels: 2}
	mgr := NewManager(cfg)
	defer mgr.Stop()

	if err := mgr.PlayMicrophone("default"); err != nil {
		t.Errorf("PlayMicrophone() error = %v", err)
	}
	if stream := mgr.GetAudioStream(); stream == nil {
		t.Error("GetAudioStream() returned nil")
	}
}

func TestSourceSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createTestWAVFile(t, tmpDir)
	cfg := &Config{SampleRate: 48000, Channels: 2}
	mgr := NewManager(cfg)
	defer mgr.Stop()

	if err := mgr.PlayFile(testFile); err != nil {
		t.Fatalf("PlayFile() error = %v", err)
	}
	if err := mgr.SwitchSource(SourceTypeMicrophone); err != nil {
		t.Errorf("SwitchSource() error = %v", err)
	}
	if stream := mgr.GetAudioStream(); stream == nil {
		t.Error("GetAudioStream() returned nil after switch")
	}
}

func TestSpectrumStream(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createTestWAVFile(t, tmpDir)
	cfg := &Config{SampleRate: 48000, Channels: 2}
	mgr := NewManager(cfg)
	defer mgr.Stop()

	if err := mgr.PlayFile(testFile); err != nil {
		t.Fatalf("PlayFile() error = %v", err)
	}

	spectrumChan := mgr.GetSpectrumStream()
	if spectrumChan == nil {
		t.Fatal("GetSpectrumStream() returned nil")
	}

	select {
	case data := <-spectrumChan:
		if len(data) == 0 {
			t.Error("Received empty spectrum data")
		}
	case <-time.After(2 * time.Second):
		t.Log("No spectrum data received (timeout)")
	}
}

func TestConcurrentSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createTestWAVFile(t, tmpDir)
	cfg := &Config{SampleRate: 48000, Channels: 2}
	mgr := NewManager(cfg)
	defer mgr.Stop()

	if err := mgr.PlayFile(testFile); err != nil {
		t.Fatalf("PlayFile() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sourceType := SourceTypeFile
			if idx%2 == 0 {
				sourceType = SourceTypeMicrophone
			}
			_ = mgr.SwitchSource(sourceType)
		}(i)
	}
	wg.Wait()

	if stream := mgr.GetAudioStream(); stream == nil {
		t.Error("GetAudioStream() returned nil after concurrent switches")
	}
}

func TestStop(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createTestWAVFile(t, tmpDir)
	cfg := &Config{SampleRate: 48000, Channels: 2}
	mgr := NewManager(cfg)

	if err := mgr.PlayFile(testFile); err != nil {
		t.Fatalf("PlayFile() error = %v", err)
	}
	if err := mgr.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
	if stream := mgr.GetAudioStream(); stream != nil {
		t.Error("GetAudioStream() should return nil after Stop()")
	}
}

func TestFileSourceLoopsOnEOF(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createTestWAVFile(t, tmpDir)

	source, err := NewFileSource(testFile)
	if err != nil {
		t.Fatalf("NewFileSource() error = %v", err)
	}
	defer source.Close()

	buf := make([]byte, 2048)
	n1, err1 := source.Read(buf[:1500])
	if err1 != nil && err1 != io.EOF {
		t.Fatalf("first read error = %v", err1)
	}
	if n1 == 0 {
		t.Fatal("first read returned no data")
	}

	n2, err2 := source.Read(buf[:1500])
	if err2 != nil && err2 != io.EOF {
		t.Fatalf("second read error = %v", err2)
	}
	if n2 == 0 {
		t.Fatal("second read returned no data; expected looped playback data")
	}
}
