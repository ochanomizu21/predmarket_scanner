package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

const (
	wsEndpoint            = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	defaultPingInterval   = 10 * time.Second
	defaultPongWait       = 15 * time.Second
	defaultWriteWait      = 10 * time.Second
	defaultReadWait       = 60 * time.Second
	maxReconnectDelay     = 5 * time.Minute
	SubscriptionBatchSize = 100
)

type Config struct {
	PingInterval time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration
	ReadWait     time.Duration
	MaxReconnect time.Duration
	InitialDump  *bool
	Level        *int
	BatchSize    int
}

type Client struct {
	config       Config
	orderBookMgr *types.OrderBookManager
	mu           sync.RWMutex
	conn         *websocket.Conn
	connected    bool
	ctx          context.Context
	cancel       context.CancelFunc
	messageChan  chan []byte
	errorChan    chan error
	subscribed   map[string]bool
	metrics      *Metrics
}

type Metrics struct {
	MessagesReceived int64
	BookEvents       int64
	PriceChanges     int64
	Connections      int64
	Reconnections    int64
	mu               sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncrementMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesReceived++
}

func (m *Metrics) IncrementBookEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BookEvents++
}

func (m *Metrics) IncrementPriceChanges() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PriceChanges++
}

func (m *Metrics) IncrementConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Connections++
}

func (m *Metrics) IncrementReconnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Reconnections++
}

func (m *Metrics) GetStats() (messages, bookEvents, priceChanges, connections, reconnections int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MessagesReceived, m.BookEvents, m.PriceChanges, m.Connections, m.Reconnections
}

func DefaultConfig() Config {
	return Config{
		PingInterval: defaultPingInterval,
		PongWait:     defaultPongWait,
		WriteWait:    defaultWriteWait,
		ReadWait:     defaultReadWait,
		MaxReconnect: maxReconnectDelay,
		InitialDump:  boolPtr(true),
		Level:        intPtr(2),
		BatchSize:    SubscriptionBatchSize,
	}
}

func NewClient(orderBookMgr *types.OrderBookManager) *Client {
	return NewClientWithConfig(orderBookMgr, DefaultConfig())
}

func NewClientWithConfig(orderBookMgr *types.OrderBookManager, config Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:       config,
		orderBookMgr: orderBookMgr,
		ctx:          ctx,
		cancel:       cancel,
		messageChan:  make(chan []byte, 1000),
		errorChan:    make(chan error, 10),
		subscribed:   make(map[string]bool),
		metrics:      NewMetrics(),
	}
}

func (c *Client) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.Dial(wsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	c.metrics.IncrementConnections()

	conn.SetReadDeadline(time.Now().Add(c.config.ReadWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(c.config.ReadWait))
		return nil
	})

	go c.readPump()
	go c.writePump()
	go c.pingPump()
	go c.processMessages()

	return nil
}

func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.cancel()

	if c.conn != nil {
		err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Printf("Error sending close message: %v", err)
		}
		c.conn.Close()
	}

	c.connected = false
	return nil
}

func (c *Client) Subscribe(assetIDs []string) error {
	if len(assetIDs) == 0 {
		return fmt.Errorf("no asset IDs provided")
	}

	batches := c.batchAssetIDs(assetIDs)

	for _, batch := range batches {
		sub := types.WSSubscriptionRequest{
			AssetIDs:             batch,
			Type:                 "market",
			InitialDump:          c.config.InitialDump,
			Level:                c.config.Level,
			CustomFeatureEnabled: boolPtr(false),
		}

		if err := c.sendSubscription(sub); err != nil {
			return fmt.Errorf("failed to send subscription: %w", err)
		}

		c.mu.Lock()
		for _, assetID := range batch {
			c.subscribed[assetID] = true
		}
		c.mu.Unlock()
	}

	log.Printf("Subscribed to %d asset IDs in %d batches", len(assetIDs), len(batches))
	return nil
}

func (c *Client) AddSubscription(assetIDs []string) error {
	if len(assetIDs) == 0 {
		return nil
	}

	batches := c.batchAssetIDs(assetIDs)

	for _, batch := range batches {
		sub := types.WSUpdateSubscription{
			Operation: "subscribe",
			AssetIDs:  batch,
		}

		if err := c.sendUpdateSubscription(sub); err != nil {
			return fmt.Errorf("failed to send update subscription: %w", err)
		}

		c.mu.Lock()
		for _, assetID := range batch {
			c.subscribed[assetID] = true
		}
		c.mu.Unlock()
	}

	return nil
}

func (c *Client) batchAssetIDs(assetIDs []string) [][]string {
	if len(assetIDs) <= c.config.BatchSize {
		return [][]string{assetIDs}
	}

	var batches [][]string
	for i := 0; i < len(assetIDs); i += c.config.BatchSize {
		end := i + c.config.BatchSize
		if end > len(assetIDs) {
			end = len(assetIDs)
		}
		batches = append(batches, assetIDs[i:end])
	}

	return batches
}

