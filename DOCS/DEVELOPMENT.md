# Development & Implementation Reference

This document provides a technical overview of the `predmarket-scanner` architecture, implementation details, and the long-term project roadmap.

## 1. System Architecture

The scanner is built in Go for high performance and fast iteration. It follows a modular design for flexibility and testability.

### 1.1 Project Structure
```
.
├── cmd/
│   ├── main.go           # Entry point
│   └── commands.go       # CLI commands (fetch-markets, scan, export, etc.)
├── pkg/
│   ├── clients/
│   │   ├── polymarket.go # Polymarket Gamma & CLOB API clients
│   │   └── slippage.go   # Core slippage calculation logic
│   ├── database/
│   │   └── sqlite.go     # SQLite integration for historical data
│   ├── providers/
│   │   └── dataprovider.go # Abstraction for live vs. historical data
│   ├── strategies/
│   │   ├── dutch_book.go # Binary Dutch-book strategy
│   │   └── multi_outcome.go # Multi-outcome strategy
│   ├── types/
│   │   └── market.go      # Core data models
│   └── output/
│       └── output.go      # Formatting for Terminal, JSON, and CSV
├── internal/
│   ├── fees/
│   │   └── fees.go        # Fee structure calculations
│   └── scoring/
│       └── scoring.go     # Risk-adjusted scoring algorithm
└── Makefile               # Build and test targets
```

### 1.2 Data Providers
The application uses a `DataProvider` interface to abstract where market data comes from:
- **`LiveDataProvider`**: Fetches real-time data from the Polymarket API.
- **`HistoricalDataProvider`**: Queries a local SQLite database for backtesting against recorded snapshots.

---

## 2. Core Implementation Details

### 2.1 Slippage & Execution Simulation
Slippage calculation is central to accurate arbitrage detection. We simulate filling an order of size `$S` by walking through the order book levels (CLOB API) until the required volume is met.
- **Slippage Formula**: `((AverageExecutionPrice - FirstLevelPrice) / FirstLevelPrice) * 100`

### 2.2 Historical Recording & Backtesting
The scanner includes a built-in daemon (`record` command) that snapshots market states into a local SQLite database (`data/history.db`).
- **`snapshots` table**: Stores timestamped market states.
- **`order_book_levels` table**: Stores full order book depth for each snapshot.
- **Backtesting**: Use `scan --historical --time "YYYY-MM-DD HH:MM:SS"` to run strategies against these snapshots.

---

## 3. Future Roadmap

### Phase 1: Performance Optimization [NEXT]
- **Order Book Caching**: Implement in-memory caching (TTL: 30s) to reduce redundant API calls.
- **Concurrent Fetching**: Use Go routines to fetch order books for multiple tokens in parallel.
- **Database Deduplication**: Optimize `record` daemon to skip inserting identical order book snapshots.

### Phase 2: Enhanced Strategies
- **NO-Basket Strategy**: Implement arbitrage for NO contract baskets.
- **Cross-Platform Support**: Add connectors for Limitless and other prediction platforms.

### Phase 3: Real-Time & Web
- **WebSocket Integration**: Live order book updates for faster detection.
- **Web Dashboard**: A Go-based web server for real-time visualization of opportunities.

---

## 4. Technical Debt & Refactoring
- **Code Organization**: Continue splitting large files into smaller, domain-focused packages.
- **Configuration Management**: Move CLI flags to a central YAML/TOML configuration file using Viper.
- **Testing**: Expand unit test coverage for slippage and strategy logic.

---

## 5. Build System
The project uses a standard `Makefile` for common tasks:
- `make build`: Compiles the binary to `bin/predmarket-scanner`.
- `make test`: Runs the Go test suite.
- `make clean`: Removes build artifacts.
- `make deps`: Updates project dependencies.
