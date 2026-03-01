package providers

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type JSONLHistoricalProvider struct {
	dataDir string
}

func NewJSONLHistoricalProvider(dataDir string) *JSONLHistoricalProvider {
	return &JSONLHistoricalProvider{
		dataDir: dataDir,
	}
}

type HistoricalMarketData struct {
	Markets   []MarketWithOutcomes `json:"markets"`
	Timestamp time.Time            `json:"timestamp"`
}

type HistoricalSnapshot struct {
	AssetID   string           `json:"asset_id"`
	Market    string           `json:"market"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	EventType string           `json:"event_type"`
	Timestamp string           `json:"timestamp"`
}

func (p *JSONLHistoricalProvider) GetAvailableDates() ([]time.Time, error) {
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
		var dateStr string

		if len(name) >= 18 && name[:13] == "market_data_" && name[len(name)-9:] == ".jsonl.gz" {
			dateStr = name[13 : len(name)-9]
		} else if len(name) >= 15 && name[:13] == "market_data_" && name[len(name)-6:] == ".jsonl" {
			dateStr = name[13 : len(name)-6]
		} else {
			continue
		}

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

func (p *JSONLHistoricalProvider) GetSnapshotsAtTime(targetTime time.Time) ([]HistoricalSnapshot, error) {
	dateStr := targetTime.Format("2006-01-02")

	gzFilename := filepath.Join(p.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", dateStr))
	jsonlFilename := filepath.Join(p.dataDir, fmt.Sprintf("market_data_%s.jsonl", dateStr))

	var file *os.File
	var err error
	var isGzipped bool

	if _, statErr := os.Stat(gzFilename); statErr == nil {
		file, err = os.Open(gzFilename)
		if err != nil {
			return nil, fmt.Errorf("opening gzipped JSONL file: %w", err)
		}
		isGzipped = true
	} else {
		file, err = os.Open(jsonlFilename)
		if err != nil {
			return nil, fmt.Errorf("opening JSONL file: %w", err)
		}
		isGzipped = false
	}
	defer file.Close()

	var snapshots []HistoricalSnapshot
	var scanner *bufio.Scanner

	if isGzipped {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("opening gzip reader: %w", err)
		}
		defer gz.Close()
		scanner = bufio.NewScanner(gz)
	} else {
		scanner = bufio.NewScanner(file)
	}

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}

		timestampStr, _ := data["timestamp"].(string)
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			continue
		}

		if !timestamp.Before(targetTime) && !timestamp.After(targetTime.Add(time.Minute)) {
			continue
		}

		eventType := ""
		if et, ok := data["event_type"].(string); ok {
			eventType = et
		}

		assetID := ""
		if aid, ok := data["asset_id"].(string); ok {
			assetID = aid
		}

		market := ""
		if m, ok := data["market"].(string); ok {
			market = m
		}

		var bids, asks []OrderBookLevel

		if eventType == "book" {
			if bidsData, ok := data["bids"].([]interface{}); ok {
				for _, b := range bidsData {
					if bidMap, ok := b.(map[string]interface{}); ok {
						price := parseFloat(bidMap, "price")
						size := parseFloat(bidMap, "size")
						bids = append(bids, OrderBookLevel{Price: price, Size: size})
					}
				}
			}

			if asksData, ok := data["asks"].([]interface{}); ok {
				for _, a := range asksData {
					if askMap, ok := a.(map[string]interface{}); ok {
						price := parseFloat(askMap, "price")
						size := parseFloat(askMap, "size")
						asks = append(asks, OrderBookLevel{Price: price, Size: size})
					}
				}
			}
		}

		snapshots = append(snapshots, HistoricalSnapshot{
			AssetID:   assetID,
			Market:    market,
			Bids:      bids,
			Asks:      asks,
			EventType: eventType,
			Timestamp: timestampStr,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning JSONL file: %w", err)
	}

	return snapshots, nil
}

func (p *JSONLHistoricalProvider) GetOrderBooksAtTime(targetTime time.Time) (map[string]OrderBookSnapshot, error) {
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

type OrderBookSnapshot struct {
	AssetID   string
	Bids      []OrderBookLevel
	Asks      []OrderBookLevel
	Timestamp string
}

func parseFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case string:
			var f float64
			fmt.Sscanf(val, "%f", &f)
			return f
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

func (p *JSONLHistoricalProvider) GetMarketsAtTime(targetTime time.Time, maxMarkets, offset int) ([]MarketWithOutcomes, error) {
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
