package config

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config — основная структура конфигурации
type Config struct {
	Engine  EngineConfig  `yaml:"engine"`
	Network NetworkConfig `yaml:"network"`
	Logging LoggingConfig `yaml:"logging"`
}

// EngineConfig — конфигурация движка
type EngineConfig struct {
	Type string `yaml:"type"` // Например, "in_memory" (наше единственное значение пока)
}

// NetworkConfig — конфигурация TCP-сервера
type NetworkConfig struct {
	Address        string `yaml:"address"`         // Например, "127.0.0.1:3223"
	MaxConnections int    `yaml:"max_connections"` // Максимальное кол-во одновременных клиентов

	// Можно хранить сырые строки и потом при запуске парсить (KB, MB и т.п.)
	MaxMessageSize string        `yaml:"max_message_size"` // напр., "4KB"
	IdleTimeout    time.Duration `yaml:"idle_timeout"`     // Можно распарсить напрямую time.ParseDuration
}

// LoggingConfig — конфигурация логирования
type LoggingConfig struct {
	Level  string `yaml:"level"`  // "debug" / "info" / "error"
	Output string `yaml:"output"` // куда писать логи
}

// LoadConfig читает YAML-файл и возвращает Config с учётом значений по умолчанию
func LoadConfig(path string) (Config, error) {
	var cfg Config

	// Сначала зададим дефолты
	cfg.Engine.Type = "in_memory"
	cfg.Network.Address = "127.0.0.1:4000"
	cfg.Network.MaxConnections = 10
	cfg.Network.MaxMessageSize = "4KB"
	cfg.Network.IdleTimeout = 5 * time.Minute
	cfg.Logging.Level = "info"
	cfg.Logging.Output = "stdout"

	// Пытаемся прочитать файл (если не нашли, не падаем, а оставляем дефолты)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		// Можно проверить, существует ли файл, если нет — вернём дефолты без ошибки
		// либо вернуть ошибку, если мы хотим, чтобы наличие файла было обязательно
		return cfg, nil // Возвращаем cfg с дефолтами
	}

	// Парсим YAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	// Валидация и до-настройка при необходимости
	if cfg.Engine.Type == "" {
		cfg.Engine.Type = "in_memory"
	}
	if cfg.Network.Address == "" {
		cfg.Network.Address = "127.0.0.1:4000"
	}
	if cfg.Network.MaxConnections <= 0 {
		cfg.Network.MaxConnections = 10
	}
	if cfg.Network.MaxMessageSize == "" {
		cfg.Network.MaxMessageSize = "4KB"
	}
	// time.Duration распарсится автоматически из YAML, если формат корректный (например "5m")
	if cfg.Network.IdleTimeout == 0 {
		cfg.Network.IdleTimeout = 5 * time.Minute
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	return cfg, nil
}

// ParseSize — вспомогательная функция для парсинга "4KB" -> 4096, "1MB" -> 1048576
func ParseSize(input string) (int, error) {
	// Допустим, поддерживаем только KB, MB
	input = strings.ToUpper(strings.TrimSpace(input))
	if strings.HasSuffix(input, "KB") {
		valStr := strings.TrimSuffix(input, "KB")
		num, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, err
		}
		return num * 1024, nil
	} else if strings.HasSuffix(input, "MB") {
		valStr := strings.TrimSuffix(input, "MB")
		num, err := strconv.Atoi(valStr)
		if err != nil {
			return 0, err
		}
		return num * 1024 * 1024, nil
	}
	// Если ничего не подошло, попробуем просто int
	num, err := strconv.Atoi(input)
	if err != nil {
		return 0, errors.New("unknown size format: " + input)
	}
	return num, nil
}
