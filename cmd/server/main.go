package main

import (
	"keyvalue/internal/server"
)

const PORT = 6379

func main() {
	srv := server.NewServer(PORT)

	srv.Start()
}
