package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ttuser/event-consumer/consumer"
	"ttuser/sms-consumer/internal/handler"
	"ttuser/sms-consumer/internal/sms"
)

func main() {
	fmt.Println("[sms-consumer] starting...")

	// ========== 初始化短信发送器 ==========
	smsConfig := sms.DefaultConfig()
	smsSender := sms.NewSender(smsConfig)

	// ========== 初始化短信事件处理器 ==========
	smsHandler := handler.NewSMSHandler(smsSender)

	// ========== RMQ配置（写死，后期从配置中心获取） ==========
	rmqConfig := &consumer.RMQConfig{
		URL:          "amqp://guest:guest@127.0.0.1:5672/",
		Exchange:     "user.events",
		ExchangeType: "topic",
		RoutingKey:   "user.registered",
		Queue:        "sms.user.registered",
		PrefetchCnt:  10,
	}

	// ========== 创建通用消费者，注入短信handler ==========
	rmqConsumer := consumer.NewRMQConsumer(rmqConfig, smsHandler)

	// ========== 启动消费者 ==========
	if err := rmqConsumer.Start(); err != nil {
		fmt.Printf("[sms-consumer] failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[sms-consumer] started successfully")

	// ========== 等待信号 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[sms-consumer] shutting down...")
	rmqConsumer.Stop()
	fmt.Println("[sms-consumer] stopped")
}
