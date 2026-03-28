package websocket

import (
	"sync"

	apperrors "github.com/liigoo/pi-fm-rds-go/internal/errors"
)

// Hub WebSocket 连接管理中心
type Hub struct {
	// 已注册的客户端
	clients map[*Client]bool

	// 广播消息到所有客户端
	broadcast chan []byte

	// 注册客户端请求
	register chan *Client

	// 注销客户端请求
	unregister chan *Client

	// 控制命令队列（串行化处理）
	controlQueue chan ControlMessage

	// 最大客户端数量
	maxClients int

	// 读写锁
	mu sync.RWMutex

	// 停止信号
	stop chan struct{}
}

// ControlMessage 控制消息结构
type ControlMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// NewHub 创建新的 Hub
func NewHub(maxClients int) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		controlQueue: make(chan ControlMessage, 100),
		maxClients:   maxClients,
		stop:         make(chan struct{}),
	}
}

// Run 启动 Hub 主循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.handleBroadcast(message)

		case controlMsg := <-h.controlQueue:
			h.handleControlMessage(controlMsg)

		case <-h.stop:
			return
		}
	}
}

// handleRegister 处理客户端注册
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查客户端数量限制
	if len(h.clients) >= h.maxClients {
		// 拒绝注册，关闭客户端连接
		if client.send != nil {
			close(client.send)
		}
		return
	}

	h.clients[client] = true
}

// handleUnregister 处理客户端注销
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		if client.send != nil {
			close(client.send)
		}
	}
}

// handleBroadcast 处理广播消息
func (h *Hub) handleBroadcast(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
			// 消息发送成功
		default:
			// 客户端发送缓冲区已满，跳过此客户端
			// 避免阻塞其他客户端
		}
	}
}

// handleControlMessage 处理控制命令（串行化）
func (h *Hub) handleControlMessage(msg ControlMessage) {
	// 这里实现控制命令的处理逻辑
	// 由于是串行化处理，自动实现 last-write-wins
	// 具体的命令处理逻辑将在后续集成时实现

	// 广播控制命令到所有客户端（示例实现）
	// 实际项目中，这里会调用播放器控制逻辑
	_ = msg // 使用 msg 避免未使用警告
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	close(h.stop)
}

// GetClientCount 获取当前客户端数量
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast 广播消息到所有客户端
func (h *Hub) Broadcast(message []byte) error {
	select {
	case h.broadcast <- message:
		return nil
	default:
		return apperrors.New(apperrors.ErrWebSocketDisconnected, "broadcast channel full")
	}
}

// SendControl 发送控制命令
func (h *Hub) SendControl(msg ControlMessage) error {
	select {
	case h.controlQueue <- msg:
		return nil
	default:
		return apperrors.New(apperrors.ErrWebSocketDisconnected, "control queue full")
	}
}
