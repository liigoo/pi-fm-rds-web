package main

import (
	"os"
	"testing"

	"github.com/liigoo/pi-fm-rds-go/internal/config"
)

// TestLoadConfig 测试配置加载
func TestLoadConfig(t *testing.T) {
	configContent := `
server:
  host: "127.0.0.1"
  port: 18080
pifmrds:
  binary_path: "/usr/local/bin/pi_fm_rds"
  default_frequency: 100.0
storage:
  upload_dir: "/tmp/test_uploads"
  transcoded_dir: "/tmp/test_transcoded"
  max_file_size: 104857600
  max_total_size: 1073741824
audio:
  sample_rate: 44100
  channels: 2
websocket:
  max_clients: 10
  spectrum_fps: 30
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != 18080 {
		t.Errorf("Expected port 18080, got %d", cfg.Server.Port)
	}

	if cfg.PiFmRds.DefaultFrequency != 100.0 {
		t.Errorf("Expected frequency 100.0, got %.1f", cfg.PiFmRds.DefaultFrequency)
	}
}

// TestInitializeManagers 测试管理器初始化
func TestInitializeManagers(t *testing.T) {
	cfg := &config.Config{
		Audio: config.AudioConfig{
			SampleRate: 44100,
			Channels:   2,
		},
		Storage: config.StorageConfig{
			UploadDir:     "/tmp/test_uploads",
			TranscodedDir: "/tmp/test_transcoded",
			MaxFileSize:   104857600,
			MaxTotalSize:  1073741824,
		},
		PiFmRds: config.PiFmRdsConfig{
			BinaryPath:       "/usr/local/bin/pi_fm_rds",
			DefaultFrequency: 100.0,
		},
		WebSocket: config.WebSocketConfig{
			MaxClients:  10,
			SpectrumFPS: 30,
		},
	}

	managers, err := initializeManagers(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize managers: %v", err)
	}

	if managers.audio == nil {
		t.Error("Audio manager not initialized")
	}
	if managers.storage == nil {
		t.Error("Storage manager not initialized")
	}
	if managers.process == nil {
		t.Error("Process manager not initialized")
	}
	if managers.playlist == nil {
		t.Error("Playlist manager not initialized")
	}
	if managers.wsHub == nil {
		t.Error("WebSocket hub not initialized")
	}
}
