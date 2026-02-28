package strategies

import (
	"fmt"

	"github.com/ochanomizu/predmarket-scanner/internal/fees"
	"github.com/ochanomizu/predmarket-scanner/internal/scoring"
	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

func FindOpportunities(markets []types.Market) []types.ArbitrageOpportunity {
	return FindOpportunitiesWithSize(markets, 1000, 0.5)
}

func FindOpportunitiesWithSize(markets []types.Market, executionSize float64, maxSlippagePercent float64) []types.ArbitrageOpportunity {
	var opportunities []types.ArbitrageOpportunity

	for _, market := range markets {
		if opp := checkDutchBookWithSlippage(market, executionSize, maxSlippagePercent); opp != nil {
			opportunities = append(opportunities, *opp)
		}
	}

	return opportunities
}

func isBinaryMarket(market types.Market) bool {
	if len(market.Outcomes) != 2 {
		return false
	}

	hasYes := false
	hasNo := false
	for _, outcome := range market.Outcomes {
		if outcome.Name == "YES" {
			hasYes = true
		}
		if outcome.Name == "NO" {
			hasNo = true
		}
	}

	return hasYes && hasNo
}

func checkDutchBookWithSlippage(market types.Market, executionSize, maxSlippagePercent float64) *types.ArbitrageOpportunity {
	if !isBinaryMarket(market) {
		return nil
	}

	var yesPrice, noPrice float64
	for _, outcome := range market.Outcomes {
		if outcome.Name == "YES" {
			yesPrice = outcome.Price
		}
		if outcome.Name == "NO" {
			noPrice = outcome.Price
		}
	}

	if yesPrice == 0 || noPrice == 0 {
		return nil
	}

	sum := yesPrice + noPrice
	if sum >= 1.0 {
		return nil
	}

	grossProfit := 1.0 - sum
		feeCost := fees.CalculatePolymarketFee(grossProfit, market)
	netProfit := grossProfit - feeCost

	if netProfit < 0.001 {
		return nil
	}

	return &types.ArbitrageOpportunity{
		Market:             market,
		Strategy:           types.DutchBook,
		GrossProfit:        grossProfit,
		NetProfit:          netProfit,
		FeeCost:            feeCost,
		Score:              scoring.CalculateScore(market, netProfit),
		ExecutionPlan:      buildExecutionPlan(yesPrice, noPrice),
		SlippageImpact:     0,
		YesSlippage:        0,
		NoSlippage:         0,
		AvailableLiquidity: market.Liquidity,
	}
}

func FindOpportunitiesNoSlippage(markets []types.Market, minProfit float64) []types.ArbitrageOpportunity {
	var opportunities []types.ArbitrageOpportunity

	for _, market := range markets {
		if opp := checkDutchBookNoSlippage(market, minProfit); opp != nil {
			opportunities = append(opportunities, *opp)
		}
	}

	return opportunities
}

func checkDutchBookNoSlippage(market types.Market, minProfit float64) *types.ArbitrageOpportunity {
	if !isBinaryMarket(market) {
		return nil
	}

	var yesPrice, noPrice float64
	for _, outcome := range market.Outcomes {
		if outcome.Name == "YES" {
			yesPrice = outcome.Price
		}
		if outcome.Name == "NO" {
			noPrice = outcome.Price
		}
	}

	if yesPrice == 0 || noPrice == 0 {
		return nil
	}

	sum := yesPrice + noPrice
	if sum >= 1.0 {
		return nil
	}

	grossProfit := 1.0 - sum
	feeCost := fees.CalculatePolymarketFee(grossProfit, market)
	netProfit := grossProfit - feeCost

	if netProfit < minProfit {
		return nil
	}

	return &types.ArbitrageOpportunity{
		Market:             market,
		Strategy:           types.DutchBook,
		GrossProfit:        grossProfit,
		NetProfit:          netProfit,
		FeeCost:            feeCost,
		Score:              scoring.CalculateScore(market, netProfit),
		ExecutionPlan:      buildExecutionPlan(yesPrice, noPrice),
		SlippageImpact:     0,
		YesSlippage:        0,
		NoSlippage:         0,
		AvailableLiquidity: market.Liquidity,
	}
}

func CheckDutchBookWithOrderBooks(market types.Market, orderBooks map[string]clients.OrderBook, executionSize float64, maxSlippagePercent float64) *types.ArbitrageOpportunity {
	if !isBinaryMarket(market) {
		return nil
	}

	var yesTokenID, noTokenID string
	var yesPrice, noPrice float64

	for _, outcome := range market.Outcomes {
		if outcome.Name == "YES" {
			yesPrice = outcome.Price
			yesTokenID = outcome.Name
		}
		if outcome.Name == "NO" {
			noPrice = outcome.Price
			noTokenID = outcome.Name
		}
	}

	if yesPrice == 0 || noPrice == 0 {
		return nil
	}

	yesBook, yesOk := orderBooks[yesTokenID]
	noBook, noOk := orderBooks[noTokenID]

	if !yesOk || !noOk {
		return nil
	}

	yesSlippage, _ := clients.CalculateSlippage(yesBook, clients.Buy, executionSize)
	noSlippage, _ := clients.CalculateSlippage(noBook, clients.Buy, executionSize)

	yesExecPrice := yesPrice
	noExecPrice := noPrice

	if yesSlippage.TotalFilled >= executionSize {
		yesExecPrice = yesSlippage.AveragePrice
	}
	if noSlippage.TotalFilled >= executionSize {
		noExecPrice = noSlippage.AveragePrice
	}

	sum := yesExecPrice + noExecPrice
	if sum >= 1.0 {
		return nil
	}

	grossProfit := 1.0 - sum
	feeCost := fees.CalculatePolymarketFee(grossProfit, market)
	netProfit := grossProfit - feeCost

	if netProfit < 0.001 {
		return nil
	}

	slippageImpact := (yesExecPrice - yesPrice) + (noExecPrice - noPrice)

	return &types.ArbitrageOpportunity{
		Market:             market,
		Strategy:           types.DutchBook,
		GrossProfit:        grossProfit,
		NetProfit:          netProfit,
		FeeCost:            feeCost,
		Score:              scoring.CalculateScore(market, netProfit),
		ExecutionPlan:      buildExecutionPlanWithSlippage(yesExecPrice, noExecPrice, executionSize, yesSlippage, noSlippage),
		SlippageImpact:     slippageImpact,
		YesSlippage:        yesSlippage.Slippage,
		NoSlippage:         noSlippage.Slippage,
		AvailableLiquidity: market.Liquidity,
	}
}

func buildExecutionPlan(yesPrice, noPrice float64) types.ExecutionPlan {
	return types.ExecutionPlan{
		Legs: []types.TradeLeg{
			{
				Outcome: "YES",
				Side:    types.Bid,
				Price:   yesPrice,
				Size:    1.0,
			},
			{
				Outcome: "NO",
				Side:    types.Bid,
				Price:   noPrice,
				Size:    1.0,
			},
		},
		TotalCost:        yesPrice + noPrice,
		GuaranteedPayout: 1.0,
	}
}

func buildExecutionPlanWithSlippage(yesPrice, noPrice, size float64, yesSlippage, noSlippage clients.SlippageResult) types.ExecutionPlan {
	yesFilled := size
	if yesSlippage.TotalFilled < size {
		yesFilled = yesSlippage.TotalFilled
	}

	noFilled := size
	if noSlippage.TotalFilled < size {
		noFilled = noSlippage.TotalFilled
	}

	return types.ExecutionPlan{
		Legs: []types.TradeLeg{
			{
				Outcome: "YES",
				Side:    types.Bid,
				Price:   yesPrice,
				Size:    yesFilled,
			},
			{
				Outcome: "NO",
				Side:    types.Bid,
				Price:   noPrice,
				Size:    noFilled,
			},
		},
		TotalCost:        yesPrice*yesFilled + noPrice*noFilled,
		GuaranteedPayout: yesFilled + noFilled,
	}
}

func FindOpportunitiesWithOrderBooks(markets []types.Market, orderBookGetter func([]string) (map[string]clients.OrderBook, error), executionSize, maxSlippagePercent float64) ([]types.ArbitrageOpportunity, error) {
	var opportunities []types.ArbitrageOpportunity

	for _, market := range markets {
		if !isBinaryMarket(market) {
			continue
		}

		var tokenIDs []string
		for _, outcome := range market.Outcomes {
			tokenIDs = append(tokenIDs, outcome.Name)
		}

		orderBooks, err := orderBookGetter(tokenIDs)
		if err != nil {
			fmt.Printf("Error fetching order books for market %s: %v\n", market.ID, err)
			continue
		}

		if opp := CheckDutchBookWithOrderBooks(market, orderBooks, executionSize, maxSlippagePercent); opp != nil {
			opportunities = append(opportunities, *opp)
		}
	}

	return opportunities, nil
}
