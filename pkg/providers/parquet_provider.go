package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/storage"
	"github.com/xitongsys/parquet-go/reader"
)

type ParquetHistoricalProvider struct {
	dataDir string
}

func NewParquetHistoricalProvider(dataDir string) *ParquetHistoricalProvider {
	return &ParquetHistoricalProvider{
		dataDir: dataDir,
	}
}

func (p *ParquetHistoricalProvider) GetAvailableDates() ([]time.Time, error) {
	entries, err := os.ReadDir(p.dataDir)
	if err != nil {
		return nil, fmt.Errorf("reading data directory: %w", err)
	}

	var dates []time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) < 15 || name[:13] != "market_data_" || name[len(name)-8:] != ".parquet" {
			continue
		}

		dateStr := name[13 : len(name)-8]
		t, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			dates = append(dates, t)
		}
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	return dates, nil
}

func (p *ParquetHistoricalProvider) GetSnapshotsAtTime(targetTime time.Time) ([]HistoricalSnapshot, error) {
	dateStr := targetTime.Format("2006-01-02")
	filename := filepath.Join(p.dataDir, fmt.Sprintf("market_data_%s.parquet", dateStr))

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening parquet file: %w", err)
	}
	defer file.Close()

	localFile := &storage.LocalFile{File: file}

	fr, err := reader.NewParquetReader(localFile, new(storage.ParquetRecord), 4)
	if err != nil {
		return nil, fmt.Errorf("creating parquet reader: %w", err)
	}
	defer fr.ReadStop()

	var snapshots []HistoricalSnapshot

	num := int(fr.GetNumRows())
	for i := 0; i < num; i++ {
		rec := new(storage.ParquetRecord)
		if err := fr.Read(rec); err != nil {
			continue
		}

		timestamp := time.UnixMilli(rec.Timestamp)

		if !timestamp.Before(targetTime) && !timestamp.After(targetTime.Add(time.Minute)) {
			continue
		}

		snapshot := HistoricalSnapshot{
			AssetID:   rec.AssetID,
			Market:    rec.Market,
			EventType: rec.EventType,
			Timestamp: timestamp.Format(time.RFC3339Nano),
		}

		if rec.EventType == "book" && rec.BidsJSON != "" {
			var bids []OrderBookLevel
			if err := json.Unmarshal([]byte(rec.BidsJSON), &bids); err == nil {
				snapshot.Bids = bids
			}

			var asks []OrderBookLevel
			if err := json.Unmarshal([]byte(rec.AsksJSON), &asks); err == nil {
				snapshot.Asks = asks
			}
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func (p *ParquetHistoricalProvider) GetOrderBooksAtTime(targetTime time.Time) (map[string]OrderBookSnapshot, error) {
	snapshots, err := p.GetSnapshotsAtTime(targetTime)
	if err != nil {
		return nil, err
	}

	books := make(map[string]OrderBookSnapshot)

	for _, snap := range snapshots {
		if snap.EventType == "book" && snap.AssetID != "" {
			books[snap.AssetID] = OrderBookSnapshot{
				AssetID:   snap.AssetID,
				Bids:      snap.Bids,
				Asks:      snap.Asks,
				Timestamp: snap.Timestamp,
			}
		}
	}

	return books, nil
}

func (p *ParquetHistoricalProvider) GetMarketsAtTime(targetTime time.Time, maxMarkets, offset int) ([]MarketWithOutcomes, error) {
	books, err := p.GetOrderBooksAtTime(targetTime)
	if err != nil {
		return nil, err
	}

	marketsFile := filepath.Join(p.dataDir, "markets.json")
	marketsData, err := os.ReadFile(marketsFile)
	if err != nil {
		return nil, fmt.Errorf("reading markets file: %w", err)
	}

	type LocalMarketData struct {
		ID           string   `json:"id"`
		Question     string   `json:"question"`
		Liquidity    float64  `json:"liquidity"`
		Volume       float64  `json:"volume"`
		ClobTokenIDs []string `json:"clob_token_ids"`
		Outcomes     []struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		} `json:"outcomes"`
	}

	var marketList []LocalMarketData

	if err := json.Unmarshal(marketsData, &marketList); err != nil {
		return nil, fmt.Errorf("unmarshaling markets: %w", err)
	}

	var sortedMarkets []*LocalMarketData

	for i := range marketList {
		sortedMarkets = append(sortedMarkets, &marketList[i])
	}

	sort.Slice(sortedMarkets, func(i, j int) bool {
		return sortedMarkets[i].Liquidity > sortedMarkets[j].Liquidity
	})

	start := offset
	end := offset + maxMarkets
	if end > len(sortedMarkets) {
		end = len(sortedMarkets)
	}

	if start >= len(sortedMarkets) {
		return []MarketWithOutcomes{}, nil
	}

	result := make([]MarketWithOutcomes, 0, end-start)

	for i := start; i < end; i++ {
		market := sortedMarkets[i]
		marketData := MarketWithOutcomes{
			MarketData: MarketData{
				ID:        market.ID,
				Question:  market.Question,
				Liquidity: market.Liquidity,
				Volume:    market.Volume,
			},
			Outcomes: make([]OutcomeData, 0, len(market.Outcomes)),
		}

		for _, outcome := range market.Outcomes {
			outcomeData := OutcomeData{
				Name: outcome.Name,
			}

			for j, tokenID := range market.ClobTokenIDs {
				if j < len(market.Outcomes) && market.Outcomes[j].Name == outcome.Name {
					if book, ok := books[tokenID]; ok && len(book.Asks) > 0 {
						outcomeData.BestAsk = book.Asks[0].Price
						outcomeData.BestBid = book.Bids[0].Price
					} else {
						outcomeData.BestAsk = outcome.Price
						outcomeData.BestBid = outcome.Price
					}
					break
				}
			}

			marketData.Outcomes = append(marketData.Outcomes, outcomeData)
		}

		result = append(result, marketData)
	}

	return result, nil
}
