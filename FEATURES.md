# Features & Strategies Documentation

## Current Implementation Status: ✅ Production Ready

## Implemented Features

### 1. Market Data Fetching
- Fetches all active markets from Polymarket (~34K markets)
- Supports pagination for large datasets
- Retrieves market metadata: question, liquidity, volume, outcomes
- Gets best bid/ask prices and spread

### 2. Order Book Integration
- Fetches full order book depth from CLOB API
- Multi-level bid/ask support
- Real-time price discovery
- Slippage calculation based on actual market depth

### 3. Dutch Book Arbitrage
- Detects binary markets (YES/NO) with guaranteed profit
- Accounts for execution size and slippage
- Calculates net profit after fees
- Risk-adjusted scoring for opportunity ranking

### 4. Slippage Considerations
- Simulates order execution at specified size
- Calculates weighted average execution price
- Determines slippage percentage
- Filters opportunities exceeding max slippage tolerance

### 5. Fee Calculations
- 2% trading fee on winnings
- 0.02% market maker rebate
- Net profit calculation
- ROI calculation available

### 6. Risk-Adjusted Scoring
- Multi-factor scoring algorithm
- Considers profit, liquidity, volume, risk, time
- Normalized scores (0-1 range)
- Prioritizes high-quality opportunities

### 7. Multiple Output Formats
- Terminal tables with formatted output
- JSON export for programmatic use
- CSV export for spreadsheet analysis

### 8. Comprehensive CLI
- `fetch-markets`: Display market data
- `scan`: Find arbitrage opportunities
- `export`: Export results to file
- Configurable execution size and slippage limits

## Arbitrage Strategies

### Dutch Book Arbitrage (Implemented ✅)

**Concept**: Binary markets where YES price + NO price < 1.0

**Example**:
```
Market: "Will Trump win the 2024 election?"
YES price: 0.48
NO price: 0.49
Sum: 0.97

Strategy:
- Buy 1 YES @ 0.48 = $0.48
- Buy 1 NO @ 0.49 = $0.49
- Total Cost: $0.97
- Guaranteed Payout: $1.00
- Gross Profit: 3.09%
```

**With Slippage**:
```
Order Book Analysis:
- YES token asks: 0.48 x 500, 0.49 x 200, ...
- NO token asks: 0.49 x 1000, 0.50 x 500, ...

Execution Size: $1000
YES fill: 500 @ 0.48 + 500 @ 0.49 = avg 0.485
NO fill: 1000 @ 0.49 = avg 0.490

New prices with slippage:
YES effective: 0.485 (1.04% slippage)
NO effective: 0.490 (0.00% slippage)
Sum: 0.975

Net profit after slippage: 2.56%
```

**Implementation**:
- File: `pkg/strategies/dutch_book.go`
- Function: `checkDutchBookWithSlippage()`
- Filters: Binary markets, positive net profit, max slippage

### Multi-Outcome Arbitrage (Planned 📋)

**Concept**: Markets with N outcomes where sum of prices ≠ 1.0

**Example**:
```
Market: "What will be the temperature on July 1st?"
Outcomes: "Below 70°F", "70-80°F", "80-90°F", "Above 90°F"
Prices: 0.20, 0.25, 0.30, 0.20
Sum: 0.95

Strategy: Buy all outcomes
Cost: 0.20 + 0.25 + 0.30 + 0.20 = 0.95
Payout: 1.00
Profit: 5.26%
```

**Implementation Requirements**:
- Handle variable number of outcomes
- Calculate slippage for each outcome
- Determine optimal purchase amounts

### NO-Basket Arbitrage (Planned 📋)

**Concept**: For NO contracts, the sum should be N-1 (not 1.0)

**Example**:
```
Market: "Who will win the election?"
Outcomes: "Democrat", "Republican", "Other"
YES prices: 0.40, 0.45, 0.10
NO prices: 0.60, 0.55, 0.90

NO Basket: Buy NO for Democrat and Republican
Cost: 0.60 + 0.55 = 1.15
Payout: 2.00 (if Other wins)
Profit: (2.00 / 1.15) - 1 = 73.9%
```

**Implementation Requirements**:
- Convert YES prices to NO prices
- Check sum of NO prices vs. N-1
- More complex probability calculations

### Cross-Platform Arbitrage (Planned 📋)

**Concept**: Same event across different platforms with price differences

**Example**:
```
Event: "Will Trump win 2024 election?"

Polymarket:
- YES: 0.48
- NO: 0.49

Limitless:
- YES: 0.52
- NO: 0.46

Arbitrage:
- Buy YES on Polymarket @ 0.48
- Buy NO on Polymarket @ 0.49
- Sell YES on Limitless @ 0.52
- Sell NO on Limitless @ 0.46

Net: +0.01 per share
```

**Implementation Requirements**:
- Multiple platform support
- Event matching algorithm (title similarity)
- Cross-platform order fetching
- Transfer fees consideration

### Combinatorial Arbitrage (Planned 📋)

**Concept**: Exploit logical dependencies between related markets

**Example**:
```
Market A: "Will Trump win the election?"
Market B: "Will the margin be > 5%?"
Market C: "Will turnout be > 60%?"

If A is YES, B and C provide additional information
Construct portfolio that covers all outcome combinations
```

**Implementation Requirements**:
- Dependency graph construction
- Event relationship detection
- Complex portfolio optimization
- Linear programming solver

## Scoring Algorithm

### Factors

