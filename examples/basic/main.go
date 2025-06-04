package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	dtraderhq "github.com/DTrader-store/level2-client-go"
)

func main() {
	// 创建客户端
	client := dtraderhq.NewClient("ws://localhost:8080/ws")

	// 连接到服务器
	log.Println("正在连接到服务器...")
	if err := client.Connect(); err != nil {
		log.Fatal("连接失败:", err)
	}
	defer client.Close()
	log.Println("连接成功")

	// 认证（请替换为实际的token）
	log.Println("正在进行认证...")
	token := "your-auth-token-here"
	if err := client.Authenticate(token); err != nil {
		log.Fatal("认证失败:", err)
	}

	// 等待认证完成
	time.Sleep(1 * time.Second)
	if !client.IsAuthenticated() {
		log.Fatal("认证未完成")
	}
	log.Println("认证成功")

	// 订阅股票数据
	stockCodes := []string{"000001", "000002", "600000"}
	dataTypes := []int{4, 8, 14} // 4: 逐笔成交数据, 8: 逐笔大单数据, 14: 逐笔委托数据

	for _, stockCode := range stockCodes {
		log.Printf("订阅股票: %s", stockCode)
		if err := client.Subscribe(stockCode, dataTypes); err != nil {
			log.Printf("订阅失败 %s: %v", stockCode, err)
		} else {
			log.Printf("订阅成功: %s", stockCode)
		}
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动数据处理goroutine
	go func() {
		for {
			select {
			case data := <-client.DataChannel():
				log.Printf("收到市场数据: 股票=%s, 类型=%d, 时间=%d",
					data.StockCode, data.DataType, data.Timestamp)
				log.Printf("数据内容: %+v", data.Data)

			case err := <-client.ErrorChannel():
				log.Printf("收到错误: %v", err)
			}
		}
	}()

	log.Println("客户端运行中，按 Ctrl+C 退出...")
	log.Printf("当前订阅: %+v", client.GetSubscriptions())

	// 等待退出信号
	<-sigChan
	log.Println("正在关闭客户端...")

	// 取消所有订阅
	for stockCode := range client.GetSubscriptions() {
		log.Printf("取消订阅: %s", stockCode)
		if err := client.Unsubscribe(stockCode); err != nil {
			log.Printf("取消订阅失败 %s: %v", stockCode, err)
		}
	}

	log.Println("客户端已关闭")
}
