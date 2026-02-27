# Prediction Market Scanner

A high-performance research tool to identify arbitrage opportunities across prediction markets, focusing on Polymarket.

## Features

- **Fast Compilation**: Builds in seconds (not minutes)
- **Order Book Integration**: Real slippage calculations based on actual market depth
- **Configurable Execution**: Set trade size and maximum acceptable slippage
- **Comprehensive API**: Fetch-markets, scan, and export commands
- **Multiple Output Formats**: Terminal tables, JSON, and CSV
- **Risk-Adjusted Scoring**: Considers liquidity, volume, and time decay

## Installation

### Prerequisites
- Go 1.21 or later

### Build from Source

```bash
# Clone the repository
git clone https://github.com/ochanomizu21/predmarket_scanner.git
cd predmarket_scanner

# Build the binary
go build -o bin/predmarket-scanner cmd/*.go

# Or use the Makefile
make build
```

## Usage

### Fetch Markets

Fetch and display markets from Polymarket:

```bash
# Display first 10 markets
./bin/predmarket-scanner fetch-markets --limit 10

# Fetch only 100 markets (for testing)
./bin/predmarket-scanner fetch-markets --max-markets 100 --limit 5
```

### Scan for Arbitrage

Scan for Dutch book arbitrage opportunities with liquidity and slippage considerations:

```bash
# Basic scan (default: $1000 size, 5% max slippage)
./bin/predmarket-scanner scan

# Scan with $500 execution size, 1% max slippage
./bin/predmarket-scanner scan --size 500 --max-slippage 1

# Scan with very tight slippage limits
./bin/predmarket-scanner scan --size 100 --max-slippage 0.5 --min-profit 0.01

# Limit to first 1000 markets
./bin/predmarket-scanner scan --max-markets 1000 --limit 20
```

**Flags:**
- `--size` (-s): Execution size in USDC (default: 1000)
- `--max-slippage`: Maximum acceptable slippage in percent (default: 5.0)
- `--min-profit` (-p): Minimum profit threshold (default: 0.001)
- `--limit` (-l): Maximum number of opportunities to display (default: 100)
- `--max-markets`: Maximum number of markets to fetch (0 = all, ~34K)

### Export Opportunities

Export opportunities to JSON or CSV:

```bash
# Export to JSON
./bin/predmarket-scanner export --format json --output opportunities

# Export to CSV
./bin/predmarket-scanner export --format csv --output opportunities
```

## Understanding the Output

### Scan Output

```
=== Arbitrage Opportunities ===

Market                                      Gross %  Net %   Fee %   Slip %  Liq $   Score
-------------------------------------------------------------------------
Will BTC hit $100k by Dec 2024?         3.500    3.430   0.070   0.120   15000   0.890
Will Trump win the 2024 election?         2.800    2.744   0.056   0.085   50000   0.925
```

**Columns:**
- **Gross %**: Theoretical profit before fees and slippage
- **Net %**: Actual profit after fees and slippage
- **Fee %**: Polymarket trading fees (2%)
- **Slip %**: Price impact from order book depth
- **Liq $**: Available market liquidity in USDC
- **Score**: Risk-adjusted score (higher = better)

### Slippage Explained

Slippage occurs when your trade size is larger than the first level of the order book.

**Example:**
```
Order Book (asks for YES):
- 0.013 x 500 USDC   (Level 1)
- 0.014 x 200 USDC   (Level 2)
- 0.015 x 300 USDC   (Level 3)

Execution Size: 1000 USDC

Fill Simulation:
- 500 @ 0.013 = $6.50  (100% of level 1)
- 200 @ 0.014 = $2.80  (100% of level 2)
- 300 @ 0.015 = $4.50  (100% of level 3)

Average Price: 0.0138
Slippage: (0.0138 - 0.013) / 0.013 = 6.15%
```

## Architecture

### Project Structure

