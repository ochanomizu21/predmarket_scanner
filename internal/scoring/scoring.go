package scoring

import (
	"math"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/types"
)

type ScoreFactors struct {
	ProfitScore    float64 `json:"profit_score"`
	LiquidityScore float64 `json:"liquidity_score"`
	VolumeScore    float64 `json:"volume_score"`
	ExecutionRisk  float64 `json:"execution_risk"`
	TimeDecay      float64 `json:"time_decay"`
}

func CalculateScore(market types.Market, netProfit float64) (float64, ScoreFactors) {
	factors := ScoreFactors{
		ProfitScore:    normalizeProfit(netProfit),
		LiquidityScore: normalizeLiquidity(market.Liquidity),
		VolumeScore:    normalizeVolume(market.Volume),
		ExecutionRisk:  1.0,
		TimeDecay:      calculateTimeDecay(market.EndTime),
	}

	score := (factors.ProfitScore * 0.4) +
		(factors.LiquidityScore * 0.25) +
		(factors.VolumeScore * 0.15) +
		(factors.ExecutionRisk * 0.15) +
		(factors.TimeDecay * 0.05)

	return score, factors
}

func normalizeProfit(profit float64) float64 {
	return 1.0 / (1.0 + math.Exp(-50.0*(profit-0.025)))
}

func normalizeLiquidity(liquidity float64) float64 {
	normalized := (liquidity / 100000.0)
	if normalized > 1.0 {
		normalized = 1.0
	}
	return math.Log1p(normalized)
}

func normalizeVolume(volume float64) float64 {
	normalized := (volume / 1000000.0)
	if normalized > 1.0 {
		normalized = 1.0
	}
	return math.Log1p(normalized)
}

func calculateTimeDecay(endTime *time.Time) float64 {
	if endTime == nil {
		return 0.5
	}

	remaining := endTime.Sub(time.Now()).Hours()
	if remaining < 0 {
		return 0.0
	}

	result := remaining / 168.0
	if result > 1.0 {
		result = 1.0
	}

	return result
}
