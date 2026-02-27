# Prediction Market Arbitrage Scanner - Implementation Plan

## Project Overview

Build a Rust-based research tool to identify arbitrage opportunities across prediction markets, focusing on Polymarket first, with Limitless Exchange as secondary.

## Requirements Summary

- **Goal**: Research tool (not real-time trading)
- **Language**: Rust
- **Strategies**: All arbitrage types (start with MVP)
- **Architecture**: Evaluate polymarket-cli SDK
- **Platforms**: Polymarket (primary), Limitless (secondary)
- **Fees**: Show both gross and net profit
- **Scoring**: Risk-adjusted scoring
- **Storage**: ~30GB limit
- **Output**: Web dashboard, JSON/CSV export, terminal output

---

## Architecture Decision: Polymarket SDK

### Analysis of `polymarket-cli`

The polymarket-cli uses `polymarket-client-sdk` (v0.4) which provides:
- **gamma** - markets/events API (read-only market data)
- **clob** - order book API (prices, order books, trades)
- **data** - on-chain data (positions, volume, leaderboard)
- **ctf** - conditional token framework (split/merge operations)

### Recommendation: **Use the SDK as a library**

**Pros:**
- Official, maintained SDK with API rate handling
- Type-safe Rust interface to all Polymarket endpoints
- Saves weeks of development time on API integration
- Proven in production (CLI is actively used)

**Cons:**
- Limited to Polymarket's supported operations (this is actually fine for a scanner)
- May need to work around some CLI-specific assumptions

**Conclusion:** Start with SDK as primary dependency. Build custom client for Limitless once their API is understood.

---

## Phase 1: MVP - Single-Market Dutch Book Arbitrage

### Objective
Identify binary markets where YES price + NO price < 1.0 (long arbitrage)

### Core Components

#### 1. Project Structure

```
predmarket_scanner/
├── Cargo.toml
├── src/
│   ├── main.rs              # CLI entry point
│   ├── config.rs            # Configuration management
│   ├── clients/
│   │   ├── mod.rs
│   │   ├── polymarket.rs    # Polymarket SDK wrapper
│   │   └── limitless.rs     # Limitless client (placeholder)
│   ├── strategies/
│   │   ├── mod.rs
│   │   └── dutch_book.rs    # Single-market YES/NO arbitrage
│   ├── types/
│   │   ├── mod.rs
│   │   └── market.rs        # Market data types
│   ├── scoring.rs           # Risk-adjusted scoring
│   ├── fees.rs              # Fee calculations
│   └── output/
│       ├── mod.rs
│       ├── terminal.rs      # Pretty terminal output
│       ├── json_csv.rs      # Export functionality
│       └── dashboard/       # Web dashboard
│           └── main.rs
└── data/                   # SQLite database for historical data
```

#### 2. Data Models

```rust
// types/market.rs
pub struct Market {
    pub id: String,
    pub question: String,
    pub platform: Platform,
    pub outcomes: Vec<Outcome>,
    pub liquidity: f64,
    pub volume: f64,
    pub end_time: Option<DateTime<Utc>>,
}

pub struct Outcome {
    pub name: String,      // "YES" or "NO"
    pub price: f64,        // 0.0 to 1.0
    pub side: Side,        // Buy/Sell price
    pub order_book_depth: usize,
}

pub enum Platform {
    Polymarket,
    Limitless,
}

pub enum Side {
    Bid,    // Best price to buy
    Ask,    // Best price to sell
}

pub struct ArbitrageOpportunity {
    pub market: Market,
    pub strategy: StrategyType,
    pub gross_profit: f64,
    pub net_profit: f64,
    pub fee_cost: f64,
    pub score: f64,        // Risk-adjusted score
    pub execution_plan: ExecutionPlan,
}

pub struct ExecutionPlan {
    pub legs: Vec<TradeLeg>,
    pub total_cost: f64,
    pub guaranteed_payout: f64,
}

pub struct TradeLeg {
    pub outcome: String,
    pub side: Side,
    pub price: f64,
    pub size: f64,
}
```

#### 3. Strategy Implementation

