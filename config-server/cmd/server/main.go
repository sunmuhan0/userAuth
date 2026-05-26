package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teou/inji"

	"ttuser/config-server/internal/service"
	"ttuser/config-server/sp"
	"ttuser/pkg/log"
)

func main() {
	name := flag.String("name", "config-server", "service name")
	port := flag.Int("port", 7963, "service port")
	env := flag.String("env", "prod", "deploy environment: prod/staging/preview")
	flag.Parse()

	fmt.Println("[config-server] starting...")

	// ========== 初始化inji依赖注入容器 ==========
	inji.InitDefault()

	// ========== 注册服务标识与运行环境 ==========
	inji.Reg("serverName", *name)
	inji.Reg("serverPort", fmt.Sprintf("%d", *port))
	inji.Reg("env", *env)

	// ========== 配置服务 ==========
	configSvc := service.NewConfigService("./config-center")
	inji.Reg("configService", configSvc)

	// ========== 注册ServiceProvider ==========
	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	sp.Init()

	// ========== 初始化日志 ==========
	log.Init(nil)
	defer log.Sync()

	// ========== 启动HTTP服务 ==========
	httpServer := sp.Get().HTTPServer
	httpServer.Start()

	fmt.Println("[config-server] started")

	// ========== 等待信号 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-quit

	fmt.Printf("\n[config-server] received signal %v, shutting down...\n", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		stdlog.Printf("[config-server] server shutdown error: %v", err)
	} else {
		fmt.Println("[config-server] HTTP server stopped gracefully")
	}

	// 关闭依赖注入容器
	inji.Close()

	fmt.Println("[config-server] stopped")
}
