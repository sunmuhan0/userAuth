package producer

// Event 通用事件结构
// 调用方自行定义Payload的具体struct，通过JSON序列化/反序列化解耦
type Event struct {
	Type    string      `json:"type"`    // 事件类型
	Payload interface{} `json:"payload"` // 事件载荷（任意结构体）
}