```rust
// strategies/dutch_book.rs
use crate::types::*;
use crate::fees;
use crate::scoring;

pub fn find_opportunities(markets: &[Market]) -> Vec<ArbitrageOpportunity> {
    markets
        .iter()
        .filter(|m| is_binary_market(m))
        .filter_map(|m| check_dutch_book(m))
        .collect()
}

fn check_dutch_book(market: &Market) -> Option<ArbitrageOpportunity> {
    // Get best YES and NO prices
    let yes_price = market.outcomes.iter().find(|o| o.name == "YES")?.price;
    let no_price = market.outcomes.iter().find(|o| o.name == "NO")?.price;

    let sum = yes_price + no_price;

    // Check if there's arbitrage (sum < 1.0)
    if sum >= 1.0 {
        return None;
    }

    let gross_profit = 1.0 - sum;  // e.g., 0.03 = 3%
    let fee_cost = fees::calculate_polymarket_fee(gross_profit, market);
    let net_profit = gross_profit - fee_cost;

    // Skip if net profit is too low
    if net_profit < 0.001 {  // 0.1% minimum threshold
        return None;
    }

    Some(ArbitrageOpportunity {
        market: market.clone(),
        strategy: StrategyType::DutchBook,
        gross_profit,
        net_profit,
        fee_cost,
        score: scoring::calculate_score(market, net_profit),
        execution_plan: build_execution_plan(market, yes_price, no_price),
    })
}

fn build_execution_plan(market: &Market, yes_price: f64, no_price: f64) -> ExecutionPlan {
    ExecutionPlan {
        legs: vec![
            TradeLeg {
                outcome: "YES".to_string(),
                side: Side::Bid,
                price: yes_price,
                size: 1.0,
            },
            TradeLeg {
                outcome: "NO".to_string(),
                side: Side::Bid,
                price: no_price,
                size: 1.0,
            },
        ],
        total_cost: yes_price + no_price,
        guaranteed_payout: 1.0,
    }
}
```

#### 4. Fee Calculations

```rust
// fees.rs
pub fn calculate_polymarket_fee(profit: f64, market: &Market) -> f64 {
    // Polymarket fees (verify actual rates)
    // Trading fee: 2% of winnings (not of position size)
    // Market maker rebate: 0.02% (if applicable)

    const TRADING_FEE_RATE: f64 = 0.02;  // 2%
    const MAKER_REBATE: f64 = 0.0002;    // 0.02%

    // Fee is on the profit portion
    let fee = profit * TRADING_FEE_RATE;
    let rebate = profit * MAKER_REBATE;

    fee - rebate
}

pub fn calculate_net_roi(gross_profit: f64, cost: f64, fees: f64) -> f64 {
    let net_payout = cost + gross_profit - fees;
    (net_payout - cost) / cost
}
```

#### 5. Risk-Adjusted Scoring

```rust
// scoring.rs
use crate::types::*;

pub fn calculate_score(market: &Market, net_profit: f64) -> f64 {
    let factors = ScoreFactors {
        profit_score: normalize_profit(net_profit),
        liquidity_score: normalize_liquidity(market.liquidity),
        volume_score: normalize_volume(market.volume),
        execution_risk: calculate_execution_risk(market),
        time_decay: calculate_time_decay(market.end_time),
    };

    // Weighted average of factors
    (factors.profit_score * 0.4) +
    (factors.liquidity_score * 0.25) +
    (factors.volume_score * 0.15) +
    (factors.execution_risk * 0.15) +
    (factors.time_decay * 0.05)
}

struct ScoreFactors {
    profit_score: f64,
    liquidity_score: f64,
    volume_score: f64,
    execution_risk: f64,  // Higher is better (less risk)
    time_decay: f64,      // Higher is better (more time)
}

fn normalize_profit(profit: f64) -> f64 {
    // Sigmoid function to normalize 0-5% profit range
    1.0 / (1.0 + std::f64::consts::E.powf(-50.0 * (profit - 0.025)))
}

fn normalize_liquidity(liquidity: f64) -> f64 {
    // Log scale: $100K baseline
    (liquidity / 100_000.0).min(1.0).ln_1p()
}

fn normalize_volume(volume: f64) -> f64 {
    // Log scale: $1M baseline
    (volume / 1_000_000.0).min(1.0).ln_1p()
}

fn calculate_execution_risk(market: &Market) -> f64 {
    // Based on order book depth and spread
    // Lower spread = better execution
    // More depth = better execution
    1.0  // Placeholder - implement based on order book data
}

fn calculate_time_decay(end_time: Option<DateTime<Utc>>) -> f64 {
    match end_time {
        None => 0.5,  // Unknown = neutral
        Some(end) => {
            let remaining = (end - Utc::now()).num_hours();
            if remaining < 0 { 0.0 }  // Already ended
            else { (remaining / 168.0).min(1.0) }  // Normalize to 1 week
        }
    }
}
```

#### 6. Polymarket Client Wrapper

