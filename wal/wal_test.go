package wal_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"imkvdb/config"
	"imkvdb/wal"

	"go.uber.org/zap"
)

func TestFileWAL_BasicBatch(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	dir, err := ioutil.TempDir("", "wal_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cfg := config.WALConfig{
		Enabled:              true,
		FlushingBatchSize:    2, // маленький батч для теста
		FlushingBatchTimeout: 50 * time.Millisecond,
		MaxSegmentSize:       "10MB",
		DataDirectory:        dir,
	}
	w, err := wal.NewFileWAL(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer w.Close()

	// Пишем первую запись
	rec1 := wal.Record{Op: wal.OpSet, Key: "k1", Value: "v1"}
	if err := w.WriteAndWait(rec1); err != nil {
		t.Fatalf("WriteAndWait rec1 error: %v", err)
	}

	// Т.к. batch_size=2, flush произойдёт только при второй записи
	// но также может сработать таймаут 50ms
	// Чтоб быть уверенным, что flush ещё не произошёл, можем быстро проверить размер файла
	files, _ := filepath.Glob(filepath.Join(dir, "wal_segment_*.log"))
	if len(files) == 0 {
		t.Errorf("expected wal segment file to exist")
	} else {
		info, _ := os.Stat(files[0])
		if info.Size() == 0 {
			t.Logf("size=0 => запись отложена, ждем вторую операцию или таймаут")
		}
	}

	// Пишем вторую запись -> batch достигнет размера 2 => flush
	rec2 := wal.Record{Op: wal.OpSet, Key: "k2", Value: "v2"}
	if err := w.WriteAndWait(rec2); err != nil {
		t.Fatalf("WriteAndWait rec2 error: %v", err)
	}

	// Теперь должно быть зафлашено
	info, _ := os.Stat(files[0])
	if info.Size() == 0 {
		t.Errorf("expected wal file to have >0 size after flush, got %d", info.Size())
	}
}
