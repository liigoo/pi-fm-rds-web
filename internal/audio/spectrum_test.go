package audio

import (
	"math"
	"testing"
)

func TestSpectrum_Downsample(t *testing.T) {
	spectrum := NewSpectrum()

	tests := []struct {
		name           string
		input          []int16
		inputRate      int
		targetRate     int
		expectedLength int
	}{
		{
			name:           "44100Hz to 8000Hz",
			input:          generateSineWave(44100, 1000, 1.0), // 1 second at 44100Hz
			inputRate:      44100,
			targetRate:     8000,
			expectedLength: 8000,
		},
		{
			name:           "22050Hz to 8000Hz",
			input:          generateSineWave(22050, 1000, 0.5), // 0.5 second at 22050Hz
			inputRate:      22050,
			targetRate:     8000,
			expectedLength: 4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := spectrum.Downsample(tt.input, tt.inputRate, tt.targetRate)
			if len(output) != tt.expectedLength {
				t.Errorf("Downsample() length = %d, want %d", len(output), tt.expectedLength)
			}
		})
	}
}

func TestSpectrum_ApplyAntiAliasingFilter(t *testing.T) {
	spectrum := NewSpectrum()

	tests := []struct {
		name       string
		input      []int16
		sampleRate int
		cutoffFreq float64
	}{
		{
			name:       "Filter 44100Hz signal",
			input:      generateSineWave(44100, 1000, 1.0),
			sampleRate: 44100,
			cutoffFreq: 4000, // Nyquist for 8kHz
		},
		{
			name:       "Filter 22050Hz signal",
			input:      generateSineWave(22050, 500, 0.5),
			sampleRate: 22050,
			cutoffFreq: 4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := spectrum.ApplyAntiAliasingFilter(tt.input, tt.sampleRate, tt.cutoffFreq)
			if len(output) != len(tt.input) {
				t.Errorf("ApplyAntiAliasingFilter() length = %d, want %d", len(output), len(tt.input))
			}

			// Check that output is not all zeros
			hasNonZero := false
			for _, v := range output {
				if v != 0 {
					hasNonZero = true
					break
				}
			}
			if !hasNonZero {
				t.Error("ApplyAntiAliasingFilter() produced all zeros")
			}
		})
	}
}

func TestSpectrum_FormatPCM(t *testing.T) {
	spectrum := NewSpectrum()

	tests := []struct {
		name   string
		input  []int16
		format PCMFormat
	}{
		{
			name:   "16-bit PCM",
			input:  []int16{0, 100, -100, 32767, -32768},
			format: PCMFormat16Bit,
		},
		{
			name:   "8-bit PCM",
			input:  []int16{0, 100, -100, 32767, -32768},
			format: PCMFormat8Bit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := spectrum.FormatPCM(tt.input, tt.format)
			if len(output) == 0 {
				t.Error("FormatPCM() returned empty output")
			}

			expectedLen := len(tt.input) * 2 // 16-bit = 2 bytes per sample
			if tt.format == PCMFormat8Bit {
				expectedLen = len(tt.input) // 8-bit = 1 byte per sample
			}

			if len(output) != expectedLen {
				t.Errorf("FormatPCM() length = %d, want %d", len(output), expectedLen)
			}
		})
	}
}

func TestSpectrum_ProcessForFM(t *testing.T) {
	spectrum := NewSpectrum()

	input := generateSineWave(44100, 1000, 1.0)

	output, err := spectrum.ProcessForFM(input, 44100)
	if err != nil {
		t.Fatalf("ProcessForFM() error = %v", err)
	}

	if len(output) == 0 {
		t.Error("ProcessForFM() returned empty output")
	}

	// Output should be downsampled to 8kHz, formatted as 16-bit PCM (2 bytes per sample)
	expectedSamples := len(input) * 8000 / 44100
	expectedLength := expectedSamples * 2 // 16-bit = 2 bytes per sample
	tolerance := expectedLength / 10      // 10% tolerance
	if math.Abs(float64(len(output)-expectedLength)) > float64(tolerance) {
		t.Errorf("ProcessForFM() length = %d, want ~%d", len(output), expectedLength)
	}
}

// Helper function to generate sine wave test data
func generateSineWave(sampleRate int, frequency float64, duration float64) []int16 {
	samples := int(float64(sampleRate) * duration)
	wave := make([]int16, samples)

	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		value := math.Sin(2 * math.Pi * frequency * t)
		wave[i] = int16(value * 32767)
	}

	return wave
}
