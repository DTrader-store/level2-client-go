package dtraderhq

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 消息类型常量
const (
	MessageTypeAuth             = "auth"
	MessageTypeSubscribe        = "subscribe"
	MessageTypeUnsubscribe      = "unsubscribe"
	MessageTypeBatchSubscribe   = "batch_subscribe"
	MessageTypeBatchUnsubscribe = "batch_unsubscribe"
	MessageTypeReset            = "reset"
	MessageTypeData             = "data"
	MessageTypeError            = "error"
	MessageTypeSuccess          = "success"
	MessageTypePing             = "ping"
	MessageTypePong             = "pong"
)

// Message WebSocket消息结构
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// AuthMessage 认证消息
type AuthMessage struct {
	Token string `json:"token"`
}

// SubscribeMessage 订阅消息
type SubscribeMessage struct {
	StockCode string `json:"stock_code"`
	DataTypes []int  `json:"data_types"`
}

// UnsubscribeMessage 取消订阅消息
type UnsubscribeMessage struct {
	StockCode string `json:"stock_code"`
}

// BatchSubscribeMessage 批量订阅消息
type BatchSubscribeMessage struct {
	Subscriptions []SubscribeMessage `json:"subscriptions"`
}

// BatchUnsubscribeMessage 批量取消订阅消息
type BatchUnsubscribeMessage struct {
	StockCodes []string `json:"stock_codes"`
}

// ResetMessage 重置订阅消息
type ResetMessage struct {
	Subscriptions []SubscribeMessage `json:"subscriptions"`
}

