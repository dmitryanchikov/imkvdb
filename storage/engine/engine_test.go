package engine

import (
	"testing"

	"go.uber.org/zap"
)

func TestInMemoryEngine(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	engine := NewInMemoryEngine(logger)

	// Проверяем, что GET неизвестного ключа
	if _, found := engine.Get("unknown"); found {
		t.Error("expected 'unknown' key to not be found")
	}

	// Проверяем SET
	if err := engine.Set("k1", "v1"); err != nil {
		t.Error("SET returned unexpected error:", err)
	}

	// Теперь GET должен вернуть k1
	if val, found := engine.Get("k1"); !found || val != "v1" {
		t.Errorf("got = %v, found=%v, want v1, true", val, found)
	}

	// Проверяем DEL
	if ok := engine.Del("k1"); !ok {
		t.Error("expected k1 deletion to succeed")
	}

	// Повторная DEL должна вернуть false
	if ok := engine.Del("k1"); ok {
		t.Error("expected k1 deletion to fail on second time")
	}
}
