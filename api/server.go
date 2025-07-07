package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"go-taskqueue/model"
	"go-taskqueue/queue"
)

var (
	TaskStore     = make(map[int]*model.Task)
	taskIDCounter int
	TaskStoreMu   sync.RWMutex
)

func getTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	TaskStoreMu.RLock()
	task, ok := TaskStore[id]
	TaskStoreMu.RUnlock()

	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func postTask(w http.ResponseWriter, r *http.Request) {
	var task model.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	TaskStoreMu.Lock()
	taskIDCounter++
	task.ID = taskIDCounter
	task.Status = "queued"
	TaskStore[task.ID] = &task
	TaskStoreMu.Unlock()

	err = queue.Enqueue(task)
	if err != nil {
		http.Error(w, "failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /tasks/{id}", getTask)
	mux.HandleFunc("POST /tasks", postTask)

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
