package sms

import (
	"fmt"

	configclient "ttuser/config-client/client"
)

// Config 短信发送配置
// Start时从配置中心获取，获取失败则用默认值
type Config struct {
	APIKey    string
	APISecret string
	SignName  string
	Template  string
}

// Start 实现 inji.Startable 接口
func (c *Config) Start() error {
	cfg := configclient.DefaultConfig()
	cfg.ServiceName = "async-handler"
	cc := configclient.New(cfg)
	if err := cc.Start(0); err != nil {
		fmt.Printf("[smsConfig] config-center unavailable, using defaults: %v\n", err)
		c.setDefaults()
	} else {
		var smsConf struct {
			APIKey    string `json:"api_key"`
			APISecret string `json:"api_secret"`
			SignName  string `json:"sign_name"`
			Template  string `json:"template"`
		}
		if err := cc.Get("sms", &smsConf); err != nil {
			fmt.Printf("[smsConfig] config key 'sms' not found, using defaults: %v\n", err)
			c.setDefaults()
		} else {
			c.APIKey = smsConf.APIKey
			c.APISecret = smsConf.APISecret
			c.SignName = smsConf.SignName
			c.Template = smsConf.Template
		}
	}
	fmt.Printf("[smsConfig] initialized: signName=%s\n", c.SignName)
	return nil
}

func (c *Config) setDefaults() {
	c.APIKey = "your-api-key"
	c.APISecret = "your-api-secret"
	c.SignName = "TT用户平台"
	c.Template = "尊敬的%s，您已成功注册TT用户平台，欢迎使用！"
}
