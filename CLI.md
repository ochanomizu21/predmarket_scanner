# CLI Reference

Complete reference for all `predmarket-scanner` commands and flags.

## Global Usage

```bash
predmarket-scanner [command] [flags]
```

## Available Commands

- `fetch-markets` - Fetch and display markets from Polymarket
- `scan` - Scan for arbitrage opportunities
- `export` - Export opportunities to file
- `record` - Record historical market data (daemon)
- `fetch-history` - Fetch historical price data from Polymarket API
- `completion` - Generate autocompletion script for your shell

---

## `fetch-markets`

Fetch and display market data from Polymarket.

### Usage

```bash
predmarket-scanner fetch-markets [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-l, --limit` | int | 10 | Number of markets to display |
| `-m, --max-markets` | int | 0 (all) | Maximum number of markets to fetch |
| `--min-outcomes` | int | 0 | Minimum number of outcomes (0 = no minimum) |
| `--max-outcomes` | int | 0 | Maximum number of outcomes (0 = no maximum) |

### Examples

```bash
# Display first 10 markets
predmarket-scanner fetch-markets

# Display first 5 markets, but only fetch up to 100
predmarket-scanner fetch-markets --limit 5 --max-markets 100

# Fetch all markets (approximately 34K) and display first 20
predmarket-scanner fetch-markets --limit 20

# Filter for binary markets (exactly 2 outcomes)
predmarket-scanner fetch-markets --min-outcomes 2 --max-outcomes 2

# Filter for multi-outcome markets (3+ outcomes)
predmarket-scanner fetch-markets --min-outcomes 3

# Filter for markets with 2-5 outcomes
predmarket-scanner fetch-markets --min-outcomes 2 --max-outcomes 5
```

---

## `fetch-history`

Fetch historical price data from Polymarket's CLOB API and store in SQLite.

### Usage

```bash
predmarket-scanner fetch-history [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | 100 | Maximum number of markets to fetch history for |
| `--max-days` | int | 30 | Maximum number of days of history to fetch |
| `--interval` | string | 1d | Price history interval: `1m`, `1h`, `6h`, `1d` |
| `--db` | string | data/history.db | Path to SQLite database |

### Examples

```bash
# Fetch 30 days of daily price history for 100 markets (default)
predmarket-scanner fetch-history

# Fetch hourly price history for last 7 days
predmarket-scanner fetch-history --interval 1h --max-days 7

# Fetch minute-level history for last day
predmarket-scanner fetch-history --interval 1m --max-days 1

