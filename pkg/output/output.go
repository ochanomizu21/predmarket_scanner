package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

func PrintOpportunities(opportunities []types.ArbitrageOpportunity) {
	if len(opportunities) == 0 {
		fmt.Println("No arbitrage opportunities found.")
		return
	}

	fmt.Println("\n=== Arbitrage Opportunities ===\n")
	fmt.Printf("%-45s %-10s %-10s %-10s %-10s %-10s %-10s\n", "Market", "Gross %", "Net %", "Fee %", "Slip %", "Liq $", "Score")
	fmt.Println(strings.Repeat("-", 115))

	limit := len(opportunities)
	if limit > 20 {
		limit = 20
	}

	for i := 0; i < limit; i++ {
		opp := opportunities[i]
		question := truncate(opp.Market.Question, 42)
		fmt.Printf("%-45s %-10.3f %-10.3f %-10.3f %-10.3f %-10.0f %-10.3f\n",
			question,
			opp.GrossProfit*100.0,
			opp.NetProfit*100.0,
			opp.FeeCost*100.0,
			opp.SlippageImpact*100.0,
			opp.AvailableLiquidity,
			opp.Score,
		)
	}

	if len(opportunities) > 20 {
		fmt.Printf("\n... and %d more opportunities\n", len(opportunities)-20)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func ExportJSON(opportunities []types.ArbitrageOpportunity, path string) error {
	data, err := json.MarshalIndent(opportunities, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func ExportCSV(opportunities []types.ArbitrageOpportunity, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"market_id",
		"question",
		"strategy",
		"gross_profit",
		"net_profit",
		"fee_cost",
		"score",
		"platform",
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("writing headers: %w", err)
	}

	for _, opp := range opportunities {
		record := []string{
			opp.Market.ID,
			opp.Market.Question,
			string(opp.Strategy),
			fmt.Sprintf("%f", opp.GrossProfit),
			fmt.Sprintf("%f", opp.NetProfit),
			fmt.Sprintf("%f", opp.FeeCost),
			fmt.Sprintf("%f", opp.Score),
			string(opp.Market.Platform),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("writing record: %w", err)
		}
	}

	return nil
}
