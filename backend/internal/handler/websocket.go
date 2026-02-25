package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type WSHub struct {
	rooms    map[string]map[*WSClient]bool
	register chan *WSClient
	unregister chan *WSClient
	broadcast chan *WSMessage
	mutex    sync.RWMutex
}

type WSClient struct {
	hub    *WSHub
	conn   interface{} // websocket.Conn
	send   chan []byte
	rooms  map[string]bool
}

type WSMessage struct {
	Type    string          `json:"type"`
	Topic   string          `json:"topic"`
	Data    json.RawMessage `json:"data"`
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
		}
	}
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

		// 验证 token (简化版)
		if token != "" {
			// 在实际实现中需要验证 JWT
			log.Printf("WebSocket connection with token")
		}

		// 升级为 WebSocket 连接
		// 这里需要使用 gorilla/websocket 或 nhooyr.io/websocket
		// 简化处理，返回一个模拟的响应
		c.JSON(http.StatusUpgradeRequired, gin.H{
			"code":    0,
			"message": "WebSocket upgrade required",
		})
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

// JWT 验证
func ValidateWSToken(tokenString, jwtSecret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return token.Claims.(jwt.MapClaims), nil
}

// 生成 Token (用于测试)
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
