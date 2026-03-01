package storage

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xitongsys/parquet-go/source"
	"github.com/xitongsys/parquet-go/writer"
)

type LocalFile struct {
	*os.File
}

func (lf *LocalFile) Create(name string) (source.ParquetFile, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return &LocalFile{File: f}, nil
}

func (lf *LocalFile) Open(name string) (source.ParquetFile, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &LocalFile{File: f}, nil
}

type ParquetRecord struct {
	EventType   string `parquet:"name=event_type, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	AssetID     string `parquet:"name=asset_id, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Market      string `parquet:"name=market, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	Timestamp   int64  `parquet:"name=timestamp, type=INT64"`
	Hash        string `parquet:"name=hash, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	BidsJSON    string `parquet:"name=bids_json, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	AsksJSON    string `parquet:"name=asks_json, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
	ChangesJSON string `parquet:"name=price_changes_json, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN"`
}

type rawMessage struct {
	EventType    string                 `json:"event_type"`
	AssetID      string                 `json:"asset_id"`
	Market       string                 `json:"market"`
	Timestamp    string                 `json:"timestamp"`
	Hash         string                 `json:"hash"`
	Bids         []map[string]string    `json:"bids,omitempty"`
	Asks         []map[string]string    `json:"asks,omitempty"`
	PriceChanges []map[string]string    `json:"price_changes,omitempty"`
	Extra        map[string]interface{} `json:"-"`
}

type ParquetConverter struct {
	dataDir string
}

func NewParquetConverter(dataDir string) *ParquetConverter {
	return &ParquetConverter{
		dataDir: dataDir,
	}
}

func (pc *ParquetConverter) ConvertDay(date string) error {
	ctx := context.Background()
	return pc.ConvertDayWithContext(ctx, date)
}

func (pc *ParquetConverter) ConvertDayWithContext(ctx context.Context, date string) error {
	jsonlPath := filepath.Join(pc.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", date))
	parquetPath := filepath.Join(pc.dataDir, fmt.Sprintf("market_data_%s.parquet", date))

	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		return fmt.Errorf("JSONL file not found: %s", jsonlPath)
	}

	file, err := os.Open(jsonlPath)
	if err != nil {
		return fmt.Errorf("opening JSONL file: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("opening gzip reader: %w", err)
	}
	defer gz.Close()

	fw, err := os.Create(parquetPath)
	if err != nil {
		return fmt.Errorf("creating parquet file: %w", err)
	}
	defer fw.Close()

	lf := &LocalFile{File: fw}

	pw, err := writer.NewParquetWriter(lf, new(ParquetRecord), int64(len(jsonlPath)))
	if err != nil {
		return fmt.Errorf("creating parquet writer: %w", err)
	}
	defer pw.WriteStop()

	scanner := bufio.NewScanner(gz)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	recordCount := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var raw rawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		timestamp := parseTimestamp(raw.Timestamp)

		var bidsJSON, asksJSON, changesJSON string
		if raw.Bids != nil {
			bidBytes, _ := json.Marshal(raw.Bids)
			bidsJSON = string(bidBytes)
		}
		if raw.Asks != nil {
			askBytes, _ := json.Marshal(raw.Asks)
			asksJSON = string(askBytes)
		}
		if raw.PriceChanges != nil {
			changeBytes, _ := json.Marshal(raw.PriceChanges)
			changesJSON = string(changeBytes)
		}

		record := &ParquetRecord{
			EventType:   raw.EventType,
			AssetID:     raw.AssetID,
			Market:      raw.Market,
			Timestamp:   timestamp,
			Hash:        raw.Hash,
			BidsJSON:    bidsJSON,
			AsksJSON:    asksJSON,
			ChangesJSON: changesJSON,
		}

		if err := pw.Write(record); err != nil {
			return fmt.Errorf("writing parquet record: %w", err)
		}

		recordCount++
		if recordCount%10000 == 0 {
			fmt.Printf("Converted %d records...\n", recordCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning JSONL file: %w", err)
	}

	if err := fw.Sync(); err != nil {
		return fmt.Errorf("syncing parquet file: %w", err)
	}

	jsonlInfo, _ := os.Stat(jsonlPath)
	parquetInfo, _ := os.Stat(parquetPath)
	compressionRatio := float64(1.0)
	if jsonlInfo != nil && parquetInfo != nil {
		compressionRatio = float64(jsonlInfo.Size()) / float64(parquetInfo.Size())
	}

	fmt.Printf("Conversion complete: %d records\n", recordCount)
	fmt.Printf("Original: %d bytes, Compressed: %d bytes (ratio: %.1fx)\n",
		jsonlInfo.Size(), parquetInfo.Size(), compressionRatio)

	return nil
}


func (pc *ParquetConverter) ConvertAllAvailableDays() error {
	ctx := context.Background()
	return pc.ConvertAllAvailableDaysWithContext(ctx, false)
}

func (pc *ParquetConverter) ConvertAllAvailableDaysAndDelete() error {
	ctx := context.Background()
	return pc.ConvertAllAvailableDaysWithContext(ctx, true)
}

func (pc *ParquetConverter) ConvertAllAvailableDaysWithContext(ctx context.Context, deleteAfter bool) error {
	entries, err := os.ReadDir(pc.dataDir)
	if err != nil {
		return fmt.Errorf("reading data directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) < 18 || name[:13] != "market_data_" || name[len(name)-9:] != ".jsonl.gz" {
			continue
		}

		dateStr := name[13 : len(name)-9]
		parquetPath := filepath.Join(pc.dataDir, fmt.Sprintf("market_data_%s.parquet", dateStr))

		if _, err := os.Stat(parquetPath); err == nil {
			fmt.Printf("Skipping %s - parquet already exists\n", dateStr)
			continue
		}

		fmt.Printf("Converting %s...\n", dateStr)
		if err := pc.ConvertDayWithContext(ctx, dateStr); err != nil {
			fmt.Printf("Error converting %s: %v\n", dateStr, err)
			continue
		}

		if deleteAfter {
			if err := pc.DeleteJSONL(dateStr); err != nil {
				fmt.Printf("Warning: failed to delete JSONL for %s: %v\n", dateStr, err)
			}
		}
	}

	return nil
}

func (pc *ParquetConverter) DeleteJSONL(date string) error {
	jsonlPath := filepath.Join(pc.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", date))

	if err := os.Remove(jsonlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting JSONL file: %w", err)
	}

	return nil
}

func parseTimestamp(s string) int64 {
	var ts int64
	_, err := fmt.Sscanf(s, "%d", &ts)
	if err != nil {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			return t.UnixNano() / 1e6
		}
	}
	return ts
}
