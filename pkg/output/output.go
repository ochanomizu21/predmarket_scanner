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
	PrintOpportunitiesDetailed(opportunities, false)
}

func PrintOpportunitiesDetailed(opportunities []types.ArbitrageOpportunity, showScoreBreakdown bool) {
	if len(opportunities) == 0 {
		fmt.Println("No arbitrage opportunities found.")
		return
	}

	if showScoreBreakdown {
		fmt.Println("\n=== Arbitrage Opportunities (with Score Breakdown) ===\n")
		fmt.Printf("%-40s %-8s %-8s %-8s %-7s %-8s | %-6s %-6s %-6s %-6s %-6s\n", 
			"Market", "Gross %", "Net %", "Fee %", "Slip %", "Liq $", "P_Sc", "L_Sc", "V_Sc", "E_Rk", "T_Dc")
		fmt.Println(strings.Repeat("-", 110))

		limit := len(opportunities)
		if limit > 20 {
			limit = 20
		}

		for i := 0; i < limit; i++ {
			opp := opportunities[i]
			question := truncate(opp.Market.Question, 37)
			fmt.Printf("%-40s %-8.3f %-8.3f %-8.3f %-7.3f %-8.0f | %-6.3f %-6.3f %-6.3f %-6.3f %-6.3f\n",
				question,
				opp.GrossProfit*100.0,
				opp.NetProfit*100.0,
				opp.FeeCost*100.0,
				opp.SlippageImpact*100.0,
				opp.AvailableLiquidity,
				opp.ScoreFactors.ProfitScore,
				opp.ScoreFactors.LiquidityScore,
				opp.ScoreFactors.VolumeScore,
				opp.ScoreFactors.ExecutionRisk,
				opp.ScoreFactors.TimeDecay,
			)
		}
	} else {
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

type MarketExport struct {
	ID          string           `json:"id"`
	Question    string           `json:"question"`
	Liquidity   float64          `json:"liquidity"`
	Volume      float64          `json:"volume"`
	Outcomes   []OutcomeExport  `json:"outcomes"`
	Sum        float64          `json:"sum"`
	IsBinary    bool             `json:"is_binary"`
	HasYesNo   bool             `json:"has_yes_no"`
}

type OutcomeExport struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func ExportMarketsJSON(markets []types.Market, path string) error {
	exports := make([]MarketExport, len(markets))
	
	for i, m := range markets {
		exp := MarketExport{
			ID:        m.ID,
			Question:  m.Question,
			Liquidity: m.Liquidity,
			Volume:    m.Volume,
			Outcomes:  make([]OutcomeExport, len(m.Outcomes)),
		}
		
		sum := 0.0
		for j, o := range m.Outcomes {
			exp.Outcomes[j] = OutcomeExport{
				Name:  o.Name,
				Price: o.Price,
			}
			sum += o.Price
		}
		exp.Sum = sum
		exp.IsBinary = len(m.Outcomes) == 2
		
		hasYes, hasNo := false, false
		for _, o := range m.Outcomes {
			if o.Name == "YES" {
				hasYes = true
			}
			if o.Name == "NO" {
				hasNo = true
			}
		}
		exp.HasYesNo = hasYes && hasNo
		
		exports[i] = exp
	}
	
	data, err := json.MarshalIndent(exports, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	
	return nil
}
