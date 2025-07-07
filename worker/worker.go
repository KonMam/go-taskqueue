package worker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go-taskqueue/model"
	"go-taskqueue/queue"
)

var (
	TaskStore   map[int]*model.Task
	TaskStoreMu *sync.RWMutex
)

func Init(store map[int]*model.Task, mu *sync.RWMutex) {
	TaskStore = store
	TaskStoreMu = mu
}

func Start() {
	go func() {
		for {
			task, err := queue.Dequeue(5 * time.Second)
			if err != nil {
				log.Printf("worker dequeue error: %v", err)
				continue
			}
			if task == nil {
				continue
			}

			task.Result = task.Result * 2
			task.Status = "completed"

			TaskStoreMu.Lock()
			if stored, ok := TaskStore[task.ID]; ok {
				stored.Status = task.Status
				stored.Result = task.Result
			}
			TaskStoreMu.Unlock()

			fmt.Printf("Processed task ID %d: result=%d\n", task.ID, task.Result)
		}
	}()
}
