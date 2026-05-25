package producer

// EventType 事件类型常量
const (
	EventUserRegistered = "user.registered" // 用户注册
	EventUserUpdated    = "user.updated"    // 用户信息更新
	EventUserDeleted    = "user.deleted"    // 用户注销
)

// Event 通用事件结构
type Event struct {
	Type    string      `json:"type"`     // 事件类型（即routing key）
	Payload interface{} `json:"payload"`  // 事件载荷
}

// UserRegisteredPayload 用户注册事件载荷
type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}
