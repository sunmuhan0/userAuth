package producer

import (
	"reflect"

	"github.com/teou/implmap"
)

func init() {
	// 注册 IEventPublisher 接口的实现
	// inji通过implmap将 inject:"eventPublisher" 映射到 *RMQPublisher
	implmap.Add("eventPublisher", reflect.TypeOf((*RMQPublisher)(nil)))
}
