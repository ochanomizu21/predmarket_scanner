package storage

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type JSONLLogger struct {
	dataDir       string
	currentDate   string
	file          *os.File
	gzipWriter    *gzip.Writer
	writer        *bufio.Writer
	messageChan   chan []byte
	done          chan struct{}
	mu            sync.Mutex
	droppedCount  int64
	writtenCount  int64
	lastStatsTime time.Time
	attemptCount  int64
	receivedCount int64
}

type LogMessage struct {
	Timestamp time.Time              `json:"timestamp"`
	Message   map[string]interface{} `json:"message"`
}

func NewJSONLLogger(dataDir string) *JSONLLogger {
	return &JSONLLogger{
		dataDir:     dataDir,
		messageChan: make(chan []byte, 10000),
		done:        make(chan struct{}),
	}
}

func (l *JSONLLogger) Start(ctx context.Context) error {
	if err := os.MkdirAll(l.dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	if err := l.rotateFile(); err != nil {
		return fmt.Errorf("initializing log file: %w", err)
	}

	go l.processMessages(ctx)
	go l.dailyRotation(ctx)
	go l.periodicRotation(ctx)
	go l.printStats(ctx)

	return nil
}

func (l *JSONLLogger) Stop() {
	close(l.done)

	time.Sleep(200 * time.Millisecond)

	l.mu.Lock()
	if l.writer != nil {
		l.writer.Flush()
		l.writer = nil
	}
	l.mu.Unlock()

	// Close gzip with timeout to avoid blocking
	if l.gzipWriter != nil {
		done := make(chan error, 1)
		go func() {
			done <- l.gzipWriter.Close()
		}()

		select {
		case <-done:
			// Gzip closed successfully
		case <-time.After(2 * time.Second):
			// Timeout - skip close, file may be partial
		}
		l.gzipWriter = nil
	}

	l.mu.Lock()
	if l.file != nil {
		l.file.Sync()
		l.file.Close()
		l.file = nil
	}
	l.mu.Unlock()

	log.Printf("Logger stopped. Total written: %d\n", l.writtenCount)
}

func (l *JSONLLogger) Log(message map[string]interface{}) error {
	message["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	select {
	case l.messageChan <- data:
		return nil
	default:
		return fmt.Errorf("message channel full, dropping message")
	}
}

func (l *JSONLLogger) LogRaw(data []byte) error {
	l.mu.Lock()
	l.attemptCount++
	l.mu.Unlock()

	select {
	case l.messageChan <- data:
		return nil
	default:
		l.mu.Lock()
		l.droppedCount++
		l.mu.Unlock()
		fmt.Printf("Logger message channel full, dropping message of %d bytes (total dropped: %d)\n", len(data), l.droppedCount)
		return fmt.Errorf("message channel full, dropping message")
	}
}

func (l *JSONLLogger) processMessages(ctx context.Context) {
	messageCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		case data := <-l.messageChan:
			l.mu.Lock()
			l.receivedCount++
			l.mu.Unlock()
			l.mu.Lock()
			if l.gzipWriter != nil {
				lineToWrite := append(data, '\n')
				if _, err := l.gzipWriter.Write(lineToWrite); err != nil {
					log.Printf("Write error: %v\n", err)
				} else {
					messageCount++
					l.writtenCount++
					if messageCount%100 == 0 {
						if l.gzipWriter != nil {
							l.gzipWriter.Flush()
						}
						if l.file != nil {
							l.file.Sync()
						}
					}
				}
			}
			l.mu.Unlock()
		}
	}
}

func (l *JSONLLogger) dailyRotation(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		case <-ticker.C:
			currentDate := time.Now().UTC().Format("2006-01-02")
			if currentDate != l.currentDate {
				l.mu.Lock()
				l.rotateFileLocked()
				l.mu.Unlock()
			}
		}
	}
}

func (l *JSONLLogger) periodicRotation(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		case <-ticker.C:
			l.mu.Lock()
			// Close current gzip and start new one to ensure valid files
			if l.gzipWriter != nil {
				l.gzipWriter.Close()
				l.gzipWriter = nil
			}
			if l.file != nil {
				l.file.Sync()
				l.file.Close()
				l.file = nil
			}
			// Reopen same file (append mode)
			currentDate := time.Now().UTC().Format("2006-01-02")
			filename := filepath.Join(l.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", currentDate))
			file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				l.file = file
				l.gzipWriter = gzip.NewWriter(file)
				log.Printf("Rotated gzip file for valid file completion")
			}
			l.mu.Unlock()
		}
	}
}

func (l *JSONLLogger) printStats(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		case <-ticker.C:
			l.mu.Lock()
			channelLen := len(l.messageChan)
			attempts := l.attemptCount
			written := l.writtenCount
			dropped := l.droppedCount
			received := l.receivedCount
			l.mu.Unlock()

			successRate := float64(0)
			receiveRate := float64(0)
			if attempts > 0 {
				successRate = float64(written) / float64(attempts) * 100
			}
			if attempts > 0 {
				receiveRate = float64(received) / float64(attempts) * 100
			}

			fmt.Printf("Logger Stats - Attempts: %d | Received: %d (%.1f%%) | Written: %d (%.1f%%) | Dropped: %d | Channel: %d/10000 | File: %s\n",
				attempts, received, receiveRate, written, successRate, dropped, channelLen, l.GetCurrentFilename())
		}
	}
}

func (l *JSONLLogger) rotateFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rotateFileLocked()
}

func (l *JSONLLogger) rotateFileLocked() error {
	if l.writer != nil {
		l.writer.Flush()
		l.writer = nil
	}
	if l.gzipWriter != nil {
		l.gzipWriter.Close()
		l.gzipWriter = nil
	}
	if l.file != nil {
		l.file.Sync()
		l.file.Close()
		l.file = nil
	}

	currentDate := time.Now().UTC().Format("2006-01-02")
	filename := filepath.Join(l.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", currentDate))

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	l.file = file
	l.gzipWriter = gzip.NewWriter(file)
	// Don't use bufio - write directly to gzip for proper closing
	l.writer = nil
	l.currentDate = currentDate

	return nil
}

func (l *JSONLLogger) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer != nil {
		l.writer.Flush()
	}
	if l.gzipWriter != nil {
		l.gzipWriter.Flush()
	}
	if l.file != nil {
		l.file.Sync()
	}
}

func (l *JSONLLogger) GetCurrentFilename() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentDate == "" {
		return ""
	}
	return filepath.Join(l.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", l.currentDate))
}
