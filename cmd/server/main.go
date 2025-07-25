package main

import (
	"flag"
	"log"

	"go-grpc-server-client/internal/server"
)

func main() {
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	log.Printf("Starting gRPC server on port %d", *port)
	
	if err := server.RunServer(*port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
} 