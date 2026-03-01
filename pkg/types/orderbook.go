package types

import (
	"sort"
	"strconv"
	"sync"
)

type OrderBookLevel struct {
	Price float64
	Size  float64
}

type OrderBook struct {
	AssetID string
	Bids    map[float64]float64
	Asks    map[float64]float64
	mu      sync.RWMutex
}

func parseOrderLevel(level WSOrderLevel) (float64, float64) {
	price, _ := strconv.ParseFloat(level.Price, 64)
	size, _ := strconv.ParseFloat(level.Size, 64)
	return price, size
}

func NewOrderBook(assetID string) *OrderBook {
	return &OrderBook{
		AssetID: assetID,
		Bids:    make(map[float64]float64),
		Asks:    make(map[float64]float64),
	}
}

func (ob *OrderBook) UpdateFromSnapshot(bids []WSOrderLevel, asks []WSOrderLevel) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Bids = make(map[float64]float64)
	ob.Asks = make(map[float64]float64)

	for _, bid := range bids {
		price, size := parseOrderLevel(bid)
		if price > 0 {
			ob.Bids[price] = size
		}
	}

	for _, ask := range asks {
		price, size := parseOrderLevel(ask)
		if price > 0 {
			ob.Asks[price] = size
		}
	}
}

func (ob *OrderBook) ApplyDelta(price, size float64, side string) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if side == "BUY" || side == "bid" {
		if size > 0 {
			ob.Bids[price] = size
		} else {
			delete(ob.Bids, price)
		}
	} else if side == "SELL" || side == "ask" {
		if size > 0 {
			ob.Asks[price] = size
		} else {
			delete(ob.Asks, price)
		}
	}
}

func (ob *OrderBook) GetBidsDesc() []OrderBookLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	levels := make([]OrderBookLevel, 0, len(ob.Bids))
	for price, size := range ob.Bids {
		levels = append(levels, OrderBookLevel{Price: price, Size: size})
	}

	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Price > levels[j].Price
	})

	return levels
}

func (ob *OrderBook) GetAsksAsc() []OrderBookLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	levels := make([]OrderBookLevel, 0, len(ob.Asks))
	for price, size := range ob.Asks {
		levels = append(levels, OrderBookLevel{Price: price, Size: size})
	}

	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Price < levels[j].Price
	})

	return levels
}

func (ob *OrderBook) GetBestBid() (float64, float64) {
	bids := ob.GetBidsDesc()
	if len(bids) == 0 {
		return 0, 0
	}
	return bids[0].Price, bids[0].Size
}

func (ob *OrderBook) GetBestAsk() (float64, float64) {
	asks := ob.GetAsksAsc()
	if len(asks) == 0 {
		return 0, 0
	}
	return asks[0].Price, asks[0].Size
}

func (ob *OrderBook) GetLiquidityAbovePrice(price float64, side string) float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var total float64
	if side == "BUY" || side == "bid" {
		for p, size := range ob.Bids {
			if p >= price {
				total += size * p
			}
		}
	} else if side == "SELL" || side == "ask" {
		for p, size := range ob.Asks {
			if p <= price {
				total += size * p
			}
		}
	}
	return total
}

func (ob *OrderBook) GetMidPrice() float64 {
	bestBid, _ := ob.GetBestBid()
	bestAsk, _ := ob.GetBestAsk()

	if bestBid == 0 || bestAsk == 0 {
		return 0
	}

	return (bestBid + bestAsk) / 2
}

func (ob *OrderBook) SnapshotToMap() map[string]interface{} {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := ob.GetBidsDesc()
	asks := ob.GetAsksAsc()

	bidLevels := make([]map[string]float64, 0, len(bids))
	for _, b := range bids {
		bidLevels = append(bidLevels, map[string]float64{"price": b.Price, "size": b.Size})
	}

	askLevels := make([]map[string]float64, 0, len(asks))
	for _, a := range asks {
		askLevels = append(askLevels, map[string]float64{"price": a.Price, "size": a.Size})
	}

	return map[string]interface{}{
		"asset_id": ob.AssetID,
		"bids":     bidLevels,
		"asks":     askLevels,
	}
}

type OrderBookManager struct {
	books map[string]*OrderBook
	mu    sync.RWMutex
}

func NewOrderBookManager() *OrderBookManager {
	return &OrderBookManager{
		books: make(map[string]*OrderBook),
	}
}

func (obm *OrderBookManager) GetOrCreate(assetID string) *OrderBook {
	obm.mu.Lock()
	defer obm.mu.Unlock()

	if book, exists := obm.books[assetID]; exists {
		return book
	}

	book := NewOrderBook(assetID)
	obm.books[assetID] = book
	return book
}

func (obm *OrderBookManager) Get(assetID string) (*OrderBook, bool) {
	obm.mu.RLock()
	defer obm.mu.RUnlock()

	book, exists := obm.books[assetID]
	return book, exists
}

func (obm *OrderBookManager) GetAllOrderBooks() map[string]*OrderBook {
	obm.mu.RLock()
	defer obm.mu.RUnlock()

	result := make(map[string]*OrderBook, len(obm.books))
	for k, v := range obm.books {
		result[k] = v
	}
	return result
}

func (obm *OrderBookManager) UpdateFromSnapshot(assetID string, bids []WSOrderLevel, asks []WSOrderLevel) {
	book := obm.GetOrCreate(assetID)
	book.UpdateFromSnapshot(bids, asks)
}

func (obm *OrderBookManager) ApplyDelta(assetID string, price, size float64, side string) {
	book := obm.GetOrCreate(assetID)
	book.ApplyDelta(price, size, side)
}
