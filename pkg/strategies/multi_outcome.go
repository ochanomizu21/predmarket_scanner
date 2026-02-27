package strategies

import (
	"fmt"

	"github.com/ochanomizu/predmarket-scanner/internal/fees"
	"github.com/ochanomizu/predmarket-scanner/internal/scoring"
	"github.com/ochanomizu/predmarket-scanner/pkg/clients"
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

func FindMultiOutcomeOpportunities(markets []types.Market, executionSize, maxSlippagePercent float64) []types.ArbitrageOpportunity {
	var opportunities []types.ArbitrageOpportunity

	for _, market := range markets {
		if opp := checkMultiOutcomeWithSlippage(market, executionSize, maxSlippagePercent); opp != nil {
			opportunities = append(opportunities, *opp)
		}
	}

	return opportunities
}

func isMultiOutcomeMarket(market types.Market) bool {
	if len(market.Outcomes) < 3 {
		return false
	}

	for _, outcome := range market.Outcomes {
		if outcome.Price <= 0 {
			return false
		}
	}

	return true
}

func checkMultiOutcomeWithSlippage(market types.Market, executionSize, maxSlippagePercent float64) *types.ArbitrageOpportunity {
	if !isMultiOutcomeMarket(market) {
		return nil
	}

	sum := 0.0
	for _, outcome := range market.Outcomes {
		sum += outcome.Price
	}

	if sum >= 1.0 {
		return nil
	}

	grossProfit := 1.0 - sum
	feeCost := fees.CalculatePolymarketFee(grossProfit, market)
	netProfit := grossProfit - feeCost

	if netProfit < 0.001 {
		return nil
	}

	legs := make([]types.TradeLeg, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		legs[i] = types.TradeLeg{
			Outcome: outcome.Name,
			Side:    types.Bid,
			Price:   outcome.Price,
			Size:    1.0,
		}
	}

	return &types.ArbitrageOpportunity{
		Market:             market,
		Strategy:           types.MultiOutcome,
		GrossProfit:        grossProfit,
		NetProfit:          netProfit,
		FeeCost:            feeCost,
		Score:              scoring.CalculateScore(market, netProfit),
		ExecutionPlan: types.ExecutionPlan{
			Legs:             legs,
			TotalCost:        sum,
			GuaranteedPayout: 1.0,
		},
		SlippageImpact:     0,
		AvailableLiquidity: market.Liquidity,
	}
}

func CheckMultiOutcomeWithOrderBooks(market types.Market, orderBooks map[string]clients.OrderBook, executionSize, maxSlippagePercent float64) *types.ArbitrageOpportunity {
	if !isMultiOutcomeMarket(market) {
		return nil
	}

	tokenIDs := make([]string, len(market.Outcomes))
	prices := make([]float64, len(market.Outcomes))
	slippageResults := make([]clients.SlippageResult, len(market.Outcomes))

	for i, outcome := range market.Outcomes {
		tokenIDs[i] = outcome.Name
		prices[i] = outcome.Price
	}

	for i, tokenID := range tokenIDs {
		book, ok := orderBooks[tokenID]
		if !ok {
			return nil
		}

		slippage, err := clients.CalculateSlippage(book, clients.Buy, executionSize)
		if err != nil {
			return nil
		}

		if slippage.TotalFilled < executionSize {
			return nil
		}

		if slippage.Slippage > maxSlippagePercent {
			return nil
		}

		slippageResults[i] = slippage
	}

	executionPrices := make([]float64, len(market.Outcomes))
	for i, slippage := range slippageResults {
		executionPrices[i] = slippage.AveragePrice
	}

	sum := 0.0
	for _, price := range executionPrices {
		sum += price
	}

	if sum >= 1.0 {
		return nil
	}

	grossProfit := 1.0 - sum
	feeCost := fees.CalculatePolymarketFee(grossProfit, market)
	netProfit := grossProfit - feeCost

	if netProfit < 0.001 {
		return nil
	}

	totalSlippage := 0.0
	for i := range prices {
		totalSlippage += executionPrices[i] - prices[i]
	}

	legs := make([]types.TradeLeg, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		legs[i] = types.TradeLeg{
			Outcome: outcome.Name,
			Side:    types.Bid,
			Price:   executionPrices[i],
			Size:    executionSize / executionPrices[i],
		}
	}

	return &types.ArbitrageOpportunity{
		Market:             market,
		Strategy:           types.MultiOutcome,
		GrossProfit:        grossProfit,
		NetProfit:          netProfit,
		FeeCost:            feeCost,
		Score:              scoring.CalculateScore(market, netProfit),
		ExecutionPlan: types.ExecutionPlan{
			Legs:             legs,
			TotalCost:        sum * float64(len(market.Outcomes)),
			GuaranteedPayout: float64(len(market.Outcomes)),
		},
		SlippageImpact:     totalSlippage,
		AvailableLiquidity: market.Liquidity,
	}
}

func FindMultiOutcomeOpportunitiesWithOrderBooks(markets []types.Market, orderBookGetter func([]string) (map[string]clients.OrderBook, error), executionSize, maxSlippagePercent float64) ([]types.ArbitrageOpportunity, error) {
	var opportunities []types.ArbitrageOpportunity

	for _, market := range markets {
		if !isMultiOutcomeMarket(market) {
			continue
		}

		tokenIDs := make([]string, len(market.Outcomes))
		for i, outcome := range market.Outcomes {
			tokenIDs[i] = outcome.Name
		}

		orderBooks, err := orderBookGetter(tokenIDs)
		if err != nil {
			fmt.Printf("Error fetching order books for market %s: %v\n", market.ID, err)
			continue
		}

		if opp := CheckMultiOutcomeWithOrderBooks(market, orderBooks, executionSize, maxSlippagePercent); opp != nil {
			opportunities = append(opportunities, *opp)
		}
	}

	return opportunities, nil
}
