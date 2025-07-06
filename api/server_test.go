package api

import (
	"bytes"
	"encoding/json"
	"go-taskqueue/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTask(t *testing.T) {
	tasks = []model.Task{
		{ID: 1, Status: "queued", Result: 0},
	}

	t.Run("existing task", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tasks/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		getTask(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var task model.Task
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if task.ID != 1 {
			t.Errorf("expected ID 1, got %d", task.ID)
		}
	})

	t.Run("non-existant task", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tasks/99", nil)
		req.SetPathValue("id", "99")
		w := httptest.NewRecorder()

		getTask(w, req)

		if w.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Result().StatusCode)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tasks/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		getTask(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})
}

func TestPostTask(t *testing.T) {
	tasks = nil

	t.Run("valid request", func(t *testing.T) {
		payload := []byte(`{"status":"new","result":123}`)

		req := httptest.NewRequest("POST", "/tasks", bytes.NewReader(payload))
		w := httptest.NewRecorder()

		postTask(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}

		var task model.Task
		json.NewDecoder(resp.Body).Decode(&task)
		if task.ID != 1 || task.Result != 123 || task.Status != "new" {
			t.Errorf("unexpected task: %+v", task)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/tasks/", bytes.NewReader([]byte(`{oops}`)))
		w := httptest.NewRecorder()

		postTask(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})
}
