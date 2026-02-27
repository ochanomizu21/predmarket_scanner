# Prediction Market Scanner (Go)

A research tool to identify arbitrage opportunities across prediction markets, focusing on Polymarket.

## Building

```bash
# Build the binary
go build -o bin/predmarket-scanner cmd/main.go

# Or use the Makefile
make build
```

## Usage

### Fetch Markets

Fetch and display markets from Polymarket:

```bash
./bin/predmarket-scanner fetch-markets --limit 10
```

### Scan for Arbitrage

Scan for Dutch book arbitrage opportunities:

```bash
./bin/predmarket-scanner scan --min-profit 0.001 --limit 50
```

### Export Opportunities

Export opportunities to JSON or CSV:

```bash
# Export to JSON
./bin/predmarket-scanner export --format json --output opportunities

# Export to CSV
./bin/predmarket-scanner export --format csv --output opportunities
```

## Features

- Fetch markets from Polymarket Gamma API
- Fetch prices from Polymarket CLOB API
- Dutch book arbitrage detection (YES + NO < 1.0)
- Fee calculations (2% trading fee)
- Risk-adjusted scoring
- Export to JSON/CSV
- Terminal output with tables

## Conversion Notes

This is a Go conversion of the original Rust implementation. The key differences:

- Uses direct HTTP calls to Polymarket APIs instead of an SDK
- Uses Cobra for CLI instead of Clap
- Uses standard library for JSON/CSV instead of serde/csv
- No external dependencies for decimal/math (uses float64)
