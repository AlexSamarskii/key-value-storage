package server

import (
	"fmt"
	"io"
	"keyvalue/internal/usecase/aof"
	"keyvalue/internal/usecase/resp"
	"keyvalue/internal/usecase/storage"
	"log"
	"net"
	"os"
	"strings"
)

type Server struct {
	port    int
	listner net.Listener
	storage *storage.Storage
	logger  *log.Logger
	aof     *aof.Aof
}

func NewServer(port int) *Server {
	return &Server{
		port:    port,
		storage: storage.NewStorage(),
		logger:  log.New(os.Stdout, "[INFO]: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (s *Server) Start() {
	listner, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		s.logger.Fatalf("Failed to start the server: %v\n", err)
	}

	aof, err := aof.NewAof("database.aof")
	if err != nil {
		s.logger.Fatalf("Failed to create AOF: %v\n", err)
	}

	defer listner.Close()
	defer aof.Close()

	s.listner = listner
	s.aof = aof

	s.aof.Read(func(value resp.Value) {
		commands.ExecuteCommand(value, s.storage)
	})

	s.serve()

	s.logger.Printf("Server started on port: %v\n", s.port)
}

func (s *Server) serve() {
	for {
		conn, err := s.listner.Accept()
		if err != nil {
			s.logger.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go s.handleConnection(conn)

	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	for {
		cmd, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				s.logger.Printf("Client disconnected: %v\n", conn.RemoteAddr())
				return
			}

			s.logger.Printf("Error reading command: %v\n", err)
			return
		}

		if cmd.Typ != "array" {
			s.logger.Println("Invalid request, expected array")
			continue
		}

		if len(cmd.Array) == 0 {
			s.logger.Println("Invalid request, expected array length > 0")
			continue
		}

		command := strings.ToUpper(cmd.Array[0].Bulk)
		if command == "SET" || command == "HSET" || command == "DEL" || command == "HDEL" || command == "HDELALL" {
			err := s.aof.Write(cmd)
			if err != nil {
				s.logger.Printf("Failed to write to the aof: %v\n", err)
			}
		}

		result := commands.ExecuteCommand(cmd, s.storage)

		writer.Write(result)
	}
}
