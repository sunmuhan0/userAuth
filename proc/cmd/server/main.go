package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teou/inji"

	configclient "ttuser/config-client/client"
	"ttuser/pkg/log"
	"ttuser/proc/router"
	"ttuser/proc/sp"
)

func main() {
	name := flag.String("name", "proc", "service name")
	port := flag.Int("port", 8080, "service port")
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
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Printf("[proc] log sync error: %v\n", err)
		}
	}()

	// ========== 从配置中心拉取配置文件 ==========
	cc := configclient.New(&configclient.Config{
		Env:         *env,
		ServiceName: *name,
	})
	if err := cc.FetchConfigs(); err != nil {
		fmt.Printf("[proc] fetch configs failed: %v\n", err)
		os.Exit(1)
	}

	// ========== 注册ServiceProvider ==========
	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	sp.Init()

	// ========== 配置路由 ==========
	r := &router.Router{}
	engine := r.Setup()

	// ========== 启动HTTP服务 ==========
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           engine,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("[proc] HTTP gateway listening on :%d\n", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		fmt.Printf("[proc] failed to start: %v\n", err)
		os.Exit(1)
	default:
	}

	// ========== 优雅关闭 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-quit

	fmt.Printf("\n[proc] received signal %v, shutting down...\n", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("[proc] server shutdown error: %v\n", err)
	} else {
		fmt.Println("[proc] HTTP server stopped gracefully")
	}

	// 关闭依赖注入容器
	if err := inji.Close(); err != nil {
		fmt.Printf("[proc] inji close error: %v\n", err)
	}

	fmt.Println("[proc] stopped")
}
