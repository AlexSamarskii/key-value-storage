package server

import (
	"fmt"
	"io"
	"keyvalue/internal/usecase/aof"
	command "keyvalue/internal/usecase/commands"
	"keyvalue/internal/usecase/resp"
	"keyvalue/internal/usecase/storage"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type Config struct {
	Port        int
	AofFilename string
}

type Server struct {
	config      Config
	listener    net.Listener
	storage     *storage.Storage
	commandExec *command.CommandExecutor
	logger      *log.Logger
	aof         *aof.Aof
	shutdown    chan struct{}
	wg          sync.WaitGroup
	conns       sync.Map
}

func NewServer(cfg Config) *Server {
	stor := storage.NewStorage()
	return &Server{
		config:      cfg,
		storage:     stor,
		commandExec: command.NewCommandExecutor(stor),
		logger:      log.New(os.Stdout, "[kv-server] ", log.Ldate|log.Ltime|log.Lshortfile),
		shutdown:    make(chan struct{}),
	}
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.aof, err = aof.NewAof(s.config.AofFilename)
	if err != nil {
		return fmt.Errorf("failed to create AOF: %w", err)
	}

	if err := s.aof.Read(func(value resp.Value) {
		s.commandExec.Execute(value)
	}); err != nil {
		return fmt.Errorf("failed to read AOF: %w", err)
	}

	s.logger.Printf("Server started on port %d", s.config.Port)
	s.wg.Add(1)
	go s.serve()

	return nil
}

func (s *Server) Stop() {
	close(s.shutdown)
	if s.listener != nil {
		s.listener.Close()
	}
	s.conns.Range(func(key any, _ any) bool {
		conn := key.(net.Conn)
		conn.Close()
		return true
	})
	if s.aof != nil {
		s.aof.Close()
	}
	s.wg.Wait()
	s.logger.Println("Server stopped gracefully")
}

func (s *Server) serve() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			s.logger.Printf("Shutdown")
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					s.logger.Printf("Failed to accept connection: %v", err)
				}
				select {
				case <-s.shutdown:
					return
				default:
					s.logger.Printf("Accept error: %v", err)
					continue
				}
			}

			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handleConnection(conn)
			}()
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Printf("Error closing connection: %v", err)
		}
	}()

	s.conns.Store(conn, struct{}{})
	defer func() {
		conn.Close()
		s.conns.Delete(conn)
	}()

	remoteAddr := conn.RemoteAddr().String()
	s.logger.Printf("New connection from %s", remoteAddr)

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	for {
		select {
		case <-s.shutdown:
			return
		default:
			cmd, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					s.logger.Printf("Client %s disconnected", remoteAddr)
					return
				}
				s.logger.Printf("Error reading from %s: %v", remoteAddr, err)
				return
			}

			if cmd.Typ != "array" || len(cmd.Array) == 0 {
				s.logger.Printf("Invalid command format from %s", remoteAddr)
				writer.Write(resp.Value{Typ: "error", Str: "ERR invalid command format"})
				continue
			}

			cmdStr := commandToString(cmd)
			s.logger.Printf("Command from %s: %s", remoteAddr, cmdStr)

			// Обработка команды
			result := s.processCommand(cmd)

			if err := writer.Write(result); err != nil {
				s.logger.Printf("Failed to write response to %s: %v", remoteAddr, err)
				return
			}
		}
	}
}

func (s *Server) processCommand(cmd resp.Value) resp.Value {
	command := strings.ToUpper(cmd.Array[0].Bulk)

	// Записываем в AOF только модифицирующие команды
	if isWriteCommand(command) {
		if err := s.aof.Write(cmd); err != nil {
			s.logger.Printf("AOF write error: %v", err)
			return resp.Value{Typ: "error", Str: "ERR internal error"}
		}
	}

	return s.commandExec.Execute(cmd)
}

func isWriteCommand(cmd string) bool {
	switch cmd {
	case "SET", "HSET", "DEL", "HDEL", "HDELALL", "EXPIRE":
		return true
	default:
		return false
	}
}

func commandToString(cmd resp.Value) string {
	var parts []string
	for _, arg := range cmd.Array {
		parts = append(parts, arg.Bulk)
	}
	return strings.Join(parts, " ")
}
