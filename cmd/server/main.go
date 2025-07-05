package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"keyvalue/internal/server"
)

const (
	defaultPort = 6379
	aofPath     = "database.aof"
)

func main() {
	srv := server.NewServer(server.Config{
		Port:        defaultPort,
		AofFilename: aofPath,
	})

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Graceful shutdown
	log.Println("Shutting down server...")
	srv.Stop()
	log.Println("Server stopped")
}