```
.
├── cmd/
│   ├── main.go           # Entry point
│   └── commands.go       # CLI commands (fetch-markets, scan, export)
├── pkg/
│   ├── clients/
│   │   ├── polymarket.go # Polymarket Gamma/CLOB API client
│   │   └── slippage.go  # Slippage calculation logic
│   ├── output/
│   │   └── output.go     # Terminal, JSON, CSV output
│   ├── strategies/
│   │   └── dutch_book.go # Dutch book arbitrage detection
│   └── types/
│       └── market.go      # Core data types
├── internal/
│   ├── fees/
│   │   └── fees.go       # Fee calculations
│   └── scoring/
│       └── scoring.go     # Risk-adjusted scoring
├── go.mod
├── Makefile
└── README.md
```

### API Integration

**Gamma API** (`https://gamma-api.polymarket.com`)
- Market metadata (questions, outcomes, liquidity, volume)
- Best bid/ask prices
- Market spread
- Pagination support (fetches all ~34K markets)

**CLOB API** (`https://clob.polymarket.com`)
- Full order book depth
- Real-time bid/ask levels
- Order book analysis for slippage

### Arbitrage Detection Flow

```
1. Fetch Markets (Gamma API)
   └─ Get market metadata, best prices, spread

2. For Binary Markets:
   └─ Check if YES price + NO price < 1.0

3. If Potential Arbitrage Found:
   ├─ Fetch Order Books (CLOB API)
   │  ├─ YES token order book
   │  └─ NO token order book
   │
   ├─ Calculate Slippage
   │  ├─ Simulate filling orders at execution size
   │  ├─ Calculate average execution price
   │  └─ Determine slippage percentage
   │
   ├─ Recalculate Profit
   │  ├─ Use slippage-impacted prices
   │  ├─ Subtract 2% trading fee
   │  └─ Check if still profitable
   │
   └─ Score Opportunity
      ├─ Profit margin
      ├─ Market liquidity
      ├─ Trading volume
      ├─ Time until resolution
      └─ Execution risk

4. Display Results
   └─ Only show if net_profit > threshold
```

## Strategies

### Dutch Book Arbitrage

Detects binary markets where buying both outcomes yields guaranteed profit.

**Example:**
```
Market: "Will BTC hit $100k by Dec 2024?"
YES price: 0.48
NO price: 0.51
Sum: 0.99

Arbitrage: Buy 1 YES @ 0.48, Buy 1 NO @ 0.51
Cost: 0.99
Guaranteed Payout: 1.00
Profit: 1.00% (before fees and slippage)
```

**With Slippage:**
```
With slippage, prices might be:
YES effective: 0.485 (1.04% slippage)
NO effective: 0.515 (0.98% slippage)
Sum: 1.000

Net result: No profit after slippage (opportunity filtered out)
```

## Configuration

### Environment Variables

Currently, no environment variables are required. All configuration is done via CLI flags.

### Fee Structure

- **Polymarket Trading Fee**: 2% of winnings
- **Market Maker Rebate**: 0.02% (when applicable)

### Scoring Algorithm

Opportunities are scored based on:
- **Profit Margin** (40%): Higher profit = better score
- **Liquidity** (25%): More liquidity = less slippage risk
- **Volume** (15%): Higher volume = more active market
- **Execution Risk** (15%): Based on order book depth and spread
- **Time Decay** (5%): More time until resolution = better

Score is normalized to 0-1 range.

## Performance

### Build Time
- **Go**: 2-5 seconds
- **Binary Size**: ~8MB

### Runtime Performance
- **Fetch All Markets**: ~30 seconds (34K markets with pagination)
- **Scan Without Order Books**: <1 second
- **Scan With Order Books**: ~1-2 minutes (depends on API rate limits)

## Contributing

Contributions are welcome! Areas for improvement:

1. **Additional Strategies**: Multi-outcome, NO-basket, cross-platform arbitrage
2. **Database Storage**: SQLite for historical opportunity tracking
3. **Web Dashboard**: Real-time visualization of opportunities
4. **WebSocket Support**: Live order book updates
5. **Backtesting**: Historical performance analysis
6. **More Platforms**: Add Limitless, Augur, etc.

## License

MIT License - See LICENSE file for details

## Acknowledgments

- Built for prediction market research
- Uses Polymarket public APIs
- Inspired by Dutch book arbitrage theory

## Support

For issues, questions, or contributions:
- GitHub: https://github.com/ochanomizu21/predmarket_scanner
- Issues: https://github.com/ochanomizu21/predmarket_scanner/issues