// MarketData 市场数据
type MarketData struct {
	StockCode string      `json:"stock_code"`
	DataType  int         `json:"data_type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// BatchSubscribeResult 批量订阅结果
type BatchSubscribeResult struct {
	SuccessCount int                      `json:"success_count"`
	ErrorCount   int                      `json:"error_count"`
	SuccessList  []map[string]interface{} `json:"success_list"`
	ErrorList    []map[string]interface{} `json:"error_list"`
}

// BatchUnsubscribeResult 批量取消订阅结果
type BatchUnsubscribeResult struct {
	SuccessCount int      `json:"success_count"`
	SuccessList  []string `json:"success_list"`
}

// ResetResult 重置订阅结果
type ResetResult struct {
	CancelledCount int                      `json:"cancelled_count"`
	CancelledList  []string                 `json:"cancelled_list"`
	SuccessCount   int                      `json:"success_count"`
	ErrorCount     int                      `json:"error_count"`
	SuccessList    []map[string]interface{} `json:"success_list"`
	ErrorList      []map[string]interface{} `json:"error_list"`
}

// Client DTraderHQ WebSocket客户端
type Client struct {
	url             string
	conn            *websocket.Conn
	mu              sync.RWMutex
	isConnected     bool
	isAuthenticated bool
	dataChan        chan *MarketData
	errorChan       chan error
	closeChan       chan struct{}
	pingTicker      *time.Ticker
	subscriptions   map[string][]int // stockCode -> dataTypes
}

// NewClient 创建新的DTraderHQ客户端
func NewClient(serverURL string) *Client {
	return &Client{
		url:           serverURL,
		dataChan:      make(chan *MarketData, 100),
		errorChan:     make(chan error, 10),
		closeChan:     make(chan struct{}),
		subscriptions: make(map[string][]int),
	}
}

// Connect 连接到服务器
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return errors.New("already connected")
	}

	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	// 启动消息处理goroutine
	go c.readMessages()
	go c.pingLoop()

	return nil
}

// Close 关闭连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return nil
	}

	c.isConnected = false
	c.isAuthenticated = false

	if c.pingTicker != nil {
		c.pingTicker.Stop()
	}

	close(c.closeChan)

	if c.conn != nil {
		c.conn.Close()
	}

	return nil
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// IsAuthenticated 检查认证状态
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isAuthenticated
}

// Authenticate 进行认证
func (c *Client) Authenticate(token string) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	authMsg := AuthMessage{Token: token}
	msg := Message{
		Type:      MessageTypeAuth,
		Data:      authMsg,
		Timestamp: time.Now().Unix(),
	}

	return c.sendMessage(msg)
}

// BatchSubscribe 批量订阅股票数据
func (c *Client) BatchSubscribe(subscriptions []SubscribeMessage) error {
	if !c.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	if len(subscriptions) == 0 {
		return errors.New("subscriptions list is empty")
	}

	if len(subscriptions) > 100 {
		return errors.New("subscriptions count exceeds limit (max 100)")
	}

	batchMsg := BatchSubscribeMessage{
		Subscriptions: subscriptions,
	}
	msg := Message{
		Type:      MessageTypeBatchSubscribe,
		Data:      batchMsg,
		Timestamp: time.Now().Unix(),
	}

	// 更新本地订阅记录
	c.mu.Lock()
	for _, sub := range subscriptions {
		c.subscriptions[sub.StockCode] = sub.DataTypes
	}
	c.mu.Unlock()

	return c.sendMessage(msg)
}

// BatchUnsubscribe 批量取消订阅
func (c *Client) BatchUnsubscribe(stockCodes []string) error {
	if !c.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	if len(stockCodes) == 0 {
		return errors.New("stock codes list is empty")
	}

	if len(stockCodes) > 100 {
		return errors.New("stock codes count exceeds limit (max 100)")
	}

	batchMsg := BatchUnsubscribeMessage{
		StockCodes: stockCodes,
	}
	msg := Message{
		Type:      MessageTypeBatchUnsubscribe,
		Data:      batchMsg,
		Timestamp: time.Now().Unix(),
	}

	// 更新本地订阅记录
	c.mu.Lock()
	for _, stockCode := range stockCodes {
		delete(c.subscriptions, stockCode)
	}
	c.mu.Unlock()

	return c.sendMessage(msg)
}

// ResetSubscriptions 重置订阅（取消所有当前订阅并设置新的订阅）
func (c *Client) ResetSubscriptions(subscriptions []SubscribeMessage) error {
	if !c.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	if len(subscriptions) > 100 {
		return errors.New("subscriptions count exceeds limit (max 100)")
	}

	resetMsg := ResetMessage{
		Subscriptions: subscriptions,
	}
	msg := Message{
		Type:      MessageTypeReset,
		Data:      resetMsg,
		Timestamp: time.Now().Unix(),
	}

	// 重置本地订阅记录
	c.mu.Lock()
	c.subscriptions = make(map[string][]int)
	for _, sub := range subscriptions {
		c.subscriptions[sub.StockCode] = sub.DataTypes
	}
	c.mu.Unlock()

	return c.sendMessage(msg)
}

// Subscribe 订阅股票数据
func (c *Client) Subscribe(stockCode string, dataTypes []int) error {
	if !c.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	subMsg := SubscribeMessage{
		StockCode: stockCode,
		DataTypes: dataTypes,
	}
	msg := Message{
		Type:      MessageTypeSubscribe,
		Data:      subMsg,
		Timestamp: time.Now().Unix(),
	}

	c.mu.Lock()
	c.subscriptions[stockCode] = dataTypes
	c.mu.Unlock()

	return c.sendMessage(msg)
}

// Unsubscribe 取消订阅
func (c *Client) Unsubscribe(stockCode string) error {
	if !c.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	unsubMsg := UnsubscribeMessage{StockCode: stockCode}
	msg := Message{
		Type:      MessageTypeUnsubscribe,
		Data:      unsubMsg,
		Timestamp: time.Now().Unix(),
	}

	c.mu.Lock()
	delete(c.subscriptions, stockCode)
	c.mu.Unlock()

	return c.sendMessage(msg)
}

// DataChannel 获取数据通道
func (c *Client) DataChannel() <-chan *MarketData {
	return c.dataChan
}

// ErrorChannel 获取错误通道
func (c *Client) ErrorChannel() <-chan error {
	return c.errorChan
}

// GetSubscriptions 获取当前订阅列表
func (c *Client) GetSubscriptions() map[string][]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]int)
	for k, v := range c.subscriptions {
		result[k] = append([]int(nil), v...)
	}
	return result
}

// sendMessage 发送消息
func (c *Client) sendMessage(msg Message) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("connection is nil")
	}

	return conn.WriteJSON(msg)
}

// readMessages 读取消息
func (c *Client) readMessages() {
	for {
		select {
		case <-c.closeChan:
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				select {
				case c.errorChan <- fmt.Errorf("read message error: %w", err):
				case <-c.closeChan:
				}
				return
			}

			c.handleMessage(&msg)
		}
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypeSuccess:
		// 处理成功消息，包括认证成功
		if dataMap, ok := msg.Data.(map[string]interface{}); ok {
			if message, exists := dataMap["message"]; exists {
				if messageStr, ok := message.(string); ok {
					if messageStr == "认证成功" {
						c.mu.Lock()
						c.isAuthenticated = true
						c.mu.Unlock()
					}
				}
			}
		}

	case MessageTypeAuth:
		// 处理认证响应
		if dataMap, ok := msg.Data.(map[string]interface{}); ok {
			if message, exists := dataMap["message"]; exists {
				if messageStr, ok := message.(string); ok {
					if messageStr == "认证成功" {
						c.mu.Lock()
						c.isAuthenticated = true
						c.mu.Unlock()
					}
				}
			}
		}

	case MessageTypeSubscribe, MessageTypeBatchSubscribe, MessageTypeUnsubscribe, MessageTypeBatchUnsubscribe, MessageTypeReset:
		// 处理订阅相关的响应消息
		// 这些消息通常包含操作结果，可以根据需要进行处理
		// 目前只是静默处理，实际应用中可以添加回调或事件通知

	case MessageTypeData:
		// 处理市场数据
		if dataBytes, err := json.Marshal(msg.Data); err == nil {
			var marketData MarketData
			if err := json.Unmarshal(dataBytes, &marketData); err == nil {
				select {
				case c.dataChan <- &marketData:
				case <-c.closeChan:
				}
			}
		}

	case MessageTypeError:
		select {
		case c.errorChan <- errors.New(msg.Error):
		case <-c.closeChan:
		}

	case MessageTypePing:
		// 响应ping
		pongMsg := Message{
			Type:      MessageTypePong,
			Timestamp: time.Now().Unix(),
		}
		c.sendMessage(pongMsg)
	}
}

// pingLoop 心跳循环
func (c *Client) pingLoop() {
	c.pingTicker = time.NewTicker(30 * time.Second)
	defer c.pingTicker.Stop()

	for {
		select {
		case <-c.pingTicker.C:
			pingMsg := Message{
				Type:      MessageTypePing,
				Timestamp: time.Now().Unix(),
			}
			if err := c.sendMessage(pingMsg); err != nil {
				select {
				case c.errorChan <- fmt.Errorf("ping error: %w", err):
				case <-c.closeChan:
				}
				return
			}
		case <-c.closeChan:
			return
		}
	}
}
