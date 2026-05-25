package sms

import (
	"fmt"
	"log"
)

// Sender 短信发送器
type Sender struct {
	Config *Config `inject:"smsConfig"`
}

// SendRegistrationSMS 发送注册成功短信
func (s *Sender) SendRegistrationSMS(phone, username string) error {
	if phone == "" {
		log.Printf("[sms-sender] phone is empty for user %s, skip sending SMS", username)
		return nil
	}

	content := fmt.Sprintf(s.Config.Template, username)

	// TODO: 对接实际短信服务商SDK（如阿里云短信、腾讯云短信等）
	// 当前仅打印日志模拟发送
	log.Printf("[sms-sender] sending SMS to %s, content: %s (sign: %s)", phone, content, s.Config.SignName)
	log.Printf("[sms-sender] SMS sent successfully to %s", phone)

	return nil
}
