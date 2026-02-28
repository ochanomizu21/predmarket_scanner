package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/providers"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	d := &DB{DB: db}
	if err := d.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return d, nil
}

func (d *DB) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS markets (
		id TEXT PRIMARY KEY,
		question TEXT NOT NULL,
		end_time TEXT,
		liquidity REAL,
		volume REAL,
		num_outcomes INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		market_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (market_id) REFERENCES markets(id)
	);

	CREATE TABLE IF NOT EXISTS outcomes_snapshot (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		snapshot_id INTEGER NOT NULL,
		outcome_name TEXT NOT NULL,
		best_bid REAL,
		best_ask REAL,
		FOREIGN KEY (snapshot_id) REFERENCES snapshots(id)
	);

	CREATE TABLE IF NOT EXISTS order_book_levels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		snapshot_id INTEGER NOT NULL,
		outcome_name TEXT NOT NULL,
		token_id TEXT NOT NULL,
		side TEXT NOT NULL,
		price REAL NOT NULL,
		size REAL NOT NULL,
		FOREIGN KEY (snapshot_id) REFERENCES snapshots(id)
	);

	CREATE TABLE IF NOT EXISTS price_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		market_id TEXT NOT NULL,
		outcome_name TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		price REAL NOT NULL,
		token_id TEXT,
		order_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (market_id) REFERENCES markets(id),
		UNIQUE(market_id, outcome_name, timestamp)
	);

	CREATE INDEX IF NOT EXISTS idx_snapshots_market_timestamp ON snapshots(market_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_order_book_snapshot_outcome ON order_book_levels(snapshot_id, outcome_name);
	CREATE INDEX IF NOT EXISTS idx_order_book_outcome_side ON order_book_levels(outcome_name, side);
	CREATE INDEX IF NOT EXISTS idx_price_history_market ON price_history(market_id);
	CREATE INDEX IF NOT EXISTS idx_price_history_market_outcome ON price_history(market_id, outcome_name);
	CREATE INDEX IF NOT EXISTS idx_price_history_timestamp ON price_history(timestamp);
	`

	_, err := d.Exec(schema)
	if err != nil {
		return err
	}

	_, err = d.Exec(`ALTER TABLE order_book_levels ADD COLUMN token_id TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	return nil
}

