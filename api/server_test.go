package api

import (
	"bytes"
	"encoding/json"
	"go-taskqueue/model"
	"go-taskqueue/queue"
	"net/http"
	"net/http/httptest"
	"testing"
)


func resetState() {
	taskIDCounter = 0
	taskStore = make(map[int]*model.Task)
	queue.Tasks = make(chan model.Task, 10)
}

func TestGetTask(t *testing.T) {
	t.Run("existing task", func(t *testing.T) {
		resetState()

		task := &model.Task{ID: 1, Status: "queued", Result: 42}
		taskStoreMu.Lock()
		taskStore[1] = task
		taskIDCounter = 1
		taskStoreMu.Unlock()

		req := httptest.NewRequest("GET", "/tasks/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		getTask(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var result model.Task
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if result.ID != 1 || result.Result != 42 {
			t.Errorf("unexpected task data: %+v", result)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		resetState()

		req := httptest.NewRequest("GET", "/tasks/99", nil)
		req.SetPathValue("id", "99")
		w := httptest.NewRecorder()

		getTask(w, req)

		if w.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Result().StatusCode)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		resetState()

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
	t.Run("valid request", func(t *testing.T) {
		resetState()

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
		if task.ID != 1 || task.Result != 123 || task.Status != "queued" {
			t.Errorf("unexpected task: %+v", task)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		resetState()

		req := httptest.NewRequest("POST", "/tasks/", bytes.NewReader([]byte(`{oops}`)))
		w := httptest.NewRecorder()

		postTask(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Result().StatusCode)
		}
	})
}
