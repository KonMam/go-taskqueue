package main

import (
	"fmt"
	"log"

	"go-taskqueue/api"
)

func main() {
	fmt.Println("Hello world!")
	addr := ":8080"
	server := api.NewServer(addr)

	log.Printf("Starting server on %s", addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
