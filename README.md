# DZH Go 客户端

这是 DZH 实时行情系统的 Go 客户端库，支持 WebSocket 连接、用户认证、股票数据订阅等功能。

## 功能特性

- WebSocket 连接管理
- 用户认证
- 单个股票订阅/取消订阅
- **批量股票订阅/取消订阅**（新增）
- **重置订阅**（清空所有订阅并设置新的订阅）（新增）
- 实时市场数据接收
- 自动心跳保持连接
- 错误处理和重连机制

## 安装

```bash
go get github.com/sooboy/DTraderHQ/client/go
```

## 快速开始

```go
package main

import (
    "log"
    "github.com/sooboy/DTraderHQ/client/go"
)

func main() {
    // 创建客户端
    client := dtraderhq.NewClient("ws://localhost:8080/ws")
    
    // 连接到服务器
    if err := client.Connect(); err != nil {
        log.Fatal("连接失败:", err)
    }
    defer client.Close()
    
    // 认证
    if err := client.Authenticate("your-token"); err != nil {
        log.Fatal("认证失败:", err)
    }
    
    // 订阅股票数据
    if err := client.Subscribe("000001", []int{1, 2}); err != nil {
        log.Fatal("订阅失败:", err)
    }
    
    // 监听数据
    for data := range client.DataChannel() {
        log.Printf("收到数据: %+v", data)
    }
}
```

## API 文档

### 客户端创建

```go
client := dtraderhq.NewClient(url string) *Client
```

### 连接管理

```go
// 连接到服务器
func (c *Client) Connect() error

// 关闭连接
func (c *Client) Close() error

// 检查连接状态
func (c *Client) IsConnected() bool
```

### 认证

```go
// 使用 token 进行认证
func (c *Client) Authenticate(token string) error
```

### 数据订阅

```go
// 订阅股票数据
func (c *Client) Subscribe(stockCode string, dataTypes []int) error

// 取消订阅
func (c *Client) Unsubscribe(stockCode string) error
```

### 数据接收

```go
// 获取数据通道
func (c *Client) DataChannel() <-chan *MarketData

// 获取错误通道
func (c *Client) ErrorChannel() <-chan error
```

## 快速开始

### 基本使用

```go
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
    
    // 认证
    token := "your-auth-token-here"
    if err := client.Authenticate(token); err != nil {
        log.Fatalf("认证失败: %v", err)
    }
    
    // 等待认证完成
    time.Sleep(2 * time.Second)
    
    // 订阅股票数据（使用开放的数据类型）
    if err := client.Subscribe("000001", []int{4, 8}); err != nil {
        log.Printf("订阅失败: %v", err)
    }
    
    // 接收数据
    go func() {
        for {
            select {
            case data := <-client.DataChannel():
                fmt.Printf("收到数据: %+v\n", data)
            case err := <-client.ErrorChannel():
                fmt.Printf("错误: %v\n", err)
            }
        }
    }()
    
    // 保持运行
    time.Sleep(30 * time.Second)
}
```

## API 参考

### 客户端创建和连接

```go
// 创建新客户端
client := dtraderhq.NewClient("ws://localhost:8080/ws")

// 连接到服务器
err := client.Connect()

// 关闭连接
client.Close()

// 检查连接状态
isConnected := client.IsConnected()
```

### 用户认证

```go
// 认证
err := client.Authenticate("your-token")

// 检查认证状态
isAuth := client.IsAuthenticated()
```

### 订阅管理

#### 单个订阅

```go
// 订阅单个股票（使用开放的数据类型）
err := client.Subscribe("000001", []int{4, 8}) // 逐笔成交+逐笔大单

// 取消订阅
err := client.Unsubscribe("000001")
```

#### 批量订阅（新增功能）

