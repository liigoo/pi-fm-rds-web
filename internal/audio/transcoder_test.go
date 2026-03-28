package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscoder_DetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     AudioFormat
		wantErr  bool
	}{
		{
			name:     "MP3 file",
			filePath: "testdata/sample.mp3",
			want:     FormatMP3,
			wantErr:  false,
		},
		{
			name:     "FLAC file",
			filePath: "testdata/sample.flac",
			want:     FormatFLAC,
			wantErr:  false,
		},
		{
			name:     "WAV file",
			filePath: "testdata/sample.wav",
			want:     FormatWAV,
			wantErr:  false,
		},
		{
			name:     "Unknown file",
			filePath: "testdata/sample.txt",
			want:     FormatUnknown,
			wantErr:  true,
		},
		{
			name:     "Non-existent file",
			filePath: "testdata/nonexistent.mp3",
			want:     FormatUnknown,
			wantErr:  true,
		},
	}

	transcoder := NewTranscoder("/tmp/pi-fm-rds-cache")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transcoder.DetectFormat(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTranscoder_Transcode(t *testing.T) {
	cacheDir := filepath.Join(os.TempDir(), "test-cache")
	defer os.RemoveAll(cacheDir)

	transcoder := NewTranscoder(cacheDir)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			// Dummy MP3 (just magic bytes) - FFmpeg will fail to decode
			name:    "Transcode MP3 to WAV (dummy file)",
			input:   "testdata/sample.mp3",
			wantErr: true,
		},
		{
			// Dummy FLAC (just magic bytes) - FFmpeg will fail to decode
			name:    "Transcode FLAC to WAV (dummy file)",
			input:   "testdata/sample.flac",
			wantErr: true,
		},
		{
			name:    "Already WAV",
			input:   "testdata/sample.wav",
			wantErr: false,
		},
		{
			name:    "Invalid file",
			input:   "testdata/invalid.mp3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := transcoder.Transcode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transcode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && output == "" {
				t.Error("Transcode() returned empty output path")
			}
		})
	}
}

func TestTranscoder_CacheManagement(t *testing.T) {
	cacheDir := filepath.Join(os.TempDir(), "test-cache-mgmt")
	defer os.RemoveAll(cacheDir)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	transcoder := NewTranscoder(cacheDir)

	// Test cache hit: pre-populate the cache then verify the same path is returned
	t.Run("Cache hit", func(t *testing.T) {
		input := "testdata/sample.mp3"

		// Pre-populate the cache with a dummy WAV file using the same key logic
		cacheKey := transcoder.GetCacheKey(input)
		cachedPath := filepath.Join(cacheDir, cacheKey+".wav")
		if err := os.WriteFile(cachedPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to pre-populate cache: %v", err)
		}

		// Transcode should return cached path
		output, err := transcoder.Transcode(input)
		if err != nil {
			t.Fatalf("Transcode() error = %v", err)
		}

		if output != cachedPath {
			t.Errorf("Cache miss: got=%s, want=%s", output, cachedPath)
		}
	})

	// Test cache cleanup
	t.Run("Cache cleanup", func(t *testing.T) {
		err := transcoder.CleanCache()
		if err != nil {
			t.Errorf("CleanCache() error = %v", err)
		}
	})
}
