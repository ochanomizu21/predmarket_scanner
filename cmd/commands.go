package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/database"
	"github.com/ochanomizu/predmarket-scanner/pkg/output"
	"github.com/ochanomizu/predmarket-scanner/pkg/providers"
	"github.com/ochanomizu/predmarket-scanner/pkg/strategies"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
	"github.com/spf13/cobra"
)

var (
	fetchLimit      int
	fetchMaxMarkets int
	fetchMinOutcomes int
	fetchMaxOutcomes int
	fetchIncludeClosed bool
	minProfit       float64
	scanLimit       int
	scanMaxMarkets  int
	executionSize   float64
	maxSlippage     float64
	strategyType    string
	exportFormat    string
	exportOutput    string
	recordInterval  int
	recordMaxMarkets int
	historicalMode  bool
	historicalTime  string
	timeRange      string
	dbPath         string
	historyLimit   int
	historyMaxDays int
	historyInterval string
	recordIncludeOrderBook bool
	recordOrderBookLevels  int
)

var FetchMarketsCmd = &cobra.Command{
	Use:   "fetch-markets",
	Short: "Fetch markets from Polymarket",
	Long:  "Fetch and display markets from Polymarket",
	RunE:  runFetchMarkets,
}

func init() {
	FetchMarketsCmd.Flags().IntVarP(&fetchLimit, "limit", "l", 10, "Number of markets to display")
	FetchMarketsCmd.Flags().IntVarP(&fetchMaxMarkets, "max-markets", "m", 0, "Maximum number of markets to fetch (0 = all)")
	FetchMarketsCmd.Flags().IntVar(&fetchMinOutcomes, "min-outcomes", 0, "Minimum number of outcomes")
	FetchMarketsCmd.Flags().IntVar(&fetchMaxOutcomes, "max-outcomes", 0, "Maximum number of outcomes")
	FetchMarketsCmd.Flags().BoolVar(&fetchIncludeClosed, "closed", false, "Include closed/resolved markets")
	ScanCmd.Flags().Float64VarP(&minProfit, "min-profit", "p", 0.001, "Minimum profit threshold")
	ScanCmd.Flags().IntVarP(&scanLimit, "limit", "l", 100, "Maximum number of opportunities to display")
	ScanCmd.Flags().IntVar(&scanMaxMarkets, "max-markets", 0, "Maximum number of markets to fetch (0 = all)")
	ScanCmd.Flags().Float64VarP(&executionSize, "size", "s", 1000, "Execution size in USDC")
	ScanCmd.Flags().Float64Var(&maxSlippage, "max-slippage", 5.0, "Maximum slippage in percent")
	ScanCmd.Flags().StringVar(&strategyType, "strategy", "all", "Strategy to use (all, dutch_book, multi_outcome)")
	ScanCmd.Flags().BoolVar(&historicalMode, "historical", false, "Enable historical backtesting mode")
	ScanCmd.Flags().StringVar(&historicalTime, "time", "", "Target historical timestamp (YYYY-MM-DD HH:MM:SS)")
	ScanCmd.Flags().StringVar(&timeRange, "time-range", "", "Time range for historical scanning (start,end)")
	ScanCmd.Flags().StringVar(&dbPath, "db", "data/history.db", "Path to SQLite database for historical data")
	ExportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format (json or csv)")
	ExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "opportunities", "Output filename prefix")
	RecordCmd.Flags().IntVarP(&recordInterval, "interval", "i", 60, "Recording interval in seconds")
	RecordCmd.Flags().IntVar(&recordMaxMarkets, "max-markets", 500, "Maximum number of markets to record")
	RecordCmd.Flags().BoolVar(&recordIncludeOrderBook, "order-book", true, "Include full order book data (uses more API calls)")
	RecordCmd.Flags().IntVar(&recordOrderBookLevels, "order-book-levels", 10, "Number of order book levels to record per side")
	FetchHistoryCmd.Flags().IntVar(&historyLimit, "limit", 100, "Maximum number of markets to fetch history for")
	FetchHistoryCmd.Flags().IntVar(&historyMaxDays, "max-days", 30, "Maximum number of days of history to fetch")
	FetchHistoryCmd.Flags().StringVar(&historyInterval, "interval", "1d", "Price history interval (1m, 1h, 6h, 1d)")
}

