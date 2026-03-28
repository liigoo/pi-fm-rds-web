package storage

import (
	"bytes"
	"mime/multipart"
	"sync"
	"testing"
)

// TestFileUpload 测试文件上传
func TestFileUpload(t *testing.T) {
	// 创建临时目录
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()

	mgr := NewManager(uploadDir, transcodedDir, 100*1024*1024, 2*1024*1024*1024)

	// 创建测试 MP3 文件（带 magic bytes）
	mp3Data := createTestMP3Data()
	header := &multipart.FileHeader{
		Filename: "test.mp3",
		Size:     int64(len(mp3Data)),
	}

	fileID, err := mgr.Upload(bytes.NewReader(mp3Data), header)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if fileID == "" {
		t.Fatal("Expected non-empty file ID")
	}

	// 验证文件信息
	info, err := mgr.GetFile(fileID)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	if info.Filename != "test.mp3" {
		t.Errorf("Expected filename 'test.mp3', got '%s'", info.Filename)
	}

	if info.Size != int64(len(mp3Data)) {
		t.Errorf("Expected size %d, got %d", len(mp3Data), info.Size)
	}

	if info.Format != "audio/mpeg" {
		t.Errorf("Expected format 'audio/mpeg', got '%s'", info.Format)
	}
}

// TestQuotaEnforcement 测试配额限制
func TestQuotaEnforcement(t *testing.T) {
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()

	// 设置小配额用于测试
	maxFileSize := int64(1024)      // 1KB
	maxTotalSize := int64(1024)     // 1KB (只能容纳两个 512 字节的文件)
	mgr := NewManager(uploadDir, transcodedDir, maxFileSize, maxTotalSize)

	// 测试单文件大小限制
	largeData := make([]byte, maxFileSize+1)
	copy(largeData, []byte("ID3"))  // MP3 magic bytes
	header := &multipart.FileHeader{
		Filename: "large.mp3",
		Size:     int64(len(largeData)),
	}

	_, err := mgr.Upload(bytes.NewReader(largeData), header)
	if err == nil {
		t.Fatal("Expected error for file exceeding max size")
	}

	// 测试总配额限制
	smallData := createTestMP3Data()
	header1 := &multipart.FileHeader{
		Filename: "file1.mp3",
		Size:     int64(len(smallData)),
	}
	header2 := &multipart.FileHeader{
		Filename: "file2.mp3",
		Size:     int64(len(smallData)),
	}

	_, err = mgr.Upload(bytes.NewReader(smallData), header1)
	if err != nil {
		t.Fatalf("First upload failed: %v", err)
	}

	_, err = mgr.Upload(bytes.NewReader(smallData), header2)
	if err != nil {
		t.Fatalf("Second upload failed: %v", err)
	}

	// 第三个文件应该超出总配额
	header3 := &multipart.FileHeader{
		Filename: "file3.mp3",
		Size:     int64(len(smallData)),
	}
	_, err = mgr.Upload(bytes.NewReader(smallData), header3)
	if err == nil {
		t.Fatal("Expected error for exceeding total quota")
	}
}

