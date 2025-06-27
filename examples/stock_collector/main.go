package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	dtraderhq "github.com/DTrader-store/level2-client-go"
)

// StockDataCollector 股票数据收集器
type StockDataCollector struct {
	client    *dtraderhq.Client
	files     map[string]*os.File // key格式: "stockCode_dataType"
	filesMu   sync.RWMutex
	dataDir   string
	stockList []string
}

// NewStockDataCollector 创建新的股票数据收集器
func NewStockDataCollector(wsURL string, dataDir string, stockList []string) *StockDataCollector {
	return &StockDataCollector{
		client:    dtraderhq.NewClient(wsURL),
		files:     make(map[string]*os.File),
		dataDir:   dataDir,
		stockList: stockList,
	}
}

// Connect 连接到服务器
func (sdc *StockDataCollector) Connect() error {
	return sdc.client.Connect()
}

// Authenticate 认证
func (sdc *StockDataCollector) Authenticate(token string) error {
	if err := sdc.client.Authenticate(token); err != nil {
		return err
	}

	// 等待认证完成
	time.Sleep(2 * time.Second)

	if !sdc.client.IsAuthenticated() {
		return fmt.Errorf("认证未完成")
	}

	return nil
}

// Subscribe 订阅股票数据
func (sdc *StockDataCollector) Subscribe() error {
	// 创建订阅消息
	var subscriptions []dtraderhq.SubscribeMessage
	for _, stockCode := range sdc.stockList {
		subscriptions = append(subscriptions, dtraderhq.SubscribeMessage{
			StockCode: stockCode,
			DataTypes: []int{4, 8, 14}, // 逐笔成交、逐笔明细、逐笔委托
		})
	}

	fmt.Printf("批量订阅 %d 只股票...\n", len(subscriptions))
	if err := sdc.client.BatchSubscribe(subscriptions); err != nil {
		return fmt.Errorf("批量订阅失败: %v", err)
	}

	fmt.Println("批量订阅请求已发送")
	return nil
}

// getDataTypeFileName 获取数据类型的文件名后缀
func getDataTypeFileName(dataType int) string {
	switch dataType {
	case 4:
		return "transaction" // 逐笔成交
	case 8:
		return "detail"      // 逐笔明细
	case 14:
		return "order"       // 逐笔委托
	default:
		return "unknown"
	}
}

