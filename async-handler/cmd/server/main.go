package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teou/inji"

	"ttuser/async-handler/sp"
	configclient "ttuser/config-client/client"
	"ttuser/pkg/log"
)

func main() {
	name := flag.String("name", "async-handler", "service name")
	env := flag.String("env", "prod", "deploy environment: prod/staging/preview")
	flag.Parse()

	fmt.Println("[sms-consumer] starting...")

	// ========== 初始化inji依赖注入容器 ==========
	inji.InitDefault()

	// ========== 注册服务标识与运行环境 ==========
	inji.Reg("serverName", *name)
	inji.Reg("env", *env)

	// ========== 初始化日志 ==========
	log.Init(nil)
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Printf("[sms-consumer] log sync error: %v\n", err)
		}
	}()

	// ========== 从配置中心拉取配置文件 ==========
	cc := configclient.New(&configclient.Config{
		Env:         *env,
		ServiceName: *name,
	})
	if err := cc.FetchConfigs(); err != nil {
		fmt.Printf("[sms-consumer] fetch configs failed: %v\n", err)
		log.Sync()
		os.Exit(1)
	}

	// ========== 注册ServiceProvider ==========
	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	sp.Init()

	fmt.Println("[sms-consumer] started successfully")

	// ========== 等待信号 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	fmt.Printf("\n[sms-consumer] received signal %v, shutting down...\n", sig)

	closeDone := make(chan struct{})
	go func() {
		if err := inji.Close(); err != nil {
			fmt.Printf("[sms-consumer] inji close error: %v\n", err)
		}
		close(closeDone)
	}()
	select {
	case <-closeDone:
		fmt.Println("[sms-consumer] resources closed gracefully")
	case <-time.After(10 * time.Second):
		fmt.Println("[sms-consumer] resource close timeout, force shutdown")
	}

	fmt.Println("[sms-consumer] stopped")
}
