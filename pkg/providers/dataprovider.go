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

type PriceHistoryPoint struct {
	Timestamp  time.Time
	Price      float64
	TokenID    string
	OrderCount int
}

type DataProvider interface {
	FetchMarkets(maxMarkets int) ([]types.Market, error)
	FetchOrderBooks(tokenIDs []string) (map[string]clients.OrderBook, error)
}

type MarketWithOutcomes struct {
	MarketData
	Outcomes []OutcomeData
}

type Database interface {
	GetLatestSnapshot(marketID string, before time.Time) (*SnapshotData, error)
	GetLatestSnapshotByTokenID(tokenID string, before time.Time) (*SnapshotData, error)
	GetSnapshotData(snapshotID int64) (*SnapshotDetail, error)
	GetOrderBookLevels(snapshotID int64, tokenID, side string) ([]OrderBookLevel, error)
	FetchMarketsAtTime(targetTime time.Time, maxMarkets, offset int) ([]MarketData, error)
	FetchMarketsWithOutcomesAtTime(targetTime time.Time, maxMarkets, offset int) ([]MarketWithOutcomes, error)
	FetchMarketsWithOrderBookAtTime(targetTime time.Time, maxMarkets, offset int) ([]MarketWithOutcomes, error)
	GetTimestampsInRange(startTime, endTime time.Time) ([]time.Time, error)
}

type LiveDataProvider struct {
	client        *clients.PolymarketClient
	offset        int
	limit         int
	includeClosed bool
}

func NewLiveDataProvider(offset, limit int, includeClosed bool) *LiveDataProvider {
	return &LiveDataProvider{
		client:        clients.NewPolymarketClient(),
		offset:        offset,
		limit:         limit,
		includeClosed: includeClosed,
	}
}

func (p *LiveDataProvider) FetchMarkets(maxMarkets int) ([]types.Market, error) {
	markets, err := p.client.FetchMarketsFilterOffset(maxMarkets, p.offset, 0, 0, p.includeClosed)
	if err != nil {
		return nil, err
	}

	if p.offset > 0 || p.limit > 0 {
		if p.offset >= len(markets) {
			return []types.Market{}, nil
		}

		end := p.offset + p.limit
		if end > len(markets) || p.limit == 0 {
			end = len(markets)
		}

		return markets[p.offset:end], nil
	}

	return markets, nil
}

func (p *LiveDataProvider) FetchOrderBooks(tokenIDs []string) (map[string]clients.OrderBook, error) {
	return p.client.FetchOrderBooks(tokenIDs)
}

type HistoricalDataProvider struct {
	db           Database
	targetTime   time.Time
	offset       int
	useOrderBook bool
}

func NewHistoricalDataProvider(db Database, targetTime time.Time, offset int) *HistoricalDataProvider {
	return &HistoricalDataProvider{
		db:         db,
		targetTime: targetTime,
		offset:     offset,
	}
}

func NewHistoricalDataProviderWithOrderBook(db Database, targetTime time.Time, offset int) *HistoricalDataProvider {
	return &HistoricalDataProvider{
		db:           db,
		targetTime:   targetTime,
		offset:       offset,
		useOrderBook: true,
	}
}

func (p *HistoricalDataProvider) FetchMarkets(maxMarkets int) ([]types.Market, error) {
	var marketData []MarketWithOutcomes
	var err error

	if p.useOrderBook {
		marketData, err = p.db.FetchMarketsWithOrderBookAtTime(p.targetTime, maxMarkets, p.offset)
		if err != nil {
			return nil, err
		}
	} else {
		marketData, err = p.db.FetchMarketsWithOutcomesAtTime(p.targetTime, maxMarkets, p.offset)
		if err != nil {
			return nil, err
		}
	}

	markets := make([]types.Market, 0, len(marketData))
	for _, md := range marketData {
		m := types.Market{
			ID:        md.ID,
			Question:  md.Question,
			Platform:  types.Polymarket,
			Liquidity: md.Liquidity,
			Volume:    md.Volume,
			EndTime:   md.EndTime,
		}

		m.Outcomes = make([]types.Outcome, 0, len(md.Outcomes))
		for _, o := range md.Outcomes {
			m.Outcomes = append(m.Outcomes, types.Outcome{
				Name:           o.Name,
				Price:          o.BestBid,
				Side:           types.Bid,
				OrderBookDepth: 0,
			})
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
		snapshot, err := p.db.GetLatestSnapshotByTokenID(tokenID, p.targetTime)
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
