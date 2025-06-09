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
				if data.DataType == 4 {
					// 处理逐笔成交数据
					transactionData, ok := data.Data.(map[string]interface{})
					if ok {
						price, _ := transactionData["Price"].(float64)
						volume, _ := transactionData["Volume"].(float64)
						timestamp, _ := transactionData["Time"].(float64)

						// 价格需要除以10000才是实际价格
						actualPrice := price / 100.0

						// 转换时间戳为可读时间
						timeStr := time.Unix(int64(timestamp), 0).Format("15:04:05")

						fmt.Printf("[逐笔成交] 股票: %s, 价格: %.4f, 成交量: %.0f, 时间: %s\n",
							data.StockCode, actualPrice, volume, timeStr)
					} else {
						fmt.Printf("[逐笔成交] 数据格式错误: %v\n", data.Data)
					}
				} else if data.DataType == 8 {
					// 处理逐笔大单数据
					bigOrderData, ok := data.Data.(map[string]interface{})
					if ok {
						buyOrderId, _ := bigOrderData["BuyOrderIdWithFlag"].(float64)
						sellOrderId, _ := bigOrderData["SellOrderIdWithFlag"].(float64)
						buyPrice, _ := bigOrderData["BuyPrice"].(float64)
						sellPrice, _ := bigOrderData["SellPrice"].(float64)
						buyVol, _ := bigOrderData["BuyVol"].(float64)
						sellVol, _ := bigOrderData["SellVol"].(float64)
						orderPackId, _ := bigOrderData["OrderPackId"].(float64)

						// 价格需要除以10000才是实际价格
						actualBuyPrice := buyPrice / 100.0
						actualSellPrice := sellPrice / 100.0

						fmt.Printf("[逐笔大单] 股票: %s, 包ID: %.0f, 买单ID: %.0f, 卖单ID: %.0f\n",
							data.StockCode, orderPackId, buyOrderId, sellOrderId)
						fmt.Printf("          买价: %.4f, 买量: %.0f, 卖价: %.4f, 卖量: %.0f\n",
							actualBuyPrice, buyVol, actualSellPrice, sellVol)
					} else {
						fmt.Printf("[逐笔大单] 数据格式错误: %v\n", data.Data)
					}
				} else if data.DataType == 14 {
					// 处理逐笔委托数据
					orderData, ok := data.Data.(map[string]interface{})
					if ok {
						price, _ := orderData["Price"].(float64)
						volume, _ := orderData["Volume"].(float64)
						dateTime, _ := orderData["DateTime"].(float64)
						index, _ := orderData["Index"].(float64)

						// 价格需要除以10000才是实际价格
						actualPrice := price / 10000.0

						// 解析委托类型
						typeArr, ok := orderData["Type"].([]interface{})
						typeDesc := ""
						if ok && len(typeArr) >= 2 {
							// 第一个字节: 66(B)表示买入, 83(S)表示卖出
							// 第二个字节: 65(A)表示报单, 68(D)表示撤单
							firstByte, _ := typeArr[0].(float64)
							secondByte, _ := typeArr[1].(float64)

							if firstByte == 66 {
								typeDesc += "买入"
							} else if firstByte == 83 {
								typeDesc += "卖出"
							}

							if secondByte == 65 {
								typeDesc += "报单"
							} else if secondByte == 68 {
								typeDesc += "撤单"
							}
						}

						// 解析时间格式 112940640 -> 11:29:40.640
						dateTimeStr := fmt.Sprintf("%09.0f", dateTime)
						timeStr := ""
						if len(dateTimeStr) >= 9 {
							hour := dateTimeStr[0:2]
							minute := dateTimeStr[2:4]
							second := dateTimeStr[4:6]
							millisecond := dateTimeStr[6:9]
							timeStr = fmt.Sprintf("%s:%s:%s.%s", hour, minute, second, millisecond)
						}

						fmt.Printf("[逐笔委托] 股票: %s, 价格: %.4f, 数量: %.0f, 类型: %s, 时间: %s, 索引: %.0f\n",
							data.StockCode, actualPrice, volume, typeDesc, timeStr, index)
					} else {
						fmt.Printf("[逐笔委托] 数据格式错误: %v\n", data.Data)
					}
				}
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
