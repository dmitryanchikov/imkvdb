package compute

import (
	"fmt"

	"go.uber.org/zap"
	"imkvdb/compute/parser"
	"imkvdb/storage"
	"imkvdb/wal"
)

// Compute – интерфейс слоя обработки команд
type Compute interface {
	Process(input string) (string, error)
	ProcessReplay(cmd parser.Command) (string, error) // для восстановления
}

type compute struct {
	parser parser.Parser
	store  storage.Storage
	logger *zap.Logger
	wal    wal.WAL
}

func NewCompute(p parser.Parser, s storage.Storage, w wal.WAL, l *zap.Logger) Compute {
	return &compute{
		parser: p,
		store:  s,
		wal:    w,
		logger: l,
	}
}

// Process – метод, который выполняет парсинг и обработку команды, возвращая результат
func (c *compute) Process(input string) (string, error) {
	cmd, err := c.parser.Parse(input)
	if err != nil {
		c.logger.Error("failed to parse command", zap.Error(err))
		return "", err
	}
	// Модифицирующие операции -> WAL
	if cmd.Type == parser.SET || cmd.Type == parser.DEL {
		// 1. Записываем в WAL
		op := wal.Record{
			Op:    wal.OpSet,
			Key:   cmd.Key,
			Value: cmd.Value,
		}
		if cmd.Type == parser.DEL {
			op.Op = wal.OpDel
		}

		if err := c.wal.WriteAndWait(op); err != nil {
			return "", fmt.Errorf("failed to write WAL: %w", err)
		}
	}
	// 2. Пишем в engine
	return c.applyCommand(cmd)
}

func (c *compute) ProcessReplay(cmd parser.Command) (string, error) {
	// вызывается при реплее WAL (не нужно записывать в WAL заново!)
	return c.applyCommand(cmd)
}

func (c *compute) applyCommand(cmd parser.Command) (string, error) {
	switch cmd.Type {
	case parser.SET:
		err := c.store.Set(cmd.Key, cmd.Value)
		return "OK: SET", err
	case parser.DEL:
		ok := c.store.Del(cmd.Key)
		if !ok {
			return "key not found", nil
		}
		return "OK: DEL", nil
	case parser.GET:
		val, ok := c.store.Get(cmd.Key)
		if !ok {
			return "", fmt.Errorf("key not found")
		}
		return val, nil
	default:
		return "", fmt.Errorf("unknown command")
	}
}