func runFetchMarkets(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets from Polymarket...")

	if fetchMinOutcomes > 0 || fetchMaxOutcomes > 0 {
		filterStr := ""
		if fetchMinOutcomes > 0 && fetchMaxOutcomes > 0 {
			filterStr = fmt.Sprintf(" (outcomes: %d-%d)", fetchMinOutcomes, fetchMaxOutcomes)
		} else if fetchMinOutcomes > 0 {
			filterStr = fmt.Sprintf(" (outcomes: %d+)", fetchMinOutcomes)
		} else if fetchMaxOutcomes > 0 {
			filterStr = fmt.Sprintf(" (outcomes: %d-)", fetchMaxOutcomes)
		}
		fmt.Printf("Applying filter%s\n", filterStr)
	}

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarketsFilter(fetchMaxMarkets, fetchMinOutcomes, fetchMaxOutcomes, fetchIncludeClosed)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	marketType := "active"
	if fetchIncludeClosed {
		marketType = "active + closed"
	}

	fmt.Printf("\nFetched %d total %s markets\n\n", len(markets), marketType)

	displayLimit := fetchLimit
	if displayLimit > len(markets) {
		displayLimit = len(markets)
	}

	fmt.Printf("First %d markets:\n", displayLimit)
	fmt.Printf("%-40s %-15s %-15s %-10s\n", "Question", "Liquidity", "Volume", "Outcomes")
	fmt.Println(strings.Repeat("-", 85))

	for i := 0; i < displayLimit; i++ {
		market := markets[i]
		question := truncate(market.Question, 37)

		outcomesStr := ""
		if len(market.Outcomes) > 0 {
			outcomeNames := make([]string, 0, 2)
			for j := 0; j < len(market.Outcomes) && j < 2; j++ {
				outcomeNames = append(outcomeNames, market.Outcomes[j].Name)
			}
			outcomesStr = strings.Join(outcomeNames, ", ")
		}

		fmt.Printf("%-40s $%-14.2f $%-14.2f %s (%d)\n",
			question,
			market.Liquidity,
			market.Volume,
			outcomesStr,
			len(market.Outcomes),
		)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

var ScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for arbitrage opportunities",
	Long:  "Scan markets for Dutch book arbitrage opportunities",
	RunE:  runScan,
}

func runScan(cmd *cobra.Command, args []string) error {
	var provider providers.DataProvider

	if historicalMode {
		if historicalTime == "" && timeRange == "" {
			return fmt.Errorf("--time or --time-range is required in historical mode")
		}

		db, err := database.Open(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close()

		if timeRange != "" {
			parts := strings.Split(timeRange, ",")
			if len(parts) != 2 {
				return fmt.Errorf("invalid time-range format, expected 'start,end'")
			}

			fmt.Printf("Scanning historical data from %s to %s...\n", parts[0], parts[1])
		} else {
			fmt.Printf("Fetching markets from historical data at %s...\n", historicalTime)

			targetTime, err := time.Parse("2006-01-02 15:04:05", historicalTime)
			if err != nil {
				return fmt.Errorf("parsing time: %w", err)
			}
			provider = providers.NewHistoricalDataProvider(db, targetTime)
		}
	} else {
		fmt.Println("Fetching markets from Polymarket...")
		provider = providers.NewLiveDataProvider()
	}

	markets, err := provider.FetchMarkets(scanMaxMarkets)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Printf("Found %d markets\n", len(markets))

	fmt.Printf("Scanning for arbitrage opportunities (size: $%.2f, max-slippage: %.2f%%, strategy: %s)...\n",
		executionSize, maxSlippage, strategyType)

	var opportunities []types.ArbitrageOpportunity

	switch strategyType {
	case "dutch_book":
		opportunities = strategies.FindOpportunitiesWithSize(markets, executionSize, maxSlippage)
	case "multi_outcome":
		opportunities = strategies.FindMultiOutcomeOpportunities(markets, executionSize, maxSlippage)
	case "all":
		dutchBookOpps := strategies.FindOpportunitiesWithSize(markets, executionSize, maxSlippage)
		multiOutcomeOpps := strategies.FindMultiOutcomeOpportunities(markets, executionSize, maxSlippage)
		opportunities = append(dutchBookOpps, multiOutcomeOpps...)
	default:
		return fmt.Errorf("invalid strategy: %s (must be 'all', 'dutch_book', or 'multi_outcome')", strategyType)
	}

	var filtered []types.ArbitrageOpportunity
	for _, opp := range opportunities {
		if opp.NetProfit >= minProfit {
			filtered = append(filtered, opp)
		}
		if len(filtered) >= scanLimit {
			break
		}
	}

	fmt.Printf("\nFound %d opportunities\n", len(filtered))
	output.PrintOpportunities(filtered)

	return nil
}

var ExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export opportunities to file",
	Long:  "Export arbitrage opportunities to JSON or CSV",
	RunE:  runExport,
}

var RecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record historical market data",
	Long:  "Run as a daemon to record market snapshots to SQLite database",
	RunE:  runRecord,
}

