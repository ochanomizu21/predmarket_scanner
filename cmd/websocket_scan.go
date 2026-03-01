package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/output"
	"github.com/ochanomizu/predmarket-scanner/pkg/strategies"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
	"github.com/ochanomizu/predmarket-scanner/pkg/websocket"
)

func runWebSocketScan() error {
	fmt.Printf("Starting WebSocket-based scanning (mode: %s)...\n", scanMode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	client := clients.NewPolymarketClient()

	fmt.Println("Fetching markets from REST API...")
	offset := scanOffset
	limit := scanMaxMarkets

	if scanStartRank > 0 {
		offset = scanStartRank - 1
		if scanEndRank > 0 {
			limit = scanEndRank - scanStartRank + 1
		}
	}

	markets, err := client.FetchMarketsFilterOffset(limit, offset, 0, 0, scanClosed)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Printf("Fetched %d markets\n", len(markets))

	var allTokenIDs []string
	marketTokenMap := make(map[string]*types.Market)

	for i := range markets {
		markets[i].Platform = types.Polymarket
		if len(markets[i].ClobTokenIDs) > 0 {
			allTokenIDs = append(allTokenIDs, markets[i].ClobTokenIDs...)
			for _, tokenID := range markets[i].ClobTokenIDs {
				marketTokenMap[tokenID] = &markets[i]
			}
		}
	}

	allTokenIDs = uniqueTokenIDs(allTokenIDs)
	fmt.Printf("Found %d unique token IDs to subscribe to\n", len(allTokenIDs))

	orderBookMgr := types.NewOrderBookManager()

	wsClient := websocket.NewClient(orderBookMgr)

	fmt.Println("Connecting to WebSocket...")
	if err := wsClient.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to websocket: %w", err)
	}

	fmt.Println("Subscribing to markets...")
	if err := wsClient.Subscribe(allTokenIDs); err != nil {
		return fmt.Errorf("subscribing to markets: %w", err)
	}

	fmt.Println("Waiting for order book initialization...")
	time.Sleep(3 * time.Second)

	fmt.Printf("Scanning for arbitrage opportunities (size: $%.2f, strategy: %s, mode: %s)...\n",
		executionSize, strategyType, scanMode)

	var scanTriggerChan chan struct{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var opportunities []types.ArbitrageOpportunity

	if scanMode == "event-driven" {
		scanTriggerChan = make(chan struct{}, 100)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-scanTriggerChan:
					mu.Lock()
					latestOpps := scanFromOrderBooks(orderBookMgr, marketTokenMap)
					opportunities = latestOpps
					mu.Unlock()
				}
			}
		}()

		msgChan := wsClient.GetMessageChannel()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-msgChan:
					var data map[string]interface{}
					if err := json.Unmarshal(msg, &data); err != nil {
						continue
					}

					select {
					case scanTriggerChan <- struct{}{}:
					default:
					}
				}
			}
		}()

	} else {
		scanInterval := time.Duration(scanScanInterval) * time.Second
		if scanInterval < 1*time.Second {
			scanInterval = 1 * time.Second
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(scanInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					mu.Lock()
					latestOpps := scanFromOrderBooks(orderBookMgr, marketTokenMap)
					opportunities = latestOpps
					mu.Unlock()
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		lastDisplayCount := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				currentOpps := opportunities
				mu.Unlock()

				if len(currentOpps) != lastDisplayCount {
					fmt.Printf("\nFound %d opportunities (press Ctrl+C to stop)\n", len(currentOpps))
					if len(currentOpps) > 0 && len(currentOpps) <= scanLimit {
						output.PrintOpportunitiesDetailed(currentOpps, scanDetailed)
					}
					lastDisplayCount = len(currentOpps)
				}

				metrics := wsClient.GetMetrics()
				messages, bookEvents, priceChanges, connections, reconnections := metrics.GetStats()
				fmt.Printf("WebSocket: %d messages, %d book events, %d price changes, %d connections, %d reconnections\n",
					messages, bookEvents, priceChanges, connections, reconnections)
			}
		}
	}()

	<-ctx.Done()
	wg.Wait()

	mu.Lock()
	finalOpps := opportunities
	mu.Unlock()

	if len(finalOpps) > scanLimit {
		finalOpps = finalOpps[:scanLimit]
	}

	fmt.Printf("\nFinal scan: Found %d opportunities\n", len(finalOpps))
	output.PrintOpportunitiesDetailed(finalOpps, scanDetailed)

	return nil
}

func scanFromOrderBooks(orderBookMgr *types.OrderBookManager, marketTokenMap map[string]*types.Market) []types.ArbitrageOpportunity {
	allBooks := orderBookMgr.GetAllOrderBooks()

	marketMap := make(map[string]*types.Market)

	for tokenID, book := range allBooks {
		if market, ok := marketTokenMap[tokenID]; ok {
			if _, exists := marketMap[market.ID]; !exists {
				marketCopy := *market
				marketCopy.Outcomes = make([]types.Outcome, len(market.Outcomes))
				copy(marketCopy.Outcomes, market.Outcomes)

				for i := range marketCopy.Outcomes {
					if i < len(market.ClobTokenIDs) && market.ClobTokenIDs[i] == tokenID {
						bestAsk, _ := book.GetBestAsk()
						if bestAsk > 0 {
							marketCopy.Outcomes[i].Price = bestAsk
						}
					}
				}

				marketMap[market.ID] = &marketCopy
			} else {
				for i := range marketMap[market.ID].Outcomes {
					if i < len(market.ClobTokenIDs) && market.ClobTokenIDs[i] == tokenID {
						bestAsk, _ := book.GetBestAsk()
						if bestAsk > 0 {
							marketMap[market.ID].Outcomes[i].Price = bestAsk
						}
					}
				}
			}
		}
	}

	markets := make([]types.Market, 0, len(marketMap))
	for _, market := range marketMap {
		markets = append(markets, *market)
	}

	var opportunities []types.ArbitrageOpportunity

	switch strategyType {
	case "dutch_book":
		opportunities = strategies.FindOpportunitiesNoSlippage(markets, minProfit)
	case "multi_outcome":
		opportunities = strategies.FindMultiOutcomeOpportunitiesNoSlippage(markets, minProfit)
	case "all":
		dutchBookOpps := strategies.FindOpportunitiesNoSlippage(markets, minProfit)
		multiOutcomeOpps := strategies.FindMultiOutcomeOpportunitiesNoSlippage(markets, minProfit)
		opportunities = append(dutchBookOpps, multiOutcomeOpps...)
	default:
		return nil
	}

	if len(opportunities) > scanLimit {
		opportunities = opportunities[:scanLimit]
	}

	return opportunities
}
