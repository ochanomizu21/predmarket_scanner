package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ochanomizu/predmarket-scanner/internal/fees"
	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/database"
	"github.com/ochanomizu/predmarket-scanner/pkg/output"
	"github.com/ochanomizu/predmarket-scanner/pkg/providers"
	"github.com/ochanomizu/predmarket-scanner/pkg/strategies"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
	"github.com/spf13/cobra"
)

func skipSlippageStr() string {
	if skipSlippage {
		return " (no slippage)"
	}
	return ""
}

var (
	fetchLimit      int
	fetchMaxMarkets int
	fetchMinOutcomes int
	fetchMaxOutcomes int
	fetchIncludeClosed bool
	minProfit       float64
	scanLimit       int
	scanMaxMarkets  int
	scanOffset      int
	scanStartRank   int
	scanEndRank     int
	executionSize   float64
	skipSlippage   bool
	strategyType    string
	exportFormat    string
	exportOutput    string
	recordInterval  int
	recordMaxMarkets int
	recordOffset     int
	recordStartRank  int
	recordEndRank    int
	historicalMode  bool
	historicalTime  string
	timeRange      string
	dbPath         string
	historyLimit   int
	historyMaxDays int
	historyInterval string
	historyOffset    int
	historyStartRank int
	historyEndRank   int
	historyWorkers  int
	historySkipExisting bool
	recordIncludeOrderBook bool
	recordOrderBookLevels  int
	fetchOffset    int
	fetchStartRank int
	fetchEndRank   int
	scanWorkers    int
	scanDebug      bool
	scanExportOpps string
	scanNoFees    bool
	scanDetailed  bool
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
	FetchMarketsCmd.Flags().IntVar(&fetchOffset, "offset", 0, "Skip first N markets (by liquidity)")
	FetchMarketsCmd.Flags().IntVar(&fetchStartRank, "start-rank", 0, "Start rank (1-based, inclusive)")
	FetchMarketsCmd.Flags().IntVar(&fetchEndRank, "end-rank", 0, "End rank (inclusive, 0 = unlimited)")
	ScanCmd.Flags().Float64VarP(&minProfit, "min-profit", "p", 0.001, "Minimum profit threshold")
	ScanCmd.Flags().IntVarP(&scanLimit, "limit", "l", 100, "Maximum number of opportunities to display")
	ScanCmd.Flags().IntVar(&scanMaxMarkets, "max-markets", 0, "Maximum number of markets to fetch (0 = all)")
	ScanCmd.Flags().IntVar(&scanOffset, "offset", 0, "Skip first N markets (by liquidity)")
	ScanCmd.Flags().IntVar(&scanStartRank, "start-rank", 0, "Start rank (1-based, inclusive)")
	ScanCmd.Flags().IntVar(&scanEndRank, "end-rank", 0, "End rank (inclusive, 0 = unlimited)")
	ScanCmd.Flags().StringVar(&strategyType, "strategy", "all", "Strategy to use (all, dutch_book, multi_outcome)")
	ScanCmd.Flags().Float64Var(&executionSize, "size", 1000, "Execution size in USDC")
	ScanCmd.Flags().BoolVar(&skipSlippage, "skip-slippage", false, "Skip order book fetch and slippage calculation (price only)")
	ScanCmd.Flags().BoolVar(&historicalMode, "historical", false, "Enable historical backtesting mode")
	ScanCmd.Flags().StringVar(&historicalTime, "time", "", "Target historical timestamp (RFC3339 format, e.g., 2026-02-28T00:00:00+01:00)")
	ScanCmd.Flags().StringVar(&timeRange, "time-range", "", "Time range for historical scanning (RFC3339 start,end)")
	ScanCmd.Flags().StringVar(&dbPath, "db", "data/history.db", "Path to SQLite database for historical data")
	ScanCmd.Flags().IntVar(&scanWorkers, "workers", 4, "Number of concurrent workers for historical scanning")
	ScanCmd.Flags().StringVar(&exportOutput, "output", "", "Export markets to JSON file (for debugging)")
	ScanCmd.Flags().BoolVar(&scanDebug, "debug", false, "Enable debug output (show all markets checked)")
	ScanCmd.Flags().StringVar(&scanExportOpps, "export-opps", "", "Export opportunities to JSON file")
	ScanCmd.Flags().BoolVar(&scanNoFees, "no-fees", false, "Ignore Polymarket fees (for markets without trading fees)")
	ScanCmd.Flags().BoolVar(&scanDetailed, "detailed", false, "Show detailed output with score breakdown")
	ExportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format (json or csv)")
	ExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "opportunities", "Output filename prefix")
	RecordCmd.Flags().IntVarP(&recordInterval, "interval", "i", 60, "Recording interval in seconds")
	RecordCmd.Flags().IntVar(&recordMaxMarkets, "max-markets", 500, "Maximum number of markets to record")
	RecordCmd.Flags().IntVar(&recordOffset, "offset", 0, "Skip first N markets (by liquidity)")
	RecordCmd.Flags().IntVar(&recordStartRank, "start-rank", 0, "Start rank (1-based, inclusive)")
	RecordCmd.Flags().IntVar(&recordEndRank, "end-rank", 0, "End rank (inclusive, 0 = unlimited)")
	RecordCmd.Flags().BoolVar(&recordIncludeOrderBook, "order-book", true, "Include full order book data (uses more API calls)")
	RecordCmd.Flags().IntVar(&recordOrderBookLevels, "order-book-levels", 10, "Number of order book levels to record per side")
	FetchHistoryCmd.Flags().IntVar(&historyLimit, "limit", 100, "Maximum number of markets to fetch history for")
	FetchHistoryCmd.Flags().IntVar(&historyMaxDays, "max-days", 30, "Maximum number of days of history to fetch")
	FetchHistoryCmd.Flags().StringVar(&historyInterval, "interval", "1d", "Price history interval (1m, 1h, 6h, 1d)")
	FetchHistoryCmd.Flags().IntVar(&historyOffset, "offset", 0, "Skip first N markets (by liquidity)")
	FetchHistoryCmd.Flags().IntVar(&historyStartRank, "start-rank", 0, "Start rank (1-based, inclusive)")
	FetchHistoryCmd.Flags().IntVar(&historyEndRank, "end-rank", 0, "End rank (inclusive, 0 = unlimited)")
	FetchHistoryCmd.Flags().IntVar(&historyWorkers, "workers", 10, "Number of concurrent workers for fetching")
	FetchHistoryCmd.Flags().BoolVar(&historySkipExisting, "skip-existing", false, "Skip markets that already have price history")
}

