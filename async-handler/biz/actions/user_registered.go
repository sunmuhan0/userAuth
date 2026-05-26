package actions

import (
	"context"

	"go.uber.org/zap"

	"ttuser/async-handler/internal/sms"
	"ttuser/pkg/log"
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
func UserRegistered(ctx context.Context, req *UserRegisteredReq) error {
	log.Info(ctx, "user registered event received",
		zap.String("user_id", req.UserID),
		zap.String("username", req.Username),
	)
	return sms.GetSender().SendRegistrationSMS(req.Phone, req.Username)
}
