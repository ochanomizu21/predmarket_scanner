package providers

import (
	"fmt"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

type MarketData struct {
	ID        string
	Question  string
	EndTime   *time.Time
	Liquidity float64
	Volume    float64
}

type SnapshotData struct {
	ID        int64
	MarketID  string
	Timestamp time.Time
}

type SnapshotDetail struct {
	ID        int64
	MarketID  string
	Timestamp time.Time
	Question  string
	Liquidity float64
	Volume    float64
	Outcomes  []OutcomeData
}

type OutcomeData struct {
	Name    string
	BestBid float64
	BestAsk float64
}

type OrderBookLevel struct {
	Price float64
	Size  float64
}

type DataProvider interface {
	FetchMarkets(maxMarkets int) ([]types.Market, error)
	FetchOrderBooks(tokenIDs []string) (map[string]clients.OrderBook, error)
}

type LiveDataProvider struct {
	client *clients.PolymarketClient
}

func NewLiveDataProvider() *LiveDataProvider {
	return &LiveDataProvider{
		client: clients.NewPolymarketClient(),
	}
}

func (p *LiveDataProvider) FetchMarkets(maxMarkets int) ([]types.Market, error) {
	return p.client.FetchMarkets(maxMarkets)
}

func (p *LiveDataProvider) FetchOrderBooks(tokenIDs []string) (map[string]clients.OrderBook, error) {
	return p.client.FetchOrderBooks(tokenIDs)
}

type HistoricalDataProvider struct {
	db         Database
	targetTime time.Time
}

type Database interface {
	GetLatestSnapshot(marketID string, before time.Time) (*SnapshotData, error)
	GetSnapshotData(snapshotID int64) (*SnapshotDetail, error)
	GetOrderBookLevels(snapshotID int64, outcomeName, side string) ([]OrderBookLevel, error)
	FetchMarketsAtTime(targetTime time.Time, maxMarkets int) ([]MarketData, error)
}

func NewHistoricalDataProvider(db Database, targetTime time.Time) *HistoricalDataProvider {
	return &HistoricalDataProvider{
		db:         db,
		targetTime: targetTime,
	}
}

func (p *HistoricalDataProvider) FetchMarkets(maxMarkets int) ([]types.Market, error) {
	marketData, err := p.db.FetchMarketsAtTime(p.targetTime, maxMarkets)
	if err != nil {
		return nil, err
	}

	var markets []types.Market
	for _, md := range marketData {
		m := types.Market{
			ID:        md.ID,
			Question:  md.Question,
			Platform:  types.Polymarket,
			Liquidity: md.Liquidity,
			Volume:    md.Volume,
			EndTime:   md.EndTime,
		}

		m.Outcomes, err = p.fetchMarketOutcomes(md.ID)
		if err != nil {
			continue
		}

		markets = append(markets, m)
	}

	return markets, nil
}

func (p *HistoricalDataProvider) fetchMarketOutcomes(marketID string) ([]types.Outcome, error) {
	snapshot, err := p.db.GetLatestSnapshot(marketID, p.targetTime)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, nil
	}

	detail, err := p.db.GetSnapshotData(snapshot.ID)
	if err != nil {
		return nil, err
	}

	var outcomes []types.Outcome
	for _, o := range detail.Outcomes {
		outcomes = append(outcomes, types.Outcome{
			Name:           o.Name,
			Price:          o.BestAsk,
			Side:           types.Ask,
			OrderBookDepth: 0,
		})
	}

	return outcomes, nil
}

func (p *HistoricalDataProvider) FetchOrderBooks(tokenIDs []string) (map[string]clients.OrderBook, error) {
	books := make(map[string]clients.OrderBook)

	for _, tokenID := range tokenIDs {
		snapshot, err := p.db.GetLatestSnapshot(tokenID, p.targetTime)
		if err != nil {
			return nil, err
		}
		if snapshot == nil {
			continue
		}

		book := clients.OrderBook{
			Market:    tokenID,
			AssetID:   tokenID,
			Timestamp: snapshot.Timestamp.Format(time.RFC3339),
		}

		bids, err := p.db.GetOrderBookLevels(snapshot.ID, tokenID, "bid")
		if err != nil {
			return nil, err
		}

		for _, bid := range bids {
			book.Bids = append(book.Bids, clients.OrderLevel{
				Price: fmt.Sprintf("%.6f", bid.Price),
				Size:  fmt.Sprintf("%.6f", bid.Size),
			})
		}

		asks, err := p.db.GetOrderBookLevels(snapshot.ID, tokenID, "ask")
		if err != nil {
			return nil, err
		}

		for _, ask := range asks {
			book.Asks = append(book.Asks, clients.OrderLevel{
				Price: fmt.Sprintf("%.6f", ask.Price),
				Size:  fmt.Sprintf("%.6f", ask.Size),
			})
		}

		books[tokenID] = book
	}

	return books, nil
}
