package types

import "time"

type Platform string

const (
	Polymarket Platform = "polymarket"
	Limitless  Platform = "limitless"
)

type Side string

const (
	Bid Side = "bid"
	Ask Side = "ask"
)

type StrategyType string

const (
	DutchBook     StrategyType = "dutch_book"
	MultiOutcome  StrategyType = "multi_outcome"
	NoBasket      StrategyType = "no_basket"
	CrossPlatform StrategyType = "cross_platform"
	Combinatorial StrategyType = "combinatorial"
)

type Market struct {
	ID          string     `json:"id"`
	Question    string     `json:"question"`
	Platform    Platform   `json:"platform"`
	Outcomes    []Outcome  `json:"outcomes"`
	Liquidity   float64    `json:"liquidity"`
	Volume      float64    `json:"volume"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	ClobTokenIDs []string  `json:"clob_token_ids,omitempty"`
}

type Outcome struct {
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	Side           Side    `json:"side"`
	OrderBookDepth int     `json:"order_book_depth"`
}

type ArbitrageOpportunity struct {
	Market             Market        `json:"market"`
	Strategy           StrategyType  `json:"strategy"`
	GrossProfit        float64       `json:"gross_profit"`
	NetProfit          float64       `json:"net_profit"`
	FeeCost            float64       `json:"fee_cost"`
	Score              float64       `json:"score"`
	ExecutionPlan      ExecutionPlan `json:"execution_plan"`
	SlippageImpact     float64       `json:"slippage_impact"`
	YesSlippage        float64       `json:"yes_slippage"`
	NoSlippage         float64       `json:"no_slippage"`
	AvailableLiquidity float64       `json:"available_liquidity"`
}

type ExecutionPlan struct {
	Legs             []TradeLeg `json:"legs"`
	TotalCost        float64    `json:"total_cost"`
	GuaranteedPayout float64    `json:"guaranteed_payout"`
}

type TradeLeg struct {
	Outcome string  `json:"outcome"`
	Side    Side    `json:"side"`
	Price   float64 `json:"price"`
	Size    float64 `json:"size"`
}
