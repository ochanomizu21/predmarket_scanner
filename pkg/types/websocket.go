package types

type WSEventType string

const (
	WSEventTypeBook           WSEventType = "book"
	WSEventTypePriceChange    WSEventType = "price_change"
	WSEventTypeLastTradePrice WSEventType = "last_trade_price"
	WSEventTypeTickSizeChange WSEventType = "tick_size_change"
	WSEventTypeBestBidAsk     WSEventType = "best_bid_ask"
	WSEventTypeNewMarket      WSEventType = "new_market"
	WSEventTypeMarketResolved WSEventType = "market_resolved"
)

type WSSubscriptionRequest struct {
	AssetIDs             []string `json:"assets_ids"`
	Type                 string   `json:"type"`
	InitialDump          *bool    `json:"initial_dump,omitempty"`
	Level                *int     `json:"level,omitempty"`
	CustomFeatureEnabled *bool    `json:"custom_feature_enabled,omitempty"`
}

type WSUpdateSubscription struct {
	Operation string   `json:"operation"`
	AssetIDs  []string `json:"assets_ids"`
}

type WSBookEvent struct {
	EventType WSEventType    `json:"event_type"`
	AssetID   string         `json:"asset_id"`
	Market    string         `json:"market"`
	Bids      []WSOrderLevel `json:"bids"`
	Asks      []WSOrderLevel `json:"asks"`
	Timestamp string         `json:"timestamp"`
	Hash      string         `json:"hash"`
}

type WSOrderLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

type WSPriceChangeEvent struct {
	EventType    string               `json:"event_type"`
	Market       string               `json:"market"`
	PriceChanges []WSPriceChangeLevel `json:"price_changes"`
	Timestamp    string               `json:"timestamp"`
}

type WSPriceChangeLevel struct {
	AssetID string `json:"asset_id"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Side    string `json:"side"`
	Hash    string `json:"hash"`
	BestBid string `json:"best_bid"`
	BestAsk string `json:"best_ask"`
}

type WSMessage interface {
	GetEventType() WSEventType
	GetTimestamp() int64
}

func (m *WSBookEvent) GetEventType() WSEventType {
	return m.EventType
}

func (m *WSBookEvent) GetTimestamp() int64 {
	timestamp, _ := ParseTimestamp(m.Timestamp)
	return timestamp
}

func (m *WSPriceChangeEvent) GetEventType() WSEventType {
	return WSEventType(m.EventType)
}

func (m *WSPriceChangeEvent) GetTimestamp() int64 {
	timestamp, _ := ParseTimestamp(m.Timestamp)
	return timestamp
}
