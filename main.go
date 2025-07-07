package main

import (
	"log"

	"go-taskqueue/api"
	"go-taskqueue/queue"
	"go-taskqueue/worker"
)

func main() {
	queue.InitRedis()
	worker.Init(api.TaskStore, &api.TaskStoreMu)
	worker.Start()

	addr := ":8080"
	server := api.NewServer(addr)

	log.Printf("Starting server on %s", addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
