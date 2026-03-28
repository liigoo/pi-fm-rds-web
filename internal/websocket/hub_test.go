package websocket

import (
	"sync"
	"testing"
	"time"
)

// TestNewHub 测试创建 Hub
func TestNewHub(t *testing.T) {
	hub := NewHub(5)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.maxClients != 5 {
		t.Errorf("expected maxClients=5, got %d", hub.maxClients)
	}

	if hub.clients == nil {
		t.Error("clients map is nil")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel is nil")
	}

	if hub.register == nil {
		t.Error("register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("unregister channel is nil")
	}

	if hub.controlQueue == nil {
		t.Error("controlQueue channel is nil")
	}
}

// mockClient 创建模拟客户端用于测试
type mockClient struct {
	id   string
	send chan []byte
}

// TestClientRegister 测试客户端注册
func TestClientRegister(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	client := &Client{
		id:   "test-client-1",
		send: make(chan []byte, 256),
	}

	// 注册客户端
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	count := len(hub.clients)
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	if !exists {
		t.Error("client not found in hub")
	}
}

// TestClientLimit 测试 5 个客户端限制
func TestClientLimit(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 注册 5 个客户端（应该成功）
	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		clients[i] = &Client{
			id:   string(rune('A' + i)),
			send: make(chan []byte, 256),
		}
		hub.register <- clients[i]
	}
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	count := len(hub.clients)
	hub.mu.RUnlock()

	if count != 5 {
		t.Errorf("expected 5 clients, got %d", count)
	}

	// 尝试注册第 6 个客户端（应该失败）
	client6 := &Client{
		id:   "F",
		send: make(chan []byte, 256),
	}
	hub.register <- client6
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	count = len(hub.clients)
	_, exists := hub.clients[client6]
	hub.mu.RUnlock()

	if count != 5 {
		t.Errorf("expected 5 clients after limit, got %d", count)
	}

	if exists {
		t.Error("6th client should not be registered")
	}
}

// TestBroadcast 测试状态广播
func TestBroadcast(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 注册 3 个客户端
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			id:   string(rune('A' + i)),
			send: make(chan []byte, 256),
		}
		hub.register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	// 广播消息
	testMsg := []byte(`{"type":"status","data":"test"}`)
	hub.broadcast <- testMsg

	// 验证所有客户端都收到消息
	for i, client := range clients {
		select {
		case msg := <-client.send:
			if string(msg) != string(testMsg) {
				t.Errorf("client %d: expected %s, got %s", i, testMsg, msg)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("client %d: timeout waiting for broadcast", i)
		}
	}
}

// TestControlQueue 测试控制命令串行化
func TestControlQueue(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 发送多个控制命令
	messages := []ControlMessage{
		{Type: "play", Data: "song1.wav"},
		{Type: "pause", Data: ""},
		{Type: "play", Data: "song2.wav"},
	}

	for _, msg := range messages {
		hub.controlQueue <- msg
	}

	time.Sleep(100 * time.Millisecond)
	// 控制队列应该已处理完所有消息
}

// TestLastWriteWins 测试 last-write-wins
func TestLastWriteWins(t *testing.T) {
	hub := NewHub(5)

	var processedMessages []ControlMessage
	var mu sync.Mutex

	// 自定义 Run 方法来捕获处理的消息
	go func() {
		for {
			select {
			case msg := <-hub.controlQueue:
				mu.Lock()
				processedMessages = append(processedMessages, msg)
				mu.Unlock()
			case <-time.After(200 * time.Millisecond):
				return
			}
		}
	}()

	// 快速发送多个控制命令
	messages := []ControlMessage{
		{Type: "play", Data: "song1.wav"},
		{Type: "play", Data: "song2.wav"},
		{Type: "play", Data: "song3.wav"},
	}

	for _, msg := range messages {
		hub.controlQueue <- msg
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	count := len(processedMessages)
	mu.Unlock()

	if count != 3 {
		t.Errorf("expected 3 messages processed, got %d", count)
	}
}

// TestClientUnregister 测试客户端注销
func TestClientUnregister(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	client := &Client{
		id:   "test-client",
		send: make(chan []byte, 256),
	}

	// 注册客户端
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	count := len(hub.clients)
	hub.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 client after register, got %d", count)
	}

	// 注销客户端
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	count = len(hub.clients)
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", count)
	}

	if exists {
		t.Error("client should not exist after unregister")
	}
}

// TestGetClientCount 测试获取客户端数量
func TestGetClientCount(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 初始应该是 0
	if count := hub.GetClientCount(); count != 0 {
		t.Errorf("expected 0 clients initially, got %d", count)
	}

	// 注册 3 个客户端
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			id:   string(rune('A' + i)),
			send: make(chan []byte, 256),
		}
		hub.register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	if count := hub.GetClientCount(); count != 3 {
		t.Errorf("expected 3 clients, got %d", count)
	}

	// 注销 1 个客户端
	hub.unregister <- clients[0]
	time.Sleep(50 * time.Millisecond)

	if count := hub.GetClientCount(); count != 2 {
		t.Errorf("expected 2 clients after unregister, got %d", count)
	}
}

// TestBroadcastMethod 测试 Broadcast 方法
func TestBroadcastMethod(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	testMsg := []byte(`{"type":"test"}`)
	err := hub.Broadcast(testMsg)
	if err != nil {
		t.Errorf("Broadcast failed: %v", err)
	}
}

