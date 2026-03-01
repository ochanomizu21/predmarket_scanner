# Prediction Market Scanner

A high-performance research tool to identify arbitrage opportunities across prediction markets, focusing on Polymarket.

---

## 🚀 Key Features

- **Arbitrage Strategies**: Implemented Dutch-book (binary) and Multi-outcome strategies.
- **Slippage Analysis**: Realistic net-profit calculations using real-time order book depth.
- **Historical Backtesting**: Record market snapshots into SQLite and run scans against any point in time.
- **Multiple Output Formats**: Export results to Terminal tables, JSON, or CSV.
- **Risk-Adjusted Scoring**: Ranks opportunities based on profit, liquidity, volume, and execution risk.

---

## 🛠 Installation

### Prerequisites
- [Go 1.21+](https://go.dev/doc/install)

### Build
```bash
# Clone the repository
git clone https://github.com/ochanomizu21/predmarket_scanner.git
cd predmarket_scanner

# Build the binary
make build
```

---

## 📖 Quick Start

### 1. Basic Scan (Live)
Scan all active markets for Dutch-book opportunities with a $1000 size and 5% max slippage.
```bash
./bin/predmarket-scanner scan --size 1000 --max-slippage 5
```

### 2. Historical Backtesting
First, start the recording daemon to collect data:
```bash
./bin/predmarket-scanner record --interval 30 --max-markets 500
```
Then, scan the recorded data at a specific timestamp (RFC3339 format):
```bash
./bin/predmarket-scanner scan --historical --time "2026-02-28T12:00:00+01:00"
```

---

## 📚 Documentation

For more detailed information, please refer to:

- **[CLI Reference](CLI.md)**: Full command list and flag descriptions.
- **[Arbitrage Strategies](DOCS/STRATEGIES.md)**: Deep dive into the logic, examples, and planned strategies.
- **[Development & Implementation](DOCS/DEVELOPMENT.md)**: Architecture details, database schema, and technical roadmap.

---

## 📜 License

MIT License - See [LICENSE](LICENSE) file for details.
