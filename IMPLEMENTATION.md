# Implementation Notes

## Current Status: ✅ Production Ready

The prediction market scanner is fully implemented and operational.

## API Integration

### Gamma API (Primary)
- **Endpoint**: `https://gamma-api.polymarket.com`
- **Purpose**: Market metadata, best prices, spread
- **Features**:
  - Fetches all active markets (~34K total)
  - Pagination support (500 markets per page)
  - Returns best bid/ask and spread
  - Includes liquidity and volume data

### CLOB API (Order Books)
- **Endpoint**: `https://clob.polymarket.com`
- **Purpose**: Full order book depth
- **Features**:
  - Fetches complete order books for tokens
  - Multi-level bid/ask depth
  - Real-time price discovery
  - Slippage calculation support

## Core Components

### 1. Market Fetching

**File**: `pkg/clients/polymarket.go`

**Methods**:
- `FetchMarkets(limit int)` - Fetch markets with optional limit
- `fetchGammaMarketsWithLimit(maxMarkets int)` - Pagination support
- `FetchOrderBooks(tokenIDs []string)` - Get order books for tokens

**Key Features**:
- Handles JSON string arrays (API quirk)
- Parses string-based decimal values
- Filters out invalid markets
- Converts API types to internal types

### 2. Slippage Calculation

**File**: `pkg/clients/slippage.go`

**Algorithm**:
1. Get order book levels (bids or asks)
2. Simulate filling order at requested size
3. Walk through order book levels until filled
4. Calculate weighted average price
5. Compute slippage percentage

**Slippage Formula**:
```
Slippage % = ((AvgExecutionPrice - FirstLevelPrice) / FirstLevelPrice) * 100
```

**Output**:
- Average execution price
- Total amount filled
- Slippage percentage
- Number of levels penetrated
- Actual orders filled

### 3. Arbitrage Detection

**File**: `pkg/strategies/dutch_book.go`

**Dutch Book Algorithm**:
1. Filter for binary markets (YES/NO only)
2. Check: YES price + NO price < 1.0
3. Fetch order books for both tokens
4. Calculate slippage at execution size
5. Compute net profit (gross - fees - slippage)
6. Filter: net_profit > threshold

**Profit Calculation**:
```
Gross Profit = 1.0 - (YES_price + NO_price)
Fee = Gross Profit * 2%
Net Profit = Gross Profit - Fee - Slippage_Impact
```

### 4. Risk-Adjusted Scoring

**File**: `internal/scoring/scoring.go`

**Scoring Factors**:
- Profit Score (40%): Sigmoid function on profit margin
- Liquidity Score (25%): Log-normalized liquidity
- Volume Score (15%): Log-normalized volume
- Execution Risk (15%): Order book depth analysis
- Time Decay (5%): Time until market resolution

**Score Formula**:
```
Score = (Profit * 0.40) +
        (Liquidity * 0.25) +
        (Volume * 0.15) +
        (Risk * 0.15) +
        (Time * 0.05)
```

### 5. CLI Interface

**File**: `cmd/commands.go`

**Commands**:
1. `fetch-markets` - Display market data
2. `scan` - Find arbitrage opportunities
3. `export` - Export to JSON/CSV

**Flags**:
- `--limit` / `-l`: Number of items to display
- `--max-markets`: Maximum markets to fetch
- `--size` / `-s`: Trade execution size
- `--max-slippage`: Maximum slippage tolerance
- `--min-profit` / `-p`: Minimum profit threshold
- `--format` / `-f`: Output format (json/csv)
- `--output` / `-o`: Output filename

## Data Types

### Market
```go
type Market struct {
    ID        string
    Question  string
    Platform  Platform  (Polymarket)
    Outcomes  []Outcome
    Liquidity float64
    Volume    float64
    EndTime   *time.Time
}
```

### Outcome
```go
type Outcome struct {
    Name           string
    Price          float64
    Side           Side  (Bid/Ask)
    OrderBookDepth int
}
```