func (d *DB) InsertOrUpdateMarket(marketID, question string, endTime *time.Time, liquidity, volume float64, numOutcomes int) error {
	var endTimeStr string
	if endTime != nil {
		endTimeStr = endTime.Format(time.RFC3339)
	}

	query := `
	INSERT INTO markets (id, question, end_time, liquidity, volume, num_outcomes, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO UPDATE SET
		question = excluded.question,
		end_time = excluded.end_time,
		liquidity = excluded.liquidity,
		volume = excluded.volume,
		num_outcomes = excluded.num_outcomes,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := d.Exec(query, marketID, question, endTimeStr, liquidity, volume, numOutcomes)
	return err
}

func (d *DB) InsertSnapshot(marketID string, timestamp time.Time) (int64, error) {
	result, err := d.Exec(`
		INSERT INTO snapshots (market_id, timestamp)
		VALUES (?, ?)
	`, marketID, timestamp.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) InsertOutcomeSnapshot(snapshotID int64, outcomeName string, bestBid, bestAsk float64) error {
	_, err := d.Exec(`
		INSERT INTO outcomes_snapshot (snapshot_id, outcome_name, best_bid, best_ask)
		VALUES (?, ?, ?, ?)
	`, snapshotID, outcomeName, bestBid, bestAsk)
	return err
}

func (d *DB) InsertOrderBookLevels(snapshotID int64, outcomeName, tokenID, side string, levels []providers.OrderBookLevel) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO order_book_levels (snapshot_id, outcome_name, token_id, side, price, size)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, level := range levels {
		_, err := stmt.Exec(snapshotID, outcomeName, tokenID, side, level.Price, level.Size)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) GetLatestSnapshot(marketID string, before time.Time) (*providers.SnapshotData, error) {
	query := `
		SELECT s.id, s.market_id, s.timestamp
		FROM snapshots s
		WHERE s.market_id = ? AND s.timestamp <= ?
		ORDER BY s.timestamp DESC
		LIMIT 1
	`

	var s providers.SnapshotData
	var timestampStr string
	err := d.QueryRow(query, marketID, before.Format(time.RFC3339)).Scan(&s.ID, &s.MarketID, &timestampStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	s.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (d *DB) GetLatestSnapshotByTokenID(tokenID string, before time.Time) (*providers.SnapshotData, error) {
	query := `
		SELECT s.id, s.market_id, s.timestamp
		FROM snapshots s
		JOIN order_book_levels obl ON s.id = obl.snapshot_id
		WHERE obl.token_id = ? AND s.timestamp <= ?
		ORDER BY s.timestamp DESC
		LIMIT 1
	`

	var s providers.SnapshotData
	var timestampStr string
	err := d.QueryRow(query, tokenID, before.Format(time.RFC3339)).Scan(&s.ID, &s.MarketID, &timestampStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	s.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (d *DB) GetTimestampsInRange(startTime, endTime time.Time) ([]time.Time, error) {
	query := `
		SELECT DISTINCT timestamp
		FROM snapshots
		WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp
	`

	rows, err := d.Query(query, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var tsStr string
		if err := rows.Scan(&tsStr); err != nil {
			return nil, err
		}
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			continue
		}
		timestamps = append(timestamps, ts)
	}

	return timestamps, nil
}

func (d *DB) GetSnapshotData(snapshotID int64) (*providers.SnapshotDetail, error) {
	query := `
		SELECT s.id, s.market_id, s.timestamp, m.question, m.liquidity, m.volume
		FROM snapshots s
		JOIN markets m ON s.market_id = m.id
		WHERE s.id = ?
	`

	var s providers.SnapshotDetail
	var timestampStr string
	err := d.QueryRow(query, snapshotID).Scan(&s.ID, &s.MarketID, &timestampStr, &s.Question, &s.Liquidity, &s.Volume)
	if err != nil {
		return nil, err
	}

	s.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, err
	}

	rows, err := d.Query(`
		SELECT outcome_name, best_bid, best_ask
		FROM outcomes_snapshot
		WHERE snapshot_id = ?
	`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o providers.OutcomeData
		if err := rows.Scan(&o.Name, &o.BestBid, &o.BestAsk); err != nil {
			return nil, err
		}
		s.Outcomes = append(s.Outcomes, o)
	}

	return &s, nil
}

func (d *DB) GetOrderBookLevels(snapshotID int64, tokenID, side string) ([]providers.OrderBookLevel, error) {
	query := `
		SELECT price, size
		FROM order_book_levels
		WHERE snapshot_id = ? AND token_id = ? AND side = ?
		ORDER BY CASE WHEN side = 'bid' THEN price END DESC,
		         CASE WHEN side = 'ask' THEN price END ASC
	`

	rows, err := d.Query(query, snapshotID, tokenID, side)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var levels []providers.OrderBookLevel
	for rows.Next() {
		var l providers.OrderBookLevel
		if err := rows.Scan(&l.Price, &l.Size); err != nil {
			return nil, err
		}
		levels = append(levels, l)
	}

	return levels, nil
}

func (d *DB) InsertPriceHistory(marketID, outcomeName string, historyPoints []providers.PriceHistoryPoint) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO price_history (market_id, outcome_name, timestamp, price, token_id, order_count)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(market_id, outcome_name, timestamp) DO UPDATE SET
			price = excluded.price,
			order_count = excluded.order_count
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, point := range historyPoints {
		_, err := stmt.Exec(
			marketID,
			outcomeName,
			point.Timestamp,
			point.Price,
			point.TokenID,
			point.OrderCount,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) GetPriceHistory(marketID, outcomeName string, startTime, endTime time.Time) ([]providers.PriceHistoryPoint, error) {
	query := `
		SELECT timestamp, price, token_id, order_count
		FROM price_history
		WHERE market_id = ? AND outcome_name = ?
	`

	args := []interface{}{marketID, outcomeName}

	if !startTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, startTime.Format(time.RFC3339))
	}

	if !endTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, endTime.Format(time.RFC3339))
	}

	query += " ORDER BY timestamp ASC"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []providers.PriceHistoryPoint
	for rows.Next() {
		var point providers.PriceHistoryPoint
		var timestampStr string
		if err := rows.Scan(&timestampStr, &point.Price, &point.TokenID, &point.OrderCount); err != nil {
			return nil, err
		}

		point.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil, err
		}

		points = append(points, point)
	}

	return points, nil
}

type SnapshotDetail struct {
	ID        int64
	MarketID  string
	Timestamp time.Time
	Question  string
	Liquidity float64
	Volume    float64
	Outcomes  []OutcomeData
}

type OutcomeData struct {
	Name    string
	BestBid float64
	BestAsk float64
}

func (d *DB) FetchMarketsAtTime(targetTime time.Time, maxMarkets int) ([]providers.MarketData, error) {
	query := `
		SELECT DISTINCT m.id, m.question, m.end_time, m.liquidity, m.volume
		FROM markets m
		JOIN snapshots s ON s.market_id = m.id
		WHERE s.timestamp <= ?
		ORDER BY m.liquidity DESC
		LIMIT ?
	`

	rows, err := d.Query(query, targetTime.Format(time.RFC3339), maxMarkets)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []providers.MarketData
	for rows.Next() {
		var m providers.MarketData
		var endTimeStr string

		if err := rows.Scan(&m.ID, &m.Question, &endTimeStr, &m.Liquidity, &m.Volume); err != nil {
			return nil, err
		}

		if endTimeStr != "" {
			endTime, err := time.Parse(time.RFC3339, endTimeStr)
			if err == nil {
				m.EndTime = &endTime
			}
		}

		markets = append(markets, m)
	}

	return markets, nil
}