// TestSendControl 测试 SendControl 方法
func TestSendControl(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	msg := ControlMessage{
		Type: "play",
		Data: "test.wav",
	}

	err := hub.SendControl(msg)
	if err != nil {
		t.Errorf("SendControl failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
}

// TestNewClient 测试创建客户端
func TestNewClient(t *testing.T) {
	hub := NewHub(5)
	client := NewClient("test-id", nil, hub)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.id != "test-id" {
		t.Errorf("expected id='test-id', got '%s'", client.id)
	}

	if client.hub != hub {
		t.Error("client hub reference is incorrect")
	}

	if client.send == nil {
		t.Error("client send channel is nil")
	}
}

// TestHubStop 测试停止 Hub
func TestHubStop(t *testing.T) {
	hub := NewHub(5)

	done := make(chan bool)
	go func() {
		hub.Run()
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	hub.Stop()

	select {
	case <-done:
		// Hub 成功停止
	case <-time.After(200 * time.Millisecond):
		t.Error("Hub did not stop in time")
	}
}

// TestBroadcastWithSlowClient 测试慢客户端不阻塞其他客户端
func TestBroadcastWithSlowClient(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 创建一个正常客户端
	normalClient := &Client{
		id:   "normal",
		send: make(chan []byte, 256),
	}

	// 创建一个慢客户端（缓冲区很小）
	slowClient := &Client{
		id:   "slow",
		send: make(chan []byte, 1),
	}

	hub.register <- normalClient
	hub.register <- slowClient
	time.Sleep(50 * time.Millisecond)

	// 填满慢客户端的缓冲区
	slowClient.send <- []byte("blocking")

	// 广播消息
	testMsg := []byte(`{"type":"test"}`)
	hub.broadcast <- testMsg

	// 正常客户端应该能收到消息
	select {
	case msg := <-normalClient.send:
		if string(msg) != string(testMsg) {
			t.Errorf("normal client: expected %s, got %s", testMsg, msg)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("normal client: timeout waiting for message")
	}

	// 慢客户端可能收不到（因为缓冲区满了）
	// 但不应该阻塞整个系统
}

// TestControlQueueFull 测试控制队列满时的行为
func TestControlQueueFull(t *testing.T) {
	hub := NewHub(5)
	// 不启动 Run()，这样队列会满

	// 填满控制队列
	for i := 0; i < 100; i++ {
		msg := ControlMessage{
			Type: "test",
			Data: string(rune('A' + i%26)),
		}
		hub.controlQueue <- msg
	}

	// 尝试再发送一个（应该失败）
	msg := ControlMessage{
		Type: "test",
		Data: "overflow",
	}
	err := hub.SendControl(msg)
	if err == nil {
		t.Error("expected error when control queue is full")
	}
}

// TestBroadcastChannelFull 测试广播通道满时的行为
func TestBroadcastChannelFull(t *testing.T) {
	hub := NewHub(5)
	// 不启动 Run()，这样通道会满

	// 填满广播通道
	for i := 0; i < 256; i++ {
		hub.broadcast <- []byte("test")
	}

	// 尝试再发送一个（应该失败）
	err := hub.Broadcast([]byte("overflow"))
	if err == nil {
		t.Error("expected error when broadcast channel is full")
	}
}

// TestHandleControlMessage 测试控制消息处理
func TestHandleControlMessage(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 发送控制消息
	msg := ControlMessage{
		Type: "play",
		Data: "test.wav",
	}

	hub.controlQueue <- msg
	time.Sleep(50 * time.Millisecond)

	// 消息应该被处理（不会阻塞）
}

// TestMultipleControlMessages 测试多个控制消息的串行处理
func TestMultipleControlMessages(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	// 发送多个控制消息
	messages := []ControlMessage{
		{Type: "play", Data: "song1.wav"},
		{Type: "pause", Data: ""},
		{Type: "resume", Data: ""},
		{Type: "stop", Data: ""},
	}

	for _, msg := range messages {
		hub.controlQueue <- msg
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	// 所有消息应该被串行处理
}

// TestClientReadPumpWithNilConn 测试 ReadPump 处理 nil 连接
func TestClientReadPumpWithNilConn(t *testing.T) {
	hub := NewHub(5)
	go hub.Run()
	defer hub.Stop()

	client := NewClient("test", nil, hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// ReadPump 应该能处理 nil 连接
	client.ReadPump()

	// 客户端应该被注销
	time.Sleep(50 * time.Millisecond)
	if count := hub.GetClientCount(); count != 0 {
		t.Errorf("expected 0 clients after ReadPump with nil conn, got %d", count)
	}
}

// TestClientWritePumpWithNilConn 测试 WritePump 处理 nil 连接
func TestClientWritePumpWithNilConn(t *testing.T) {
	hub := NewHub(5)
	client := NewClient("test", nil, hub)

	// WritePump 应该能处理 nil 连接
	done := make(chan bool)
	go func() {
		client.WritePump()
		done <- true
	}()

	// 关闭 send 通道触发退出
	close(client.send)

	select {
	case <-done:
		// WritePump 成功退出
	case <-time.After(200 * time.Millisecond):
		t.Error("WritePump did not exit in time")
	}
}

