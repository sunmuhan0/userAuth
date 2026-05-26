package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/teou/inji"

	"ttuser/config-server/sp"
)

func main() {
	fmt.Println("[config-server] starting...")

	inji.InitDefault()
	defer inji.Close()

	inji.Reg("serviceProvider", (*sp.ServiceProvider)(nil))
	sp.Init()

	fmt.Println("[config-server] started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[config-server] stopped")
}
