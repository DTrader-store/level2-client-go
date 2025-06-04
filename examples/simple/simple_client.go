package main

import (
	"fmt"
	"log"
	"time"

	dtraderhq "github.com/sooboy/dzh/client/go"
)

func main() {
	// 创建客户端
	client := dtraderhq.NewClient("ws://localhost:8080/ws")

	// 连接到服务器
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	fmt.Println("已连接到服务器")

	// 认证
	token := "your-auth-token-here"
	if err := client.Authenticate(token); err != nil {
		log.Fatalf("认证失败: %v", err)
	}

	// 等待认证完成
	time.Sleep(2 * time.Second)

	if !client.IsAuthenticated() {
		log.Fatal("认证未完成")
	}

	fmt.Println("认证成功")

	// 启动数据接收协程
	go func() {
		for {
			select {
			case data := <-client.DataChannel():
				fmt.Printf("收到市场数据: 股票=%s, 类型=%d, 价格=%.2f, 时间=%d\n",
					data.StockCode, data.DataType, data.Price, data.Timestamp)
			case err := <-client.ErrorChannel():
				fmt.Printf("收到错误: %v\n", err)
			}
		}
	}()

	// 订阅股票数据
	stockCode := "000001"    // 平安银行
	dataTypes := []int{4, 8} // 数据类型：4=逐笔成交数据, 8=逐笔大单数据

	fmt.Printf("订阅股票 %s...\n", stockCode)
	if err := client.Subscribe(stockCode, dataTypes); err != nil {
		log.Printf("订阅失败: %v", err)
	} else {
		fmt.Println("订阅成功")
	}

	// 等待接收数据
	time.Sleep(10 * time.Second)

	// 订阅更多股票
	moreStocks := []string{"600000", "600036", "600519"}
	for _, stock := range moreStocks {
		fmt.Printf("订阅股票 %s...\n", stock)
		if err := client.Subscribe(stock, []int{4}); err != nil { // 数据类型：4=逐笔成交数据
			log.Printf("订阅 %s 失败: %v", stock, err)
		}
		time.Sleep(1 * time.Second)
	}

	// 继续接收数据
	time.Sleep(15 * time.Second)

	// 取消订阅
	fmt.Printf("取消订阅股票 %s...\n", stockCode)
	if err := client.Unsubscribe(stockCode); err != nil {
		log.Printf("取消订阅失败: %v", err)
	} else {
		fmt.Println("取消订阅成功")
	}

	// 显示当前订阅
	currentSubs := client.GetSubscriptions()
	fmt.Printf("当前订阅数量: %d\n", len(currentSubs))
	for stockCode, dataTypes := range currentSubs {
		fmt.Printf("  %s: %v\n", stockCode, dataTypes)
	}

	// 继续运行一段时间
	time.Sleep(10 * time.Second)

	fmt.Println("客户端演示完成")
}
