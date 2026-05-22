package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/teou/inji"

	"ttuser/auth-server/sp"
)

func main() {
	// ========== 初始化inji依赖注入容器 ==========
	inji.InitDefault()
	defer inji.Close()

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
	<-quit

	fmt.Println("\n[auth-server] shutting down...")
	sp.Get().GRPCServer.Stop()
	fmt.Println("[auth-server] stopped")
}
