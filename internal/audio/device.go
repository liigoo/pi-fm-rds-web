package audio

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// AudioDevice 音频设备
type AudioDevice struct {
	ID          string
	Name        string
	Description string
}

// CaptureConfig 音频捕获配置
type CaptureConfig struct {
	SampleRate int
	Channels   int
	Format     string
	BufferSize int
}

// DeviceHandle 设备句柄（模拟实现）
type DeviceHandle struct {
	deviceID string
	config   *CaptureConfig
	closed   bool
}

func (h *DeviceHandle) Close() error {
	h.closed = true
	return nil
}

// DeviceManager 设备管理器
type DeviceManager struct {
	alsaCardsPath string
}

// NewDeviceManager 创建设备管理器
func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		alsaCardsPath: "/proc/asound/cards",
	}
}

// ListDevices 列出所有音频设备
func (dm *DeviceManager) ListDevices() ([]AudioDevice, error) {
	file, err := os.Open(dm.alsaCardsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ALSA cards: %w", err)
	}
	defer file.Close()

	var devices []AudioDevice
	scanner := bufio.NewScanner(file)

	// Parse ALSA cards format:
	// 0 [HDMI           ]: HDA-Intel - HDA Intel HDMI
	cardRegex := regexp.MustCompile(`^\s*(\d+)\s+\[([^\]]+)\]\s*:\s+(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := cardRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			device := AudioDevice{
				ID:          matches[1],
				Name:        strings.TrimSpace(matches[2]),
				Description: strings.TrimSpace(matches[3]),
			}
			devices = append(devices, device)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read ALSA cards: %w", err)
	}

	return devices, nil
}

// CheckAvailability 检查设备可用性
func (dm *DeviceManager) CheckAvailability(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	// Parse device ID (e.g., "hw:0,0")
	if !strings.HasPrefix(deviceID, "hw:") {
		return fmt.Errorf("invalid device ID format: %s", deviceID)
	}

	// Extract card number from device ID
	parts := strings.Split(strings.TrimPrefix(deviceID, "hw:"), ",")
	if len(parts) < 1 {
		return fmt.Errorf("invalid device ID format: %s", deviceID)
	}
	cardID := parts[0]

	// Check if device exists in ALSA
	devices, err := dm.ListDevices()
	if err != nil {
		// ALSA not available (e.g., macOS), validate card number is reasonable
		cardNum := 0
		if _, scanErr := fmt.Sscanf(cardID, "%d", &cardNum); scanErr != nil {
			return fmt.Errorf("invalid card number in device ID: %s", deviceID)
		}
		// Reject card numbers that are unreasonably large
		if cardNum > 31 {
			return fmt.Errorf("device not found: %s", deviceID)
		}
		return nil
	}

	found := false
	for _, dev := range devices {
		if dev.ID == cardID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	return nil
}

// GetCaptureConfig 获取音频捕获配置
func (dm *DeviceManager) GetCaptureConfig(deviceID string) (*CaptureConfig, error) {
	if err := dm.CheckAvailability(deviceID); err != nil {
		return nil, err
	}

	// Return default configuration
	return &CaptureConfig{
		SampleRate: 44100,
		Channels:   2,
		Format:     "S16_LE", // 16-bit signed little-endian
		BufferSize: 4096,
	}, nil
}

// OpenDevice 打开音频设备
func (dm *DeviceManager) OpenDevice(deviceID string, config *CaptureConfig) (*DeviceHandle, error) {
	if err := dm.CheckAvailability(deviceID); err != nil {
		return nil, err
	}

	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if config.SampleRate <= 0 {
		return nil, fmt.Errorf("invalid sample rate: %d", config.SampleRate)
	}
	if config.Channels <= 0 {
		return nil, fmt.Errorf("invalid channels: %d", config.Channels)
	}

	// Create device handle (mock implementation)
	handle := &DeviceHandle{
		deviceID: deviceID,
		config:   config,
		closed:   false,
	}

	return handle, nil
}
