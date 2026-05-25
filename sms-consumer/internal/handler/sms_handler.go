package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"ttuser/sms-consumer/internal/sms"
)

// Event 通用事件结构（与event-producer对应）
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// UserRegisteredPayload 用户注册事件载荷
type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// SMSHandler 短信事件处理器，实现 event-consumer 的 IEventHandler 接口
type SMSHandler struct {
	Sender *sms.Sender
}

// NewSMSHandler 创建短信处理器
func NewSMSHandler(sender *sms.Sender) *SMSHandler {
	return &SMSHandler{Sender: sender}
}

// Handle 处理原始消息体
func (h *SMSHandler) Handle(body []byte) error {
	// 解析通用事件包装
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("unmarshal event failed: %w", err)
	}

	// 根据事件类型分发
	switch event.Type {
	case "user.registered":
		return h.handleUserRegistered(event.Payload)
	default:
		log.Printf("[sms-handler] unknown event type: %s, skip", event.Type)
		return nil
	}
}

// handleUserRegistered 处理用户注册事件
func (h *SMSHandler) handleUserRegistered(raw json.RawMessage) error {
	var payload UserRegisteredPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("unmarshal user registered payload failed: %w", err)
	}

	log.Printf("[sms-handler] received user registered: userID=%s, username=%s", payload.UserID, payload.Username)
	return h.Sender.SendRegistrationSMS(payload.Phone, payload.Username)
}
