package playlist

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddItem 测试添加文件到播放列表
func TestAddItem(t *testing.T) {
	m := NewManager()

	// 添加第一个文件
	err := m.Add("file1", "song1.mp3", 3*time.Minute)
	require.NoError(t, err)

	// 验证文件已添加
	items := m.GetAll()
	assert.Len(t, items, 1)
	assert.Equal(t, "file1", items[0].FileID)
	assert.Equal(t, "song1.mp3", items[0].Filename)
	assert.Equal(t, 3*time.Minute, items[0].Duration)
	assert.Equal(t, 0, items[0].Index)

	// 添加第二个文件
	err = m.Add("file2", "song2.mp3", 4*time.Minute)
	require.NoError(t, err)

	items = m.GetAll()
	assert.Len(t, items, 2)
	assert.Equal(t, 1, items[1].Index)
}

// TestAddDuplicateItem 测试添加重复文件
func TestAddDuplicateItem(t *testing.T) {
	m := NewManager()

	err := m.Add("file1", "song1.mp3", 3*time.Minute)
	require.NoError(t, err)

	// 尝试添加相同的文件
	err = m.Add("file1", "song1.mp3", 3*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestMaxItems 测试 50 个文件限制
func TestMaxItems(t *testing.T) {
	m := NewManager()

	// 添加 50 个文件
	for i := 0; i < 50; i++ {
		err := m.Add(
			fmt.Sprintf("file%d", i),
			fmt.Sprintf("song%d.mp3", i),
			time.Duration(i+1)*time.Minute,
		)
		require.NoError(t, err)
	}

	items := m.GetAll()
	assert.Len(t, items, 50)

	// 尝试添加第 51 个文件，应该失败
	err := m.Add("file51", "song51.mp3", 5*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum")

	// 验证仍然只有 50 个文件
	items = m.GetAll()
	assert.Len(t, items, 50)
}

// TestRemoveItem 测试移除文件
func TestRemoveItem(t *testing.T) {
	m := NewManager()

	// 添加三个文件
	m.Add("file1", "song1.mp3", 3*time.Minute)
	m.Add("file2", "song2.mp3", 4*time.Minute)
	m.Add("file3", "song3.mp3", 5*time.Minute)

	// 移除中间的文件
	err := m.Remove("file2")
	require.NoError(t, err)

	items := m.GetAll()
	assert.Len(t, items, 2)
	assert.Equal(t, "file1", items[0].FileID)
	assert.Equal(t, "file3", items[1].FileID)

	// 验证索引已重新计算
	assert.Equal(t, 0, items[0].Index)
	assert.Equal(t, 1, items[1].Index)
}

// TestRemoveNonExistentItem 测试移除不存在的文件
func TestRemoveNonExistentItem(t *testing.T) {
	m := NewManager()

	err := m.Remove("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestReorder 测试拖拽排序
func TestReorder(t *testing.T) {
	m := NewManager()

	// 添加五个文件
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)
	m.Add("file3", "song3.mp3", 3*time.Minute)
	m.Add("file4", "song4.mp3", 4*time.Minute)
	m.Add("file5", "song5.mp3", 5*time.Minute)

	// 将索引 1 的文件移动到索引 3
	err := m.Reorder(1, 3)
	require.NoError(t, err)

	items := m.GetAll()
	// 原顺序: file1, file2, file3, file4, file5
	// 新顺序: file1, file3, file4, file2, file5
	assert.Equal(t, "file1", items[0].FileID)
	assert.Equal(t, "file3", items[1].FileID)
	assert.Equal(t, "file4", items[2].FileID)
	assert.Equal(t, "file2", items[3].FileID)
	assert.Equal(t, "file5", items[4].FileID)

	// 验证索引正确
	for i, item := range items {
		assert.Equal(t, i, item.Index)
	}
}

// TestReorderInvalidIndex 测试无效的排序索引
func TestReorderInvalidIndex(t *testing.T) {
	m := NewManager()

	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)

	// 测试越界索引
	err := m.Reorder(-1, 1)
	assert.Error(t, err)

	err = m.Reorder(0, 10)
	assert.Error(t, err)
}

// TestNext 测试自动播放下一首
func TestNext(t *testing.T) {
	m := NewManager()

	// 添加三个文件
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)
	m.Add("file3", "song3.mp3", 3*time.Minute)

	// 第一次调用 Next，应该返回第一个文件
	fileID, err := m.Next()
	require.NoError(t, err)
	assert.Equal(t, "file1", fileID)

	// 验证当前播放项
	current := m.GetCurrent()
	require.NotNil(t, current)
	assert.Equal(t, "file1", current.FileID)

	// 第二次调用 Next，应该返回第二个文件
	fileID, err = m.Next()
	require.NoError(t, err)
	assert.Equal(t, "file2", fileID)

	// 第三次调用 Next，应该返回第三个文件
	fileID, err = m.Next()
	require.NoError(t, err)
	assert.Equal(t, "file3", fileID)

	// 第四次调用 Next，播放列表结束，应该返回错误
	fileID, err = m.Next()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end of playlist")
}

// TestNextEmptyPlaylist 测试空播放列表调用 Next
func TestNextEmptyPlaylist(t *testing.T) {
	m := NewManager()

	fileID, err := m.Next()
	assert.Error(t, err)
	assert.Empty(t, fileID)
	assert.Contains(t, err.Error(), "empty")
}

// TestSkip 测试跳过当前文件
func TestSkip(t *testing.T) {
	m := NewManager()

	// 添加三个文件
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)
	m.Add("file3", "song3.mp3", 3*time.Minute)

	// 开始播放第一个文件
	m.Next()

	// 跳过当前文件，应该播放第二个文件
	err := m.Skip()
	require.NoError(t, err)

	current := m.GetCurrent()
	require.NotNil(t, current)
	assert.Equal(t, "file2", current.FileID)

	// 再次跳过
	err = m.Skip()
	require.NoError(t, err)

	current = m.GetCurrent()
	require.NotNil(t, current)
	assert.Equal(t, "file3", current.FileID)

	// 最后一首歌跳过后，应该返回错误
	err = m.Skip()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end of playlist")
}

// TestSkipWithoutPlaying 测试未开始播放时跳过
func TestSkipWithoutPlaying(t *testing.T) {
	m := NewManager()

	m.Add("file1", "song1.mp3", 1*time.Minute)

	err := m.Skip()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not playing")
}

// TestGetCurrent 测试获取当前播放项
func TestGetCurrent(t *testing.T) {
	m := NewManager()

	// 未开始播放时，应该返回 nil
	current := m.GetCurrent()
	assert.Nil(t, current)

	// 添加文件并开始播放
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Next()

	current = m.GetCurrent()
	require.NotNil(t, current)
	assert.Equal(t, "file1", current.FileID)
	assert.Equal(t, "song1.mp3", current.Filename)
}

// TestClear 测试清空播放列表
func TestClear(t *testing.T) {
	m := NewManager()

	// 添加文件并开始播放
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)
	m.Next()

	// 清空播放列表
	m.Clear()

	items := m.GetAll()
	assert.Len(t, items, 0)

	current := m.GetCurrent()
	assert.Nil(t, current)
}

