# Implementation Plan: Agentic Coding Roadmap

This document outlines the development roadmap for evolving `predmarket-scanner` from an arbitrage scanner into a **Quantitative Prediction Market Engine**. It integrates the existing codebase structure with the advanced simulation and risk management concepts derived from the "Quant Desk" methodology.

The plan is broken into actionable **Sprints** suitable for an AI coding agent. Each sprint contains specific implementation tasks, file targets, and verification criteria.

---

## Sprint 1: Foundation & Microstructure Refinement
**Goal:** Enhance the data models to support complex derivatives and robust simulation.

### Task 1.1: Enrich Market Data Models
The current `Market` struct is optimized for simple binary outcomes. We need to support the "dozens of parameters" mentioned in the quantitative analysis (volatility surfaces, correlation groups).

*   **File:** `pkg/types/market.go`
*   **Action:**
    *   Add `UnderlyingAsset` (string) and `Volatility` (float64) fields to the `Market` struct.
    *   Add `CorrelationGroup` (string) to tag related markets (e.g., "2024 Election", "Fed Rates").
    *   Add `SettlementSource` (string) to distinguish between "Oracle", "Sports", "Chainlink".
*   **Verification:** Ensure `json` marshaling/unmarshaling works for new fields. Update `FetchMarkets` to parse these if available from API metadata.

### Task 1.2: Implement Order Book "Shape" Analysis
Basic slippage calculation is insufficient for tail-risk pricing. We need to characterize the liquidity depth.

*   **File:** `pkg/clients/slippage.go`
*   **Action:**
    *   Create a function `AnalyzeDepth(book OrderBook, threshold float64) (liquidityDepth float64)`.
    *   This function calculates the total USD available within `threshold` percent of the mid-price.
    *   Update `SlippageCalculation` to return a `LiquidityScore` alongside slippage.
*   **Verification:** Unit test with mock order books (steep vs. flat curves) to ensure accurate depth scoring.

---

## Sprint 2: The Simulation Engine (Monte Carlo Core)
**Goal:** Implement the "Quantitative Core" to price contracts based on probabilistic models rather than just order book snapshots.

### Task 2.1: Basic Monte Carlo Pricer
Implement a generic simulation engine for binary contracts. This moves beyond "YES+NO < 1" to "Model Price vs. Market Price".

*   **File:** `pkg/simulation/monte_carlo.go` (New File)
*   **Action:**
    *   Define a struct `Simulator` with configuration for iterations (default: 100,000).
    *   Implement `SimulateBinaryContract(S0, K, mu, sigma, T float64) (prob, ci_lower, ci_upper float64)`.
    *   Use Geometric Brownian Motion (GBM) for asset-linked contracts (e.g., "BTC > 100k").
*   **Verification:** Run simulation for a known option price (Black-Scholes check) to ensure accuracy within confidence intervals.

### Task 2.2: Rare Event Simulation (Importance Sampling)
Standard Monte Carlo fails for low-probability events (e.g., "S&P drops 20% in a week").

*   **File:** `pkg/simulation/rare_events.go` (New File)
*   **Action:**
    *   Implement `EstimateRareEventProbability(S0, K_crash, sigma, T float64) float64`.
    *   Use Exponential Tilting (change of measure) to force the simulation into the "tail" region and re-weight the results.
*   **Verification:** Compare the error convergence of `EstimateRareEventProbability` vs naive simulation for a 5-sigma event. The Importance Sampling version should converge 10x-100x faster.

---

## Sprint 3: Strategy Engine 2.0
**Goal:** Decouple strategies from the scanner loop and enable model-based arbitrage.

### Task 3.1: Strategy Interface Refactoring
Move from hardcoded checks to a plugin-style architecture.

*   **File:** `pkg/strategies/interface.go` (New File)
*   **Action:**
    *   Define interface `Strategy`:
        ```go
        type Strategy interface {
            Name() string
            IdentifyOpportunities(markets []types.Market, sim *simulation.Simulator) ([]types.ArbitrageOpportunity, error)
            // 'sim' allows strategies to use Monte Carlo pricing
        }
        ```
    *   Refactor `DutchBook` and `MultiOutcome` to implement this interface.
*   **Verification:** Ensure `cmd/scan` still works using the new interface dispatch.

### Task 3.2: Model-vs-Market Strategy
Implement a strategy that arbitrages the difference between the Simulated Price and the Order Book Price.