### OrderBook
```go
type OrderBook struct {
    Market         string
    AssetID        string
    Bids           []OrderLevel
    Asks           []OrderLevel
    MinOrderSize   string
    TickSize       string
    LastTradePrice string
}
```

### ArbitrageOpportunity
```go
type ArbitrageOpportunity struct {
    Market         Market
    Strategy       StrategyType
    GrossProfit    float64
    NetProfit      float64
    FeeCost        float64
    Score          float64
    ExecutionPlan  ExecutionPlan
    SlippageImpact float64
    YesSlippage    float64
    NoSlippage     float64
    AvailableLiquidity float64
}
```

## Build System

### Makefile Targets

- `make build` - Build binary
- `make clean` - Remove build artifacts
- `make run` - Build and run help
- `make test` - Run tests
- `make deps` - Download dependencies

### Go Modules

- Uses Go modules for dependency management
- Dependencies:
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/spf13/viper` - Configuration (future use)

## Performance Considerations

### API Rate Limits
- Gamma API: ~3-4 requests/second
- CLOB API: ~1-2 requests/second (order books are heavier)

### Caching Opportunities
Future enhancement to cache:
- Market metadata (TTL: 5 minutes)
- Order books (TTL: 30 seconds)
- Opportunity results (TTL: 1 minute)

### Concurrent Fetching
Could implement concurrent order book fetching:
```go
var wg sync.WaitGroup
semaphore := make(chan struct{}, 10)  // Limit to 10 concurrent

for _, tokenID := range tokenIDs {
    wg.Add(1)
    go func(id string) {
        defer wg.Done()
        semaphore <- struct{}{}
        defer func() { <-semaphore }()
        fetchOrderBook(id)
    }(tokenID)
}

wg.Wait()
```

## Error Handling

### API Errors
- Network timeouts (30 second timeout)
- Rate limiting (429 status)
- Invalid responses (malformed JSON)
- Missing data (empty fields)

### Graceful Degradation
- Skip markets without token IDs
- Skip markets with invalid prices
- Continue scanning if individual order books fail
- Log errors but don't crash

## Future Enhancements

### Short Term
1. **Order Book Caching**: Reduce API calls
2. **Database Storage**: SQLite for historical tracking
3. **Web Dashboard**: Go web server for visualization

### Medium Term
1. **WebSocket Support**: Real-time order book updates
2. **More Strategies**: Multi-outcome, NO-basket, cross-platform
3. **Backtesting**: Historical performance analysis
4. **Alert System**: Notifications for new opportunities

### Long Term
1. **ML-Based Scoring**: Predict profitability
2. **Multiple Platforms**: Limitless, Augur integration
3. **Automated Trading**: Execution via smart contracts
4. **Portfolio Management**: Track positions and P&L

## Testing

### Manual Testing

**Test Market Fetching**:
```bash
./bin/predmarket-scanner fetch-markets --limit 10
```

**Test Arbitrage Scan**:
```bash
./bin/predmarket-scanner scan --max-markets 100 --size 500
```

**Test Export**:
```bash
./bin/predmarket-scanner export --format json
```

### Automated Testing

Future: Add unit tests with `go test`
- Test slippage calculation
- Test arbitrage detection logic
- Test fee calculations
- Test scoring algorithms

## Troubleshooting

### Common Issues

**No Opportunities Found**
- Markets are efficient (arbitrage is rare)
- Try lowering `--min-profit` threshold
- Try increasing `--max-slippage` tolerance

**Slow Performance**
- Order book fetching is rate-limited
- Reduce `--max-markets` to limit API calls
- Use caching when implemented

**Build Errors**
- Ensure Go 1.21+ is installed
- Run `go mod tidy` to update dependencies
- Clear Go cache: `go clean -cache`

## Security Considerations

### API Security
- No authentication required for read operations
- HTTPS used for all API calls
- No secrets stored in code

### Data Privacy
- No personal data collected
- All market data is public
- No wallet connection required (read-only)

### Best Practices
- Validate all API responses
- Handle network errors gracefully
- Rate limit API calls
- Sanitize user inputs
