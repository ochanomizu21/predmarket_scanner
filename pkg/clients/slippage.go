package clients

import (
	"fmt"
	"strconv"
)

type SlippageResult struct {
	AveragePrice     float64
	TotalFilled      float64
	Slippage         float64
	PenetratedLevels int
	OrderBookUsed    []OrderLevel
}

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

func CalculateSlippage(book OrderBook, side Side, size float64) (SlippageResult, error) {
	var levels []OrderLevel
	if side == Buy {
		levels = book.Asks
	} else {
		levels = book.Bids
	}

	if len(levels) == 0 {
		return SlippageResult{}, fmt.Errorf("no order book levels available")
	}

	var remainingSize float64 = size
	var totalCost float64 = 0
	var totalFilled float64 = 0
	var usedLevels []OrderLevel
	var penetratedLevels int

	for _, level := range levels {
		if remainingSize <= 0 {
			break
		}

		levelPrice, _ := strconv.ParseFloat(level.Price, 64)
		levelSize, _ := strconv.ParseFloat(level.Size, 64)

		fillSize := remainingSize
		if levelSize < remainingSize {
			fillSize = levelSize
		}

		totalCost += levelPrice * fillSize
		totalFilled += fillSize
		remainingSize -= fillSize
		penetratedLevels++

		usedLevels = append(usedLevels, OrderLevel{
			Price: level.Price,
			Size:  fmt.Sprintf("%.2f", fillSize),
		})
	}

	if totalFilled == 0 {
		return SlippageResult{}, fmt.Errorf("no liquidity available")
	}

	averagePrice := totalCost / totalFilled

	var slippage float64
	if penetratedLevels > 0 {
		firstPrice, _ := strconv.ParseFloat(levels[0].Price, 64)
		if firstPrice > 0 {
			slippage = ((averagePrice - firstPrice) / firstPrice) * 100
			if side == Sell {
				slippage = -slippage
			}
		}
	}

	return SlippageResult{
		AveragePrice:     averagePrice,
		TotalFilled:      totalFilled,
		Slippage:         slippage,
		PenetratedLevels: penetratedLevels,
		OrderBookUsed:    usedLevels,
	}, nil
}

func (c *PolymarketClient) CalculateExecutionPrice(book OrderBook, side Side, size float64) (float64, error) {
	slippage, err := CalculateSlippage(book, side, size)
	if err != nil {
		return 0, err
	}
	return slippage.AveragePrice, nil
}

func (c *PolymarketClient) GetAvailableLiquidity(book OrderBook, side Side, maxSlippagePercent float64) float64 {
	var levels []OrderLevel
	if side == Buy {
		levels = book.Asks
	} else {
		levels = book.Bids
	}

	if len(levels) == 0 {
		return 0
	}

	firstPrice, _ := strconv.ParseFloat(levels[0].Price, 64)
	if firstPrice == 0 {
		return 0
	}

	var totalLiquidity float64
	var acceptablePrice float64

	if side == Buy {
		acceptablePrice = firstPrice * (1 + maxSlippagePercent/100)
	} else {
		acceptablePrice = firstPrice * (1 - maxSlippagePercent/100)
	}

	for _, level := range levels {
		levelPrice, _ := strconv.ParseFloat(level.Price, 64)
		if side == Buy && levelPrice > acceptablePrice {
			break
		}
		if side == Sell && levelPrice < acceptablePrice {
			break
		}

		levelSize, _ := strconv.ParseFloat(level.Size, 64)
		totalLiquidity += levelSize
	}

	return totalLiquidity
}
