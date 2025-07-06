package worker

import (
	"sync"
	"testing"
	"time"

	"go-taskqueue/model"
	"go-taskqueue/queue"
)

func TestWorkerProcessesTask(t *testing.T) {
	testStore := make(map[int]*model.Task)
	var testMu sync.RWMutex
	queue.Tasks = make(chan model.Task, 1)

	Init(testStore, &testMu)
	Start()

	task := model.Task{ID: 1, Status: "queued", Result: 21}
	testStore[task.ID] = &task
	queue.Tasks <- task

	time.Sleep(50 * time.Millisecond)

	testMu.RLock()
	updated, ok := testStore[1]
	testMu.RUnlock()

	if !ok {
		t.Fatalf("task not found in store")
	}
	if updated.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", updated.Status)
	}
	if updated.Result != 42 {
		t.Errorf("expected result 42, got %d", updated.Result)
	}
}
