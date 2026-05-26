package sms

import (
	"fmt"

	"github.com/teou/inji"

	configclient "ttuser/config-client/client"
)

type Config struct {
	APIKey    string
	APISecret string
	SignName  string
	Template  string
}

func (c *Config) Start() error {
	var smsConf struct {
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
		SignName  string `json:"sign_name"`
		Template  string `json:"template"`
	}
	svc := "async-handler"
	if v, ok := inji.Find("serverName"); ok {
		svc = v.(string)
	}
	if err := configclient.LoadFile(svc, "sms.json", &smsConf); err != nil {
		return fmt.Errorf("[smsConfig] load sms config failed: %w", err)
	}
	c.APIKey = smsConf.APIKey
	c.APISecret = smsConf.APISecret
	c.SignName = smsConf.SignName
	c.Template = smsConf.Template
	fmt.Printf("[smsConfig] initialized: signName=%s\n", c.SignName)
	return nil
}
