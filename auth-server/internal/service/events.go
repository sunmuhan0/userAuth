package service

// RocketMQ Topic 和 Tag 常量
const (
	TopicUser         = "UserTopic"
	TagUserRegistered = "registered"
	TagUserUpdated    = "updated"
	TagUserDeleted    = "deleted"
)

// UserRegisteredPayload 用户注册事件载荷
type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}