*   **File:** `pkg/strategies/model_mispricing.go` (New File)
*   **Action:**
    *   Strategy Logic:
        1. Filter for asset-linked markets (Crypto/Stocks).
        2. Fetch `S0` (current price) and `sigma` (volatility) from external oracle.
        3. Use `SimulateBinaryContract` to calculate `ModelPrice`.
        4. If `ModelPrice > MarketAsk + Margin`, signal BUY.
    *   This implements the "Alpha" strategy from the quantitative articles.
*   **Verification:** Paper trade against historical data for a volatile asset (e.g., BTC). Does it correctly identify undervalued tail risk?

---

## Sprint 4: Backtesting & Risk ("The Overfitting Guard")
**Goal:** Build a backtester that simulates reality, including execution lag and information decay.

### Task 4.1: Vectorized Backtesting Engine
Current historical scanning is point-in-time. We need a "Time-Machine" that respects causality.

*   **File:** `pkg/backtest/engine.go` (New File)
*   **Action:**
    *   Create `BacktestEngine` struct that loads `LiveDataProvider` snapshots sequentially.
    *   Implement `RunBacktest(strategy Strategy, startTime, endTime time.Time) BacktestResult`.
    *   **Crucial Logic:** When a trade is signaled at Time T, fill the order at Time T+1 (simulating network/API latency).
*   **Verification:** Run a known profitable strategy. The backtest should show *lower* returns than the theoretical "instant fill" model, validating the latency simulation.

### Task 4.2: Performance Attribution & Brier Score
We need to know if we are earning alpha or just riding beta (market trends).

*   **File:** `pkg/backtest/metrics.go` (New File)
*   **Action:**
    *   Implement `BrierScore(predictions []float64, outcomes []bool) float64`.
    *   Implement `DecomposeReturns(trades []Trade) (alpha, beta float64)`.
    *   Calculate the Sharpe Ratio of the strategy output.
*   **Verification:** Run the backtest on a random strategy. Ensure Sharpe Ratio is near 0 and Brier Score is near 0.25 (random guessing).

---

## Sprint 5: Advanced Execution & Portfolio Construction
**Goal:** Handle complex portfolios and cross-market dependencies.

### Task 5.1: Correlation & Combinatorial Strategy
Implement logic to scan for logical inconsistencies across markets (e.g., "Dem Wins" vs "Rep Wins Senate").

*   **File:** `pkg/strategies/combinatorial.go`
*   **Action:**
    *   Define `LogicalDependency` struct (Parent Market, Child Market, ConstraintType).
        *   Example: If Market A (Dem Win) is True, Market B (Rep Win) must be False.
    *   Scan for violations: `Price(A) + Price(B) != 1.0` (after accounting for platform fees).
    *   Use the Monte Carlo engine to simulate correlated paths (e.g., using a Copula if needed, or simple conditional probability trees).
*   **Verification:** Create a synthetic market dataset with a logical inconsistency. The strategy must detect the arbitrage.

### Task 5.2: Order Construction & Gas Awareness
Prepare for live execution by constructing valid trade payloads.

*   **File:** `pkg/execution/trade_builder.go` (New File)
*   **Action:**
    *   Create `BuildTrade(opportunity types.ArbitrageOpportunity) ([]OrderPayload, error)`.
    *   Include logic to batch orders if necessary.
    *   Calculate "Gas/Transaction Cost" estimate and subtract from `NetProfit`.
*   **Verification:** Dry-run the trade builder against the Polymarket testnet (if available) or mock API. Ensure orders are signed correctly.

---

## Summary of Architecture Changes

```
predmarket-scanner/
├── pkg/
│   ├── simulation/       <-- NEW: Monte Carlo & Pricing Models
│   ├── backtest/         <-- NEW: Time-series engine & Metrics
│   ├── strategies/
│   │   ├── interface.go  <-- NEW: Strategy Plugin System
│   │   ├── dutch_book.go
│   │   ├── model_mispricing.go <-- NEW: Model-based Arb
│   │   └── combinatorial.go    <-- NEW: Logic Arb
│   ├── execution/        <-- NEW: Order construction
│   └── types/
│       └── market.go     <-- MODIFIED: Rich data fields
└── cmd/
    └── backtest.go       <-- NEW: CLI command for rigorous backtesting
```
