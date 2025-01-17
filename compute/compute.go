package compute

import (
	"fmt"

	"go.uber.org/zap"
	"imkvdb/compute/parser"
	"imkvdb/storage"
)

// Compute – интерфейс слоя обработки команд
type Compute interface {
	Process(input string) (string, error)
}

type compute struct {
	parser  parser.Parser
	storage storage.Storage
	logger  *zap.Logger
}

// NewCompute – конструктор для compute
func NewCompute(p parser.Parser, s storage.Storage, logger *zap.Logger) Compute {
	return &compute{
		parser:  p,
		storage: s,
		logger:  logger,
	}
}

// Process – метод, который выполняет парсинг и обработку команды, возвращая результат
func (c *compute) Process(input string) (string, error) {
	cmd, err := c.parser.Parse(input)
	if err != nil {
		c.logger.Error("failed to parse command", zap.Error(err))
		return "", err
	}

	switch cmd.Type {
	case parser.SET:
		err := c.storage.Set(cmd.Key, cmd.Value)
		if err != nil {
			c.logger.Error("failed to SET", zap.Error(err))
			return "", err
		}
		return fmt.Sprintf("OK: key=%s set", cmd.Key), nil

	case parser.GET:
		val, ok := c.storage.Get(cmd.Key)
		if !ok {
			return "", fmt.Errorf("key %s not found", cmd.Key)
		}
		return val, nil

	case parser.DEL:
		ok := c.storage.Del(cmd.Key)
		if !ok {
			return "", fmt.Errorf("key %s not found", cmd.Key)
		}
		return fmt.Sprintf("OK: key=%s deleted", cmd.Key), nil

	default:
		// Теоретически сюда не дойдём, т.к. unknown command отлавливается парсером
		return "", fmt.Errorf("unknown command type: %v", cmd.Type)
	}
}
