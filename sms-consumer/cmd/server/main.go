package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/teou/inji"

	"ttuser/sms-consumer/sp"
)

func main() {
	fmt.Println("[sms-consumer] starting...")

	// ========== 初始化inji依赖注入容器 ==========
	inji.InitDefault()
	defer inji.Close()

	// ========== 注册ServiceProvider ==========
	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	sp.Init()

	fmt.Println("[sms-consumer] started successfully")

	// ========== 等待信号 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[sms-consumer] shutting down...")
	fmt.Println("[sms-consumer] stopped")
}
