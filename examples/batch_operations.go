package main

import (
	"fmt"
	"log"
	"math"
	"time"

	dtraderhq "github.com/DTrader-store/level2-client-go"
)

func Type(t int) string {
	switch t {
	case 4:
		return "逐笔成交"
	case 8:
		return "逐笔明细"
	case 14:
		return "逐笔委托"
	default:
		return "未知"
	}
}

func main() {
	// 创建客户端
	client := dtraderhq.NewClient("ws://127.0.0.1:8080/ws")

	// 连接到服务器
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	// 认证
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJ1c2VybmFtZSI6InNvb2JveTIiLCJleHAiOjE3ODA3OTgwMDIsImlhdCI6MTc0OTY5NDAwMn0.qmERZ7GZFEcGrq2076G-AtcK0MN0EPn8eRlqtCZAu-o"
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
		{StockCode: "000001", DataTypes: []int{4, 8}},     // 平安银行：逐笔成交+逐笔明细
		{StockCode: "000002", DataTypes: []int{4, 14}},    // 万科A：逐笔成交+逐笔委托
		{StockCode: "600000", DataTypes: []int{4, 8, 14}}, // 浦发银行：逐笔成交+逐笔明细+逐笔委托
		{StockCode: "600036", DataTypes: []int{4}},        // 招商银行：逐笔成交
		{StockCode: "600519", DataTypes: []int{4, 8}},     // 贵州茅台：逐笔成交+逐笔明细
		{StockCode: "002177", DataTypes: []int{14}},       // 贵州茅台：逐笔成交+逐笔明细
		{StockCode: "002094", DataTypes: []int{4, 8}},     // 贵州茅台：逐笔成交+逐笔明细
		// {StockCode: "000001", DataTypes: []int{4, 14}},
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
				log.Printf("[数据流转] 收到数据: 股票=%s, 类型=%d, 数据结构=%T", data.StockCode, data.DataType, data.Data)

				if data.DataType == 4 {
					// 处理逐笔成交数据 - 支持单个对象和数组两种格式
					var transactionDatas []map[string]interface{}

					// 尝试解析为数组格式
					if dataArray, ok := data.Data.([]interface{}); ok {
						log.Printf("[数据流转] 逐笔成交数据为数组格式，长度: %d", len(dataArray))
						for _, item := range dataArray {
							if itemMap, ok := item.(map[string]interface{}); ok {
								transactionDatas = append(transactionDatas, itemMap)
							}
						}
					} else if dataMap, ok := data.Data.(map[string]interface{}); ok {
						// 单个对象格式
						log.Printf("[数据流转] 逐笔成交数据为单个对象格式")
						transactionDatas = append(transactionDatas, dataMap)
					} else {
						log.Printf("[数据流转] 逐笔成交数据格式无法识别: %T, 内容: %v", data.Data, data.Data)
						continue
					}

					log.Printf("[数据流转] 成功解析 %d 条逐笔成交记录", len(transactionDatas))
					for i, transactionData := range transactionDatas {
						price, _ := transactionData["Price"].(float64)
						volume, _ := transactionData["Volume"].(float64)
						timestamp, _ := transactionData["Time"].(float64)
						orderPackId, _ := transactionData["OrderPackId"].(float64)

						// 价格需要除以100才是实际价格
						actualPrice := price / 100.0

						// 转换时间戳为可读时间
						timeStr := time.Unix(int64(timestamp), 0).Format("15:04:05")

						if i < 3 || math.Abs(volume) > 100_00 { // 显示前3条
							fmt.Printf("[逐笔成交] 股票: %s, 价格: %.4f, 成交量: %.0f, 时间: %s, 成交ID: %.0f\n",
								data.StockCode, actualPrice, volume, timeStr, orderPackId)
						}
					}
				} else if data.DataType == 8 {
					// 处理逐笔明细数据 - 支持单个对象和数组两种格式
					var bigOrderDatas []map[string]interface{}

					// 尝试解析为数组格式
					if dataArray, ok := data.Data.([]interface{}); ok {
						log.Printf("[数据流转] 逐笔明细数据为数组格式，长度: %d", len(dataArray))
						for _, item := range dataArray {
							if itemMap, ok := item.(map[string]interface{}); ok {
								bigOrderDatas = append(bigOrderDatas, itemMap)
							}
						}
					} else if dataMap, ok := data.Data.(map[string]interface{}); ok {
						// 单个对象格式
						log.Printf("[数据流转] 逐笔明细数据为单个对象格式")
						bigOrderDatas = append(bigOrderDatas, dataMap)
					} else {
						log.Printf("[数据流转] 逐笔明细数据格式无法识别: %T, 内容: %v", data.Data, data.Data)
						continue
					}

					log.Printf("[数据流转] 成功解析 %d 条逐笔明细记录", len(bigOrderDatas))
					for i, bigOrderData := range bigOrderDatas {
						buyOrderId, _ := bigOrderData["BuyOrderPackId"].(float64)
						sellOrderId, _ := bigOrderData["SellOrderPackId"].(float64)
						buyOrderFlag, _ := bigOrderData["BuyFlag"].(float64)
						sellOrderFlag, _ := bigOrderData["SellFlag"].(float64)
						buyPrice, _ := bigOrderData["BuyPrice"].(float64)
						sellPrice, _ := bigOrderData["SellPrice"].(float64)
						buyVol, _ := bigOrderData["BuyVol"].(float64)
						sellVol, _ := bigOrderData["SellVol"].(float64)
						orderPackId, _ := bigOrderData["OrderPackId"].(float64)

						// 价格需要除以100才是实际价格
						actualBuyPrice := buyPrice / 100.0
						actualSellPrice := sellPrice / 100.0

						if i < 3 { // 只显示前3条大单记录
							fmt.Printf("[逐笔明细] 股票: %s, 包ID: %.0f, 买单ID: %.0f,买标识:%.0f, 卖单ID: %.0f,卖标识:%.0f \n",
								data.StockCode, orderPackId, buyOrderId, buyOrderFlag, sellOrderId, sellOrderFlag)
							fmt.Printf("          买价: %.4f, 买量: %.0f, 卖价: %.4f, 卖量: %.0f\n",
								actualBuyPrice, buyVol, actualSellPrice, sellVol)
						}
					}

				} else if data.DataType == 14 {
					// 处理逐笔委托数据 - 支持单个对象和数组两种格式
					var orderDatas []map[string]interface{}

					// 尝试解析为数组格式
					if dataArray, ok := data.Data.([]interface{}); ok {
						log.Printf("[数据流转] 逐笔委托数据为数组格式，长度: %d", len(dataArray))
						for _, item := range dataArray {
							if itemMap, ok := item.(map[string]interface{}); ok {
								orderDatas = append(orderDatas, itemMap)
							}
						}
					} else if dataMap, ok := data.Data.(map[string]interface{}); ok {
						// 单个对象格式
						log.Printf("[数据流转] 逐笔委托数据为单个对象格式")
						orderDatas = append(orderDatas, dataMap)
					} else {
						log.Printf("[数据流转] 逐笔委托数据格式无法识别: %T, 内容: %v", data.Data, data.Data)
						continue
					}

					log.Printf("[数据流转] 成功解析 %d 条逐笔委托记录", len(orderDatas))
					for i, orderData := range orderDatas {
						price, _ := orderData["Price"].(float64)
						volume, _ := orderData["Volume"].(float64)
						dateTime, _ := orderData["DateTime"].(float64)
						index, _ := orderData["Index"].(float64)

						// 价格需要除以100才是实际价格
						actualPrice := price / 100.0

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

						if i < 3 || volume > 100_00 { // 只显示前3条记录或大额委托
							fmt.Printf("[逐笔委托] 股票: %s, 价格: %.4f, 数量: %.0f, 类型: %s, 时间: %s, 索引: %.0f\n",
								data.StockCode, actualPrice, volume, typeDesc, timeStr, index)
						}
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
	// fmt.Println("添加单个股票订阅...")
	// if err := client.Subscribe("300750", []int{4, 8}); err != nil { // 逐笔成交+逐笔明细
	// 	log.Printf("单个订阅失败: %v", err)
	// } else {
	// 	fmt.Println("单个订阅请求已发送")
	// }

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
	// 	{StockCode: "002415", DataTypes: []int{4, 8}}, // 海康威视：逐笔成交+逐笔明细
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
