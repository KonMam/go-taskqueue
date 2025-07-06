package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"go-taskqueue/model"
)


var tasks = []model.Task{
	{ID: 1, Status: "pending", Result: 0},
	{ID: 2, Status: "completed", Result: 42},
}

func getTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task *model.Task
	for i := range tasks {
		if tasks[i].ID == id {
			task = &tasks[i]
		}
	}

	if task == nil {
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

	task.ID = len(tasks) + 1
	tasks = append(tasks, task)

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
