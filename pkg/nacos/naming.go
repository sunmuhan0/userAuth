package nacos

import (
	"fmt"
	"net"
	"os"
	"reflect"

	configclient "ttuser/config-client/client"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/teou/implmap"
)

func init() {
	implmap.Add("serviceRegistrar", reflect.TypeOf((*Registry)(nil)))
}

// IServiceRegistrar 服务注册接口
// 业务层依赖此接口，由基础设施层（如 Nacos）实现
type IServiceRegistrar interface {
	Start() error
	Close()
}

type Config struct {
	ServerAddr  string `json:"server_addr"`
	ServerPort  uint64 `json:"server_port"`
	NamespaceID string `json:"namespace_id"`
	LogDir      string `json:"log_dir"`
	CacheDir    string `json:"cache_dir"`
}

type Registry struct {
	client      naming_client.INamingClient
	serviceName string
	ip          string
	port        uint64
	ServerName  string `inject:"serverName"`
	PortStr     string `inject:"serverPort"`
}

func (r *Registry) Start() error {
	if r.client == nil {
		if err := r.initLazy(); err != nil {
			return err
		}
	}
	if r.client == nil {
		return nil
	}
	if _, err := r.client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          r.ip,
		Port:        r.port,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		ServiceName: r.serviceName,
		GroupName:   "DEFAULT_GROUP",
		Ephemeral:   true,
	}); err != nil {
		return fmt.Errorf("register to nacos failed: %w", err)
	}
	fmt.Printf("[nacos] registered: %s -> %s:%d\n", r.serviceName, r.ip, r.port)
	return nil
}

func (r *Registry) Close() {
	if r.client == nil {
		return
	}
	if _, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          r.ip,
		Port:        r.port,
		ServiceName: r.serviceName,
		GroupName:   "DEFAULT_GROUP",
		Ephemeral:   true,
	}); err != nil {
		fmt.Printf("[nacos] deregister error: %v\n", err)
		return
	}
	fmt.Printf("[nacos] deregistered: %s -> %s:%d\n", r.serviceName, r.ip, r.port)
}

func (r *Registry) initLazy() error {
	if os.Getenv("NACOS_DISABLE") == "true" {
		fmt.Println("[nacos] skip registration (NACOS_DISABLE=true)")
		return nil
	}

	serviceName := r.ServerName
	if serviceName == "" {
		return fmt.Errorf("serverName not injected")
	}

	ip := getLocalIP()

	port := uint64(9090)
	if r.PortStr != "" {
		fmt.Sscanf(r.PortStr, "%d", &port)
	}

	var cfg Config
	if err := configclient.LoadFile(serviceName, "nacos.json", &cfg); err != nil {
		fmt.Printf("[nacos] skip registration (no config): %v\n", err)
		return nil
	}
	if cfg.ServerAddr == "" || cfg.ServerPort == 0 {
		fmt.Println("[nacos] skip registration (server_addr empty)")
		return nil
	}
	client, err := newNamingClient(&cfg)
	if err != nil {
		return err
	}
	r.client = client
	r.serviceName = serviceName
	r.ip = ip
	r.port = port
	return nil
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

func Discover(myServiceName, targetServiceName string) (string, error) {
	if os.Getenv("NACOS_DISABLE") == "true" {
		return "", fmt.Errorf("nacos disabled")
	}
	var cfg Config
	if err := configclient.LoadFile(myServiceName, "nacos.json", &cfg); err != nil {
		return "", fmt.Errorf("load nacos config failed: %w", err)
	}
	client, err := newNamingClient(&cfg)
	if err != nil {
		return "", err
	}
	defer client.CloseClient()

	instance, err := client.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: targetServiceName,
		GroupName:   "DEFAULT_GROUP",
		Clusters:    []string{"DEFAULT"},
	})
	if err != nil {
		return "", fmt.Errorf("discover %s failed: %w", targetServiceName, err)
	}
	if instance == nil {
		return "", fmt.Errorf("no healthy instance found for %s", targetServiceName)
	}
	return fmt.Sprintf("%s:%d", instance.Ip, instance.Port), nil
}

func newNamingClient(cfg *Config) (naming_client.INamingClient, error) {
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
