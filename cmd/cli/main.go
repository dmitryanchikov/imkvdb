package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"imkvdb/compute"
	"imkvdb/compute/parser"
	"imkvdb/storage"
	"imkvdb/storage/engine"

	"go.uber.org/zap"
)

func main() {
	// Инициализируем логгер zap
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Создаем in-memory движок
	inMemEngine := engine.NewInMemoryEngine(logger)

	// Так как наш storage.Storage пока совпадает по интерфейсу с engine.Engine,
	// мы можем его передать напрямую
	var store storage.Storage = inMemEngine

	// Создаем парсер
	p := parser.NewParser()

	// Создаем слой compute
	cmp := compute.NewCompute(p, store, logger)

	// Запускаем цикл чтения команд из stdin
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("In-memory KV store. Enter command (SET/GET/DEL) or type 'exit' to quit.")

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("failed to read string", zap.Error(err))
			break
		}

		line = strings.TrimSpace(line)
		if line == "exit" {
			logger.Info("exiting application")
			break
		}

		if line == "" {
			// Пустая строка, просто продолжаем
			continue
		}

		result, err := cmp.Process(line)
		if err != nil {
			fmt.Println("ERROR:", err)
		} else {
			// Показываем результат (например, значение при GET или сообщение при SET/DEL)
			fmt.Println(result)
		}
	}
}
