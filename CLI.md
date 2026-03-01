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
- `convert-parquet` - Convert JSONL files to Parquet format
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
| `--offset` | int | 0 | Skip first N markets (by liquidity) |
| `--start-rank` | int | 0 | Start rank (1-based, inclusive) |
| `--end-rank` | int | 0 | End rank (inclusive, 0 = unlimited) |

### Examples

```bash
# Display first 10 markets
predmarket-scanner fetch-markets

# Display first 5 markets, but only fetch up to 100
predmarket-scanner fetch-markets --limit 5 --max-markets 100

# Fetch markets 101-200 (by liquidity ranking)
predmarket-scanner fetch-markets --start-rank 101 --end-rank 200

# Skip first 1000 markets, fetch next 500
predmarket-scanner fetch-markets --offset 1000 --max-markets 500

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
| `--offset` | int | 0 | Skip first N markets (by liquidity) |
| `--start-rank` | int | 0 | Start rank (1-based, inclusive) |
| `--end-rank` | int | 0 | End rank (inclusive, 0 = unlimited) |
| `--db` | string | data/history.db | Path to SQLite database |

### Examples

```bash
# Fetch 30 days of daily price history for 100 markets (default)
predmarket-scanner fetch-history

# Fetch hourly price history for last 7 days
predmarket-scanner fetch-history --interval 1h --max-days 7

# Fetch markets 101-200 (by liquidity ranking)
predmarket-scanner fetch-history --start-rank 101 --end-rank 200

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
| `-p, --min-profit` | float | 0.001 | Minimum profit threshold |
| `-l, --limit` | int | 100 | Maximum number of opportunities to display |
| `--max-markets` | int | 0 (all) | Maximum number of markets to fetch |
| `--strategy` | string | all | Strategy to use: `all`, `dutch_book`, `multi_outcome` |
| `--mode` | string | rest | Scan mode: `rest` (API polling), `event-driven` (WebSocket trigger), `periodic` (WebSocket interval) |
| `--scan-interval` | int | 1 | Scan interval in seconds (for periodic mode) |
| `--historical` | bool | false | Enable historical backtesting mode |
| `--time` | string | "" | Target historical timestamp (RFC3339 format) |
| `--time-range` | string | "" | Time range for historical scanning (format: start,end) |
| `--db` | string | data/history.db | Path to SQLite database for historical data |
| `--workers` | int | 4 | Number of concurrent workers for historical scanning |
| `--skip-slippage` | bool | false | Skip order book fetch and slippage calculation |
| `--use-order-book` | bool | false | Use real order book prices (best ask) instead of Gamma API prices |
| `--closed` | bool | false | Include closed/resolved markets |
| `--no-fees` | bool | false | Ignore Polymarket fees (for non-crypto markets) |
| `--detailed` | bool | false | Show detailed output with score breakdown |
| `--all-snapshots` | bool | false | Scan all available snapshots in database |
| `--output` | string | "" | Export markets to JSON file (for debugging) |
| `--export-opps` | string | "" | Export opportunities to JSON file |
| `--debug` | bool | false | Enable debug output (show all markets checked) |

### Examples

**Live Scanning:**

```bash
# Basic scan with default settings (REST polling)
./bin/predmarket-scanner scan

# Scan with WebSocket event-driven mode (reacts to order book changes)
./bin/predmarket-scanner scan --mode event-driven

# Scan with WebSocket periodic mode (scan every 2 seconds)
./bin/predmarket-scanner scan --mode periodic --scan-interval 2

# Scan with WebSocket periodic mode using real order book prices
./bin/predmarket-scanner scan --mode periodic --scan-interval 1 --use-order-book

# Scan with $500 size
./bin/predmarket-scanner scan --size 500

# Scan only for Dutch book opportunities
./bin/predmarket-scanner scan --strategy dutch_book

# Scan only for Multi-Outcome opportunities
./bin/predmarket-scanner scan --strategy multi_outcome

# Tight filters with high profit threshold
./bin/predmarket-scanner scan --size 100 --min-profit 0.01

# Limit to first 1000 markets
./bin/predmarket-scanner scan --max-markets 1000 --limit 20

# Skip order book fetch for faster scanning (no slippage calculation)
./bin/predmarket-scanner scan --skip-slippage

# Show score breakdown
./bin/predmarket-scanner scan --detailed

# Export opportunities to file
./bin/predmarket-scanner scan --export-opps opportunities.json

# Debug: export raw markets and show all checked
./bin/predmarket-scanner scan --output markets.json --debug
```

**Scan Mode Comparison:**

| Mode | Behavior | Latency | CPU Usage | Best For |
|------|-----------|----------|------------|-----------|
| `rest` | Polls API periodically (every scan) | High (~1-2s) | Low | Simple use cases |
| `event-driven` | Scans immediately when WebSocket order book updates | Very Low (~100ms) | Medium | Reacting to fast market moves |
| `periodic` | Scans at fixed intervals using WebSocket order book | Medium (configurable) | Low-Medium | Regular monitoring without API limits |

**Historical Backtesting:**

