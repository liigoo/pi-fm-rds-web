package audio

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AudioFormat 音频格式
type AudioFormat int

const (
	FormatUnknown AudioFormat = iota
	FormatMP3
	FormatFLAC
	FormatWAV
)

// Magic bytes for audio format detection
var magicBytes = map[AudioFormat][]byte{
	FormatMP3:  {0xFF, 0xFB}, // MP3 frame sync
	FormatFLAC: {0x66, 0x4C, 0x61, 0x43}, // "fLaC"
	FormatWAV:  {0x52, 0x49, 0x46, 0x46}, // "RIFF"
}

// Transcoder 音频转码器
type Transcoder struct {
	cacheDir string
}

// NewTranscoder 创建转码器
func NewTranscoder(cacheDir string) *Transcoder {
	return &Transcoder{
		cacheDir: cacheDir,
	}
}

// DetectFormat 检测音频格式
func (t *Transcoder) DetectFormat(filePath string) (AudioFormat, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return FormatUnknown, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 12 bytes for magic byte detection
	header := make([]byte, 12)
	n, err := file.Read(header)
	if err != nil || n < 4 {
		return FormatUnknown, fmt.Errorf("failed to read file header: %w", err)
	}

	// Check magic bytes
	for format, magic := range magicBytes {
		if len(header) >= len(magic) {
			match := true
			for i, b := range magic {
				if header[i] != b {
					match = false
					break
				}
			}
			if match {
				return format, nil
			}
		}
	}

	return FormatUnknown, fmt.Errorf("unknown audio format")
}

// Transcode 转码音频文件到 WAV
func (t *Transcoder) Transcode(inputPath string) (string, error) {
	// Detect format
	format, err := t.DetectFormat(inputPath)
	if err != nil {
		return "", err
	}

	// If already WAV, return original path
	if format == FormatWAV {
		return inputPath, nil
	}

	// Check cache
	cacheKey := t.getCacheKey(inputPath)
	cachedPath := filepath.Join(t.cacheDir, cacheKey+".wav")

	// Return cached file if exists
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath, nil
	}

	// Create cache directory if not exists
	if err := os.MkdirAll(t.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Transcode using FFmpeg
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ar", "44100", // Sample rate
		"-ac", "2",     // Stereo
		"-f", "wav",
		"-y", // Overwrite
		cachedPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg transcode failed: %w, output: %s", err, string(output))
	}

	return cachedPath, nil
}

// GetCacheKey 生成缓存键（导出供测试使用）
func (t *Transcoder) GetCacheKey(filePath string) string {
	return t.getCacheKey(filePath)
}

// getCacheKey 生成缓存键
func (t *Transcoder) getCacheKey(filePath string) string {
	// Use file path + modification time as cache key
	stat, err := os.Stat(filePath)
	if err != nil {
		// Fallback to path only
		hash := md5.Sum([]byte(filePath))
		return hex.EncodeToString(hash[:])
	}

	key := fmt.Sprintf("%s-%d", filePath, stat.ModTime().Unix())
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CleanCache 清理缓存
func (t *Transcoder) CleanCache() error {
	if _, err := os.Stat(t.cacheDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(t.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".wav") {
			path := filepath.Join(t.cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove cached file %s: %w", path, err)
			}
		}
	}

	return nil
}
