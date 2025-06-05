package main

import (
	"fmt"
	"log"
	"time"

	dtraderhq "github.com/DTrader-store/level2-client-go"
)

func Type(t int) string {
	switch t {
	case 4:
		return "逐笔成交"
	case 8:
		return "逐笔大单"
	case 14:
		return "逐笔委托"
	default:
		return "未知"
	}
}

func main() {
	// 创建客户端
	client := dtraderhq.NewClient("ws://localhost:8080/ws")

	// 连接到服务器
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	// 认证
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InNvb2JveSIsImV4cCI6MTc4MDE5MTMyNSwiaWF0IjoxNzQ5MDg3MzI1fQ.qwYExSfL2qz5G4u6rhXxPYr3wkezmwOCYb6OfL3tVZk"
	if err := client.Authenticate(token); err != nil {
		log.Fatalf("认证失败: %v", err)
	}

	// 等待认证完成
	time.Sleep(2 * time.Second)

	if !client.IsAuthenticated() {
		log.Fatal("认证未完成")
	}

	fmt.Println("认证成功，开始演示批量操作...")

	// 演示批量订阅
	subscriptions := []dtraderhq.SubscribeMessage{
		{StockCode: "000001", DataTypes: []int{4, 8}},     // 平安银行：逐笔成交+逐笔大单
		{StockCode: "000002", DataTypes: []int{4, 14}},    // 万科A：逐笔成交+逐笔委托
		{StockCode: "600000", DataTypes: []int{4, 8, 14}}, // 浦发银行：逐笔成交+逐笔大单+逐笔委托
		{StockCode: "600036", DataTypes: []int{4}},        // 招商银行：逐笔成交
		{StockCode: "600519", DataTypes: []int{4, 8}},     // 贵州茅台：逐笔成交+逐笔大单
		{StockCode: "002177", DataTypes: []int{4, 8}},     // 贵州茅台：逐笔成交+逐笔大单
		{StockCode: "002094", DataTypes: []int{4, 8}},     // 贵州茅台：逐笔成交+逐笔大单
	}

	fmt.Printf("批量订阅 %d 只股票...\n", len(subscriptions))
	if err := client.BatchSubscribe(subscriptions); err != nil {
		log.Printf("批量订阅失败: %v", err)
	} else {
		fmt.Println("批量订阅请求已发送")
	}

	// 启动数据接收协程
	go func() {
		for {
			select {
			case data := <-client.DataChannel():
				t := time.Unix(0, data.Timestamp)
				fmt.Printf("收到数据: 股票=%s, 类型=%s, 时间=%s\n",
					data.StockCode, Type(data.DataType), t.Format("2006-01-02 15:04:05"))
			case err := <-client.ErrorChannel():
				fmt.Printf("收到错误: %v\n", err)
			}
		}
	}()

	// 等待一段时间接收数据
	time.Sleep(10 * time.Second)

	// 演示单个订阅
	fmt.Println("添加单个股票订阅...")
	if err := client.Subscribe("300750", []int{4, 8}); err != nil { // 逐笔成交+逐笔大单
		log.Printf("单个订阅失败: %v", err)
	} else {
		fmt.Println("单个订阅请求已发送")
	}

	// time.Sleep(5 * time.Second)

	// // 演示批量取消订阅
	// stockCodesToUnsubscribe := []string{"000001", "000002"}
	// fmt.Printf("批量取消订阅 %d 只股票...\n", len(stockCodesToUnsubscribe))
	// if err := client.BatchUnsubscribe(stockCodesToUnsubscribe); err != nil {
	// 	log.Printf("批量取消订阅失败: %v", err)
	// } else {
	// 	fmt.Println("批量取消订阅请求已发送")
	// }

	// time.Sleep(5 * time.Second)

	// // 演示重置订阅
	// newSubscriptions := []dtraderhq.SubscribeMessage{
	// 	{StockCode: "002415", DataTypes: []int{4, 8}}, // 海康威视：逐笔成交+逐笔大单
	// 	{StockCode: "002594", DataTypes: []int{14}},   // 比亚迪：逐笔委托
	// }

	// fmt.Printf("重置订阅为 %d 只新股票...\n", len(newSubscriptions))
	// if err := client.ResetSubscriptions(newSubscriptions); err != nil {
	// 	log.Printf("重置订阅失败: %v", err)
	// } else {
	// 	fmt.Println("重置订阅请求已发送")
	// }

	// 继续接收数据
	time.Sleep(10 * time.Second)

	// 显示当前订阅
	currentSubs := client.GetSubscriptions()
	fmt.Printf("当前订阅数量: %d\n", len(currentSubs))
	for stockCode, dataTypes := range currentSubs {
		fmt.Printf("  %s: %v\n", stockCode, dataTypes)
	}
	// 输入任意键退出
	fmt.Println("演示完成，按任意键退出...")
	fmt.Scanln()
	fmt.Println("演示完成")
}
