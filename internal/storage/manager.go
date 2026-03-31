package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
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

type fileMetadata struct {
	Filename string `json:"filename"`
	Format   string `json:"format"`
}

// NewManager 创建存储管理器
func NewManager(uploadDir, transcodedDir string, maxFileSize, maxTotalSize int64) Manager {
	// 确保目录存在
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(transcodedDir, 0755)

	m := &manager{
		uploadDir:     uploadDir,
		transcodedDir: transcodedDir,
		maxFileSize:   maxFileSize,
		maxTotalSize:  maxTotalSize,
		files:         make(map[string]*FileInfo),
	}
	m.mu.Lock()
	_ = m.refreshFilesLocked()
	m.mu.Unlock()
	return m
}

// Upload 上传文件
func (m *manager) Upload(file io.Reader, header *multipart.FileHeader) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.refreshFilesLocked(); err != nil {
		return "", err
	}

	// 检查单文件大小限制
	if header.Size > m.maxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum %d", header.Size, m.maxFileSize)
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

	fileID := sanitizeFilename(header.Filename)
	if fileID == "" {
		return "", fmt.Errorf("invalid filename")
	}

	replacedSize := int64(0)
	if existing, ok := m.files[fileID]; ok && existing != nil {
		replacedSize = existing.Size
	}

	currentUsed := m.calculateUsedSpace()
	currentUsed = currentUsed - replacedSize
	if currentUsed+header.Size > m.maxTotalSize {
		return "", fmt.Errorf("total quota exceeded: used %d + new %d > max %d",
			currentUsed, header.Size, m.maxTotalSize)
	}

	// 保存文件
	filePath := filepath.Join(m.uploadDir, fileID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if err := m.writeMetadata(fileID, &fileMetadata{
		Filename: header.Filename,
		Format:   format,
	}); err != nil {
		_ = os.Remove(filePath)
		return "", err
	}

	// 记录文件信息
	m.files[fileID] = &FileInfo{
		ID:       fileID,
		Filename: header.Filename,
		Size:     int64(len(data)),
		Format:   format,
		Duration: 0,
	}

	return fileID, nil
}

// Delete 删除文件
func (m *manager) Delete(fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.refreshFilesLocked(); err != nil {
		return err
	}

	// 检查文件是否存在
	if _, exists := m.files[fileID]; !exists {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// 删除原始文件
	filePath := filepath.Join(m.uploadDir, fileID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	_ = os.Remove(m.metadataPath(fileID))

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

	if err := m.refreshFilesLocked(); err != nil {
		return nil, err
	}

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

	if err := m.refreshFilesLocked(); err != nil {
		return "", err
	}

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

	if err := m.refreshFilesLocked(); err != nil {
		return nil, err
	}

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

	_ = m.refreshFilesLocked()

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

func (m *manager) refreshFilesLocked() error {
	entries, err := os.ReadDir(m.uploadDir)
	if err != nil {
		return fmt.Errorf("failed to scan upload dir: %w", err)
	}

	refreshed := make(map[string]*FileInfo, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".meta.json") {
			continue
		}

		filePath := filepath.Join(m.uploadDir, name)
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to inspect file %s: %w", name, err)
		}

		fileInfo := &FileInfo{
			ID:       name,
			Filename: name,
			Size:     info.Size(),
			Duration: 0,
		}

		if meta, err := m.readMetadata(name); err == nil {
			if meta.Filename != "" {
				fileInfo.Filename = meta.Filename
			}
			fileInfo.Format = meta.Format
		} else {
			fileInfo.Format = detectAudioFormatFromFile(filePath)
		}

		refreshed[name] = fileInfo
	}

	m.files = refreshed
	return nil
}

func (m *manager) metadataPath(fileID string) string {
	return filepath.Join(m.uploadDir, fileID+".meta.json")
}

func (m *manager) writeMetadata(fileID string, meta *fileMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}
	if err := os.WriteFile(m.metadataPath(fileID), data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	return nil
}

func (m *manager) readMetadata(fileID string) (*fileMetadata, error) {
	data, err := os.ReadFile(m.metadataPath(fileID))
	if err != nil {
		return nil, err
	}

	var meta fileMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
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

func detectAudioFormatFromFile(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	header := make([]byte, 12)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return ""
	}

	format, err := detectAudioFormat(header[:n])
	if err != nil {
		return ""
	}
	return format
}

func sanitizeFilename(name string) string {
	cleaned := strings.TrimSpace(filepath.Base(name))
	if cleaned == "." || cleaned == "" {
		return ""
	}
	return cleaned
}