```go
// 批量订阅（使用开放的数据类型）
subscriptions := []dtraderhq.SubscribeMessage{
    {StockCode: "000001", DataTypes: []int{4, 8}},    // 逐笔成交+逐笔大单
    {StockCode: "600000", DataTypes: []int{4, 14}},   // 逐笔成交+逐笔委托
}
err := client.BatchSubscribe(subscriptions)

// 批量取消订阅
stockCodes := []string{"000001", "600000"}
err := client.BatchUnsubscribe(stockCodes)
```

#### 重置订阅（新增功能）

```go
// 重置订阅（清空所有现有订阅，设置新的订阅）
newSubscriptions := []dtraderhq.SubscribeMessage{
    {StockCode: "002415", DataTypes: []int{4, 8}},   // 逐笔成交+逐笔大单
    {StockCode: "002594", DataTypes: []int{14}},     // 逐笔委托
}
err := client.ResetSubscriptions(newSubscriptions)
```

#### 查询订阅

```go
// 获取当前所有订阅
subscriptions := client.GetSubscriptions()
for stockCode, dataTypes := range subscriptions {
    fmt.Printf("股票: %s, 数据类型: %v\n", stockCode, dataTypes)
}
```

### 数据接收

```go
// 获取数据通道
dataChan := client.DataChannel()

// 获取错误通道
errorChan := client.ErrorChannel()

// 接收数据
go func() {
    for {
        select {
        case data := <-dataChan:
            // 处理市场数据
            fmt.Printf("股票: %s, 价格: %.2f\n", data.StockCode, data.Price)
        case err := <-errorChan:
            // 处理错误
            fmt.Printf("错误: %v\n", err)
        }
    }
}()
```

## 消息类型

### 数据类型说明

当前开放的数据类型：
- `4`: 逐笔成交数据（TRANSACTION）
- `8`: 逐笔大单数据（BIG_ORDER）
- `14`: 逐笔委托数据（ZBWT）

完整数据类型列表（仅供参考）：
- `0`: 5档盘口（LOW_FIVE）
- `1`: 10档盘口（HIGH_TEN，已废弃）
- `2`: L1数据包（L1_DATAGRAM）
- `3`: L2数据包（L2_DATAGRAM）
- `4`: 逐笔成交数据（TRANSACTION）✅
- `5`: 逐笔委托数据（ORDER_QUEUE）
- `6`: 高档位行情数据（HIGH_FIVE）
- `7`: 百档盘口（HUNDRED）
- `8`: 逐笔大单数据（BIG_ORDER）✅
- `9`: 免费行情数据（FREE_QUOTE）
- `10`: 期权数据（OPTION）
- `11`: K线数据（KLINE）
- `12`: 财务数据（FINANCE）
- `13`: IOPV数据（IOPV）
- `14`: 逐笔委托（ZBWT）✅

### 市场数据结构

```go
type MarketData struct {
    StockCode string  `json:"stock_code"` // 股票代码
    DataType  int     `json:"data_type"`  // 数据类型
    Price     float64 `json:"price"`      // 价格
    Volume    int64   `json:"volume"`     // 成交量
    Timestamp int64   `json:"timestamp"`  // 时间戳
    // 其他字段...
}
```

## 示例程序

项目包含两个示例程序：

1. `examples/simple_client.go` - 基本使用示例
2. `examples/batch_operations.go` - 批量操作示例

运行示例：

```bash
# 基本示例
go run examples/simple_client.go

# 批量操作示例
go run examples/batch_operations.go
```

## 性能优化

客户端已针对高频交易场景进行了优化：

- 支持批量订阅操作，减少网络往返
- 内置连接池和心跳机制
- 异步消息处理
- 缓冲通道避免阻塞

### 批量操作限制

- 批量订阅/取消订阅：最大 100 个股票
- 重置订阅：最大 100 个股票

## 注意事项

1. 确保在使用前先进行认证
2. 批量操作有数量限制（最大100个）
3. 建议使用协程处理数据接收，避免阻塞
4. 程序退出前记得调用 `Close()` 方法
5. 网络异常时客户端会自动重连