// TestFileValidation 测试文件格式验证
func TestFileValidation(t *testing.T) {
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()
	mgr := NewManager(uploadDir, transcodedDir, 100*1024*1024, 2*1024*1024*1024)

	tests := []struct {
		name      string
		data      []byte
		filename  string
		wantError bool
	}{
		{
			name:      "valid MP3",
			data:      createTestMP3Data(),
			filename:  "test.mp3",
			wantError: false,
		},
		{
			name:      "valid WAV",
			data:      createTestWAVData(),
			filename:  "test.wav",
			wantError: false,
		},
		{
			name:      "invalid format",
			data:      []byte("not an audio file"),
			filename:  "test.txt",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     int64(len(tt.data)),
			}
			_, err := mgr.Upload(bytes.NewReader(tt.data), header)
			if (err != nil) != tt.wantError {
				t.Errorf("Upload() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestFileDelete 测试文件删除
func TestFileDelete(t *testing.T) {
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()
	mgr := NewManager(uploadDir, transcodedDir, 100*1024*1024, 2*1024*1024*1024)

	// 上传文件
	mp3Data := createTestMP3Data()
	header := &multipart.FileHeader{
		Filename: "test.mp3",
		Size:     int64(len(mp3Data)),
	}

	fileID, err := mgr.Upload(bytes.NewReader(mp3Data), header)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// 删除文件
	err = mgr.Delete(fileID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证文件已删除
	_, err = mgr.GetFile(fileID)
	if err == nil {
		t.Fatal("Expected error when getting deleted file")
	}

	// 验证配额已释放
	quota := mgr.GetQuotaInfo()
	if quota.Used != 0 {
		t.Errorf("Expected used quota to be 0, got %d", quota.Used)
	}
}

// TestConcurrentUpload 测试并发上传
func TestConcurrentUpload(t *testing.T) {
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()
	mgr := NewManager(uploadDir, transcodedDir, 100*1024*1024, 2*1024*1024*1024)

	const numUploads = 10
	var wg sync.WaitGroup
	errors := make(chan error, numUploads)

	for i := 0; i < numUploads; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			mp3Data := createTestMP3Data()
			header := &multipart.FileHeader{
				Filename: "test.mp3",
				Size:     int64(len(mp3Data)),
			}

			_, err := mgr.Upload(bytes.NewReader(mp3Data), header)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent upload error: %v", err)
	}

	// 验证所有文件都已上传
	files, err := mgr.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != numUploads {
		t.Errorf("Expected %d files, got %d", numUploads, len(files))
	}
}

// TestListFiles 测试文件列表
func TestListFiles(t *testing.T) {
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()
	mgr := NewManager(uploadDir, transcodedDir, 100*1024*1024, 2*1024*1024*1024)

	// 上传多个文件
	for i := 0; i < 3; i++ {
		mp3Data := createTestMP3Data()
		header := &multipart.FileHeader{
			Filename: "test.mp3",
			Size:     int64(len(mp3Data)),
		}
		_, err := mgr.Upload(bytes.NewReader(mp3Data), header)
		if err != nil {
			t.Fatalf("Upload %d failed: %v", i, err)
		}
	}

	files, err := mgr.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

// TestGetQuotaInfo 测试配额信息
func TestGetQuotaInfo(t *testing.T) {
	uploadDir := t.TempDir()
	transcodedDir := t.TempDir()
	maxTotal := int64(2 * 1024 * 1024 * 1024)
	mgr := NewManager(uploadDir, transcodedDir, 100*1024*1024, maxTotal)

	quota := mgr.GetQuotaInfo()
	if quota.Total != maxTotal {
		t.Errorf("Expected total %d, got %d", maxTotal, quota.Total)
	}
	if quota.Used != 0 {
		t.Errorf("Expected used 0, got %d", quota.Used)
	}
	if quota.Available != maxTotal {
		t.Errorf("Expected available %d, got %d", maxTotal, quota.Available)
	}

	// 上传文件后检查配额
	mp3Data := createTestMP3Data()
	header := &multipart.FileHeader{
		Filename: "test.mp3",
		Size:     int64(len(mp3Data)),
	}
	_, err := mgr.Upload(bytes.NewReader(mp3Data), header)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	quota = mgr.GetQuotaInfo()
	if quota.Used == 0 {
		t.Error("Expected used quota > 0")
	}
	if quota.Available != maxTotal-quota.Used {
		t.Errorf("Available quota mismatch: %d != %d - %d", quota.Available, maxTotal, quota.Used)
	}
}

// Helper functions

func createTestMP3Data() []byte {
	// MP3 magic bytes: ID3
	data := make([]byte, 512)
	copy(data, []byte("ID3"))
	return data
}

func createTestWAVData() []byte {
	// WAV magic bytes: RIFF....WAVE
	data := make([]byte, 512)
	copy(data, []byte("RIFF"))
	copy(data[8:], []byte("WAVE"))
	return data
}
