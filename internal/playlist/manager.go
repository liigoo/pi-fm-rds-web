package playlist

import (
	"fmt"
	"sync"
	"time"
)

const MaxPlaylistItems = 50

// Manager 播放列表管理器接口
type Manager interface {
	Add(fileID, filename string, duration time.Duration) error
	Remove(fileID string) error
	Reorder(fromIndex, toIndex int) error
	Next() (string, error)
	Prev() (string, error)
	Skip() error
	SetCurrent(index int) (string, error)
	IndexOf(fileID string) int
	CurrentIndex() int
	ResetCurrent()
	GetCurrent() *PlaylistItem
	GetAll() []PlaylistItem
	Clear()
}

// PlaylistItem 播放列表项
type PlaylistItem struct {
	FileID   string
	Filename string
	Duration time.Duration
	Index    int
}

// manager 播放列表管理器实现
type manager struct {
	mu           sync.Mutex
	items        []PlaylistItem
	fileIDMap    map[string]int // fileID -> index mapping
	currentIndex int            // -1 表示未开始播放
}

// NewManager 创建播放列表管理器
func NewManager() Manager {
	return &manager{
		items:        make([]PlaylistItem, 0, MaxPlaylistItems),
		fileIDMap:    make(map[string]int),
		currentIndex: -1,
	}
}

// Add 添加文件到播放列表
func (m *manager) Add(fileID, filename string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查文件是否已存在
	if _, exists := m.fileIDMap[fileID]; exists {
		return fmt.Errorf("file %s already exists in playlist", fileID)
	}

	// 检查播放列表是否已满
	if len(m.items) >= MaxPlaylistItems {
		return fmt.Errorf("playlist has reached maximum capacity of %d items", MaxPlaylistItems)
	}

	// 添加新项
	item := PlaylistItem{
		FileID:   fileID,
		Filename: filename,
		Duration: duration,
		Index:    len(m.items),
	}

	m.items = append(m.items, item)
	m.fileIDMap[fileID] = item.Index

	return nil
}

// Remove 从播放列表移除文件
func (m *manager) Remove(fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查文件是否存在
	index, exists := m.fileIDMap[fileID]
	if !exists {
		return fmt.Errorf("file %s not found in playlist", fileID)
	}

	// 如果移除的是当前播放的文件，清除当前播放状态
	if m.currentIndex == index {
		m.currentIndex = -1
	} else if m.currentIndex > index {
		// 如果移除的文件在当前播放位置之前，调整当前索引
		m.currentIndex--
	}

	// 移除文件
	m.items = append(m.items[:index], m.items[index+1:]...)

	// 重建索引映射
	m.rebuildIndexMap()

	return nil
}

// Reorder 重新排序播放列表（拖拽排序）
func (m *manager) Reorder(fromIndex, toIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证索引有效性
	if fromIndex < 0 || fromIndex >= len(m.items) {
		return fmt.Errorf("invalid fromIndex: %d", fromIndex)
	}
	if toIndex < 0 || toIndex >= len(m.items) {
		return fmt.Errorf("invalid toIndex: %d", toIndex)
	}

	if fromIndex == toIndex {
		return nil
	}

	// 保存要移动的元素
	item := m.items[fromIndex]

	// 根据移动方向调整数组
	if fromIndex < toIndex {
		// 向后移动：将 fromIndex+1 到 toIndex 的元素向前移动
		copy(m.items[fromIndex:toIndex], m.items[fromIndex+1:toIndex+1])
		m.items[toIndex] = item
	} else {
		// 向前移动：将 toIndex 到 fromIndex-1 的元素向后移动
		copy(m.items[toIndex+1:fromIndex+1], m.items[toIndex:fromIndex])
		m.items[toIndex] = item
	}

	// 重建索引映射
	m.rebuildIndexMap()

	return nil
}

// Next 播放下一首
func (m *manager) Next() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查播放列表是否为空
	if len(m.items) == 0 {
		return "", fmt.Errorf("playlist is empty")
	}

	// 移动到下一首
	m.currentIndex++

	// 检查是否到达播放列表末尾
	if m.currentIndex >= len(m.items) {
		m.currentIndex = len(m.items) - 1
		return "", fmt.Errorf("reached end of playlist")
	}

	return m.items[m.currentIndex].FileID, nil
}

// Prev 播放上一首
func (m *manager) Prev() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.items) == 0 {
		return "", fmt.Errorf("playlist is empty")
	}

	if m.currentIndex < 0 {
		m.currentIndex = 0
		return m.items[m.currentIndex].FileID, nil
	}

	if m.currentIndex == 0 {
		return "", fmt.Errorf("already at beginning of playlist")
	}

	m.currentIndex--
	return m.items[m.currentIndex].FileID, nil
}

// Skip 跳过当前文件
func (m *manager) Skip() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否正在播放
	if m.currentIndex < 0 {
		return fmt.Errorf("not playing any file")
	}

	// 移动到下一首
	m.currentIndex++

	// 检查是否到达播放列表末尾
	if m.currentIndex >= len(m.items) {
		m.currentIndex = len(m.items) - 1
		return fmt.Errorf("reached end of playlist")
	}

	return nil
}

// SetCurrent 设置当前播放项
func (m *manager) SetCurrent(index int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.items) {
		return "", fmt.Errorf("invalid index: %d", index)
	}

	m.currentIndex = index
	return m.items[m.currentIndex].FileID, nil
}

// IndexOf 根据 fileID 获取索引
func (m *manager) IndexOf(fileID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idx, ok := m.fileIDMap[fileID]; ok {
		return idx
	}
	return -1
}

// CurrentIndex 获取当前播放索引
func (m *manager) CurrentIndex() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentIndex
}

// ResetCurrent 重置当前播放索引
func (m *manager) ResetCurrent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentIndex = -1
}

// GetCurrent 获取当前播放项
func (m *manager) GetCurrent() *PlaylistItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentIndex < 0 || m.currentIndex >= len(m.items) {
		return nil
	}

	item := m.items[m.currentIndex]
	return &item
}

// GetAll 获取所有播放列表项
func (m *manager) GetAll() []PlaylistItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 返回副本以避免外部修改
	items := make([]PlaylistItem, len(m.items))
	copy(items, m.items)
	return items
}

// Clear 清空播放列表
func (m *manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make([]PlaylistItem, 0, MaxPlaylistItems)
	m.fileIDMap = make(map[string]int)
	m.currentIndex = -1
}

// rebuildIndexMap 重建索引映射（需要持有锁）
func (m *manager) rebuildIndexMap() {
	m.fileIDMap = make(map[string]int)
	for i := range m.items {
		m.items[i].Index = i
		m.fileIDMap[m.items[i].FileID] = i
	}
}
