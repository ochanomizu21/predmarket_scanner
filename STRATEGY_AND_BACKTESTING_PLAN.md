# Strategy & Backtesting Implementation Plan

This document outlines the implementation plan for adding the **Multi-Outcome Arbitrage Strategy** and a **Historical Backtesting Mode** backed by **SQLite**.

## 1. Multi-Outcome Arbitrage Strategy

**Concept:**
Extend the current Dutch Book logic (which only handles binary YES/NO markets) to N-outcome markets. If the sum of all mutually exclusive "YES" outcomes in a market is less than `1.0`, buying one share of each outcome guarantees a payout of `1.0` with a cost less than `1.0`.

**Implementation Steps:**
1. **Market Filtering:**
   - Modify the market filtering logic to include non-binary markets (markets with `> 2` outcomes).
   - Exclude combinatorial or loosely correlated markets unless they are strictly mutually exclusive.
2. **Gross Profit Calculation:**
   - Calculate the sum of the best ask prices for all "YES" tokens in the market.
   - If `sum(YES asks) < 1.0`, a potential arbitrage opportunity exists.
3. **Slippage & Net Profit Calculation:**
   - Fetch the order book for all `N` tokens.
   - For a given execution size `$S`, simulate buying `$S` worth of shares for *each* outcome token.
   - Calculate the average execution price and slippage for each token.
   - Deduct Polymarket trading fees (2% on winnings).
   - If the net cost to guarantee a `$1.00` payout is still `< 1.0`, it's a valid opportunity.
4. **Integration:**
   - Add a new `StrategyType` (e.g., `MultiOutcomeDutchBook`).
   - Update the CLI `scan` command to run both binary and multi-outcome strategies (or allow users to filter via a `--strategy` flag).

---

## 2. Historical Data Recording (SQLite)

To support backtesting, we need to record market states and order book snapshots over time.

**Implementation Steps:**
1. **SQLite Setup:**
   - Use `github.com/mattn/go-sqlite3` to interface with a local SQLite database (e.g., `data/history.db`).
2. **Database Schema:**
   - `markets`: ID, Question, EndTime, etc.
   - `snapshots`: ID, MarketID, Timestamp.
   - `outcomes_snapshot`: SnapshotID, OutcomeID, BestBid, BestAsk.
   - `order_book_levels`: SnapshotID, OutcomeID, Side (Bid/Ask), Price, Size.
3. **Recording Command (`predmarket-scanner record`):**
   - Create a new long-running CLI command that acts as a daemon.
   - Periodically fetches the top `N` most liquid or volatile markets.
   - Saves their metadata and full order books into the SQLite database.
   - *Optimization:* Only insert new order book levels if the top of the book or depth has changed significantly to save space.

---

## 3. Backtesting Mode

The backtesting mode will allow the user to run the scanner against historical data stored in the SQLite database to find opportunities that *would have* worked at that specific point in time.

**Implementation Steps:**
1. **Data Source Abstraction:**
   - Refactor the scanner to accept an interface for market and order book fetching (e.g., `DataProvider`).
   - Create two implementations:
     - `LiveDataProvider` (current behavior, hits Polymarket API).
     - `HistoricalDataProvider` (queries the local SQLite database).
2. **CLI Integration:**
   - Add new flags to the `scan` command:
     - `--historical`: Enables backtesting mode.
     - `--time "YYYY-MM-DD HH:MM:SS"`: Specifies the target historical timestamp.
     - `--time-range "start,end"`: Alternatively, sweep through a range of historical snapshots.
3. **Historical Execution Logic:**
   - When `--historical` is used, the scanner will look up the closest order book snapshot in SQLite for the specified time.
   - Run the exact same arbitrage detection and slippage calculation logic on the historical order book.
   - Output the results indicating what the profit *would have been* based on the recorded market depth.

---

## 4. Suggested Execution Order

1. **Phase A (Multi-Outcome Live):** Implement and test the Multi-Outcome Arbitrage Strategy against the live Polymarket API.
2. **Phase B (Data Recording):** Set up SQLite, create the schema, and implement the `record` command to start gathering historical data immediately.
3. **Phase C (Backtesting Engine):** Abstract the data fetching logic, implement the `HistoricalDataProvider`, and add the `--historical` flags to the `scan` command to query the collected SQLite data.