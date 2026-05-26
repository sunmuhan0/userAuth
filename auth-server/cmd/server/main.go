package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teou/inji"

	"ttuser/auth-server/sp"
	configclient "ttuser/config-client/client"
	"ttuser/pkg/log"
)

func main() {
	name := flag.String("name", "auth-server", "service name")
	port := flag.Int("port", 9090, "service port")
	env := flag.String("env", "prod", "deploy environment: prod/staging/preview")
	flag.Parse()

	// ========== 初始化inji依赖注入容器 ==========
	inji.InitDefault()

	// ========== 注册服务标识与运行环境 ==========
	inji.Reg("serverName", *name)
	inji.Reg("serverPort", fmt.Sprintf("%d", *port))
	inji.Reg("env", *env)

	// ========== 初始化日志 ==========
	log.Init(nil)
	defer log.Sync()

	// ========== 从配置中心拉取配置文件 ==========
	cc := configclient.New(&configclient.Config{
		Env:         *env,
		ServiceName: *name,
	})
	if err := cc.FetchConfigs(); err != nil {
		fmt.Printf("[auth-server] fetch configs failed: %v\n", err)
		os.Exit(1)
	}

	// ========== 注册ServiceProvider ==========
	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	sp.Init()

	// ========== 启动gRPC服务 ==========
	go func() {
		if err := sp.Get().GRPCServer.Run(); err != nil {
			fmt.Printf("[auth-server] failed to start: %v\n", err)
			os.Exit(1)
		}
	}()

	// ========== 等待信号 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	fmt.Printf("\n[auth-server] received signal %v, shutting down...\n", sig)

	// 1. 优雅停止gRPC服务
	stopDone := make(chan struct{})
	go func() {
		sp.Get().GRPCServer.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
		fmt.Println("[auth-server] gRPC server stopped gracefully")
	case <-time.After(10 * time.Second):
		fmt.Println("[auth-server] gRPC server stop timeout, force shutdown")
	}

	// 2. 关闭依赖注入容器
	inji.Close()

	fmt.Println("[auth-server] stopped")
}
