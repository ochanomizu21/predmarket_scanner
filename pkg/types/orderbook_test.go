package types

import (
	"math"
	"sync"
	"testing"
)

func TestNewOrderBook(t *testing.T) {
	assetID := "test-asset-id"
	ob := NewOrderBook(assetID)

	if ob.AssetID != assetID {
		t.Errorf("Expected AssetID %s, got %s", assetID, ob.AssetID)
	}

	if len(ob.Bids) != 0 {
		t.Errorf("Expected empty Bids map, got %d entries", len(ob.Bids))
	}

	if len(ob.Asks) != 0 {
		t.Errorf("Expected empty Asks map, got %d entries", len(ob.Asks))
	}
}

func TestUpdateFromSnapshot(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	bids := []WSOrderLevel{
		{Price: "0.50", Size: "100"},
		{Price: "0.49", Size: "50"},
	}

	asks := []WSOrderLevel{
		{Price: "0.51", Size: "75"},
		{Price: "0.52", Size: "25"},
	}

	ob.UpdateFromSnapshot(bids, asks)

	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.Bids) != 2 {
		t.Errorf("Expected 2 bid levels, got %d", len(ob.Bids))
	}

	if len(ob.Asks) != 2 {
		t.Errorf("Expected 2 ask levels, got %d", len(ob.Asks))
	}

	if _, exists := ob.Bids[0.50]; !exists {
		t.Errorf("Expected bid at price 0.50 to exist")
	}

	if _, exists := ob.Asks[0.51]; !exists {
		t.Errorf("Expected ask at price 0.51 to exist")
	}
}

func TestApplyDelta(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	ob.ApplyDelta(0.50, 100, "BUY")
	ob.ApplyDelta(0.51, 75, "SELL")
	ob.ApplyDelta(0.50, 0, "BUY")

	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.Bids) != 1 {
		t.Errorf("Expected 1 bid level after deletion, got %d", len(ob.Bids))
	}

	if len(ob.Asks) != 1 {
		t.Errorf("Expected 1 ask level, got %d", len(ob.Asks))
	}

	if bidSize, exists := ob.Bids[0.50]; !exists || bidSize != 100 {
		t.Errorf("Expected bid at price 0.50 with size 100, got exists=%v, size=%v", exists, bidSize)
	}

	if askSize, exists := ob.Asks[0.51]; !exists || askSize != 75 {
		t.Errorf("Expected ask at price 0.51 with size 75, got exists=%v, size=%v", exists, askSize)
	}
}

func TestGetBidsDesc(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	bids := []WSOrderLevel{
		{Price: "0.50", Size: "100"},
		{Price: "0.49", Size: "50"},
		{Price: "0.48", Size: "25"},
	}

	ob.UpdateFromSnapshot(bids, []WSOrderLevel{})

	levels := ob.GetBidsDesc()

	if len(levels) != 3 {
		t.Errorf("Expected 3 bid levels, got %d", len(levels))
	}

	for i := 0; i < len(levels)-1; i++ {
		if levels[i].Price <= levels[i+1].Price {
			t.Errorf("Expected bids to be in descending order at index %d", i)
		}
	}
}

func TestGetAsksAsc(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	asks := []WSOrderLevel{
		{Price: "0.52", Size: "25"},
		{Price: "0.51", Size: "75"},
		{Price: "0.50", Size: "100"},
	}

	ob.UpdateFromSnapshot([]WSOrderLevel{}, asks)

	levels := ob.GetAsksAsc()

	if len(levels) != 3 {
		t.Errorf("Expected 3 ask levels, got %d", len(levels))
	}

	for i := 0; i < len(levels)-1; i++ {
		if levels[i].Price >= levels[i+1].Price {
			t.Errorf("Expected asks to be in ascending order at index %d", i)
		}
	}
}

func TestGetBestBid(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	ob.UpdateFromSnapshot([]WSOrderLevel{{Price: "0.50", Size: "100"}}, []WSOrderLevel{})

	bestBid, bestSize := ob.GetBestBid()

	if bestBid != 0.50 {
		t.Errorf("Expected best bid price 0.50, got %f", bestBid)
	}

	if bestSize != 100 {
		t.Errorf("Expected best bid size 100, got %f", bestSize)
	}

	ob.mu.RLock()
	ob.Bids[0.50] = 50
	ob.mu.RUnlock()

	bestBid, bestSize = ob.GetBestBid()

	if bestBid != 0.50 {
		t.Errorf("Expected best bid price 0.50 after update, got %f", bestBid)
	}

	if bestSize != 50 {
		t.Errorf("Expected best bid size 50 after update, got %f", bestSize)
	}
}

