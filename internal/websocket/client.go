package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写入等待时间
	writeWait = 10 * time.Second

	// Pong 等待时间
	pongWait = 60 * time.Second

	// Ping 周期（必须小于 pongWait）
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

// Client WebSocket 客户端
type Client struct {
	// 客户端 ID
	id string

	// WebSocket 连接
	conn *websocket.Conn

	// Hub 引用
	hub *Hub

	// 发送消息的缓冲通道
	send chan []byte
}

// NewClient 创建新的客户端
func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		id:   id,
		conn: conn,
		hub:  hub,
		send: make(chan []byte, 256),
	}
}

// ReadPump 从 WebSocket 连接读取消息
// 这个方法在实际的 WebSocket 连接中运行，主要用于集成测试
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	if c.conn == nil {
		return
	}

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 记录错误（后续可以添加日志）
			}
			break
		}

		// 处理接收到的消息
		// 这里可以解析控制命令并发送到 controlQueue
		_ = message
	}
}

// WritePump 向 WebSocket 连接写入消息
// 这个方法在实际的 WebSocket 连接中运行，主要用于集成测试
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	if c.conn == nil {
		return
	}

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 将队列中的其他消息也一起发送
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

