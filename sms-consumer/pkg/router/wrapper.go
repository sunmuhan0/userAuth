package router

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// WrapHandleFunc 将强类型的action函数包装为Handler接口
// 支持的函数签名：func(req *T) error
// 自动完成 JSON body → *T 的反序列化
func WrapHandleFunc(fn interface{}) (Handler, error) {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("WrapHandleFunc: expected func, got %s", fnType.Kind())
	}
	if fnType.NumIn() != 1 {
		return nil, fmt.Errorf("WrapHandleFunc: expected func with 1 param, got %d", fnType.NumIn())
	}
	if fnType.NumOut() != 1 {
		return nil, fmt.Errorf("WrapHandleFunc: expected func with 1 return (error), got %d", fnType.NumOut())
	}

	// 第一个参数必须是指针类型
	paramType := fnType.In(0)
	if paramType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("WrapHandleFunc: first param must be a pointer, got %s", paramType.Kind())
	}

	return &funcHandler{
		fn:        fnValue,
		paramType: paramType,
	}, nil
}

// funcHandler 将函数包装为Handler
type funcHandler struct {
	fn        reflect.Value
	paramType reflect.Type
}

func (h *funcHandler) Handle(body []byte) error {
	// 创建请求对象并反序列化
	reqPtr := reflect.New(h.paramType.Elem())
	if err := json.Unmarshal(body, reqPtr.Interface()); err != nil {
		return fmt.Errorf("unmarshal request failed: %w", err)
	}

	// 调用函数
	results := h.fn.Call([]reflect.Value{reqPtr})

	// 处理返回值
	errVal := results[0]
	if errVal.IsNil() {
		return nil
	}
	return errVal.Interface().(error)
}