// getOrCreateFile 获取或创建股票数据文件
func (sdc *StockDataCollector) getOrCreateFile(stockCode string, dataType int) (*os.File, error) {
	fileKey := fmt.Sprintf("%s_%d", stockCode, dataType)
	
	sdc.filesMu.RLock()
	file, exists := sdc.files[fileKey]
	sdc.filesMu.RUnlock()

	if exists {
		return file, nil
	}

	sdc.filesMu.Lock()
	defer sdc.filesMu.Unlock()

	// 双重检查
	if file, exists := sdc.files[fileKey]; exists {
		return file, nil
	}

	// 确保数据目录存在
	if err := os.MkdirAll(sdc.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 创建文件，按股票代码和数据类型分类
	dataTypeName := getDataTypeFileName(dataType)
	filename := fmt.Sprintf("%s_%s_%s.json", stockCode, dataTypeName, time.Now().Format("20060102"))
	filePath := filepath.Join(sdc.dataDir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}

	sdc.files[fileKey] = file
	fmt.Printf("为股票 %s 数据类型 %s 创建数据文件: %s\n", stockCode, dataTypeName, filePath)
	return file, nil
}

// writeDataToFile 将数据写入文件
func (sdc *StockDataCollector) writeDataToFile(stockCode string, dataType int, data interface{}) error {
	file, err := sdc.getOrCreateFile(stockCode, dataType)
	if err != nil {
		return err
	}

	// 构造数据记录
	record := map[string]interface{}{
		"timestamp":  time.Now().Unix(),
		"stock_code": stockCode,
		"data_type":  dataType,
		"data":       data,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 写入文件
	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// getDataTypeName 获取数据类型名称
func getDataTypeName(dataType int) string {
	switch dataType {
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

// handleTransactionData 处理逐笔成交数据
func handleTransactionData(data interface{}, stockCode string) {
	// 数据是ZhubiData数组格式
	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				record := DataRecord{
					StockCode: stockCode,
					DataType:  "transaction",
					Timestamp: time.Now().Unix(),
					Data:      make(map[string]interface{}),
				}

				// 解析字段 - 对应ZhubiData结构体
				if price, ok := itemMap["Price"]; ok {
					if priceFloat, ok := price.(float64); ok {
						record.Data["Price"] = priceFloat / 100 // 价格需要除以100
					}
				}
				if volume, ok := itemMap["Volume"]; ok {
					record.Data["Volume"] = volume
				}
				if timeVal, ok := itemMap["Time"]; ok {
					if timeFloat, ok := timeVal.(float64); ok {
						timestamp := int64(timeFloat)
						record.Data["Time"] = time.Unix(timestamp, 0).Format("15:04:05")
					}
				}
				// 注意：实际数据结构中没有OrderPackId字段

				writeToFile(stockCode, record)
			}
		}
	default:
		log.Printf("未知的逐笔成交数据格式: %T", data)
	}
}

// 处理逐笔明细数据
func handleDetailData(data interface{}, stockCode string) {
	// 数据是BigOrder数组格式
	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				record := DataRecord{
					StockCode: stockCode,
					DataType:  "detail",
					Timestamp: time.Now().Unix(),
					Data:      make(map[string]interface{}),
				}

				// 解析字段 - 对应BigOrder结构体
				if orderPackId, ok := itemMap["OrderPackId"]; ok {
					record.Data["OrderPackId"] = orderPackId
				}
				if buyOrderId, ok := itemMap["BuyOrderIdWithFlag"]; ok {
					record.Data["BuyOrderIdWithFlag"] = buyOrderId
				}
				if sellOrderId, ok := itemMap["SellOrderIdWithFlag"]; ok {
					record.Data["SellOrderIdWithFlag"] = sellOrderId
				}
				if buyPrice, ok := itemMap["BuyPrice"]; ok {
					if priceFloat, ok := buyPrice.(float64); ok {
						record.Data["BuyPrice"] = priceFloat / 100 // 价格需要除以100
					}
				}
				if sellPrice, ok := itemMap["SellPrice"]; ok {
					if priceFloat, ok := sellPrice.(float64); ok {
						record.Data["SellPrice"] = priceFloat / 100 // 价格需要除以100
					}
				}
				if buyVol, ok := itemMap["BuyVol"]; ok {
					record.Data["BuyVol"] = buyVol
				}
				if sellVol, ok := itemMap["SellVol"]; ok {
					record.Data["SellVol"] = sellVol
				}

				writeToFile(stockCode, record)
			}
		}
	default:
		log.Printf("未知的逐笔明细数据格式: %T", data)
	}
}

// 处理逐笔委托数据
func handleOrderData(data interface{}, stockCode string) {
	// 数据是ZBWTData数组格式
	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				record := DataRecord{
					StockCode: stockCode,
					DataType:  "order",
					Timestamp: time.Now().Unix(),
					Data:      make(map[string]interface{}),
				}

				// 解析字段 - 对应ZBWTData结构体
				if index, ok := itemMap["Index"]; ok {
					record.Data["Index"] = index
				}
				if dateTime, ok := itemMap["DateTime"]; ok {
					record.Data["DateTime"] = dateTime
				}
				if price, ok := itemMap["Price"]; ok {
					if priceFloat, ok := price.(float64); ok {
						record.Data["Price"] = priceFloat / 100 // 价格需要除以100
					}
				}
				if volume, ok := itemMap["Volume"]; ok {
					record.Data["Volume"] = volume
				}
				if typeVal, ok := itemMap["Type"]; ok {
					// Type是[2]byte，表示BA(买),SA(卖),BD(买撤),SD(卖撤)
					if typeStr, ok := typeVal.(string); ok {
						record.Data["Type"] = typeStr
						if len(typeStr) >= 2 {
							// 解析买卖方向和操作类型
							buySell := string(typeStr[0])   // B=买, S=卖
							orderType := string(typeStr[1]) // A=报单, D=撤单
							record.Data["BuySell"] = buySell
							record.Data["OrderType"] = orderType
						}
					}
				}

				writeToFile(stockCode, record)
			}
		}
	default:
		log.Printf("未知的逐笔委托数据格式: %T", data)
	}
}

