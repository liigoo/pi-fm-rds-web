package audio

import (
	"fmt"
)

// PCMFormat PCM 数据格式
type PCMFormat int

const (
	PCMFormat16Bit PCMFormat = iota
	PCMFormat8Bit
)

// Spectrum 频谱数据处理器
type Spectrum struct {
	filterCoeffs []float64
}

// NewSpectrum 创建频谱处理器
func NewSpectrum() *Spectrum {
	return &Spectrum{}
}

// Downsample 降采样
func (s *Spectrum) Downsample(input []int16, inputRate, targetRate int) []int16 {
	if inputRate == targetRate {
		return input
	}

	ratio := float64(inputRate) / float64(targetRate)
	outputLen := int(float64(len(input)) / ratio)
	output := make([]int16, outputLen)

	for i := 0; i < outputLen; i++ {
		srcIdx := int(float64(i) * ratio)
		if srcIdx < len(input) {
			output[i] = input[srcIdx]
		}
	}

	return output
}

// ApplyAntiAliasingFilter 应用抗混叠滤波器
func (s *Spectrum) ApplyAntiAliasingFilter(input []int16, sampleRate int, cutoffFreq float64) []int16 {
	// Simple low-pass filter using moving average
	windowSize := int(float64(sampleRate) / cutoffFreq / 2)
	if windowSize < 3 {
		windowSize = 3
	}
	if windowSize%2 == 0 {
		windowSize++
	}

	output := make([]int16, len(input))
	halfWindow := windowSize / 2

	for i := range input {
		sum := 0.0
		count := 0

		for j := -halfWindow; j <= halfWindow; j++ {
			idx := i + j
			if idx >= 0 && idx < len(input) {
				sum += float64(input[idx])
				count++
			}
		}

		if count > 0 {
			output[i] = int16(sum / float64(count))
		}
	}

	return output
}

// FormatPCM 格式化 PCM 数据
func (s *Spectrum) FormatPCM(input []int16, format PCMFormat) []byte {
	switch format {
	case PCMFormat16Bit:
		// 16-bit little-endian
		output := make([]byte, len(input)*2)
		for i, sample := range input {
			output[i*2] = byte(sample & 0xFF)
			output[i*2+1] = byte((sample >> 8) & 0xFF)
		}
		return output

	case PCMFormat8Bit:
		// 8-bit unsigned
		output := make([]byte, len(input))
		for i, sample := range input {
			// Convert signed 16-bit to unsigned 8-bit
			output[i] = byte((int(sample) + 32768) >> 8)
		}
		return output

	default:
		return nil
	}
}

// ProcessForFM 处理音频数据用于 FM 传输
func (s *Spectrum) ProcessForFM(input []int16, inputRate int) ([]byte, error) {
	if inputRate <= 0 {
		return nil, fmt.Errorf("invalid input sample rate: %d", inputRate)
	}

	targetRate := 8000 // 8kHz for FM

	// Apply anti-aliasing filter before downsampling
	filtered := s.ApplyAntiAliasingFilter(input, inputRate, 4000)

	// Downsample to 8kHz
	downsampled := s.Downsample(filtered, inputRate, targetRate)

	// Format as 16-bit PCM
	output := s.FormatPCM(downsampled, PCMFormat16Bit)

	return output, nil
}
