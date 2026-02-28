# Arbitrage Strategies Reference

This document provides a comprehensive guide to the arbitrage strategies for prediction markets. It combines theoretical research with practical implementation details for the `predmarket-scanner`.

---

## 1. Same-Market Arbitrage (Single Platform)
These are the simplest, most “textbook” arbitrages: all the logic sits within one market on one platform.

### 1.1 Single-Condition Dutch-Book Arbitrage (Binary YES/NO) [✅ IMPLEMENTED]
**What it is**
In a binary market, “YES” and “NO” should sum to 1 (or very close). When `YES price + NO price < 1`, you can buy both sides and lock in a riskless profit.

**Example**
- YES @ 0.50, NO @ 0.47 → sum = 0.97
- Buy 1 YES and 1 NO for 0.97; at resolution you receive 1.
- Profit = 0.03 (≈3.1%).

**Implementation Notes**
- **File**: `pkg/strategies/dutch_book.go`
- **Logic**: Filters for binary markets, calculates `1.0 - (YES + NO)`, and performs slippage simulation on both order books via the CLOB API.
- **Fees**: Accounts for Polymarket's 2% fee on winnings.
- **Competition**: Extreme. On Polymarket, windows often close in ~200ms and are dominated by HFT bots.

### 1.2 Multi-Condition Dutch-Book Arbitrage (Multi-Outcome) [✅ IMPLEMENTED]
**What it is**
Many markets have more than two mutually exclusive outcomes (e.g., “Trump / Biden / Other wins election”). The prices of all YES contracts should sum to 1. When the sum of all YES prices < 1, you can perform a "Long Arbitrage" by buying all YES contracts.

**Example (Long)**
- YES-Trump = 0.40, YES-Biden = 0.35, YES-Other = 0.20
- Sum = 0.95 < 1
- Buy one of each YES for 0.95; you are guaranteed 1 at resolution.
- Profit = 0.05 (5.3%).

**Implementation Notes**
- **File**: `pkg/strategies/multi_outcome.go`
- **Logic**: Sums the best asks for all outcomes. If `< 1.0`, it fetches order books for all outcomes and simulates a multi-leg execution.
- **Slippage**: Calculates weighted average execution price for all legs independently.

### 1.3 Multi-Outcome NO-Basket “Negative Risk” Arbitrage [📋 PLANNED]
**What it is**
In an N-outcome market, the NO contracts should sum to N−1. If `sum(NO) < N-1`, you can construct a basket of NO bets that hedges principal risk and yields deterministic profit in most scenarios.

**Intuition**
- Each NO contract pays 1 if that outcome does not occur.
- If the market misprices the total probability of “something else happening”, buying a set of NO contracts can lead to an over-payment for risk.

---

## 2. Cross-Market and Cross-Platform Arbitrage
Here you exploit mispricings between markets or platforms.

### 2.1 Identical Event, Different Platforms [📋 PLANNED]
**What it is**
The same event trades on multiple platforms (Polymarket, Kalshi, etc.) at different implied probabilities. You buy the underpriced side on one platform and the opposite side on the other, locking in a spread.

**Example**
- Polymarket YES @ 0.45
- Kalshi NO @ 0.52
- Cost: 0.45 + 0.52 = 0.97 → payout = 1 → profit ≈ 3.1% (before fees).

### 2.2 Identical Contract, Different Exchanges [📋 PLANNED]
**What it is**
The same contract (same question, same resolution rules) trades at different prices on different platforms. This is essentially "ETF arbitrage" for event contracts.

### 2.3 Logically Related Markets (Same Platform) [📋 PLANNED]
**What it is**
Exploiting mispricings between logically related contracts on the same exchange (e.g., “Candidate X wins presidency” vs “Party Y wins presidency”).

### 2.4 Combinatorial Arbitrage (Strict Dependencies) [📋 PLANNED]
**What it is**
Exploiting strict dependencies between markets.
**Example**: Market M1 (D wins presidency) @ 0.48 + Market M2 (All R-margin buckets) @ 0.40. Total cost = 0.88, payout = 1.

---

## 3. Time-Dynamic and Microstructure Strategies

### 3.1 Information-Lag Arbitrage [📋 PLANNED]
Trading on faster information sources (e.g., live TV feed, official API) before the prediction market's order book updates.

### 3.2 Volatility / Panic-Driven Rebalancing [📋 PLANNED]
During big news, participants trade emotionally. This pushes `YES+NO` away from 1 for short periods. Automated strategies buy both sides when the spread opens.

### 3.3 End-of-Day / End-of-Event “Sweep” Strategies [📋 PLANNED]
Near market close, the outcome is almost certain but the price hasn’t fully converged. Buying the heavily favored outcome when its probability is very high but not 1.

### 3.4 Market-Making and Spread Capture [📋 PLANNED]
Placing limit orders on both sides in low-liquidity markets to capture the wide bid-ask spread.

---

## 4. Advanced / Hybrid Strategies

### 4.1 Cross-Asset Arbitrage vs Options / Betting / Crypto Derivatives [📋 PLANNED]
Arbitraging prediction market probabilities against options markets, sports betting odds, or crypto perpetual/futures pricing.

### 4.2 Structural / Rule-Based Arbitrage [📋 PLANNED]
Exploiting differences in platform rules, oracles, or settlement mechanisms (e.g., how different exchanges handle a composite index vs. a single exchange price).

### 4.3 Manipulation-Adjacent Strategies [❌ NOT PLANNED]
Using large orders on primary exchanges to push spot prices and force a prediction market to resolve in your favor. This sits in a gray/illegal zone and is not a target for this tool.

---

## 5. Strategy Comparison Matrix

| Layer | Strategy Type | Risk Profile | Status |
| :--- | :--- | :--- | :--- |
| **Same-Market** | Dutch-Book (Binary) | Very Low | ✅ Implemented |
| **Same-Market** | Multi-Outcome | Low | ✅ Implemented |
| **Same-Market** | NO-Basket | Low | 📋 Planned |
| **Cross-Platform** | Identical Event | Low (Settlement Risk) | 📋 Planned |
| **Cross-Market** | Logically Related | Medium | 📋 Planned |
| **Time-Dynamic** | Information-Lag | Low-Medium | 📋 Planned |
| **Market-Making**| Spread Capture | Medium (Inventory Risk)| 📋 Planned |
| **Advanced** | Cross-Asset | Medium | 📋 Planned |

---

## 6. Scoring & Risk Management

Opportunities are ranked using a multi-factor **Risk-Adjusted Score (0-1)**:

1.  **Profit Score (40%)**: Sigmoid function on net profit margin.
2.  **Liquidity Score (25%)**: Log-normalized market liquidity.
3.  **Volume Score (15%)**: Log-normalized 24h trading volume.
4.  **Execution Risk (15%)**: Based on order book depth and spread width.
5.  **Time Decay (5%)**: Based on the remaining time until market resolution.

### Risk Mitigation
- **Slippage Simulation**: We never assume the "best price." Every opportunity is re-calculated by simulating the actual execution size against the full order book depth.
- **Minimum Profit Threshold**: Filters out "micro-arbs" that might be erased by network latency or price movements during execution.
- **Fee Awareness**: Automatically deducts Polymarket's 2% fee on winnings before calculating net profit.
