package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeviceManager_ListDevices(t *testing.T) {
	// Create temporary test directory
	tmpDir := filepath.Join(os.TempDir(), "test-alsa")
	defer os.RemoveAll(tmpDir)

	// Create mock ALSA cards file
	cardsPath := filepath.Join(tmpDir, "cards")
	cardsContent := ` 0 [HDMI           ]: HDA-Intel - HDA Intel HDMI
                      HDA Intel HDMI at 0xf7e34000 irq 45
 1 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7e30000 irq 46
 2 [Device         ]: USB-Audio - USB Audio Device
                      Generic USB Audio Device at usb-0000:00:14.0-1
`
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if err := os.WriteFile(cardsPath, []byte(cardsContent), 0644); err != nil {
		t.Fatalf("Failed to write cards file: %v", err)
	}

	dm := NewDeviceManager()
	dm.alsaCardsPath = cardsPath

	devices, err := dm.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}

	if len(devices) != 3 {
		t.Errorf("ListDevices() returned %d devices, want 3", len(devices))
	}

	// Check first device
	if devices[0].ID != "0" {
		t.Errorf("Device 0 ID = %s, want 0", devices[0].ID)
	}
	if devices[0].Name != "HDMI" {
		t.Errorf("Device 0 Name = %s, want HDMI", devices[0].Name)
	}
}

func TestDeviceManager_CheckAvailability(t *testing.T) {
	dm := NewDeviceManager()

	tests := []struct {
		name     string
		deviceID string
		wantErr  bool
	}{
		{
			name:     "Valid device",
			deviceID: "hw:0,0",
			wantErr:  false,
		},
		{
			name:     "Invalid device",
			deviceID: "hw:99,99",
			wantErr:  true,
		},
		{
			name:     "Empty device ID",
			deviceID: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dm.CheckAvailability(tt.deviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeviceManager_GetCaptureConfig(t *testing.T) {
	dm := NewDeviceManager()

	tests := []struct {
		name     string
		deviceID string
		wantErr  bool
		validate func(*testing.T, *CaptureConfig)
	}{
		{
			name:     "Default config",
			deviceID: "hw:0,0",
			wantErr:  false,
			validate: func(t *testing.T, cfg *CaptureConfig) {
				if cfg.SampleRate != 44100 {
					t.Errorf("SampleRate = %d, want 44100", cfg.SampleRate)
				}
				if cfg.Channels != 2 {
					t.Errorf("Channels = %d, want 2", cfg.Channels)
				}
				if cfg.Format != "S16_LE" {
					t.Errorf("Format = %s, want S16_LE", cfg.Format)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := dm.GetCaptureConfig(tt.deviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCaptureConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestDeviceManager_OpenDevice(t *testing.T) {
	dm := NewDeviceManager()

	tests := []struct {
		name     string
		deviceID string
		config   *CaptureConfig
		wantErr  bool
	}{
		{
			name:     "Open valid device",
			deviceID: "hw:0,0",
			config: &CaptureConfig{
				SampleRate: 44100,
				Channels:   2,
				Format:     "S16_LE",
				BufferSize: 4096,
			},
			wantErr: false,
		},
		{
			name:     "Open invalid device",
			deviceID: "hw:99,99",
			config: &CaptureConfig{
				SampleRate: 44100,
				Channels:   2,
				Format:     "S16_LE",
				BufferSize: 4096,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, err := dm.OpenDevice(tt.deviceID, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && handle == nil {
				t.Error("OpenDevice() returned nil handle")
			}
			if handle != nil {
				handle.Close()
			}
		})
	}
}
