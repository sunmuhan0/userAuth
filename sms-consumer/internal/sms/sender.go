package sms

import (
	"fmt"
	"log"
)

// SMSConfig 短信发送配置
// 当前先写死，后期从配置中心获取
type SMSConfig struct {
	// 短信服务商相关配置
	APIKey    string
	APISecret string
	SignName  string // 短信签名
	Template string // 短信模板
}

// DefaultConfig 默认短信配置（写死，后期从配置中心获取）
func DefaultConfig() *SMSConfig {
	return &SMSConfig{
		APIKey:    "your-api-key",
		APISecret: "your-api-secret",
		SignName:  "TT用户平台",
		Template:  "尊敬的%s，您已成功注册TT用户平台，欢迎使用！",
	}
}

// Sender 短信发送器
type Sender struct {
	config *SMSConfig
}

// NewSender 创建短信发送器
func NewSender(config *SMSConfig) *Sender {
	return &Sender{config: config}
}

// SendRegistrationSMS 发送注册成功短信
func (s *Sender) SendRegistrationSMS(phone, username string) error {
	if phone == "" {
		log.Printf("[sms-sender] phone is empty for user %s, skip sending SMS", username)
		return nil
	}

	content := fmt.Sprintf(s.config.Template, username)

	// TODO: 对接实际短信服务商SDK（如阿里云短信、腾讯云短信等）
	// 当前仅打印日志模拟发送
	log.Printf("[sms-sender] sending SMS to %s, content: %s (sign: %s)", phone, content, s.config.SignName)
	log.Printf("[sms-sender] SMS sent successfully to %s", phone)

	return nil
}
