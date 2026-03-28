package audio

import (
	"fmt"
	"os"
	"time"
)

// AudioMetadata 音频元数据
type AudioMetadata struct {
	Format     AudioFormat
	Duration   time.Duration
	SampleRate int
	Channels   int
	Bitrate    int
}

// Validator 音频格式验证器
type Validator struct {
	transcoder *Transcoder
}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{
		transcoder: NewTranscoder("/tmp/pi-fm-rds-cache"),
	}
}

// ValidateFormat 验证音频格式
func (v *Validator) ValidateFormat(filePath string) error {
	format, err := v.transcoder.DetectFormat(filePath)
	if err != nil {
		return err
	}

	if format == FormatUnknown {
		return fmt.Errorf("unsupported audio format")
	}

	return nil
}

// ExtractMetadata 提取音频元数据
func (v *Validator) ExtractMetadata(filePath string) (*AudioMetadata, error) {
	// Detect format
	format, err := v.transcoder.DetectFormat(filePath)
	if err != nil {
		return nil, err
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Basic metadata (simplified implementation)
	meta := &AudioMetadata{
		Format:     format,
		Duration:   time.Second * 10, // Placeholder
		SampleRate: 44100,
		Channels:   2,
		Bitrate:    1411200,
	}

	// Estimate duration from file size (rough approximation)
	if format == FormatWAV {
		// WAV: fileSize / (sampleRate * channels * bytesPerSample)
		bytesPerSample := 2 // 16-bit
		dataSize := fileInfo.Size() - 44 // Subtract WAV header
		if dataSize > 0 {
			samples := dataSize / int64(meta.Channels*bytesPerSample)
			meta.Duration = time.Duration(samples*1000000000/int64(meta.SampleRate)) * time.Nanosecond
		}
	}

	return meta, nil
}

// CheckCompatibility 检查格式兼容性
func (v *Validator) CheckCompatibility(meta *AudioMetadata) error {
	// Check sample rate (minimum 22050 Hz for FM broadcast)
	if meta.SampleRate < 22050 {
		return fmt.Errorf("sample rate too low: %d Hz (minimum 22050 Hz)", meta.SampleRate)
	}

	// Check channels (must be stereo)
	if meta.Channels != 2 {
		return fmt.Errorf("must be stereo audio (2 channels), got %d", meta.Channels)
	}

	// Check duration (must be > 0)
	if meta.Duration <= 0 {
		return fmt.Errorf("invalid duration: %v", meta.Duration)
	}

	return nil
}
