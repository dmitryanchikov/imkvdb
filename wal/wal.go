package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	"imkvdb/config"
)

type OperationType int

const (
	OpSet OperationType = iota
	OpDel
)

type Record struct {
	Op    OperationType
	Key   string
	Value string
	// LSN присваивается внутри самой WAL-системы
	LSN uint64
}
type WAL interface {
	WriteAndWait(rec Record) error
	Close() error
}

// NoOpWAL - пустая реализация на случай, если wal.enabled=false
type NoOpWAL struct{}

func (n *NoOpWAL) WriteAndWait(_ Record) error {
	// Ничего не делаем
	return nil
}
func (n *NoOpWAL) Close() error {
	return nil
}

type FileWAL struct {
	cfg    config.WALConfig
	logger *zap.Logger
	dir    string

	mu          sync.Mutex
	currentFile *os.File
	currentSize int64
	nextLSN     uint64

	// Батч (очередь), мьютекс/канал
	batchCh      chan walRequest
	flushTrigger chan struct{}
	quitCh       chan struct{}
	wg           sync.WaitGroup

	maxSegmentBytes int
}

// walRequest - запрос на запись в WAL
type walRequest struct {
	rec  Record
	done chan error // чтобы вернуть ошибку/ОК тому, кто вызвал WriteAndWait
}

// NewFileWAL - создает FileWAL + запускает goroutine для батчирования
func NewFileWAL(cfg config.WALConfig, logger *zap.Logger) (*FileWAL, error) {
	if cfg.DataDirectory == "" {
		return nil, fmt.Errorf("wal directory is not set")
	}

	maxSize, err := parseSize(cfg.MaxSegmentSize) // например, "10MB" -> 10*1024*1024
	if err != nil {
		return nil, fmt.Errorf("invalid max_segment_size: %v", err)
	}

	fw := &FileWAL{
		cfg:    cfg,
		logger: logger,
		dir:    cfg.DataDirectory,

		batchCh:      make(chan walRequest),
		flushTrigger: make(chan struct{}, 1),
		quitCh:       make(chan struct{}),

		maxSegmentBytes: maxSize,
	}
	// Создадим директорию, если не существует
	if err := os.MkdirAll(fw.dir, 0755); err != nil {
		return nil, err
	}

	// Откроем (или создадим) сегмент (для упрощения всегда создаём новый)
	if err := fw.rotateSegment(); err != nil {
		return nil, err
	}

	// Запускаем горутину для батчирования
	fw.wg.Add(1)
	go fw.runBatcher()

	return fw, nil
}

// WriteAndWait добавляет запись в очередь и блокируется до тех пор, пока запись не будет зафлашена
func (fw *FileWAL) WriteAndWait(rec Record) error {
	doneCh := make(chan error, 1)
	fw.batchCh <- walRequest{
		rec:  rec,
		done: doneCh,
	}
	// Подумать о триггере-флашере
	return <-doneCh
}

// runBatcher - основной цикл, который собирает записи и флашит
func (fw *FileWAL) runBatcher() {
	defer fw.wg.Done()

	ticker := time.NewTicker(fw.cfg.FlushingBatchTimeout)
	defer ticker.Stop()

	var buffer []walRequest

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		fw.flushBatch(buffer)
		buffer = buffer[:0]
	}

	for {
		select {
		case <-fw.quitCh:
			// Флашим, завершаем
			flush()
			return

		case req := <-fw.batchCh:
			buffer = append(buffer, req)
			if len(buffer) >= fw.cfg.FlushingBatchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// flushBatch - пишет все записи батча на диск (одним write) + fsync + завершает walRequest
func (fw *FileWAL) flushBatch(batch []walRequest) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Готовим буфер строк
	lines := make([]byte, 0, 256*len(batch))
	for i := range batch {
		fw.nextLSN++
		batch[i].rec.LSN = fw.nextLSN

		line := encodeRecord(batch[i].rec) // например: "LSN=1 SET key val\n"
		lines = append(lines, line...)
	}

	// Пишем в файл
	n, err := fw.currentFile.Write(lines)
	if err != nil {
		// Всем возвращаем ошибку
		for _, r := range batch {
			r.done <- fmt.Errorf("wal write error: %w", err)
		}
		return
	}
	fw.currentSize += int64(n)

	// fsync
	if err := fw.currentFile.Sync(); err != nil {
		for _, r := range batch {
			r.done <- fmt.Errorf("wal fsync error: %w", err)
		}
		return
	}

	// Если превысили лимит сегмента -> rotate
	if fw.currentSize >= int64(fw.maxSegmentBytes) {
		if err := fw.rotateSegment(); err != nil {
			for _, r := range batch {
				r.done <- fmt.Errorf("wal rotate error: %w", err)
			}
			return
		}
	}

	// Всем отдать nil, значит OK
	for _, r := range batch {
		r.done <- nil
	}
}

// rotateSegment - закрывает текущий файл (если есть) и открывает новый
func (fw *FileWAL) rotateSegment() error {
	if fw.currentFile != nil {
		if err := fw.currentFile.Close(); err != nil {
			return err
		}
	}
	segName := fmt.Sprintf("wal_segment_%d.log", time.Now().UnixNano())
	path := filepath.Join(fw.dir, segName)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	fw.currentFile = f
	fw.currentSize = 0

	fw.logger.Info("Opened new WAL segment", zap.String("file", path))
	return nil
}

// encodeRecord - преобразует структуру в строку для WAL
func encodeRecord(r Record) []byte {
	// например: "LSN=1 SET key val\n"
	var opStr string
	if r.Op == OpSet {
		opStr = "SET"
	} else {
		opStr = "DEL"
	}
	return []byte(fmt.Sprintf("LSN=%d %s %s %s\n", r.LSN, opStr, r.Key, r.Value))
}

// Close закрывает WAL
func (fw *FileWAL) Close() error {
	// Останавливаем batcher
	close(fw.quitCh)
	fw.wg.Wait()

	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.currentFile != nil {
		return fw.currentFile.Close()
	}
	return nil
}

// parseSize - пример парсинга "10MB" -> 10*1024*1024
func parseSize(s string) (int, error) {
	// Упрощённый вариант
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid size string")
	}
	suffix := s[len(s)-2:]
	numStr := s[:len(s)-2]
	switch suffix {
	case "KB":
		n, err := strconv.Atoi(numStr)
		return n * 1024, err
	case "MB":
		n, err := strconv.Atoi(numStr)
		return n * 1024 * 1024, err
	default:
		// Пытаемся как int
		return strconv.Atoi(s)
	}
}
