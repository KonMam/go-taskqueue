package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-taskqueue/model"
	"go-taskqueue/queue"
)

func resetState() {
	taskIDCounter = 0
	TaskStore = make(map[int]*model.Task)
	queue.Tasks = make(chan model.Task, 10)
}

func TestGetTask(t *testing.T) {
	t.Run("existing task", func(t *testing.T) {
		resetState()

		task := &model.Task{ID: 1, Status: "queued", Result: 42}
		TaskStoreMu.Lock()
		TaskStore[1] = task
		taskIDCounter = 1
		TaskStoreMu.Unlock()

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

func TestNewServer_Integration(t *testing.T) {
	resetState()

	server := NewServer(":0")
	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	// POST /tasks
	payload := []byte(`{"status":"test","result":21}`)
	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to post: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var posted model.Task
	json.NewDecoder(resp.Body).Decode(&posted)
	if posted.ID != 1 {
		t.Errorf("expected ID 1, got %d", posted.ID)
	}

	// GET /tasks/1
	getResp, err := http.Get(ts.URL + "/tasks/1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}

	var fetched model.Task
	json.NewDecoder(getResp.Body).Decode(&fetched)
	if fetched.ID != 1 {
		t.Errorf("expected ID 1, got %d", fetched.ID)
	}
}
