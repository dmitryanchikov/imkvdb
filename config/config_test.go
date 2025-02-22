package config_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"imkvdb/config"
)

func TestLoadConfig_DefaultsIfFileNotFound(t *testing.T) {
	// Пытаемся загрузить несуществующий файл
	cfg, err := config.LoadConfig("file_that_does_not_exist.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем, что подставились дефолты
	if cfg.Engine.Type != "in_memory" {
		t.Errorf("expected default engine.type = in_memory, got %s", cfg.Engine.Type)
	}
	if cfg.Network.Address != "127.0.0.1:4000" {
		t.Errorf("expected default network.address, got %s", cfg.Network.Address)
	}
	if cfg.Network.MaxConnections != 10 {
		t.Errorf("expected default max_connections=10, got %d", cfg.Network.MaxConnections)
	}
	if cfg.Network.MaxMessageSize != "4KB" {
		t.Errorf("expected default max_message_size=4KB, got %s", cfg.Network.MaxMessageSize)
	}
	if cfg.Network.IdleTimeout != 5*time.Minute {
		t.Errorf("expected default idle_timeout=5m, got %v", cfg.Network.IdleTimeout)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default logging.level=info, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Output != "stdout" {
		t.Errorf("expected default logging.output=stdout, got %s", cfg.Logging.Output)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Создаём временный файл с YAML
	content := `
engine:
  type: "in_memory"
network:
  address: "127.0.0.1:3223"
  max_connections: 100
  max_message_size: "8KB"
  idle_timeout: 10m
logging:
  level: "debug"
  output: "/tmp/test.log"
`
	tmpFile, err := ioutil.TempFile("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	// Загружаем конфиг
	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Проверяем, что значения считались корректно
	if cfg.Engine.Type != "in_memory" {
		t.Errorf("got engine.type=%s, want in_memory", cfg.Engine.Type)
	}
	if cfg.Network.Address != "127.0.0.1:3223" {
		t.Errorf("got network.address=%s, want 127.0.0.1:3223", cfg.Network.Address)
	}
	if cfg.Network.MaxConnections != 100 {
		t.Errorf("got max_connections=%d, want 100", cfg.Network.MaxConnections)
	}
	if cfg.Network.MaxMessageSize != "8KB" {
		t.Errorf("got max_message_size=%s, want 8KB", cfg.Network.MaxMessageSize)
	}
	if cfg.Network.IdleTimeout != 10*time.Minute {
		t.Errorf("got idle_timeout=%v, want 10m", cfg.Network.IdleTimeout)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("got logging.level=%s, want debug", cfg.Logging.Level)
	}
	if cfg.Logging.Output != "/tmp/test.log" {
		t.Errorf("got logging.output=%s, want /tmp/test.log", cfg.Logging.Output)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"4KB", 4 * 1024, false},
		{"1MB", 1 * 1024 * 1024, false},
		{"512", 512, false},
		{"abc", 0, true},
		{"4MBx", 0, true},
	}

	for _, tt := range tests {
		got, err := config.ParseSize(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSize(%q) error = %v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestLoadConfig_EmptyFields(t *testing.T) {
	// Пример YAML, где часть полей не заполнена
	content := `
engine:
  type: ""
network:
  address: ""
  max_connections: 0
logging:
  level: ""
  output: ""
`
	tmpFile, err := ioutil.TempFile("", "config_test_empty_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Проверяем, что подставились значения по умолчанию
	defaults := config.Config{}
	defaults.Engine.Type = "in_memory"
	defaults.Network.Address = "127.0.0.1:4000"
	defaults.Network.MaxConnections = 10
	defaults.Network.MaxMessageSize = "4KB"
	defaults.Network.IdleTimeout = 5 * time.Minute
	defaults.Logging.Level = "info"
	defaults.Logging.Output = "stdout"
	defaults.WAL.Enabled = false
	defaults.WAL.DataDirectory = "/tmp/wal"
	defaults.WAL.FlushingBatchSize = 100
	defaults.WAL.FlushingBatchTimeout = 10 * time.Millisecond
	defaults.WAL.MaxSegmentSize = "10MB"

	if !reflect.DeepEqual(cfg, defaults) {
		t.Errorf("config not matching defaults after empty fields.\nGot: %#v\nWant: %#v", cfg, defaults)
	}
}
