package nacos

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type Config struct {
	ServerAddr  string `json:"server_addr"`
	ServerPort  uint64 `json:"server_port"`
	NamespaceID string `json:"namespace_id"`
	LogDir      string `json:"log_dir"`
	CacheDir    string `json:"cache_dir"`
}

func NewNamingClient(cfg *Config) (naming_client.INamingClient, error) {
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = "127.0.0.1"
	}
	if cfg.ServerPort == 0 {
		cfg.ServerPort = 8848
	}
	if cfg.NamespaceID == "" {
		cfg.NamespaceID = "public"
	}

	cc := constant.NewClientConfig(
		constant.WithNamespaceId(cfg.NamespaceID),
		constant.WithTimeoutMs(5000),
		constant.WithBeatInterval(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(cfg.LogDir),
		constant.WithCacheDir(cfg.CacheDir),
	)
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(cfg.ServerAddr, cfg.ServerPort),
	}

	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return nil, fmt.Errorf("new naming client failed: %w", err)
	}
	return client, nil
}