var FetchHistoryCmd = &cobra.Command{
	Use:   "fetch-history",
	Short: "Fetch historical price data",
	Long:  "Fetch historical price data from Polymarket for markets and store in database",
	RunE:  runFetchHistory,
}

func runExport(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets...")

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarketsFilter(0, 0, 0, false)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Println("Scanning for opportunities...")
	var opportunities []types.ArbitrageOpportunity

	switch strategyType {
	case "dutch_book":
		opportunities = strategies.FindOpportunitiesWithSize(markets, executionSize, maxSlippage)
	case "multi_outcome":
		opportunities = strategies.FindMultiOutcomeOpportunities(markets, executionSize, maxSlippage)
	case "all":
		dutchBookOpps := strategies.FindOpportunitiesWithSize(markets, executionSize, maxSlippage)
		multiOutcomeOpps := strategies.FindMultiOutcomeOpportunities(markets, executionSize, maxSlippage)
		opportunities = append(dutchBookOpps, multiOutcomeOpps...)
	default:
		return fmt.Errorf("invalid strategy: %s (must be 'all', 'dutch_book', or 'multi_outcome')", strategyType)
	}

	var filename string
	switch exportFormat {
	case "json":
		filename = exportOutput + ".json"
	case "csv":
		filename = exportOutput + ".csv"
	default:
		return fmt.Errorf("invalid format: %s (must be 'json' or 'csv')", exportFormat)
	}

	switch exportFormat {
	case "json":
		if err := output.ExportJSON(opportunities, filename); err != nil {
			return fmt.Errorf("exporting JSON: %w", err)
		}
	case "csv":
		if err := output.ExportCSV(opportunities, filename); err != nil {
			return fmt.Errorf("exporting CSV: %w", err)
		}
	}

	fmt.Printf("Exported %d opportunities to %s\n", len(opportunities), filename)
	return nil
}

func runRecord(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting historical data recording daemon...")

	db, err := database.Open("data/history.db")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Recording to data/history.db\n")
	fmt.Printf("Recording interval: %d seconds\n", recordInterval)
	fmt.Printf("Max markets to record: %d\n", recordMaxMarkets)
	fmt.Printf("Order book recording: %v (max %d levels per side)\n", recordIncludeOrderBook, recordOrderBookLevels)
	fmt.Println("Press Ctrl+C to stop recording...")

	client := clients.NewPolymarketClient()
	ticker := time.NewTicker(time.Duration(recordInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n[%s] Fetching markets...\n", time.Now().Format(time.RFC3339))

			markets, err := client.FetchMarketsFilter(recordMaxMarkets, 0, 0, false)
			if err != nil {
				fmt.Printf("Error fetching markets: %v\n", err)
				continue
			}

			fmt.Printf("Fetched %d markets\n", len(markets))
			fmt.Printf("Recording snapshots...\n")

			var recorded int
			timestamp := time.Now()

			for _, market := range markets {
				err := db.InsertOrUpdateMarket(
					market.ID,
					market.Question,
					market.EndTime,
					market.Liquidity,
					market.Volume,
					len(market.Outcomes),
				)
				if err != nil {
					fmt.Printf("Error inserting market %s: %v\n", market.ID, err)
					continue
				}

				snapshotID, err := db.InsertSnapshot(market.ID, timestamp)
				if err != nil {
					fmt.Printf("Error inserting snapshot for market %s: %v\n", market.ID, err)
					continue
				}

				for _, outcome := range market.Outcomes {
					err := db.InsertOutcomeSnapshot(snapshotID, outcome.Name, outcome.Price, outcome.Price)
					if err != nil {
						fmt.Printf("Error inserting outcome snapshot for market %s: %v\n", market.ID, err)
						continue
					}

					if recordIncludeOrderBook {
						book, err := client.FetchOrderBook(outcome.Name)
						if err != nil {
							continue
						}

						bidLevels := make([]providers.OrderBookLevel, 0, recordOrderBookLevels)
						for i, bid := range book.Bids {
							if i >= recordOrderBookLevels {
								break
							}
							price, _ := strconv.ParseFloat(bid.Price, 64)
							size, _ := strconv.ParseFloat(bid.Size, 64)
							bidLevels = append(bidLevels, providers.OrderBookLevel{Price: price, Size: size})
						}

						if len(bidLevels) > 0 {
							err := db.InsertOrderBookLevels(snapshotID, outcome.Name, "bid", bidLevels)
							if err != nil {
								fmt.Printf("Error inserting bid levels for market %s: %v\n", market.ID, err)
							}
						}

						askLevels := make([]providers.OrderBookLevel, 0, recordOrderBookLevels)
						for i, ask := range book.Asks {
							if i >= recordOrderBookLevels {
								break
							}
							price, _ := strconv.ParseFloat(ask.Price, 64)
							size, _ := strconv.ParseFloat(ask.Size, 64)
							askLevels = append(askLevels, providers.OrderBookLevel{Price: price, Size: size})
						}

						if len(askLevels) > 0 {
							err := db.InsertOrderBookLevels(snapshotID, outcome.Name, "ask", askLevels)
							if err != nil {
								fmt.Printf("Error inserting ask levels for market %s: %v\n", market.ID, err)
							}
						}
					}
				}

			recorded++
			}

			fmt.Printf("Recorded %d market snapshots\n", recorded)
		}
	}
}

