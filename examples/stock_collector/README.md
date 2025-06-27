# 股票数据收集器

这个程序用于订阅指定的8只股票数据，并将数据按股票分类写入JSON文件。

## 功能特性

- 订阅指定的8只股票：603065, 002530, 603256, 002062, 002240, 002930, 002104, 603166
- 收集三种类型的数据：
  - 逐笔成交数据 (类型4)
  - 逐笔明细数据 (类型8) 
  - 逐笔委托数据 (类型14)
- 按股票代码分类，每只股票的数据写入独立的JSON文件
- 文件名格式：`{股票代码}_{日期}.json`
- 支持并发安全的文件写入

## 使用方法

### 1. 编译程序

```bash
cd /Users/regan/work/go/src/github.com/sooboy/dzh/client/go/examples/stock_collector
go build -o stock_data_collector main.go
```

### 2. 运行程序

```bash
./stock_data_collector
```

### 3. 程序输出

程序会：
1. 连接到WebSocket服务器 (ws://127.0.0.1:8080/ws)
2. 进行身份认证
3. 批量订阅8只股票的数据
4. 创建 `./stock_data` 目录
5. 为每只股票创建独立的JSON文件
6. 实时收集并写入数据
7. 运行30分钟后自动停止

### 4. 数据文件格式

### 数据结构对应关系

本程序已根据服务器端实际数据结构进行修复：

- **逐笔成交数据 (data_type=4)**: 对应服务器端 `ZhubiData` 结构体
  - 字段: `Time`, `Price`, `Volume`
  - 注意: 实际数据中没有 `OrderPackId` 字段

- **逐笔明细数据 (data_type=8)**: 对应服务器端 `BigOrder` 结构体  
  - 字段: `OrderPackId`, `BuyOrderIdWithFlag`, `SellOrderIdWithFlag`, `BuyPrice`, `SellPrice`, `BuyVol`, `SellVol`
  - 注意: 使用 `BuyOrderIdWithFlag` 和 `SellOrderIdWithFlag` 而不是文档中的 `BuyOrderPackId` 和 `SellOrderPackId`

- **逐笔委托数据 (data_type=14)**: 对应服务器端 `ZBWTData` 结构体
  - 字段: `Index`, `DateTime`, `Price`, `Volume`, `Type`
  - 注意: `Type` 字段格式为 "BA"(买入报单), "SA"(卖出报单), "BD"(买入撤单), "SD"(卖出撤单)

每行数据的JSON格式：

**逐笔成交数据 (data_type=4)**：
```json
{
  "stock_code": "000001",
  "data_type": "transaction",
  "timestamp": 1703123456,
  "data": {
    "Price": 12.34,
    "Volume": 1000,
    "Time": "14:30:15"
  }
}
```

**逐笔明细数据 (data_type=8)**：
```json
{
  "stock_code": "000001",
  "data_type": "detail",
  "timestamp": 1703123456,
  "data": {
    "OrderPackId": 12345,
    "BuyOrderIdWithFlag": 67890,
    "SellOrderIdWithFlag": 54321,
    "BuyPrice": 12.34,
    "SellPrice": 12.35,
    "BuyVol": 500,
    "SellVol": 500
  }
}
```

**逐笔委托数据 (data_type=14)**：
```json
{
  "stock_code": "000001",
  "data_type": "order",
  "timestamp": 1703123456,
  "data": {
    "Index": 1,
    "DateTime": 143015,
    "Price": 12.34,
    "Volume": 1000,
    "Type": "BA",
    "BuySell": "B",
    "OrderType": "A"
  }
}
```

### 5. 数据类型说明

- `data_type: 4` - 逐笔成交数据
- `data_type: 8` - 逐笔明细数据  
- `data_type: 14` - 逐笔委托数据

## 配置说明

### 修改股票列表

在 `main()` 函数中修改 `stockList` 变量：

```go
stockList := []string{
    "603065",
    "002530",
    // 添加或修改股票代码
}
```

### 修改数据保存目录

在 `main()` 函数中修改 `dataDir` 变量：

```go
dataDir := "./your_custom_directory"
```

### 修改运行时间

在 `main()` 函数的最后修改超时时间：

```go
case <-time.After(60 * time.Minute): // 改为60分钟
```

## 注意事项

1. 确保WebSocket服务器正在运行 (127.0.0.1:8080)
2. 确保认证token有效
3. 程序会自动创建数据目录，确保有写入权限
4. 数据文件按日期命名，每天会创建新文件
5. 程序支持优雅关闭，Ctrl+C 会正确关闭所有文件

## 故障排除

### 连接失败
- 检查WebSocket服务器是否运行
- 检查网络连接
- 确认服务器地址和端口正确

### 认证失败
- 检查token是否有效
- 确认token格式正确
- 检查用户权限

### 文件写入失败
- 检查目录权限
- 确保磁盘空间充足
- 检查文件是否被其他程序占用