package engine

import (
	"sync"

	"go.uber.org/zap"
)

// Engine – это интерфейс, определяющий методы для работы с хранилищем.
// В данном случае повторяет методы storage.Storage, но может быть расширен,
// если у нас появятся специфичные для engine методы (например, сброс на диск, статистика и т.д.).
type Engine interface {
	Set(key, value string) error
	Get(key string) (string, bool)
	Del(key string) bool
}

// InMemoryEngine – простая in-memory реализация Engine
type InMemoryEngine struct {
	mu   sync.RWMutex
	data map[string]string

	logger *zap.Logger
}

// NewInMemoryEngine – конструктор для InMemoryEngine
func NewInMemoryEngine(logger *zap.Logger) *InMemoryEngine {
	return &InMemoryEngine{
		data:   make(map[string]string),
		logger: logger,
	}
}

func (e *InMemoryEngine) Set(key, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.data[key] = value
	e.logger.Info("Set value",
		zap.String("key", key),
		zap.String("value", value),
	)
	return nil
}

func (e *InMemoryEngine) Get(key string) (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	val, ok := e.data[key]
	if ok {
		e.logger.Info("Get value",
			zap.String("key", key),
			zap.String("value", val),
		)
	} else {
		e.logger.Info("Get value - not found",
			zap.String("key", key),
		)
	}

	return val, ok
}

func (e *InMemoryEngine) Del(key string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, ok := e.data[key]
	if ok {
		delete(e.data, key)
		e.logger.Info("Del value",
			zap.String("key", key),
		)
	} else {
		e.logger.Info("Del value - not found",
			zap.String("key", key),
		)
	}

	return ok
}
