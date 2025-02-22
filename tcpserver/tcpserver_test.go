package tcpserver_test

import (
	"bufio"
	"fmt"
	"imkvdb/wal"
	"net"
	"strings"
	"testing"
	"time"

	"imkvdb/compute"
	"imkvdb/compute/parser"
	"imkvdb/config"
	"imkvdb/storage"
	"imkvdb/storage/engine"
	"imkvdb/tcpserver"

	"go.uber.org/zap"
)

// TestTCPServer_SimpleFlow — пример минимального интеграционного теста TCP-сервера.
func TestTCPServer_SimpleFlow(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Сконфигурируем минимальные параметры
	cfg := config.Config{}
	cfg.Engine.Type = "in_memory"
	cfg.Network.Address = "127.0.0.1:0" // :0 -> выбрать свободный порт
	cfg.Network.MaxConnections = 5
	cfg.Network.MaxMessageSize = "4KB"
	cfg.Network.IdleTimeout = 2 * time.Second

	// Инициализация compute + in-memory storage
	st := engine.NewInMemoryEngine(logger)
	var store storage.Storage = st
	p := parser.NewParser()
	wl := &wal.NoOpWAL{}
	cmp := compute.NewCompute(p, store, wl, logger)

	// Создаём и запускаем сервер
	srv := tcpserver.NewTCPServer(cfg, cmp, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start TCP server: %v", err)
	}

	// Узнаём фактический адрес (порт) сервера
	actualAddr := getServerAddr(srv)
	defer srv.Stop() // в конце теста остановим сервер

	// Подключаемся к серверу
	conn, err := net.Dial("tcp", actualAddr)
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer conn.Close()

	// Готовим reader
	reader := bufio.NewReader(conn)

	// Отправляем команду: SET key1 value1
	if _, err := fmt.Fprintf(conn, "SET key1 value1\n"); err != nil {
		t.Fatalf("failed to send SET command: %v", err)
	}
	resp, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read SET response: %v", err)
	}
	if !strings.Contains(resp, "OK: SET") {
		t.Errorf("unexpected SET response: %v", resp)
	}

	// Отправляем команду: GET key1
	if _, err := fmt.Fprintf(conn, "GET key1\n"); err != nil {
		t.Fatalf("failed to send GET command: %v", err)
	}
	resp, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read GET response: %v", err)
	}
	if !strings.Contains(resp, "value1") {
		t.Errorf("unexpected GET response: %v", resp)
	}

	// Отправляем команду: DEL key1
	if _, err := fmt.Fprintf(conn, "DEL key1\n"); err != nil {
		t.Fatalf("failed to send DEL command: %v", err)
	}
	resp, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read DEL response: %v", err)
	}
	if !strings.Contains(resp, "OK: DEL") {
		t.Errorf("unexpected DEL response: %v", resp)
	}
}

// getServerAddr — вспомогательная функция для получения адреса
func getServerAddr(s *tcpserver.TCPServer) string {
	addr, _ := s.Addr()
	return addr
}
