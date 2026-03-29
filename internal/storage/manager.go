package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager 存储管理器接口
type Manager interface {
	Upload(file io.Reader, header *multipart.FileHeader) (string, error)
	Delete(fileID string) error
	GetFile(fileID string) (*FileInfo, error)
	GetFilePath(fileID string) (string, error)
	ListFiles() ([]*FileInfo, error)
	GetQuotaInfo() QuotaInfo
}

// FileInfo 文件信息
type FileInfo struct {
	ID       string
	Filename string
	Size     int64
	Format   string
	Duration time.Duration
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	Used      int64
	Total     int64
	Available int64
}

// manager 存储管理器实现
type manager struct {
	uploadDir     string
	transcodedDir string
	maxFileSize   int64
	maxTotalSize  int64
	mu            sync.Mutex
	files         map[string]*FileInfo
}

// NewManager 创建存储管理器
func NewManager(uploadDir, transcodedDir string, maxFileSize, maxTotalSize int64) Manager {
	// 确保目录存在
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(transcodedDir, 0755)

	return &manager{
		uploadDir:     uploadDir,
		transcodedDir: transcodedDir,
		maxFileSize:   maxFileSize,
		maxTotalSize:  maxTotalSize,
		files:         make(map[string]*FileInfo),
	}
}

// Upload 上传文件
func (m *manager) Upload(file io.Reader, header *multipart.FileHeader) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查单文件大小限制
	if header.Size > m.maxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum %d", header.Size, m.maxFileSize)
	}

	// 检查总配额
	currentUsed := m.calculateUsedSpace()
	if currentUsed+header.Size > m.maxTotalSize {
		return "", fmt.Errorf("total quota exceeded: used %d + new %d > max %d",
			currentUsed, header.Size, m.maxTotalSize)
	}

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 验证文件格式（检查 magic bytes）
	format, err := detectAudioFormat(data)
	if err != nil {
		return "", fmt.Errorf("invalid audio format: %w", err)
	}

	// 生成文件 ID（使用时间戳 + SHA256 哈希确保唯一性）
	timestamp := time.Now().UnixNano()
	hashInput := append(data, []byte(fmt.Sprintf("%d", timestamp))...)
	hash := sha256.Sum256(hashInput)
	fileID := hex.EncodeToString(hash[:])

	// 保存文件
	filePath := filepath.Join(m.uploadDir, fileID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 记录文件信息
	m.files[fileID] = &FileInfo{
		ID:       fileID,
		Filename: header.Filename,
		Size:     int64(len(data)),
		Format:   format,
		Duration: 0, // TODO: 实际项目中需要解析音频文件获取时长
	}

	return fileID, nil
}

// Delete 删除文件
func (m *manager) Delete(fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查文件是否存在
	if _, exists := m.files[fileID]; !exists {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// 删除原始文件
	filePath := filepath.Join(m.uploadDir, fileID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// 删除转码文件（如果存在）
	transcodedPath := filepath.Join(m.transcodedDir, fileID+".wav")
	os.Remove(transcodedPath) // 忽略错误，文件可能不存在

	// 从内存中删除记录
	delete(m.files, fileID)

	return nil
}

// GetFile 获取文件信息
func (m *manager) GetFile(fileID string) (*FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.files[fileID]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}

	return info, nil
}

// GetFilePath 获取文件物理路径
func (m *manager) GetFilePath(fileID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[fileID]; !exists {
		return "", fmt.Errorf("file not found: %s", fileID)
	}

	filePath := filepath.Join(m.uploadDir, fileID)
	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("file path not available: %w", err)
	}
	return filePath, nil
}

// ListFiles 列出所有文件
func (m *manager) ListFiles() ([]*FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	files := make([]*FileInfo, 0, len(m.files))
	for _, info := range m.files {
		files = append(files, info)
	}

	return files, nil
}

// GetQuotaInfo 获取配额信息
func (m *manager) GetQuotaInfo() QuotaInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	used := m.calculateUsedSpace()
	return QuotaInfo{
		Used:      used,
		Total:     m.maxTotalSize,
		Available: m.maxTotalSize - used,
	}
}

// calculateUsedSpace 计算已使用空间（需要持有锁）
func (m *manager) calculateUsedSpace() int64 {
	var total int64
	for _, info := range m.files {
		total += info.Size
	}
	return total
}

// detectAudioFormat 检测音频格式（通过 magic bytes）
func detectAudioFormat(data []byte) (string, error) {
	if len(data) < 12 {
		return "", fmt.Errorf("file too small to detect format")
	}

	// MP3: ID3 或 0xFF 0xFB
	if len(data) >= 3 && string(data[:3]) == "ID3" {
		return "audio/mpeg", nil
	}
	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "audio/mpeg", nil
	}

	// WAV: RIFF....WAVE
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return "audio/wav", nil
	}

	// FLAC: fLaC
	if len(data) >= 4 && string(data[:4]) == "fLaC" {
		return "audio/flac", nil
	}

	// OGG: OggS
	if len(data) >= 4 && string(data[:4]) == "OggS" {
		return "audio/ogg", nil
	}

	return "", fmt.Errorf("unsupported audio format")
}
