package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	PiFmRds   PiFmRdsConfig   `yaml:"pifmrds"`
	Storage   StorageConfig   `yaml:"storage"`
	Audio     AudioConfig     `yaml:"audio"`
	WebSocket WebSocketConfig `yaml:"websocket"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// PiFmRdsConfig pi_fm_rds 配置
type PiFmRdsConfig struct {
	BinaryPath       string  `yaml:"binary_path"`
	DefaultFrequency float64 `yaml:"default_frequency"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	UploadDir     string `yaml:"upload_dir"`
	TranscodedDir string `yaml:"transcoded_dir"`
	MaxFileSize   int64  `yaml:"max_file_size"`
	MaxTotalSize  int64  `yaml:"max_total_size"`
}

// AudioConfig 音频配置
type AudioConfig struct {
	SampleRate int `yaml:"sample_rate"`
	Channels   int `yaml:"channels"`
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	MaxClients  int `yaml:"max_clients"`
	SpectrumFPS int `yaml:"spectrum_fps"`
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 应用环境变量覆盖
	if err := cfg.applyEnvOverrides(); err != nil {
		return nil, fmt.Errorf("failed to apply env overrides: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides 应用环境变量覆盖
func (c *Config) applyEnvOverrides() error {
	// SERVER_PORT
	if port := os.Getenv("SERVER_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err != nil {
			return fmt.Errorf("invalid SERVER_PORT: %w", err)
		}
		c.Server.Port = p
	}

	// SERVER_HOST
	if host := os.Getenv("SERVER_HOST"); host != "" {
		c.Server.Host = host
	}

	// PIFMRDS_BINARY_PATH
	if path := os.Getenv("PIFMRDS_BINARY_PATH"); path != "" {
		c.PiFmRds.BinaryPath = path
	}

	// PIFMRDS_DEFAULT_FREQUENCY
	if freq := os.Getenv("PIFMRDS_DEFAULT_FREQUENCY"); freq != "" {
		var f float64
		if _, err := fmt.Sscanf(freq, "%f", &f); err != nil {
			return fmt.Errorf("invalid PIFMRDS_DEFAULT_FREQUENCY: %w", err)
		}
		c.PiFmRds.DefaultFrequency = f
	}

	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d (must be 1-65535)", c.Server.Port)
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	// 验证 FM 频率 (87.5 - 108.0 MHz)
	if c.PiFmRds.DefaultFrequency < 87.5 || c.PiFmRds.DefaultFrequency > 108.0 {
		return fmt.Errorf("invalid FM frequency: %.1f (must be 87.5-108.0 MHz)", c.PiFmRds.DefaultFrequency)
	}
	if c.PiFmRds.BinaryPath == "" {
		return fmt.Errorf("pi_fm_rds binary path cannot be empty")
	}

	// 验证存储配置
	if c.Storage.UploadDir == "" {
		return fmt.Errorf("upload directory cannot be empty")
	}
	if c.Storage.TranscodedDir == "" {
		return fmt.Errorf("transcoded directory cannot be empty")
	}
	if c.Storage.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	if c.Storage.MaxTotalSize <= 0 {
		return fmt.Errorf("max total size must be positive")
	}

	// 验证音频配置
	if c.Audio.SampleRate <= 0 {
		return fmt.Errorf("sample rate must be positive")
	}
	if c.Audio.Channels < 1 || c.Audio.Channels > 2 {
		return fmt.Errorf("channels must be 1 or 2")
	}

	// 验证 WebSocket 配置
	if c.WebSocket.MaxClients <= 0 {
		return fmt.Errorf("max clients must be positive")
	}
	if c.WebSocket.SpectrumFPS <= 0 {
		return fmt.Errorf("spectrum FPS must be positive")
	}

	return nil
}
