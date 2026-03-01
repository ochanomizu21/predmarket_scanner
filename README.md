# Prediction Market Scanner

A high-performance research tool to identify arbitrage opportunities across prediction markets, focusing on Polymarket.

---

## 🚀 Key Features

- **Arbitrage Strategies**: Implemented Dutch-book (binary) and Multi-outcome strategies.
- **Slippage Analysis**: Realistic net-profit calculations using real-time order book depth.
- **Real-time WebSocket**: Subscribe to Polymarket's WebSocket for zero-latency market data.
- **Historical Backtesting**: Record market snapshots to JSONL files and run scans against any point in time.
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

### 2. WebSocket-based Real-time Scanning
Scan markets in real-time using WebSocket data (event-driven or periodic mode):
```bash
# Event-driven mode (scans on order book changes)
./bin/predmarket-scanner scan --mode event-driven --size 1000

# Periodic mode (scans at fixed intervals)
./bin/predmarket-scanner scan --mode periodic --scan-interval 1 --size 1000
```

### 3. Historical Backtesting
First, record market data using WebSocket:
```bash
./bin/predmarket-scanner record --max-markets 500
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
- **[Implementation Plan](DOCS/IMPLEMENTATION_PLAN_0.md)**: WebSocket integration and storage overhaul roadmap.

---

## 📜 License

MIT License - See [LICENSE](LICENSE) file for details.
