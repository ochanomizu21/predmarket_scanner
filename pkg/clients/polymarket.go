package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

const (
	gammaAPIBase = "https://gamma-api.polymarket.com"
	clobAPIBase  = "https://clob.polymarket.com"
)

type PolymarketClient struct {
	httpClient *http.Client
}

func NewPolymarketClient() *PolymarketClient {
	return &PolymarketClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type GammaMarket struct {
	ID            string          `json:"id"`
	Question      string          `json:"question"`
	ConditionID   string          `json:"conditionId"`
	Slug          string          `json:"slug"`
	EndDate       string          `json:"endDate"`
	Category      string          `json:"category"`
	AmmType       string          `json:"amm_type"`
	Liquidity     string          `json:"liquidity"`
	Volume        string          `json:"volume"`
	Outcomes      JSONStringSlice `json:"outcomes"`
	OutcomePrices JSONStringSlice `json:"outcomePrices"`
	ClobTokenIDs  JSONStringSlice `json:"clobTokenIds"`
	BestBid       float64         `json:"bestBid"`
	BestAsk       float64         `json:"bestAsk"`
	Spread        float64         `json:"spread"`
}

type JSONStringSlice []string

func (j *JSONStringSlice) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), (*[]string)(j))
}

type OrderLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

type OrderBook struct {
	Market         string       `json:"market"`
	AssetID        string       `json:"asset_id"`
	Timestamp      string       `json:"timestamp"`
	Bids           []OrderLevel `json:"bids"`
	Asks           []OrderLevel `json:"asks"`
	MinOrderSize   string       `json:"min_order_size"`
	TickSize       string       `json:"tick_size"`
	LastTradePrice string       `json:"last_trade_price"`
}

func (c *PolymarketClient) FetchAllMarkets() ([]types.Market, error) {
	return c.FetchMarkets(0)
}

func (c *PolymarketClient) FetchMarkets(limit int) ([]types.Market, error) {
	return c.FetchMarketsFilter(limit, 0, 0)
}

func (c *PolymarketClient) FetchMarketsFilter(limit, minOutcomes, maxOutcomes int) ([]types.Market, error) {
	gammaMarkets, err := c.fetchGammaMarketsWithLimit(limit)
	if err != nil {
		return nil, fmt.Errorf("fetching gamma markets: %w", err)
	}

	var markets []types.Market
	for _, gammaMarket := range gammaMarkets {
		market, ok := c.convertMarket(gammaMarket)
		if !ok {
			continue
		}

		outcomeCount := len(market.Outcomes)
		if minOutcomes > 0 && outcomeCount < minOutcomes {
			continue
		}
		if maxOutcomes > 0 && outcomeCount > maxOutcomes {
			continue
		}

		markets = append(markets, market)
	}

	return markets, nil
}

func (c *PolymarketClient) fetchGammaMarkets() ([]GammaMarket, error) {
	return c.fetchGammaMarketsWithLimit(0)
}

func (c *PolymarketClient) fetchGammaMarketsWithLimit(maxMarkets int) ([]GammaMarket, error) {
	var allMarkets []GammaMarket
	offset := 0
	limit := 500

	for {
		if maxMarkets > 0 && len(allMarkets) >= maxMarkets {
			break
		}

		url := fmt.Sprintf("%s/markets?limit=%d&offset=%d&closed=false", gammaAPIBase, limit, offset)

		resp, err := c.httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		var marketsResponse []GammaMarket
		if err := json.Unmarshal(body, &marketsResponse); err != nil {
			return nil, fmt.Errorf("unmarshaling response: %w", err)
		}

		if len(marketsResponse) == 0 {
			break
		}

		allMarkets = append(allMarkets, marketsResponse...)

		if len(marketsResponse) < limit {
			break
		}

		offset += limit
	}

	if maxMarkets > 0 && len(allMarkets) > maxMarkets {
		allMarkets = allMarkets[:maxMarkets]
	}

	return allMarkets, nil
}

func (c *PolymarketClient) convertMarket(gammaMarket GammaMarket) (types.Market, bool) {
	liquidity, _ := strconv.ParseFloat(gammaMarket.Liquidity, 64)
	volume, _ := strconv.ParseFloat(gammaMarket.Volume, 64)

	var endTime *time.Time
	if gammaMarket.EndDate != "" {
		t, err := time.Parse(time.RFC3339, gammaMarket.EndDate)
		if err == nil {
			endTime = &t
		}
	}

	outcomes := c.extractOutcomes(gammaMarket)
	if len(outcomes) == 0 {
		return types.Market{}, false
	}

	return types.Market{
		ID:        gammaMarket.ID,
		Question:  gammaMarket.Question,
		Platform:  types.Polymarket,
		Outcomes:  outcomes,
		Liquidity: liquidity,
		Volume:    volume,
		EndTime:   endTime,
	}, true
}

func (c *PolymarketClient) extractOutcomes(gammaMarket GammaMarket) []types.Outcome {
	var outcomes []types.Outcome

	for i, outcomeName := range gammaMarket.Outcomes {
		price := 0.0
		if i < len(gammaMarket.OutcomePrices) {
			price, _ = strconv.ParseFloat(gammaMarket.OutcomePrices[i], 64)
		}

		outcomes = append(outcomes, types.Outcome{
			Name:           outcomeName,
			Price:          price,
			Side:           types.Bid,
			OrderBookDepth: 0,
		})
	}

	return outcomes
}

func (c *PolymarketClient) FetchOrderBooks(tokenIDs []string) (map[string]OrderBook, error) {
	books := make(map[string]OrderBook)

	for _, tokenID := range tokenIDs {
		book, err := c.FetchOrderBook(tokenID)
		if err != nil {
			return nil, fmt.Errorf("fetching order book for token %s: %w", tokenID, err)
		}
		books[tokenID] = book
	}

	return books, nil
}

func (c *PolymarketClient) FetchOrderBook(tokenID string) (OrderBook, error) {
	url := fmt.Sprintf("%s/book?token_id=%s", clobAPIBase, tokenID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return OrderBook{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OrderBook{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrderBook{}, fmt.Errorf("reading response body: %w", err)
	}

	var book OrderBook
	if err := json.Unmarshal(body, &book); err != nil {
		return OrderBook{}, fmt.Errorf("unmarshaling response: %w", err)
	}

	return book, nil
}

func (c *PolymarketClient) parseHexToUint(hexStr string) (uint64, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return strconv.ParseUint(hexStr, 16, 64)
}
