package main

import (
	"fmt"
	"strings"

	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/output"
	"github.com/ochanomizu/predmarket-scanner/pkg/strategies"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
	"github.com/spf13/cobra"
)

var (
	fetchLimit      int
	fetchMaxMarkets int
	minProfit       float64
	scanLimit       int
	scanMaxMarkets  int
	executionSize   float64
	maxSlippage     float64
	exportFormat    string
	exportOutput    string
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
	ScanCmd.Flags().Float64VarP(&minProfit, "min-profit", "p", 0.001, "Minimum profit threshold")
	ScanCmd.Flags().IntVarP(&scanLimit, "limit", "l", 100, "Maximum number of opportunities to display")
	ScanCmd.Flags().IntVar(&scanMaxMarkets, "max-markets", 0, "Maximum number of markets to fetch (0 = all)")
	ScanCmd.Flags().Float64VarP(&executionSize, "size", "s", 1000, "Execution size in USDC")
	ScanCmd.Flags().Float64Var(&maxSlippage, "max-slippage", 5.0, "Maximum slippage in percent")
	ExportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format (json or csv)")
	ExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "opportunities", "Output filename prefix")
}

func runFetchMarkets(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets from Polymarket...")

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarkets(fetchMaxMarkets)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Printf("\nFetched %d total markets\n\n", len(markets))

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
	fmt.Println("Fetching markets from Polymarket...")

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarkets(scanMaxMarkets)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Printf("Found %d markets\n", len(markets))

	fmt.Println("Scanning for arbitrage opportunities...")
	opportunities := strategies.FindOpportunities(markets)

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

func runExport(cmd *cobra.Command, args []string) error {
	fmt.Println("Fetching markets...")

	client := clients.NewPolymarketClient()
	markets, err := client.FetchMarkets(0)
	if err != nil {
		return fmt.Errorf("fetching markets: %w", err)
	}

	fmt.Println("Scanning for opportunities...")
	opportunities := strategies.FindOpportunities(markets)

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