// TestRemoveCurrentlyPlaying 测试移除正在播放的文件
func TestRemoveCurrentlyPlaying(t *testing.T) {
	m := NewManager()

	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)
	m.Add("file3", "song3.mp3", 3*time.Minute)

	// 开始播放第一个文件
	m.Next()

	// 移除正在播放的文件
	err := m.Remove("file1")
	require.NoError(t, err)

	// 当前播放应该被清除
	current := m.GetCurrent()
	assert.Nil(t, current)

	// 播放列表应该只剩两个文件
	items := m.GetAll()
	assert.Len(t, items, 2)
}

func TestSetCurrentPrevAndReset(t *testing.T) {
	m := NewManager()
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)
	m.Add("file3", "song3.mp3", 3*time.Minute)

	fileID, err := m.SetCurrent(1)
	require.NoError(t, err)
	assert.Equal(t, "file2", fileID)
	assert.Equal(t, 1, m.CurrentIndex())

	fileID, err = m.Prev()
	require.NoError(t, err)
	assert.Equal(t, "file1", fileID)
	assert.Equal(t, 0, m.CurrentIndex())

	_, err = m.Prev()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "beginning")

	m.ResetCurrent()
	assert.Equal(t, -1, m.CurrentIndex())
	assert.Nil(t, m.GetCurrent())
}

func TestIndexOf(t *testing.T) {
	m := NewManager()
	m.Add("file1", "song1.mp3", 1*time.Minute)
	m.Add("file2", "song2.mp3", 2*time.Minute)

	assert.Equal(t, 0, m.IndexOf("file1"))
	assert.Equal(t, 1, m.IndexOf("file2"))
	assert.Equal(t, -1, m.IndexOf("missing"))
}