func (c *Client) sendSubscription(sub types.WSSubscriptionRequest) error {
	c.mu.RLock()
	conn := c.conn
	connected := c.connected
	c.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("marshaling subscription: %w", err)
	}

	conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("writing subscription: %w", err)
	}

	return nil
}

func (c *Client) sendUpdateSubscription(sub types.WSUpdateSubscription) error {
	c.mu.RLock()
	conn := c.conn
	connected := c.connected
	c.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("marshaling update subscription: %w", err)
	}

	conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("writing update subscription: %w", err)
	}

	return nil
}

func (c *Client) readPump() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("readPump panic: %v", r)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("WebSocket read error: %v", err)
					c.errorChan <- err
				}
				return
			}

			if messageType == websocket.TextMessage {
				msgStr := string(message)

				if msgStr == "PONG" {
					continue
				}

				c.metrics.IncrementMessages()
				c.messageChan <- message
			}
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("writePump panic: %v", r)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (c *Client) pingPump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			connected := c.connected
			c.mu.RUnlock()

			if connected && conn != nil {
				conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
				if err := conn.WriteMessage(websocket.TextMessage, []byte("PING")); err != nil {
					log.Printf("Error sending ping: %v", err)
					c.errorChan <- err
					return
				}
			}
		}
	}
}

func (c *Client) processMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case message := <-c.messageChan:
			c.handleMessage(message)
		case err := <-c.errorChan:
			log.Printf("WebSocket error: %v", err)
			c.triggerReconnect()
		}
	}
}

func (c *Client) handleMessage(message []byte) {
	var baseMsg map[string]interface{}
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return
	}

	eventType, ok := baseMsg["event_type"].(string)
	if !ok {
		return
	}

	switch types.WSEventType(eventType) {
	case types.WSEventTypeBook:
		var bookEvent types.WSBookEvent
		if err := json.Unmarshal(message, &bookEvent); err != nil {
			log.Printf("Error unmarshaling book event: %v", err)
			return
		}
		c.handleBookEvent(&bookEvent)
		c.metrics.IncrementBookEvents()

	case types.WSEventTypePriceChange:
		var priceChangeEvent types.WSPriceChangeEvent
		if err := json.Unmarshal(message, &priceChangeEvent); err != nil {
			log.Printf("Error unmarshaling price change event: %v", err)
			return
		}
		c.handlePriceChangeEvent(&priceChangeEvent)
		c.metrics.IncrementPriceChanges()

	default:
		log.Printf("Unhandled event type: %s", eventType)
	}
}

func (c *Client) handleBookEvent(event *types.WSBookEvent) {
	log.Printf("Processing book event for asset %s with %d bids and %d asks\n", event.AssetID, len(event.Bids), len(event.Asks))
	c.orderBookMgr.UpdateFromSnapshot(event.AssetID, event.Bids, event.Asks)
}

func (c *Client) handlePriceChangeEvent(event *types.WSPriceChangeEvent) {
	log.Printf("Processing price change event for market %s with %d changes\n", event.Market, len(event.PriceChanges))
	for _, change := range event.PriceChanges {
		price, _ := parseStringToFloat(change.Price)
		size, _ := parseStringToFloat(change.Size)
		c.orderBookMgr.ApplyDelta(change.AssetID, price, size, change.Side)
	}
}

func (c *Client) triggerReconnect() {
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false
	c.mu.Unlock()

	reconnectDelay := 1 * time.Second
	maxAttempts := int(c.config.MaxReconnect.Seconds())

	for i := 0; i < maxAttempts; i++ {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(reconnectDelay):
			log.Printf("Attempting to reconnect (attempt %d/%d)...", i+1, maxAttempts)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := c.Connect(ctx); err != nil {
				log.Printf("Reconnect failed: %v", err)
				cancel()
				reconnectDelay = min(reconnectDelay*2, c.config.MaxReconnect)
				continue
			}
			cancel()

			c.metrics.IncrementReconnections()

			c.mu.RLock()
			subscribed := make([]string, 0, len(c.subscribed))
			for assetID := range c.subscribed {
				subscribed = append(subscribed, assetID)
			}
			c.mu.RUnlock()

			if len(subscribed) > 0 {
				if err := c.Subscribe(subscribed); err != nil {
					log.Printf("Failed to resubscribe: %v", err)
				} else {
					log.Printf("Successfully reconnected and resubscribed to %d assets", len(subscribed))
					return
				}
			}

			reconnectDelay = min(reconnectDelay*2, c.config.MaxReconnect)
		}
	}

	log.Printf("Failed to reconnect after %d attempts", maxAttempts)
}

func (c *Client) GetOrderBookManager() *types.OrderBookManager {
	return c.orderBookMgr
}

func (c *Client) GetMetrics() *Metrics {
	return c.metrics
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *Client) GetMessageChannel() <-chan []byte {
	return c.messageChan
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func parseStringToFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
