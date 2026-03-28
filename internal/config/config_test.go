package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigLoad 测试配置文件加载
func TestConfigLoad(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  host: "0.0.0.0"

pifmrds:
  binary_path: "/usr/local/bin/pi_fm_rds"
  default_frequency: 107.9

storage:
  upload_dir: "./uploads"
  transcoded_dir: "./transcoded"
  max_file_size: 104857600
  max_total_size: 2147483648

audio:
  sample_rate: 22050
  channels: 1

websocket:
  max_clients: 5
  spectrum_fps: 10
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 测试加载配置
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证配置值
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %s, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.PiFmRds.DefaultFrequency != 107.9 {
		t.Errorf("PiFmRds.DefaultFrequency = %f, want 107.9", cfg.PiFmRds.DefaultFrequency)
	}
	if cfg.Storage.MaxFileSize != 104857600 {
		t.Errorf("Storage.MaxFileSize = %d, want 104857600", cfg.Storage.MaxFileSize)
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				PiFmRds: PiFmRdsConfig{
					BinaryPath:       "/usr/local/bin/pi_fm_rds",
					DefaultFrequency: 107.9,
				},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio: AudioConfig{
					SampleRate: 22050,
					Channels:   1,
				},
				WebSocket: WebSocketConfig{
					MaxClients:  5,
					SpectrumFPS: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Server: ServerConfig{
					Port: 0,
					Host: "0.0.0.0",
				},
				PiFmRds: PiFmRdsConfig{
					BinaryPath:       "/usr/local/bin/pi_fm_rds",
					DefaultFrequency: 107.9,
				},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio: AudioConfig{
					SampleRate: 22050,
					Channels:   1,
				},
				WebSocket: WebSocketConfig{
					MaxClients:  5,
					SpectrumFPS: 10,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid frequency - out of range",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				PiFmRds: PiFmRdsConfig{
					BinaryPath:       "/usr/local/bin/pi_fm_rds",
					DefaultFrequency: 50.0,
				},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio: AudioConfig{
					SampleRate: 22050,
					Channels:   1,
				},
				WebSocket: WebSocketConfig{
					MaxClients:  5,
					SpectrumFPS: 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEnvOverride 测试环境变量覆盖
func TestEnvOverride(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  host: "0.0.0.0"

pifmrds:
  binary_path: "/usr/local/bin/pi_fm_rds"
  default_frequency: 107.9

storage:
  upload_dir: "./uploads"
  transcoded_dir: "./transcoded"
  max_file_size: 104857600
  max_total_size: 2147483648

audio:
  sample_rate: 22050
  channels: 1

websocket:
  max_clients: 5
  spectrum_fps: 10
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 设置环境变量
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("PIFMRDS_DEFAULT_FREQUENCY", "88.5")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("PIFMRDS_DEFAULT_FREQUENCY")
	}()

	// 加载配置
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证环境变量覆盖
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090 (from env)", cfg.Server.Port)
	}
	if cfg.PiFmRds.DefaultFrequency != 88.5 {
		t.Errorf("PiFmRds.DefaultFrequency = %f, want 88.5 (from env)", cfg.PiFmRds.DefaultFrequency)
	}
}

// TestConfigLoadFileNotFound 测试配置文件不存在
func TestConfigLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
}

// TestConfigLoadInvalidYAML 测试无效的 YAML 格式
func TestConfigLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
server:
  port: invalid_port
  host: "0.0.0.0"
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}

// TestEnvOverrideInvalidValues 测试无效的环境变量值
func TestEnvOverrideInvalidValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  host: "0.0.0.0"

pifmrds:
  binary_path: "/usr/local/bin/pi_fm_rds"
  default_frequency: 107.9

storage:
  upload_dir: "./uploads"
  transcoded_dir: "./transcoded"
  max_file_size: 104857600
  max_total_size: 2147483648

audio:
  sample_rate: 22050
  channels: 1

websocket:
  max_clients: 5
  spectrum_fps: 10
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 测试无效的端口号
	os.Setenv("SERVER_PORT", "invalid")
	_, err := Load(configPath)
	os.Unsetenv("SERVER_PORT")
	if err == nil {
		t.Error("Load() expected error for invalid SERVER_PORT, got nil")
	}

	// 测试无效的频率
	os.Setenv("PIFMRDS_DEFAULT_FREQUENCY", "invalid")
	_, err = Load(configPath)
	os.Unsetenv("PIFMRDS_DEFAULT_FREQUENCY")
	if err == nil {
		t.Error("Load() expected error for invalid PIFMRDS_DEFAULT_FREQUENCY, got nil")
	}
}

// TestValidationEdgeCases 测试验证边界情况
func TestValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty host",
			config: &Config{
				Server: ServerConfig{Port: 8080, Host: ""},
				PiFmRds: PiFmRdsConfig{
					BinaryPath:       "/usr/local/bin/pi_fm_rds",
					DefaultFrequency: 107.9,
				},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio:     AudioConfig{SampleRate: 22050, Channels: 1},
				WebSocket: WebSocketConfig{MaxClients: 5, SpectrumFPS: 10},
			},
			wantErr: true,
			errMsg:  "server host cannot be empty",
		},
		{
			name: "empty binary path",
			config: &Config{
				Server: ServerConfig{Port: 8080, Host: "0.0.0.0"},
				PiFmRds: PiFmRdsConfig{
					BinaryPath:       "",
					DefaultFrequency: 107.9,
				},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio:     AudioConfig{SampleRate: 22050, Channels: 1},
				WebSocket: WebSocketConfig{MaxClients: 5, SpectrumFPS: 10},
			},
			wantErr: true,
			errMsg:  "pi_fm_rds binary path cannot be empty",
		},
		{
			name: "invalid channels - zero",
			config: &Config{
				Server:  ServerConfig{Port: 8080, Host: "0.0.0.0"},
				PiFmRds: PiFmRdsConfig{BinaryPath: "/usr/local/bin/pi_fm_rds", DefaultFrequency: 107.9},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio:     AudioConfig{SampleRate: 22050, Channels: 0},
				WebSocket: WebSocketConfig{MaxClients: 5, SpectrumFPS: 10},
			},
			wantErr: true,
			errMsg:  "channels must be 1 or 2",
		},
		{
			name: "invalid channels - three",
			config: &Config{
				Server:  ServerConfig{Port: 8080, Host: "0.0.0.0"},
				PiFmRds: PiFmRdsConfig{BinaryPath: "/usr/local/bin/pi_fm_rds", DefaultFrequency: 107.9},
				Storage: StorageConfig{
					UploadDir:     "./uploads",
					TranscodedDir: "./transcoded",
					MaxFileSize:   104857600,
					MaxTotalSize:  2147483648,
				},
				Audio:     AudioConfig{SampleRate: 22050, Channels: 3},
				WebSocket: WebSocketConfig{MaxClients: 5, SpectrumFPS: 10},
			},
			wantErr: true,
			errMsg:  "channels must be 1 or 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

