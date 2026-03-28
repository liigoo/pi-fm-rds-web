package audio

import (
	"testing"
	"time"
)

func TestValidator_ValidateFormat(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid MP3",
			filePath: "testdata/sample.mp3",
			wantErr:  false,
		},
		{
			name:     "Valid FLAC",
			filePath: "testdata/sample.flac",
			wantErr:  false,
		},
		{
			name:     "Valid WAV",
			filePath: "testdata/sample.wav",
			wantErr:  false,
		},
		{
			name:     "Invalid format",
			filePath: "testdata/sample.txt",
			wantErr:  true,
		},
		{
			name:     "Non-existent file",
			filePath: "testdata/nonexistent.mp3",
			wantErr:  true,
		},
	}

	validator := NewValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFormat(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ExtractMetadata(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
		validate func(*testing.T, *AudioMetadata)
	}{
		{
			name:     "MP3 metadata",
			filePath: "testdata/sample.mp3",
			wantErr:  false,
			validate: func(t *testing.T, meta *AudioMetadata) {
				if meta.Format != FormatMP3 {
					t.Errorf("Expected format MP3, got %v", meta.Format)
				}
			},
		},
		{
			name:     "FLAC metadata",
			filePath: "testdata/sample.flac",
			wantErr:  false,
			validate: func(t *testing.T, meta *AudioMetadata) {
				if meta.Format != FormatFLAC {
					t.Errorf("Expected format FLAC, got %v", meta.Format)
				}
			},
		},
		{
			name:     "WAV metadata",
			filePath: "testdata/sample.wav",
			wantErr:  false,
			validate: func(t *testing.T, meta *AudioMetadata) {
				if meta.Format != FormatWAV {
					t.Errorf("Expected format WAV, got %v", meta.Format)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := validator.ExtractMetadata(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, meta)
			}
		})
	}
}

func TestValidator_CheckCompatibility(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		meta    *AudioMetadata
		wantErr bool
	}{
		{
			name: "Compatible audio",
			meta: &AudioMetadata{
				Format:     FormatWAV,
				Duration:   time.Second * 10,
				SampleRate: 44100,
				Channels:   2,
				Bitrate:    1411200,
			},
			wantErr: false,
		},
		{
			name: "Low sample rate",
			meta: &AudioMetadata{
				Format:     FormatWAV,
				Duration:   time.Second * 10,
				SampleRate: 8000,
				Channels:   2,
				Bitrate:    128000,
			},
			wantErr: true,
		},
		{
			name: "Mono audio",
			meta: &AudioMetadata{
				Format:     FormatWAV,
				Duration:   time.Second * 10,
				SampleRate: 44100,
				Channels:   1,
				Bitrate:    705600,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CheckCompatibility(tt.meta)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