```rust
// clients/polymarket.rs
use polymarket_client_sdk::gamma::Client;

pub struct PolymarketClient {
    gamma: Client,
}

impl PolymarketClient {
    pub fn new() -> Self {
        Self {
            gamma: Client::default(),
        }
    }

    pub async fn fetch_all_markets(&self) -> Result<Vec<Market>, anyhow::Error> {
        let markets = self.gamma.list_markets(/* params */).await?;

        markets.iter().map(|m| {
            // Convert SDK type to our Market type
            Ok(Market {
                id: m.id.clone(),
                question: m.question.clone(),
                platform: Platform::Polymarket,
                outcomes: fetch_outcomes(&self.gamma, &m.id).await?,
                liquidity: m.liquidity,
                volume: m.volume,
                end_time: m.end_time,
            })
        }).collect()
    }

    pub async fn fetch_order_book(&self, market_id: &str) -> Result<OrderBook, anyhow::Error> {
        // Use CLOB client for order book data
        let clob_client = ClobClient::default();
        let book = clob_client.order_book(market_id).await?;

        Ok(OrderBook { ... })
    }
}
```

#### 7. Main CLI

```rust
// main.rs
use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "predmarket-scanner")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Scan {
        #[arg(short, long)]
        min_profit: Option<f64>,

        #[arg(short, long)]
        limit: Option<usize>,

        #[arg(short, long)]
        format: OutputFormat,
    },
    History {
        #[arg(short, long)]
        days: u32,
    },
    Dashboard {
        #[arg(short, long)]
        port: u16,
    },
}

#[derive(clap::ValueEnum, Clone)]
enum OutputFormat {
    Table,
    Json,
    Csv,
}

#[tokio::main]
async fn main() -> Result<(), anyhow::Error> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Scan { min_profit, limit, format } => {
            run_scan(min_profit.unwrap_or(0.001), limit, format).await
        },
        Commands::History { days } => {
            run_history(days).await
        },
        Commands::Dashboard { port } => {
            run_dashboard(port).await
        },
    }
}
```

---

## Phase 2: Storage & Historical Data

### Database Schema (SQLite)

```sql
-- Schema for storing scan results and historical market data

CREATE TABLE scans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    markets_scanned INTEGER,
    opportunities_found INTEGER
);

CREATE TABLE opportunities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scan_id INTEGER,
    market_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    strategy TEXT NOT NULL,
    gross_profit REAL,
    net_profit REAL,
    fee_cost REAL,
    score REAL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (scan_id) REFERENCES scans(id)
);

CREATE TABLE market_prices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    market_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    yes_price REAL,
    no_price REAL,
    liquidity REAL,
    volume REAL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for queries
CREATE INDEX idx_opportunities_timestamp ON opportunities(timestamp);
CREATE INDEX idx_opportunities_market ON opportunities(market_id);
CREATE INDEX idx_market_prices_timestamp ON market_prices(timestamp);
CREATE INDEX idx_market_prices_market ON market_prices(market_id);
```

### Storage Decision: SQLite

**Why SQLite:**
- ~30GB is plenty for SQLite (handles terabytes)
- No separate database server needed
- Fast read/write for this use case
- Easy to export/backup
- Rust has excellent `sqlx` and `rusqlite` libraries

**Storage Estimation:**
- 1000 markets scanned every 10 minutes = 6K scans/day
- Each scan with price data: ~10KB
- ~60MB/day for price history
- ~1MB/day for opportunities (less frequent)
- **Total for 1 year: ~22GB** (fits well within 30GB)

---

## Phase 3: Web Dashboard

### Tech Stack
- **Backend**: Axum (Rust web framework)
- **Frontend**: Simple HTML/CSS/JavaScript (or Tauri for desktop)
- **Real-time**: SSE (Server-Sent Events) for live updates

### Features
1. **Live Opportunities Table** - Auto-refreshing table of current opportunities
2. **Historical Charts** - Profit distribution over time, strategy performance
3. **Market Explorer** - Filter/search markets, view order books
4. **Strategy Comparison** - Which strategies yield best results
5. **Export** - Download current scan as JSON/CSV

### API Endpoints

```
GET  /api/opportunities         # Current opportunities
GET  /api/opportunities/history # Historical data
GET  /api/markets               # All markets with filters
GET  /api/scan                  # Trigger new scan (async)
SSE  /api/events                # Real-time updates
GET  /api/export/:format        # JSON/CSV export
```

---

## Phase 4: Additional Strategies

### 2. Multi-Outcome Dutch Book

For markets with N outcomes where sum(YES) ≠ 1.0:

```rust
fn check_multi_outcome_arb(market: &Market) -> Option<ArbitrageOpportunity> {
    let total: f64 = market.outcomes.iter().map(|o| o.price).sum();

    if (total - 1.0).abs() < 0.001 {
        return None;  // No arbitrage
    }

    if total < 1.0 {
        // Long arbitrage: buy all YES
        Some(build_long_arb(market, total))
    } else {
        // Short arbitrage: sell all YES (if possible)
        Some(build_short_arb(market, total))
    }
}
```

### 3. NO-Basket Arbitrage

For NO contracts (should sum to N-1):