func TestGetBestAsk(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	ob.UpdateFromSnapshot([]WSOrderLevel{}, []WSOrderLevel{{Price: "0.51", Size: "75"}})

	bestAsk, bestSize := ob.GetBestAsk()

	if bestAsk != 0.51 {
		t.Errorf("Expected best ask price 0.51, got %f", bestAsk)
	}

	if bestSize != 75 {
		t.Errorf("Expected best ask size 75, got %f", bestSize)
	}

	ob.mu.RLock()
	ob.Asks[0.51] = 100
	ob.mu.RUnlock()

	bestAsk, bestSize = ob.GetBestAsk()

	if bestAsk != 0.51 {
		t.Errorf("Expected best ask price 0.51 after update, got %f", bestAsk)
	}

	if bestSize != 100 {
		t.Errorf("Expected best ask size 100 after update, got %f", bestSize)
	}
}

func TestGetLiquidityAbovePrice(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	ob.UpdateFromSnapshot(
		[]WSOrderLevel{{Price: "0.50", Size: "100"}, {Price: "0.49", Size: "50"}},
		[]WSOrderLevel{{Price: "0.51", Size: "75"}},
	)

	liquidity := ob.GetLiquidityAbovePrice(0.50, "bid")

	expected := 0.50*100 + 0.49*50
	if math.Abs(liquidity-expected) > 0.01 {
		t.Errorf("Expected liquidity %f for bids above 0.50, got %f", expected, liquidity)
	}

	liquidity = ob.GetLiquidityAbovePrice(0.51, "ask")

	expected = 0.51 * 75
	if math.Abs(liquidity-expected) > 0.01 {
		t.Errorf("Expected liquidity %f for asks below 0.51, got %f", expected, liquidity)
	}
}

func TestGetMidPrice(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	ob.UpdateFromSnapshot(
		[]WSOrderLevel{{Price: "0.50", Size: "100"}},
		[]WSOrderLevel{{Price: "0.51", Size: "75"}},
	)

	midPrice := ob.GetMidPrice()

	expected := (0.50 + 0.51) / 2
	if math.Abs(midPrice-expected) > 0.001 {
		t.Errorf("Expected mid price %f, got %f", expected, midPrice)
	}
}

func TestOrderBookManager(t *testing.T) {
	obm := NewOrderBookManager()

	ob1 := obm.GetOrCreate("asset-1")
	ob2 := obm.GetOrCreate("asset-1")
	ob3 := obm.GetOrCreate("asset-2")

	if ob1 != ob2 {
		t.Errorf("Expected same order book instance for same asset ID")
	}

	if ob1 == ob3 {
		t.Errorf("Expected different order book instances for different asset IDs")
	}

	ob1.UpdateFromSnapshot([]WSOrderLevel{{Price: "0.50", Size: "100"}}, []WSOrderLevel{})

	ob1.mu.RLock()
	bestBid1, _ := ob1.GetBestBid()
	ob1.mu.RUnlock()

	ob2.mu.RLock()
	bestBid2, _ := ob2.GetBestBid()
	ob2.mu.RUnlock()

	if bestBid1 != bestBid2 {
		t.Errorf("Expected best bid to be the same for same asset instance")
	}

	allBooks := obm.GetAllOrderBooks()

	if len(allBooks) != 2 {
		t.Errorf("Expected 2 order books, got %d", len(allBooks))
	}

	obm.mu.RLock()
	book, exists := obm.books["asset-1"]
	obm.mu.RUnlock()

	if !exists {
		t.Errorf("Expected asset-1 to exist in manager")
	}

	if book == nil {
		t.Errorf("Expected non-nil order book for asset-1")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ob := NewOrderBook("test-asset-id")

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			price := 0.49 + float64(n%5)*0.01
			ob.ApplyDelta(price, float64(n+1), "BUY")
		}(i)

		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ob.GetBestBid()
			ob.GetBestAsk()
			ob.GetMidPrice()
		}(i)
	}

	wg.Wait()

	ob.mu.RLock()
	numBids := len(ob.Bids)
	ob.mu.RUnlock()

	if numBids != 5 {
		t.Errorf("Expected 5 bid levels after concurrent updates, got %d", numBids)
	}
}
