package fees

import (
	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

const (
	tradingFeeRate = 0.02
	makerRebate    = 0.0002
)

func CalculatePolymarketFee(profit float64, _ types.Market) float64 {
	fee := profit * tradingFeeRate
	rebate := profit * makerRebate

	return fee - rebate
}

func CalculateNetROI(grossProfit, cost, fees float64) float64 {
	netPayout := cost + grossProfit - fees
	return (netPayout - cost) / cost
}
