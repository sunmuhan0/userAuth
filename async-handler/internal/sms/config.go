package sms

import "fmt"

// Config 短信发送配置
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type Config struct {
	APIKey    string
	APISecret string
	SignName  string
	Template  string
}

// Start 实现 inji.Startable 接口
func (c *Config) Start() error {
	c.APIKey = "your-api-key"
	c.APISecret = "your-api-secret"
	c.SignName = "TT用户平台"
	c.Template = "尊敬的%s，您已成功注册TT用户平台，欢迎使用！"
	fmt.Println("[smsConfig] initialized")
	return nil
}
