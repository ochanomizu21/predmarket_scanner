package fees

import (
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

var (
	FeeRateBPS     = 625
	ApplyFees      = true
)

func CalculatePolymarketFee(profit float64, market types.Market) float64 {
	if !ApplyFees || profit <= 0 {
		return 0
	}

	var avgPrice float64
	for _, outcome := range market.Outcomes {
		avgPrice += outcome.Price
	}
	if len(market.Outcomes) > 0 {
		avgPrice /= float64(len(market.Outcomes))
	}

	feeRate := float64(FeeRateBPS) / 10000.0
	effectiveFeeRate := feeRate * avgPrice * (1 - avgPrice)

	return profit * effectiveFeeRate
}

func CalculateNetROI(grossProfit, cost, fees float64) float64 {
	netPayout := cost + grossProfit - fees
	return (netPayout - cost) / cost
}
