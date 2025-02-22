package main

import (
	"flag"
	"fmt"
	"imkvdb/wal"
	"os"
	"strings"

	"go.uber.org/zap"

	// Собственные пакеты (примерно так, либо относительные пути):
	"imkvdb/compute"
	"imkvdb/compute/parser"
	"imkvdb/config"
	"imkvdb/storage"
	"imkvdb/storage/engine"
	"imkvdb/tcpserver"
)

func main() {
	// Флаг для пути к файлу конфигурации
	configPath := flag.String("config", "config.yaml", "Path to YAML config file (optional)")
	flag.Parse()

	// Грузим конфигурацию (с дефолтами, если файл не найден)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	// Инициализируем логгер (zap)
	logger, _ := zap.NewProduction() // или NewDevelopment()
	defer logger.Sync()

	// Настраиваем уровень логирования, если нужно
	// (В упрощённом примере пропущено; при желании можно zap.Config сконфигурировать)

	// Создаем in-memory движок (другого типа пока нет)
	var eng storage.Storage = engine.NewInMemoryEngine(logger)

	// 4. Создаем parser
	p := parser.NewParser()

	// 5. Инициализируем compute
	//    (но внутри compute проверяем, включен ли WAL, если да -> FileWAL, иначе NoOpWAL)
	var wl wal.WAL
	if cfg.WAL.Enabled {
		w, err := wal.NewFileWAL(cfg.WAL, logger)
		if err != nil {
			logger.Fatal("failed to create WAL", zap.Error(err))
		}
		wl = w
		// Восстанавливаем данные из WAL:
		err = wal.ReplayWAL(cfg.WAL.DataDirectory, eng, logger)
	} else {
		wl = &wal.NoOpWAL{}
	}

	cmp := compute.NewCompute(p, eng, wl, logger)
	// Создаем и запускаем TCP-сервер
	srv := tcpserver.NewTCPServer(cfg, cmp, logger)
	if err := srv.Start(); err != nil {
		logger.Fatal("Failed to start TCP server", zap.Error(err))
	}

	// Чтобы сервер не выходил сразу — читаем команду "exit" из stdin
	waitForExit(logger)

	// Останавливаем сервер
	srv.Stop()
}

// waitForExit — простой способ «подождать» до ввода "exit" в консоль
func waitForExit(logger *zap.Logger) {
	for {
		var cmd string
		fmt.Print("Type 'exit' to stop server: ")
		_, err := fmt.Scanln(&cmd)
		if err != nil {
			logger.Error("Failed to read exit command", zap.Error(err))
		}
		if strings.ToLower(cmd) == "exit" {
			logger.Info("Stopping server...")
			return
		}
	}
}