```rust
fn check_no_basket_arb(market: &Market) -> Option<ArbitrageOpportunity> {
    let num_outcomes = market.outcomes.len();
    let expected_sum = (num_outcomes - 1) as f64;
    let actual_sum: f64 = market.outcomes.iter().map(|o| o.price).sum();

    if (actual_sum - expected_sum).abs() < 0.001 {
        return None;
    }

    // Build NO basket strategy
    Some(build_no_basket(market, actual_sum, expected_sum))
}
```

### 4. Cross-Platform Arbitrage

Identical events across Polymarket and Limitless:

```rust
async fn find_cross_platform_arbs(
    poly_markets: Vec<Market>,
    limi_markets: Vec<Market>
) -> Vec<ArbitrageOpportunity> {
    // Match events by title/description similarity
    let matches = find_matching_events(poly_markets, limi_markets);

    matches.iter()
        .filter_map(|(poly, limi)| check_cross_arb(poly, limi))
        .collect()
}
```

### 5. Combinatorial Arbitrage

Exploit logical dependencies (e.g., "Democrats win" vs margin buckets):

```rust
fn check_combinatorial_arb(markets: &[Market]) -> Vec<ArbitrageOpportunity> {
    // Build dependency graph from market questions
    // Find subsets that cover all possible outcomes
    // Calculate arbitrage if sum of weighted prices < 1.0
    vec![]  // Placeholder
}
```

---

## Phase 5: Limitless Exchange Integration

### API Discovery Tasks

1. **Inspect Network Traffic**
   - Use browser DevTools (Network tab) while browsing Limitless
   - Identify API endpoints, request/response formats
   - Check for authentication requirements

2. **Document API**
   - Markets endpoint (pagination, filters)
   - Order book data
   - Fee structure
   - Rate limits

3. **Build Client**
   ```rust
   // clients/limitless.rs
   pub struct LimitlessClient {
       base_url: String,
       client: reqwest::Client,
   }

   impl LimitlessClient {
       pub async fn fetch_markets(&self) -> Result<Vec<Market>> {
           // Implement based on discovered API
       }
   }
   ```

---

## Implementation Timeline

### Week 1-2: Foundation
- [ ] Set up Rust project with dependencies
- [ ] Implement core data types
- [ ] Create Polymarket SDK wrapper
- [ ] Basic CLI structure

### Week 3-4: MVP
- [ ] Implement single-market Dutch book strategy
- [ ] Fee calculations
- [ ] Risk-adjusted scoring
- [ ] Terminal output (pretty tables)

### Week 5-6: Storage & Dashboard
- [ ] SQLite integration
- [ ] Historical data tracking
- [ ] Simple web dashboard (Axum)
- [ ] JSON/CSV export

### Week 7-8: Additional Strategies
- [ ] Multi-outcome arbitrage
- [ ] NO-basket arbitrage
- [ ] Historical analysis features

### Week 9-10: Cross-Platform
- [ ] Limitless API discovery
- [ ] Limitless client implementation
- [ ] Cross-platform arbitrage detection

### Week 11-12: Polish
- [ ] Combinatorial arbitrage (if time)
- [ ] Documentation
- [ ] Performance optimization
- [ ] Testing

---

## Dependencies

```toml
[dependencies]
# Core
tokio = { version = "1", features = ["full"] }
anyhow = "1.0"
thiserror = "1.0"

# CLI
clap = { version = "4", features = ["derive"] }

# Polymarket
polymarket-client-sdk = "0.4"
alloy = "1.6"

# HTTP clients
reqwest = { version = "0.11", features = ["json"] }

# Database
sqlx = { version = "0.7", features = ["runtime-tokio", "sqlite", "chrono"] }
rusqlite = "0.30"

# Output
serde = { version = "1", features = ["derive"] }
serde_json = "1"
csv = "1.3"
tabled = "0.17"

# Web dashboard
axum = "0.7"
tower = "0.4"
tower-http = { version = "0.5", features = ["fs", "trace"] }

# Utilities
chrono = { version = "0.4", features = ["serde"] }
regex = "1.10"
log = "0.4"
env_logger = "0.11"

# Testing
mockall = "0.12"
```

---

## Next Steps

1. **Verify Polymarket SDK availability** - Check if `polymarket-client-sdk` is published to crates.io
2. **Test API access** - Set up test credentials and verify market data retrieval
3. **Create project skeleton** - Initialize Cargo project with structure
4. **Implement first fetch** - Get market data from Polymarket
5. **Build MVP** - Single-market Dutch book scanner

---

## Questions for Further Development

1. Should we implement real-time WebSocket connections for live order book updates?
2. What's the minimum profit threshold that makes sense for research purposes?
3. Should we track user-defined watchlists of specific markets?
4. Do we need backtesting capabilities to validate historical arbitrage performance?
5. Should the dashboard support multiple users or is single-user sufficient?