// DataRecord 数据记录结构
type DataRecord struct {
	StockCode string                 `json:"stock_code"`
	DataType  string                 `json:"data_type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// writeToFile 写入数据到文件
func writeToFile(stockCode string, record DataRecord) {
	// 这里应该实现实际的文件写入逻辑
	// 为了简化，这里只是打印
	fmt.Printf("写入文件: %s, 数据类型: %s\n", stockCode, record.DataType)
}

// StartDataCollection 开始数据收集
func (sdc *StockDataCollector) StartDataCollection() {
	go func() {
		for {
			select {
			case data := <-sdc.client.DataChannel():
				// 根据数据类型处理数据
				switch data.DataType {
				case 4: // 逐笔成交
					handleTransactionData(data.Data, data.StockCode)
				case 8: // 逐笔明细
					handleDetailData(data.Data, data.StockCode)
				case 14: // 逐笔委托
					handleOrderData(data.Data, data.StockCode)
				default:
					log.Printf("未知数据类型: %d", data.DataType)
				}

				// 写入数据到文件
				if err := sdc.writeDataToFile(data.StockCode, data.DataType, data.Data); err != nil {
					log.Printf("写入数据失败: %v", err)
					continue
				}

				// 打印数据摘要
				fmt.Printf("[%s] 收到 %s 数据: %s\n",
					time.Now().Format("15:04:05"),
					getDataTypeName(data.DataType),
					data.StockCode)

			case err := <-sdc.client.ErrorChannel():
				log.Printf("客户端错误: %v", err)
			}
		}
	}()
}

// Close 关闭收集器
func (sdc *StockDataCollector) Close() {
	// 关闭客户端
	sdc.client.Close()

	// 关闭所有文件
	sdc.filesMu.Lock()
	defer sdc.filesMu.Unlock()

	for fileKey, file := range sdc.files {
		if err := file.Close(); err != nil {
			log.Printf("关闭文件失败 %s: %v", fileKey, err)
		}
	}
}

func main() {
	// 指定的8只股票
	stockList := []string{
		"603065",
		"002530",
		"603256",
		"002062",
		"002240",
		"002930",
		"002104",
		"603166",
	}

	// 创建数据收集器
	dataDir := "./stock_data"
	collector := NewStockDataCollector("ws://45.152.65.97:8080/ws", dataDir, stockList)

	// 连接到服务器
	if err := collector.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer collector.Close()

	// 认证
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InNvb2JveSIsImV4cCI6MTc4MDU0Nzk2MCwiaWF0IjoxNzQ5NDQzOTYwfQ.XghmQtFLVdY9NFAMWGSUkTqpe0NNcKLQVljN0cQ62CA"
	if err := collector.Authenticate(token); err != nil {
		log.Fatalf("认证失败: %v", err)
	}

	fmt.Println("认证成功！")

	// 订阅股票数据
	if err := collector.Subscribe(); err != nil {
		log.Fatalf("订阅失败: %v", err)
	}

	// 开始数据收集
	collector.StartDataCollection()

	fmt.Printf("开始收集股票数据，数据将保存到 %s 目录\n", dataDir)
	fmt.Println("按 Ctrl+C 停止收集...")

	// 运行指定时间或直到用户中断
	select {
	case <-time.After(30 * time.Minute): // 运行30分钟
		fmt.Println("\n数据收集完成（30分钟）")
	}
}
