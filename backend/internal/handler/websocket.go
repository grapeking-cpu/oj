package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有 origin（开发环境）
		// 生产环境应该做更严格的检查
		return true
	},
}

type WSHub struct {
	rooms      map[string]map[*WSClient]bool
	register   chan *WSClient
	unregister chan *WSClient
	broadcast  chan *WSMessage
	mutex      sync.RWMutex
}

type WSClient struct {
	hub    *WSHub
	conn   *websocket.Conn
	send   chan []byte
	rooms  map[string]bool
	userID int64
}

type WSMessage struct {
	Type  string          `json:"type"`
	Topic string          `json:"topic"`
	Data  json.RawMessage `json:"data"`
}

func NewWSHub() *WSHub {
	return &WSHub{
		rooms:      make(map[string]map[*WSClient]bool),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		broadcast:  make(chan *WSMessage, 256),
	}
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if _, ok := h.rooms["lobby"]; !ok {
				h.rooms["lobby"] = make(map[*WSClient]bool)
			}
			h.rooms["lobby"][client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			for room := range client.rooms {
				if clients, ok := h.rooms[room]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						close(client.send)
						if len(clients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			h.mutex.Unlock()

		case msg := <-h.broadcast:
			h.mutex.RLock()
			if clients, ok := h.rooms[msg.Topic]; ok {
				for client := range clients {
					select {
					case client.send <- marshalMessage(msg):
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func marshalMessage(msg *WSMessage) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func (h *WSHub) Broadcast(topic string, message interface{}) {
	data, _ := json.Marshal(message)
	h.broadcast <- &WSMessage{
		Type:  topic,
		Topic: topic,
		Data:  data,
	}
}

func (h *WSHub) Subscribe(client *WSClient, topic string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.rooms[topic]; !ok {
		h.rooms[topic] = make(map[*WSClient]bool)
	}
	h.rooms[topic][client] = true
	client.rooms[topic] = true
}

func (h *WSHub) Unsubscribe(client *WSClient, topic string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, ok := h.rooms[topic]; ok {
		delete(clients, client)
		delete(client.rooms, topic)
		if len(clients) == 0 {
			delete(h.rooms, topic)
		}
	}
}

func HandleWebSocket(hub *WSHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 query 参数获取 token
		token := c.Query("token")
		if token == "" {
			// 尝试从 header 获取
			token = c.GetHeader("Authorization")
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
		}

		var userID int64
		// 验证 token
		if token != "" {
			// 在实际实现中需要验证 JWT
			// 这里简化处理，仅记录
			log.Printf("WebSocket connection with token")
		}

		// 升级为 WebSocket 连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		client := &WSClient{
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 256),
			rooms:  make(map[string]bool),
			userID: userID,
		}

		hub.register <- client

		// 启动读协程
		go client.readPump()
		// 启动写协程
		go client.writePump()
	}
}

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// 处理订阅消息
		if msg.Type == "subscribe" {
			c.hub.Subscribe(c, msg.Topic)
		} else if msg.Type == "unsubscribe" {
			c.hub.Unsubscribe(c, msg.Topic)
		}
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendStatusUpdate 发送状态更新
func (h *WSHub) SendStatusUpdate(submitID, status string, score int) {
	h.Broadcast("submit_status", gin.H{
		"submit_id": submitID,
		"status":    status,
		"score":     score,
	})
}

// SendRankUpdate 发送榜单更新
func (h *WSHub) SendRankUpdate(contestID int64, rankData interface{}) {
	topic := "contest:" + string(rune(contestID))
	h.Broadcast(topic, gin.H{
		"contest_id": contestID,
		"data":       rankData,
	})
}

// ValidateWSToken 验证 WebSocket JWT token
func ValidateWSToken(tokenString, jwtSecret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return token.Claims.(jwt.MapClaims), nil
}

// GenerateTestToken 生成测试用 Token
func GenerateTestToken(userID int64, username, role, jwtSecret string) string {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}
