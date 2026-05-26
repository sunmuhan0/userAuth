package actions

import (
	"log"

	"ttuser/async-handler/internal/sms"
)

// UserRegisteredReq 用户注册事件请求体
type UserRegisteredReq struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// UserRegistered 处理用户注册事件，发送短信
// 符合 func(req *T) error 签名，由 router.WrapHandleFunc 自动反序列化
func UserRegistered(req *UserRegisteredReq) error {
	log.Printf("[action] user registered: userID=%s, username=%s", req.UserID, req.Username)
	return sms.GetSender().SendRegistrationSMS(req.Phone, req.Username)
}
