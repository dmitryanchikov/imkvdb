package wal

import (
	"bufio"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// Replayer — тот, кто умеет применять команды из WAL (Set/Del).
type Replayer interface {
	Set(key, value string) error
	Del(key string) bool
}

// ReplayWAL читает все *.log файлы в каталоге WAL
// и последовательно применяет операции
func ReplayWAL(dir string, replayer Replayer, logger *zap.Logger) error {
	files, err := filepath.Glob(filepath.Join(dir, "wal_segment_*.log"))
	if err != nil {
		return err
	}
	sort.Strings(files) // по имени (в нашем случае по времени), чтобы идти от старого к новому

	for _, f := range files {
		logger.Info("Replaying WAL segment", zap.String("file", f))
		if err := replayFile(f, replayer); err != nil {
			return fmt.Errorf("replay file %s error: %w", f, err)
		}
	}
	return nil
}

func replayFile(path string, replayer Replayer) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			if err != nil {
				err = errors.Join(err, closeErr)
			} else {
				err = closeErr
			}
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if err := applyLine(line, replayer); err != nil {
			return fmt.Errorf("apply line %q error: %w", line, err)
		}
	}
	return scanner.Err()
}

// applyLine парсит строку вида "LSN=3 SET key1 value1" и вызывает compute
func applyLine(line string, replayer Replayer) error {
	// Простой разбор: уберём "LSN=3 " в начале
	// Или используем regexp: ^LSN=(\d+)\s+(SET|DEL)\s+(\S+)\s+(.*)$
	r := regexp.MustCompile(`^LSN=\d+\s+(SET|DEL)\s+(\S+)\s+(.*)$`)
	m := r.FindStringSubmatch(line)
	if len(m) != 4 {
		return fmt.Errorf("invalid WAL line format")
	}
	cmdType := m[1]
	key := m[2]
	val := m[3] // для DEL тоже что-то может быть
	switch cmdType {
	case "SET":
		return replayer.Set(key, val)
	case "DEL":
		replayer.Del(key)
		return nil
	default:
		return fmt.Errorf("unknown op: %s", cmdType)
	}
}