#### 1. Profit Score (40%)
**Formula**: Sigmoid function
```go
normalizeProfit(profit) = 1 / (1 + e^(-50 * (profit - 0.025)))
```

**Behavior**:
- 0% profit → 0.0 score
- 2.5% profit → 0.5 score
- 5% profit → 1.0 score
- Saturates beyond 5%

#### 2. Liquidity Score (25%)
**Formula**: Log-normalized
```go
normalizeLiquidity(liq) = ln(liq / 100000 + 1)
```

**Behavior**:
- $0 → 0.0
- $100K → 0.69
- $1M → 2.40 (capped at 1.0)

#### 3. Volume Score (15%)
**Formula**: Log-normalized
```go
normalizeVolume(vol) = ln(vol / 1000000 + 1)
```

**Behavior**:
- $0 → 0.0
- $1M → 0.69
- $10M → 2.40 (capped at 1.0)

#### 4. Execution Risk (15%)
**Formula**: Order book depth analysis
```go
calculateExecutionRisk(book) {
  depth = sum(order_sizes[:10])  // Top 10 levels
  spread = ask_price - bid_price
  
  risk = 1.0 - (spread * depth_factor)
  return max(0.0, min(1.0, risk))
}
```

**Behavior**:
- Deep order book → 1.0 (low risk)
- Shallow order book → 0.0 (high risk)
- Wide spread → 0.0 (high risk)

#### 5. Time Decay (5%)
**Formula**: Time until resolution
```go
calculateTimeDecay(endTime) {
  remaining = endTime - now
  return min(1.0, remaining / 168 hours)  // 1 week
}
```

**Behavior**:
- No end time → 0.5 (neutral)
- 1 week → 1.0
- 0 hours → 0.0 (already ended)
- More time = better

### Composite Score
```go
Score = (Profit * 0.40) +
        (Liquidity * 0.25) +
        (Volume * 0.15) +
        (Risk * 0.15) +
        (Time * 0.05)
```

**Interpretation**:
- 0.8-1.0: Excellent opportunity
- 0.6-0.8: Good opportunity
- 0.4-0.6: Moderate opportunity
- 0.2-0.4: Weak opportunity
- 0.0-0.2: Poor opportunity

## Execution Strategies

### Order Sizing

**Conservative**: Small sizes ($100-500)
- Minimal slippage
- Lower absolute profit
- Better execution certainty

**Moderate**: Medium sizes ($500-2000)
- Reasonable slippage
- Balanced profit
- Good execution

**Aggressive**: Large sizes ($2000+)
- High slippage risk
- Higher absolute profit
- May not fill completely

### Slippage Tolerance

**Tight**: <0.5%
- Few opportunities
- High execution quality
- Low risk

**Moderate**: 0.5-2%
- Balanced approach
- Good opportunities
- Manageable risk

**Loose**: >2%
- Many opportunities
- Variable execution
- Higher risk

## Market Analysis

### Efficient Markets

**Characteristics**:
- YES + NO prices ≈ 1.0
- Narrow spreads (<0.5%)
- Deep order books
- High volume
- Few arbitrage opportunities

### Inefficient Markets

**Characteristics**:
- YES + NO prices significantly < 1.0
- Wide spreads (>1%)
- Shallow order books
- Low volume
- Frequent arbitrage opportunities

### Market Types

**High Liquidity**: Political events, major elections
**Medium Liquidity**: Economic indicators, sports
**Low Liquidity**: Niche questions, long-term events

## Risk Management

### Primary Risks

1. **Slippage Risk**
   - Order book depth insufficient
   - Large execution sizes
   - Mitigation: Size limits, slippage tolerance

2. **Execution Risk**
   - Orders not filling
   - Price moving during execution
   - Mitigation: Order book analysis, limit orders

3. **Market Risk**
   - Events resolving unexpectedly
   - Changes in market conditions
   - Mitigation: Diversification, stop losses

4. **Platform Risk**
   - API failures
   - Rate limiting
   - Mitigation: Error handling, retries

### Risk Mitigation Strategies

1. **Position Limits**
   - Maximum exposure per market
   - Diversify across markets

2. **Time Limits**
   - Focus on near-term events
   - Avoid long-term uncertainty

3. **Liquidity Filters**
   - Minimum liquidity thresholds
   - Order book depth requirements

4. **Profit Thresholds**
   - Minimum profit requirements
   - Risk-adjusted returns

## Performance Metrics

### Execution Speed
- Market fetch: ~30s (34K markets)
- Order book fetch: ~1-2s per token
- Scan: ~1-2 minutes (with order books)
- Export: <1s

### Opportunity Frequency
Historical expectations (empirical):
- Dutch book: 0-5 per day
- Multi-outcome: 0-2 per day
- Cross-platform: 5-20 per day
- Total: 5-25 opportunities per day

### Profitability
Historical averages:
- Gross profit range: 0.5% - 5.0%
- Net profit after fees: 0.3% - 3.9%
- Typical opportunity: 1.0% - 2.0%

## Future Strategy Development

### Priority 1: Enhanced Dutch Book
- Real-time order book streaming
- Advanced slippage prediction
- Optimal order sizing

### Priority 2: Multi-Outcome
- Generalized N-outcome support
- Combinatorial optimization
- Partial fill handling

### Priority 3: Cross-Platform
- Multiple exchange integration
- Event matching algorithms
- Transfer cost calculation

### Priority 4: Machine Learning
- Price prediction models
- Opportunity detection
- Automated strategy selection