func runFetchMarkets(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets from Polymarket...")

	offset := fetchOffset
	limit := fetchMaxMarkets

	if fetchStartRank > 0 {
		offset = fetchStartRank - 1
		if fetchEndRank > 0 {
			limit = fetchEndRank - fetchStartRank + 1
		}
	}

	if offset > 0 || limit > 0 {
		fmt.Printf("Fetching markets: offset=%d, limit=%d\n", offset, limit)
	}

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
	markets, err := client.FetchMarketsFilterOffset(limit, offset, fetchMinOutcomes, fetchMaxOutcomes, fetchIncludeClosed)
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
	fees.ApplyFees = !scanNoFees

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

			startTime, err := time.Parse(time.RFC3339, parts[0])
			if err != nil {
				return fmt.Errorf("parsing start time: %w", err)
			}

			endTime, err := time.Parse(time.RFC3339, parts[1])
			if err != nil {
				return fmt.Errorf("parsing end time: %w", err)
			}

			fmt.Printf("Scanning historical data from %s to %s...\n", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

			timestamps, err := db.GetTimestampsInRange(startTime, endTime)
			if err != nil {
				return fmt.Errorf("getting timestamps: %w", err)
			}

			if len(timestamps) == 0 {
				fmt.Println("No snapshots found in the specified time range")
				return nil
			}

			fmt.Printf("Found %d timestamps to scan\n", len(timestamps))

			offset := scanOffset
			limit := scanMaxMarkets

			if scanStartRank > 0 {
				offset = scanStartRank - 1
				if scanEndRank > 0 {
					limit = scanEndRank - scanStartRank + 1
				}
			}

			if offset > 0 || limit > 0 {
				fmt.Printf("Scanning markets: offset=%d, limit=%d\n", offset, limit)
			}

			if scanWorkers <= 0 {
				scanWorkers = 4
			}
			fmt.Printf("Using %d concurrent workers\n", scanWorkers)

			type timestampResult struct {
				timestamp    time.Time
				opportunities []types.ArbitrageOpportunity
				err          error
			}

			taskChan := make(chan time.Time, len(timestamps))
			resultChan := make(chan timestampResult, len(timestamps))

			var wg sync.WaitGroup

			for w := 0; w < scanWorkers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					workerDB, err := database.Open(dbPath)
					if err != nil {
						return
					}
					defer workerDB.Close()

					for ts := range taskChan {
						provider = providers.NewHistoricalDataProvider(workerDB, ts, offset)
						markets, err := provider.FetchMarkets(scanMaxMarkets)
						if err != nil {
							resultChan <- timestampResult{timestamp: ts, err: err}
							continue
						}

						var opportunities []types.ArbitrageOpportunity
						switch strategyType {
						case "dutch_book":
							if skipSlippage {
								opportunities = strategies.FindOpportunitiesNoSlippage(markets, minProfit)
							} else {
								opportunities = strategies.FindOpportunitiesWithSizeAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
							}
						case "multi_outcome":
							if skipSlippage {
								opportunities = strategies.FindMultiOutcomeOpportunitiesNoSlippage(markets, minProfit)
							} else {
								opportunities = strategies.FindMultiOutcomeOpportunitiesAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
							}
						case "all":
							if skipSlippage {
								dutchBookOpps := strategies.FindOpportunitiesNoSlippage(markets, minProfit)
								multiOutcomeOpps := strategies.FindMultiOutcomeOpportunitiesNoSlippage(markets, minProfit)
								opportunities = append(dutchBookOpps, multiOutcomeOpps...)
							} else {
								dutchBookOpps := strategies.FindOpportunitiesWithSizeAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
								multiOutcomeOpps := strategies.FindMultiOutcomeOpportunitiesAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
								opportunities = append(dutchBookOpps, multiOutcomeOpps...)
							}
						default:
							resultChan <- timestampResult{timestamp: ts, err: fmt.Errorf("invalid strategy: %s", strategyType)}
							continue
						}

						resultChan <- timestampResult{timestamp: ts, opportunities: opportunities}
					}
				}(w)
			}

			for _, ts := range timestamps {
				taskChan <- ts
			}
			close(taskChan)

			go func() {
				wg.Wait()
				close(resultChan)
			}()

			var allOpportunities []types.ArbitrageOpportunity
			completed := 0
			for result := range resultChan {
				completed++
				if completed%10 == 0 || completed == len(timestamps) {
					fmt.Printf("\rProgress: %d/%d timestamps", completed, len(timestamps))
				}

				if result.err != nil {
					fmt.Printf("\nError at %s: %v\n", result.timestamp.Format("2006-01-02 15:04:05"), result.err)
					continue
				}

				for _, opp := range result.opportunities {
					if opp.NetProfit >= minProfit {
						allOpportunities = append(allOpportunities, opp)
					}
				}
			}

			fmt.Printf("\n\nTotal opportunities across %d timestamps: %d\n", len(timestamps), len(allOpportunities))

			if scanExportOpps != "" {
				exportPath := scanExportOpps
				if !strings.HasSuffix(exportPath, ".json") {
					exportPath += ".json"
				}
				if err := output.ExportJSON(allOpportunities, exportPath); err != nil {
					return fmt.Errorf("exporting opportunities: %w", err)
				}
				fmt.Printf("Exported %d opportunities to %s\n", len(allOpportunities), exportPath)
			}

			displayCount := scanLimit
			if len(allOpportunities) < displayCount {
				displayCount = len(allOpportunities)
			}

			fmt.Printf("\nFound %d opportunities (top %d displayed)\n", len(allOpportunities), displayCount)
			output.PrintOpportunitiesDetailed(allOpportunities[:displayCount], scanDetailed)
			return nil
		}

		if historicalTime != "" {
			fmt.Printf("Fetching markets from historical data at %s...\n", historicalTime)

			targetTime, err := time.Parse(time.RFC3339, historicalTime)
			if err != nil {
				return fmt.Errorf("parsing time: %w", err)
			}

			offset := scanOffset
			limit := scanMaxMarkets

			if scanStartRank > 0 {
				offset = scanStartRank - 1
				if scanEndRank > 0 {
					limit = scanEndRank - scanStartRank + 1
				}
			}

			if offset > 0 || limit > 0 {
				fmt.Printf("Scanning markets: offset=%d, limit=%d\n", offset, limit)
			}

			provider = providers.NewHistoricalDataProvider(db, targetTime, offset)
		}
	} else {
		fmt.Println("Fetching markets from Polymarket...")

		offset := scanOffset
		limit := scanMaxMarkets

		if scanStartRank > 0 {
			offset = scanStartRank - 1
			if scanEndRank > 0 {
				limit = scanEndRank - scanStartRank + 1
			}
		}

		if offset > 0 || limit > 0 {
			fmt.Printf("Scanning markets: offset=%d, limit=%d\n", offset, limit)
		}

		provider = providers.NewLiveDataProvider(offset, limit)
	}

	markets, err := provider.FetchMarkets(scanMaxMarkets)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Printf("Found %d markets\n", len(markets))

	if exportOutput != "" {
		exportPath := exportOutput
		if !strings.HasSuffix(exportPath, ".json") {
			exportPath += ".json"
		}
		if err := output.ExportMarketsJSON(markets, exportPath); err != nil {
			return fmt.Errorf("exporting markets: %w", err)
		}
		fmt.Printf("Exported %d markets to %s\n", len(markets), exportPath)
	}

	if scanDebug {
		fmt.Println("\n=== Debug: Markets being scanned ===")
		for i, m := range markets {
			sum := 0.0
			for _, o := range m.Outcomes {
				sum += o.Price
			}
			hasYes, hasNo := false, false
			for _, o := range m.Outcomes {
				if strings.EqualFold(o.Name, "yes") {
					hasYes = true
				}
				if strings.EqualFold(o.Name, "no") {
					hasNo = true
				}
			}
			question := m.Question
			if len(question) > 40 {
				question = question[:40]
			}
			fmt.Printf("[%d] %s | outcomes=%d sum=%.4f binary=%v yesno=%v\n",
				i, question, len(m.Outcomes), sum, len(m.Outcomes) == 2, hasYes && hasNo)
		}
	}

	fmt.Printf("Scanning for arbitrage opportunities (size: $%.2f, strategy: %s%s)...\n",
		executionSize, strategyType, skipSlippageStr())

	var opportunities []types.ArbitrageOpportunity

	if !skipSlippage {
		fmt.Println("Fetching order books for slippage calculation...")

		var allTokenIDs []string
		marketTokenIDs := make(map[int][]string)
		for i, m := range markets {
			if len(m.ClobTokenIDs) > 0 {
				marketTokenIDs[i] = m.ClobTokenIDs
				allTokenIDs = append(allTokenIDs, m.ClobTokenIDs...)
			}
		}

		orderBooks, err := provider.FetchOrderBooks(allTokenIDs)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch order books: %v\n", err)
			fmt.Println("Falling back to price-only calculation...")
		} else {
			fmt.Printf("Fetched %d order books\n", len(orderBooks))
		}

		orderBookGetter := func(tokenIDs []string) (map[string]clients.OrderBook, error) {
			result := make(map[string]clients.OrderBook)
			for _, tid := range tokenIDs {
				if ob, ok := orderBooks[tid]; ok {
					result[tid] = ob
				}
			}
			return result, nil
		}

		switch strategyType {
		case "dutch_book":
			opps, err := strategies.FindOpportunitiesWithOrderBooks(markets, orderBookGetter, executionSize, 100.0)
			if err != nil {
				return fmt.Errorf("finding opportunities: %w", err)
			}
			opportunities = opps
		case "multi_outcome":
			opps, err := strategies.FindMultiOutcomeOpportunitiesWithOrderBooks(markets, orderBookGetter, executionSize, 100.0)
			if err != nil {
				return fmt.Errorf("finding opportunities: %w", err)
			}
			opportunities = opps
		case "all":
			dutchBookOpps, _ := strategies.FindOpportunitiesWithOrderBooks(markets, orderBookGetter, executionSize, 100.0)
			multiOutcomeOpps, _ := strategies.FindMultiOutcomeOpportunitiesWithOrderBooks(markets, orderBookGetter, executionSize, 100.0)
			opportunities = append(dutchBookOpps, multiOutcomeOpps...)
		default:
			return fmt.Errorf("invalid strategy: %s", strategyType)
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
		opportunities = filtered
	} else {
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
			return fmt.Errorf("invalid strategy: %s", strategyType)
		}

		if len(opportunities) > scanLimit {
			opportunities = opportunities[:scanLimit]
		}
	}

	if scanExportOpps != "" {
		exportPath := scanExportOpps
		if !strings.HasSuffix(exportPath, ".json") {
			exportPath += ".json"
		}
		if err := output.ExportJSON(opportunities, exportPath); err != nil {
			return fmt.Errorf("exporting opportunities: %w", err)
		}
		fmt.Printf("Exported %d opportunities to %s\n", len(opportunities), exportPath)
	}

	fmt.Printf("\nFound %d opportunities\n", len(opportunities))
	output.PrintOpportunitiesDetailed(opportunities, scanDetailed)

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
		opportunities = strategies.FindOpportunitiesWithSizeAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
	case "multi_outcome":
		opportunities = strategies.FindMultiOutcomeOpportunitiesAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
	case "all":
		dutchBookOpps := strategies.FindOpportunitiesWithSizeAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
		multiOutcomeOpps := strategies.FindMultiOutcomeOpportunitiesAndMinProfit(markets, executionSize, 100.0, minProfit, scanLimit)
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

	offset := recordOffset
	limit := recordMaxMarkets

	if recordStartRank > 0 {
		offset = recordStartRank - 1
		if recordEndRank > 0 {
			limit = recordEndRank - recordStartRank + 1
		}
	}

	db, err := database.Open("data/history.db")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Recording to data/history.db\n")
	fmt.Printf("Recording interval: %d seconds\n", recordInterval)
	fmt.Printf("Recording markets: offset=%d, limit=%d\n", offset, limit)
	fmt.Printf("Order book recording: %v (max %d levels per side)\n", recordIncludeOrderBook, recordOrderBookLevels)
	fmt.Println("Press Ctrl+C to stop recording...")

	client := clients.NewPolymarketClient()

	for {
		fmt.Printf("\n[%s] Fetching markets...\n", time.Now().Format(time.RFC3339))

			markets, err := client.FetchMarketsFilterOffset(limit, offset, 0, 0, false)
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

				for i, outcome := range market.Outcomes {
					err := db.InsertOutcomeSnapshot(snapshotID, outcome.Name, outcome.Price, outcome.Price)
					if err != nil {
						fmt.Printf("Error inserting outcome snapshot for market %s: %v\n", market.ID, err)
						continue
					}

					if recordIncludeOrderBook {
						if i >= len(market.ClobTokenIDs) {
							continue
						}

						tokenID := market.ClobTokenIDs[i]
						if tokenID == "" {
							continue
						}

						book, err := client.FetchOrderBook(tokenID)
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
							err := db.InsertOrderBookLevels(snapshotID, outcome.Name, tokenID, "bid", bidLevels)
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
							err := db.InsertOrderBookLevels(snapshotID, outcome.Name, tokenID, "ask", askLevels)
							if err != nil {
								fmt.Printf("Error inserting ask levels for market %s: %v\n", market.ID, err)
							}
						}
					}
				}

			recorded++

			fmt.Printf("Recorded %d market snapshots\n", recorded)

			fmt.Printf("Waiting %d seconds before next recording...\n", recordInterval)
			time.Sleep(time.Duration(recordInterval) * time.Second)
		}
	}
}

