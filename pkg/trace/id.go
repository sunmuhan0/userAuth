package trace

import (
	"crypto/rand"
	"encoding/hex"
)

// NewTraceID 生成32位hex trace_id（128-bit随机数）
// 格式：0af7651916cd43dd8448eb211c80319c
func NewTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
