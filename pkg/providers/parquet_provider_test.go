package providers_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ochanomizu/predmarket-scanner/pkg/providers"
	"github.com/ochanomizu/predmarket-scanner/pkg/storage"
)

func TestParquetProvider(t *testing.T) {
	tmpDir := t.TempDir()
	converter := storage.NewParquetConverter(tmpDir)

	date := time.Now().Format("2006-01-02")

	if err := converter.ConvertDay(date); err != nil {
		t.Skipf("Skipping test - no data file found: %v", err)
	}

	provider := providers.NewParquetHistoricalProvider(tmpDir)

	dates, err := provider.GetAvailableDates()
	if err != nil {
		t.Fatalf("GetAvailableDates failed: %v", err)
	}

	if len(dates) == 0 {
		t.Fatal("No dates available")
	}

	targetTime := dates[0]
	snapshots, err := provider.GetSnapshotsAtTime(targetTime)
	if err != nil {
		t.Fatalf("GetSnapshotsAtTime failed: %v", err)
	}

	t.Logf("Found %d snapshots at %s", len(snapshots), targetTime)
}
