package tcpserver

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"imkvdb/compute" // Или относительные пути, если так требуется
	"imkvdb/config"
)

type TCPServer struct {
	cfg       config.Config
	cmp       compute.Compute
	logger    *zap.Logger
	listener  net.Listener
	quitCh    chan struct{}
	wg        sync.WaitGroup
	connLimit chan struct{}
}

// NewTCPServer конструктор
func NewTCPServer(cfg config.Config, cmp compute.Compute, logger *zap.Logger) *TCPServer {
	return &TCPServer{
		cfg:       cfg,
		cmp:       cmp,
		logger:    logger,
		quitCh:    make(chan struct{}),
		connLimit: make(chan struct{}, cfg.Network.MaxConnections), // Ограничитель
	}
}

// Start — запускает слушание порта и приём подключений
func (s *TCPServer) Start() error {
	ln, err := net.Listen("tcp", s.cfg.Network.Address)
	if err != nil {
		s.logger.Error("failed to listen", zap.Error(err))
		return err
	}
	s.listener = ln
	s.logger.Info("TCP server started", zap.String("address", s.cfg.Network.Address))

	// Запуск goroutine, которая будет приём соединений
	go s.acceptLoop()

	return nil
}
func (s *TCPServer) Addr() (string, error) {
	if s.listener == nil {
		return "", errors.New("server is not listening")
	}

	return s.listener.Addr().String(), nil
}

// acceptLoop — бесконечный цикл ожидания новых подключений
func (s *TCPServer) acceptLoop() {
	defer s.logger.Info("acceptLoop stopped")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quitCh:
				s.logger.Info("server shutting down, stop accept loop")
				return
			default:
				s.logger.Error("failed to accept", zap.Error(err))
				continue
			}
		}

		// Пытаемся занять "слот" для нового клиента
		select {
		case s.connLimit <- struct{}{}:
			// Успех, обрабатываем
			s.wg.Add(1)
			go s.handleConnection(conn)
		default:
			// Нет свободного "слота" -> отклоняем соединение
			s.logger.Warn("too many connections, rejecting client")
			_ = conn.Close()
		}
	}
}

// handleConnection — обработка конкретного клиента
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		<-s.connLimit // освобождаем слот
		err := conn.Close()
		if err != nil {
			s.logger.Error("failed to close connection", zap.Error(err))
		}
	}()

	// Устанавливаем idle timeout, если нужно
	if s.cfg.Network.IdleTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(s.cfg.Network.IdleTimeout))
	}

	// Для ограничения сообщения по размеру можно "обёртку" делать или читать посимвольно
	maxSizeBytes, _ := config.ParseSize(s.cfg.Network.MaxMessageSize) // Обработка ошибки опущена для примера

	reader := bufio.NewReader(conn)

	for {
		// Обновим дедлайн на каждый запрос (если хочется сбрасывать таймер)
		if s.cfg.Network.IdleTimeout > 0 {
			_ = conn.SetDeadline(time.Now().Add(s.cfg.Network.IdleTimeout))
		}

		// Читаем строку (до \n)
		line, err := reader.ReadString('\n')
		if err != nil {
			s.logger.Info("client disconnected", zap.Error(err))
			return
		}

		// Ограничение размера (упрощённо)
		if len(line) > maxSizeBytes {
			s.logger.Warn("message too large, closing connection")
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Обработка
		result, err := s.cmp.Process(line)
		if err != nil {
			fmt.Fprintf(conn, "ERROR: %v\n", err)
		} else {
			// Отправляем ответ
			fmt.Fprintf(conn, "%v\n", result)
		}
	}
}

// Stop — останавливает сервер
func (s *TCPServer) Stop() {
	// Закрываем listener -> acceptLoop завершится
	close(s.quitCh)
	s.listener.Close()

	// Ждём завершения всех текущих goroutine
	s.wg.Wait()
	s.logger.Info("server stopped")
}
