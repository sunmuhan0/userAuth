package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"ttuser/sms-consumer/internal/sms"
)

// UserRegisteredPayload 用户注册事件载荷
type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// SMSHandler 短信事件处理器
// 消息体即为payload JSON（无Event包装），routing key已由RMQ层面路由到对应队列
type SMSHandler struct {
	Sender *sms.Sender `inject:"smsSender"`
}

// Handle 处理原始消息体（直接就是payload JSON）
func (h *SMSHandler) Handle(body []byte) error {
	var payload UserRegisteredPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("unmarshal payload failed: %w", err)
	}

	log.Printf("[sms-handler] received user registered: userID=%s, username=%s", payload.UserID, payload.Username)
	return h.Sender.SendRegistrationSMS(payload.Phone, payload.Username)
}
