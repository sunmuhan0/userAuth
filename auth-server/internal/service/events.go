package service

// 事件类型常量（本服务发布的事件routing key）
const (
	EventTypeUserRegistered = "user.registered"
	EventTypeUserUpdated    = "user.updated"
	EventTypeUserDeleted    = "user.deleted"
)

// UserRegisteredPayload 用户注册事件载荷
type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}