```bash
# Start recording market data using WebSocket
./bin/predmarket-scanner record --max-markets 500

# Scan data from a specific point in time (RFC3339 format with timezone)
./bin/predmarket-scanner scan --historical --time "2026-02-28T12:00:00+01:00"

# Use a custom database path
./bin/predmarket-scanner scan --historical --time "2026-02-28T12:00:00+01:00" --db /path/to/history.db

# Scan across a time range
./bin/predmarket-scanner scan --historical --time-range "2026-02-28T00:00:00+01:00,2026-02-28T23:59:59+01:00"

# Use multiple workers for faster scanning
./bin/predmarket-scanner scan --historical --time-range "..." --workers 8

# Scan without fees (for non-crypto markets)
./bin/predmarket-scanner scan --historical --time-range "..." --no-fees
```

### Output Columns

**Default view:**

| Column | Description |
|--------|-------------|
| Market | Market question |
| Gross % | Theoretical profit before fees and slippage |
| Net % | Actual profit after fees and calculated slippage |
| Fee % | Polymarket trading fees (most markets are fee-free, only 15-min crypto/Serie A/NCAAB have fees) |
| Slip % | Calculated price impact from order book depth (based on --size) |
| Liq $ | Available market liquidity in USDC |
| Score | Risk-adjusted score (higher = better) |

**Detailed view (with `--detailed` flag):**

| Column | Description |
|--------|-------------|
| P_Sc | Profit Score (40% weight) |
| L_Sc | Liquidity Score (25% weight) |
| V_Sc | Volume Score (15% weight) |
| E_Rk | Execution Risk (15% weight) |
| T_Dc | Time Decay (5% weight) |

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

Run as a daemon to continuously record market data using WebSocket to JSONL files for historical backtesting.

### Usage

```bash
predmarket-scanner record [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-i, --interval` | int | 60 | **Deprecated** - Kept for backward compatibility, but recording is now continuous |
| `--max-markets` | int | 500 | Maximum number of markets to record |
| `--offset` | int | 0 | Skip first N markets (by liquidity) |
| `--start-rank` | int | 0 | Start rank (1-based, inclusive) |
| `--end-rank` | int | 0 | End rank (inclusive, 0 = unlimited) |

### Examples

```bash
# Record top 500 markets (default)
predmarket-scanner record

# Record top 100 most liquid markets
predmarket-scanner record --max-markets 100

# Record markets 101-500 (by liquidity ranking)
predmarket-scanner record --start-rank 101 --end-rank 500

# Skip first 1000 markets, record next 500
predmarket-scanner record --offset 1000 --max-markets 500
```

### Data Storage

All recorded data is stored in `data/` directory as daily compressed JSONL files (`market_data_YYYY-MM-DD.jsonl.gz`).

**Note:** This command uses WebSocket for real-time data recording, providing zero-latency tick-by-tick order book updates. Data is automatically compressed with gzip for efficient storage (10-20x compression ratio).

### Converting to Parquet (Optional)

After recording, you can convert JSONL files to Parquet format for even better compression and faster querying:

```bash
# Convert a single day to Parquet
./bin/predmarket-scanner convert-parquet --date 2026-03-01

# Convert all available days to Parquet
./bin/predmarket-scanner convert-parquet --all

# Convert and delete original JSONL files (saves space)
./bin/predmarket-scanner convert-parquet --all --delete
```

**Storage Comparison:**
- Original JSON (uncompressed): ~200-300 MB/day (500 markets)
- Compressed JSONL (gzip): ~10-20 MB/day (10-20x compression)
- Parquet (columnar): ~2-5 MB/day (additional 5-10x compression)

### Stopping the Daemon

Press `Ctrl+C` to gracefully stop the recording daemon.

---

## `convert-parquet`

Convert compressed JSONL files to Parquet format for efficient archival and faster querying.

### Usage

```bash
predmarket-scanner convert-parquet [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--date` | string | "" | Date to convert in `YYYY-MM-DD` format (or empty for all) |
| `--all` | bool | false | Convert all available days |
| `--delete` | bool | false | Delete JSONL files after successful conversion |

### Examples

```bash
# Convert a specific day to Parquet
./bin/predmarket-scanner convert-parquet --date 2026-03-01

# Convert all available days
./bin/predmarket-scanner convert-parquet --all

# Convert all and delete original JSONL files to save space
./bin/predmarket-scanner convert-parquet --all --delete
```

### Output

```
Converting 2026-03-01...
Converted 50000 records...
Original: 167755 bytes, Compressed: 23421 bytes (ratio: 7.2x)
```

**Storage Savings:**
- JSONL (gzip): ~15 MB/day
- Parquet (Snappy): ~2 MB/day
- **Total compression: ~85x reduction vs uncompressed JSON**

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
# Start recording live data using WebSocket
./bin/predmarket-scanner record --max-markets 500

# After recording, analyze specific time periods
./bin/predmarket-scanner scan --historical --time "2026-02-28 12:30:00" --strategy all
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
2. **Adjust `--size` to test different execution sizes** - The slippage shown is based on this size.
3. **Use `--min-profit` to filter low-quality opportunities** - Set higher values (e.g., 0.01 for 1%) to focus on better trades.
4. **Start recording data early** - Historical backtesting requires data. Run `record` in the background to build your dataset.
5. **Combine with export for analysis** - Export opportunities and analyze in your preferred tool (Excel, Python, etc.).
6. **Use `--skip-slippage` for faster scanning** - Skips order book API calls.
7. **Use `--detailed` to see score breakdown** - Shows profit, liquidity, volume, execution risk, and time decay components.
8. **Use `--output` for debugging** - Export raw market data to see what's being scanned.

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
