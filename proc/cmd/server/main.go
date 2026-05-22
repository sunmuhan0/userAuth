package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/teou/inji"

	"ttuser/proc/router"
	"ttuser/proc/sp"
)

func main() {
	// ========== 初始化inji依赖注入容器 ==========
	inji.InitDefault()
	defer inji.Close()

	// ========== 注册ServiceProvider ==========
	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	err := sp.Init()
	if err != nil {
		panic("service provider init fail!")
	}

	// ========== 配置路由 ==========
	httpPort := getEnv("HTTP_PORT", "8080")
	r := &router.Router{}
	engine := r.Setup()

	// ========== 启动HTTP服务 ==========
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: engine,
	}

	go func() {
		fmt.Printf("[proc] HTTP gateway listening on :%s\n", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[proc] failed to start: %v", err)
		}
	}()

	// ========== 关闭 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[proc] shutting down...")
	fmt.Println("[proc] stopped")
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
