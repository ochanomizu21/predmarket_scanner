Implementation Plan: WebSocket Integration & Storage Overhaul
Objective: Upgrade predmarket_scanner from a synchronous REST-polling architecture to an asynchronous, real-time WebSocket pipeline. This will enable zero-latency arbitrage scanning and highly efficient tick-level historical data recording.

Phase 1: Architecture Mapping & Setup
[ ] Analyze Existing Structs: Review the internal/models or pkg/ directories to understand how the current scanner represents Orders, Bids, Asks, and Market Snapshots.

[ ] Define WebSocket Structs: Create new Go structs representing Polymarket's CLOB WebSocket messages. You will need structs for:

SubscriptionRequest (to send to the WS server)

SnapshotMessage (the initial full book received upon connection)

DeltaMessage (tick-by-tick updates with size and price changes)

[ ] Add Dependencies: Add a robust WebSocket library for Go, such as github.com/gorilla/websocket, and a Parquet writer like github.com/xitongsys/parquet-go if upgrading the storage layer.

Phase 2: WebSocket Client Implementation
[ ] Initial Market Fetch: Keep a single REST call at startup to fetch all active asset_ids (markets) you want to track.

[ ] Connection Manager: Create a ws_client.go service that connects to Polymarket's WSS endpoint (e.g., wss://ws-subscriptions-clob.polymarket.com/ws/market).

[ ] Subscription Logic: Implement a function to send subscription payloads for the fetched asset_ids. Note: If tracking 500+ markets, you may need to batch subscriptions or open multiple WebSocket connections to avoid payload limits.

[ ] Ping/Pong & Reconnection: Implement connection health checks. If the websocket drops or stops receiving pongs, automatically trigger a reconnection and re-subscribe protocol.

Phase 3: In-Memory Order Book (State Management)
Since the scanner needs to read the book while the WS updates it, concurrency control is critical.

[ ] Create the OrderBook Map: Implement an in-memory map[float64]float64 (Price -> Size) for Bids and Asks per market.

[ ] Implement sync.RWMutex: Wrap the maps in a Read-Write Mutex.

Write Lock (Lock()): Used by the WebSocket listener when applying deltas.

Read Lock (RLock()): Used by the arbitrage scanner when calculating slippage.

[ ] Delta Processing Logic: - On SnapshotMessage: Overwrite the existing in-memory map.

On DeltaMessage: If size > 0, update/insert the price level. If size == 0, delete() the price level from the map.

Phase 4: Overhauling the Storage Engine (Recording)
Move away from spamming full snapshots into SQLite.

[ ] Decouple the Scanner from SQLite: Update the scan command so it reads from the new In-Memory Order Book rather than querying SQLite.

[ ] Implement Append-Only Logging (Phase A): Create a background goroutine that listens to a Go channel. Have the WebSocket client push raw JSON messages into this channel. The background worker simply appends these messages to a daily market_data_YYYYMMDD.jsonl file.

[ ] Implement Parquet Batching (Phase B - Optional but Recommended): Write a nightly cron job (or an end-of-day trigger in the Go app) that reads the .jsonl file, compresses it into Apache Parquet format, and deletes the raw JSON file.

Phase 5: CLI & Daemon Integration
[ ] Update cmd/record.go: Modify the record command. Remove the --interval 30 flag. The command should now initialize the WebSocket client and the appending file logger.

[ ] Update cmd/scan.go: Modify the live scanner so it runs concurrently with the WebSocket listener, evaluating the In-Memory Order Book on a fast loop (e.g., every 1 second, or purely event-driven whenever a delta changes the top-of-book).

[ ] Update Historical Backtesting: Modify the historical scan logic to read from your new .jsonl or .parquet files instead of SQLite.

Phase 6: Testing & Validation
[ ] Reconciliation Check: Write a test that runs the WebSocket delta stream for 10 minutes, then compares your calculated In-Memory Order Book against a fresh REST API snapshot. They must match exactly.

[ ] Load Testing: Subscribe to 1,000+ Polymarket assets simultaneously to ensure your Go channels and Mutex locks don't become bottlenecks.

[ ] Graceful Shutdown: Ensure os.Interrupt (Ctrl+C) flushes any remaining data in memory to disk before exiting.
