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
	dataDir     string
	currentDate string
	file        *os.File
	gzipWriter  *gzip.Writer
	writer      *bufio.Writer
	messageChan chan []byte
	done        chan struct{}
	mu          sync.Mutex
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

	return nil
}

func (l *JSONLLogger) Stop() {
	close(l.done)
	l.mu.Lock()
	if l.writer != nil {
		l.writer.Flush()
	}
	if l.gzipWriter != nil {
		l.gzipWriter.Close()
	}
	if l.file != nil {
		l.file.Close()
	}
	l.mu.Unlock()
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
	select {
	case l.messageChan <- data:
		return nil
	default:
		fmt.Printf("Logger message channel full, dropping message of %d bytes\n", len(data))
		return fmt.Errorf("message channel full, dropping message")
	}
}

func (l *JSONLLogger) processMessages(ctx context.Context) {
	messageCount := 0
	for {
		select {
		case <-ctx.Done():
			l.Stop()
			return
		case <-l.done:
			l.Stop()
			return
		case data := <-l.messageChan:
			l.mu.Lock()
			if l.writer != nil {
				lineToWrite := append(data, '\n')
				if _, err := l.writer.Write(lineToWrite); err != nil {
					log.Printf("Write error: %v\n", err)
				} else {
					messageCount++
					if messageCount%100 == 0 {
						l.writer.Flush()
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

func (l *JSONLLogger) rotateFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rotateFileLocked()
}

func (l *JSONLLogger) rotateFileLocked() error {
	if l.writer != nil {
		l.writer.Flush()
	}
	if l.gzipWriter != nil {
		l.gzipWriter.Close()
	}

	if l.file != nil {
		l.file.Close()
	}

	currentDate := time.Now().UTC().Format("2006-01-02")
	filename := filepath.Join(l.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", currentDate))

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	l.file = file
	l.gzipWriter = gzip.NewWriter(file)
	l.writer = bufio.NewWriterSize(l.gzipWriter, 64*1024)
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
}

func (l *JSONLLogger) GetCurrentFilename() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentDate == "" {
		return ""
	}
	return filepath.Join(l.dataDir, fmt.Sprintf("market_data_%s.jsonl.gz", l.currentDate))
}
