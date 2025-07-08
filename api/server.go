package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"go-taskqueue/model"
	"go-taskqueue/queue"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	dbPool *pgxpool.Pool
	server *http.Server
}

func NewServer(addr string, dbPool *pgxpool.Pool) *http.Server {
	mux := http.NewServeMux()

	srv := &Server{dbPool: dbPool}
	mux.HandleFunc("GET /tasks/{id}", srv.getTask)
	mux.HandleFunc("POST /tasks", srv.postTask)

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var task model.Task
	err = s.dbPool.QueryRow(context.Background(), `
		SELECT id, type, payload, status, retries, result, created_at, updated_at
		FROM tasks WHERE id = $1`, id).Scan(
		&task.ID, &task.Type, &task.Payload, &task.Status, &task.Retries, &task.Result, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s Server) postTask(w http.ResponseWriter, r *http.Request) {
	var task model.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task.Status = "queued"
	err = s.dbPool.QueryRow(context.Background(), `
		INSERT INTO tasks (type, payload, status, retries)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		task.Type, task.Payload, task.Status, task.Retries,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		http.Error(w, "failed to insert task", http.StatusInternalServerError)
		return
	}

	err = queue.Enqueue(task)
	if err != nil {
		http.Error(w, "failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}