# Fetch full historical data
predmarket-scanner fetch-history --interval max
```

---

## `scan`

Scan markets for arbitrage opportunities (Dutch book or Multi-Outcome).

### Usage

```bash
predmarket-scanner scan [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-s, --size` | float | 1000 | Execution size in USDC |
| `--max-slippage` | float | 5.0 | Maximum slippage in percent |
| `-p, --min-profit` | float | 0.001 | Minimum profit threshold |
| `-l, --limit` | int | 100 | Maximum number of opportunities to display |
| `--max-markets` | int | 0 (all) | Maximum number of markets to fetch |
| `--strategy` | string | all | Strategy to use: `all`, `dutch_book`, `multi_outcome` |
| `--historical` | bool | false | Enable historical backtesting mode |
| `--time` | string | "" | Target historical timestamp (format: YYYY-MM-DD HH:MM:SS) |
| `--time-range` | string | "" | Time range for historical scanning (format: start,end) |
| `--db` | string | data/history.db | Path to SQLite database for historical data |

### Examples

**Live Scanning:**

```bash
# Basic scan with default settings
predmarket-scanner scan

# Scan with $500 size and 1% max slippage
predmarket-scanner scan --size 500 --max-slippage 1

# Scan only for Dutch book opportunities
predmarket-scanner scan --strategy dutch_book

# Scan only for Multi-Outcome opportunities
predmarket-scanner scan --strategy multi_outcome

# Tight filters with high profit threshold
predmarket-scanner scan --size 100 --max-slippage 0.5 --min-profit 0.01

# Limit to first 1000 markets
predmarket-scanner scan --max-markets 1000 --limit 20
```

**Historical Backtesting:**

```bash
# Scan data from a specific point in time
predmarket-scanner scan --historical --time "2026-02-28 12:00:00"

# Use a custom database path
predmarket-scanner scan --historical --time "2026-02-28 12:00:00" --db /path/to/history.db

# Scan across a time range
predmarket-scanner scan --historical --time-range "2026-02-28 00:00:00,2026-02-28 23:59:59"
```

### Output Columns

| Column | Description |
|--------|-------------|
| Market | Market question |
| Gross % | Theoretical profit before fees and slippage |
| Net % | Actual profit after fees and slippage |
| Fee % | Polymarket trading fees (2%) |
| Slip % | Price impact from order book depth |
| Liq $ | Available market liquidity in USDC |
| Score | Risk-adjusted score (higher = better) |

---

## `export`

Export arbitrage opportunities to JSON or CSV files.

### Usage

```bash
predmarket-scanner export [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-f, --format` | string | json | Export format: `json` or `csv` |
| `-o, --output` | string | opportunities | Output filename prefix |
| `-s, --size` | float | 1000 | Execution size in USDC |
| `--max-slippage` | float | 5.0 | Maximum slippage in percent |
| `--max-markets` | int | 0 (all) | Maximum number of markets to fetch |
| `--strategy` | string | all | Strategy to use: `all`, `dutch_book`, `multi_outcome` |

### Examples

```bash
# Export to JSON (creates opportunities.json)
predmarket-scanner export --format json

# Export to CSV (creates opportunities.csv)
predmarket-scanner export --format csv

# Custom output filename
predmarket-scanner export --format json --output my-opportunities

# Export only Dutch book opportunities
predmarket-scanner export --strategy dutch_book --format csv
```

### Output Format

**JSON:**
```json
[
  {
    "market": {
      "id": "...",
      "question": "...",
      "outcomes": [...]
    },
    "strategy": "dutch_book",
    "gross_profit": 0.03,
    "net_profit": 0.0294,
    "score": 0.89,
    "execution_plan": {...}
  }
]
```

**CSV:**
```csv
market_id,question,strategy,gross_profit,net_profit,score
...,dutch_book,0.03,0.0294,0.89
```

---

## `record`

Run as a daemon to continuously record market snapshots to SQLite database for historical backtesting.

### Usage

```bash
predmarket-scanner record [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-i, --interval` | int | 60 | Recording interval in seconds |
| `--max-markets` | int | 500 | Maximum number of markets to record |

### Examples

```bash
# Record every 60 seconds (default)
predmarket-scanner record

# Record every 10 seconds for faster data collection
predmarket-scanner record --interval 10

# Record top 100 most liquid markets only
predmarket-scanner record --max-markets 100

# Record every 30 seconds with 200 markets
predmarket-scanner record --interval 30 --max-markets 200
```

### Data Storage

All recorded data is stored in `data/history.db` (or custom path specified in scan command).

**Database Schema:**

- `markets` - Market metadata (ID, question, end time, liquidity, volume)
- `snapshots` - Timestamped market snapshots
- `outcomes_snapshot` - Outcome prices per snapshot
- `order_book_levels` - Full order book depth per snapshot

### Stopping the Daemon

Press `Ctrl+C` to gracefully stop the recording daemon.

---

## Strategies

### Dutch Book (`dutch_book`)

Detects binary markets (YES/NO) where `YES price + NO price < 1.0`.

**Example:**
```
Market: "Will BTC hit $100k by Dec 2024?"
YES price: 0.48
NO price: 0.51
Sum: 0.99
Arbitrage: Buy 1 YES @ 0.48, Buy 1 NO @ 0.51
Cost: 0.99, Payout: 1.00, Profit: 1.01%
```

### Multi-Outcome (`multi_outcome`)

Detects markets with N outcomes where `sum of all YES asks < 1.0`.

**Example:**
```
Market: "What will be the temperature on July 1st?"
Outcomes: "Below 70°F", "70-80°F", "80-90°F", "Above 90°F"
Prices: 0.20, 0.25, 0.30, 0.20
Sum: 0.95
Arbitrage: Buy all outcomes
Cost: 0.95, Payout: 1.00, Profit: 5.26%
```

### All (`all`)

Runs both Dutch Book and Multi-Outcome strategies.

---

## Common Workflows

### Quick Market Check

```bash
# Check top 20 markets
predmarket-scanner fetch-markets --limit 20

# Quick scan for opportunities
predmarket-scanner scan --limit 5
```

### Detailed Analysis

```bash
# Find best opportunities with conservative settings
predmarket-scanner scan --size 500 --max-slippage 1 --limit 10

# Export results for further analysis
predmarket-scanner export --format json --output results
```

### Historical Research

```bash
# Fetch historical price data from Polymarket API (one-time backfill)
predmarket-scanner fetch-history --interval 1d --max-days 30

# Start recording live data
predmarket-scanner record --interval 30 --max-markets 500

# After recording, analyze specific time periods
predmarket-scanner scan --historical --time "2026-02-28 12:30:00" --strategy all

# Compare multiple time points
predmarket-scanner scan --historical --time "2026-02-28 00:00:00" --limit 20
predmarket-scanner scan --historical --time "2026-02-28 12:00:00" --limit 20
predmarket-scanner scan --historical --time "2026-02-28 23:00:00" --limit 20
```

### Strategy-Specific Analysis

```bash
# Focus on binary markets only
predmarket-scanner scan --strategy dutch_book --max-markets 1000

# Focus on multi-outcome opportunities
predmarket-scanner scan --strategy multi_outcome --max-markets 1000
```

---

## Tips

1. **Use `--max-markets` to limit scanning time** - The full market dataset is ~34K markets, which can take several minutes to scan with order books.
2. **Adjust `--size` for your trading capital** - Smaller sizes have less slippage but lower absolute profits.
3. **Use `--min-profit` to filter low-quality opportunities** - Set higher values (e.g., 0.01 for 1%) to focus on better trades.
4. **Start recording data early** - Historical backtesting requires data. Run `record` in the background to build your dataset.
5. **Combine with export for analysis** - Export opportunities and analyze in your preferred tool (Excel, Python, etc.).

---

## Help

Get help for any command:

```bash
predmarket-scanner --help
predmarket-scanner [command] --help
```

Example:
```bash
predmarket-scanner scan --help
```