func runFetchHistory(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets from Polymarket...")

	offset := historyOffset
	limit := historyLimit

	if historyStartRank > 0 {
		offset = historyStartRank - 1
		if historyEndRank > 0 {
			limit = historyEndRank - historyStartRank + 1
		}
	}

	if offset > 0 || limit > 0 {
		fmt.Printf("Fetching markets: offset=%d, limit=%d\n", offset, limit)
	}

	if historyWorkers <= 0 {
		historyWorkers = 10
	}

	fmt.Printf("Using %d concurrent workers\n", historyWorkers)
	if historySkipExisting {
		fmt.Printf("Skipping markets with existing price history\n")
	}

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarketsFilterOffset(limit, offset, 0, 0, false)
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

	type fetchTask struct {
		market     types.Market
		outcomeIndex int
		tokenID    string
	}

	var tasks []fetchTask
	for _, market := range markets {
		for i := range market.Outcomes {
			if i >= len(market.ClobTokenIDs) {
				continue
			}

			tokenID := market.ClobTokenIDs[i]
			if tokenID == "" {
				continue
			}

			tasks = append(tasks, fetchTask{
				market:     market,
				outcomeIndex: i,
				tokenID:    tokenID,
			})
		}
	}

	type fetchResult struct {
		success    bool
		points     int
		err        error
		taskIndex  int
	}

	taskChan := make(chan fetchTask, len(tasks))
	resultChan := make(chan fetchResult, len(tasks))

	var wg sync.WaitGroup

	for w := 0; w < historyWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				result := fetchResult{taskIndex: -1}

				if historySkipExisting {
					exists, err := db.HasPriceHistory(task.market.ID, task.market.Outcomes[task.outcomeIndex].Name)
					if err != nil {
						result.err = err
						resultChan <- result
						continue
					}
					if exists {
						result.success = true
						resultChan <- result
						continue
					}
				}

				history, err := client.GetPriceHistory(task.tokenID, historyInterval)
				if err != nil {
					errStr := err.Error()
					if !strings.Contains(errStr, "minimum 'fidelity'") {
						result.err = fmt.Errorf("market %s (token %s): %v", task.market.ID, task.tokenID, err)
					}
					resultChan <- result
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

				if len(validPoints) > 0 {
					err := db.InsertPriceHistory(task.market.ID, task.market.Outcomes[task.outcomeIndex].Name, validPoints)
					if err != nil {
						result.err = fmt.Errorf("inserting %s: %v", task.market.ID, err)
						resultChan <- result
						continue
					}
				}

				result.success = true
				result.points = len(validPoints)
				resultChan <- result
			}
		}()
	}

	for i, task := range tasks {
		taskChan <- fetchTask{
			market:     task.market,
			outcomeIndex: task.outcomeIndex,
			tokenID:    task.tokenID,
		}
		i++
	}
	close(taskChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	totalMarkets := 0
	totalHistoryPoints := 0
	failedMarkets := 0
	completedTasks := 0
	lastProgressMilestone := 0

	for result := range resultChan {
		completedTasks++

		progress := float64(completedTasks) / float64(len(tasks)) * 100
		currentMilestone := int(progress / 10) * 10

		if currentMilestone > lastProgressMilestone {
			fmt.Printf("\rProgress: %.1f%% (%d/%d tasks)", progress, completedTasks, len(tasks))
			lastProgressMilestone = currentMilestone
		}

		if !result.success {
			failedMarkets++
			continue
		}

		totalHistoryPoints += result.points
	}

	fmt.Println()

	totalMarkets = 0
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

		totalMarkets++
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total markets: %d\n", totalMarkets)
	fmt.Printf("  Failed tasks: %d\n", failedMarkets)
	fmt.Printf("  Total history points: %d\n", totalHistoryPoints)
	fmt.Printf("  Data saved to: %s\n", dbPath)

	return nil
}