func runFetchHistory(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets from Polymarket...")

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarketsFilter(historyLimit, 0, 0, false)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Printf("Found %d markets\n", len(markets))
	fmt.Printf("Fetching historical price data for up to %d days...\n", historyMaxDays)

	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	cutoffDate := time.Now().AddDate(0, 0, -historyMaxDays)

	totalMarkets := 0
	totalHistoryPoints := 0
	failedMarkets := 0

	for _, market := range markets {
		totalMarkets++

		err := db.InsertOrUpdateMarket(
			market.ID,
			market.Question,
			market.EndTime,
			market.Liquidity,
			market.Volume,
			len(market.Outcomes),
		)
		if err != nil {
			failedMarkets++
			fmt.Printf("Error inserting market %s: %v\n", market.ID, err)
			continue
		}

		for i, outcome := range market.Outcomes {
			questionShort := market.Question
			if len(questionShort) > 30 {
				questionShort = questionShort[:30]
			}
			fmt.Printf("Fetching price history for %s - %s...\n", questionShort, outcome.Name)

			if i >= len(market.ClobTokenIDs) {
				fmt.Printf("No token ID for %s - %s\n", questionShort, outcome.Name)
				continue
			}

			tokenID := market.ClobTokenIDs[i]
			if tokenID == "" {
				fmt.Printf("Empty token ID for %s - %s\n", questionShort, outcome.Name)
				continue
			}

			history, err := client.GetPriceHistory(tokenID, historyInterval)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "minimum 'fidelity'") {
				} else {
					fmt.Printf("Error fetching price history for market %s (token %s): %v\n", market.ID, tokenID, err)
				}
				continue
			}

			var validPoints []providers.PriceHistoryPoint
			for _, point := range history {
				timestamp, err := time.Parse(time.RFC3339, point.Timestamp)
				if err != nil {
					continue
				}

				if timestamp.After(cutoffDate) {
					validPoints = append(validPoints, providers.PriceHistoryPoint{
						Timestamp:   timestamp,
						Price:      point.Price,
						TokenID:    point.TokenID,
						OrderCount: point.OrderCount,
					})
				}
			}

			if len(validPoints) == 0 {
				fmt.Printf("No valid price points for %s - %s (before cutoff)\n", market.Question[:30], outcome.Name)
				continue
			}

			err = db.InsertPriceHistory(market.ID, outcome.Name, validPoints)
			if err != nil {
				fmt.Printf("Error inserting price history for %s - %s: %v\n", market.ID, outcome.Name, err)
				continue
			}

			totalHistoryPoints += len(validPoints)
		}

		if totalMarkets%10 == 0 {
			fmt.Printf("Processed %d/%d markets...\n", totalMarkets, historyLimit)
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total markets: %d\n", totalMarkets)
	fmt.Printf("  Failed markets: %d\n", failedMarkets)
	fmt.Printf("  Total history points: %d\n", totalHistoryPoints)
	fmt.Printf("  Data saved to: %s\n", dbPath)
	
	return nil
}

