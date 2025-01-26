package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"go.uber.org/zap"
)

func main() {
	address := flag.String("address", "127.0.0.1:4000", "Address of the DB server")
	flag.Parse()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Подключаемся к серверу
	conn, err := net.Dial("tcp", *address)
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}
	defer conn.Close()
	logger.Info("Connected to server", zap.String("address", *address))

	reader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(conn)

	fmt.Println("Connected to DB server. Enter commands (SET/GET/DEL). Type 'exit' to quit.")

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("failed to read input", zap.Error(err))
			return
		}
		line = strings.TrimSpace(line)
		if line == "exit" {
			logger.Info("exiting client...")
			return
		}
		if line == "" {
			continue
		}

		// Отправляем команду на сервер
		_, err = conn.Write([]byte(line + "\n"))
		if err != nil {
			logger.Error("failed to send command", zap.Error(err))
			return
		}

		// Читаем ответ
		resp, err := serverReader.ReadString('\n')
		if err != nil {
			logger.Error("failed to read server response", zap.Error(err))
			return
		}
		fmt.Printf("SERVER: %s", resp)
	}
}
