package worker

import (
	"context"
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

func Start(ctx context.Context, workerCount int, wg *sync.WaitGroup) {
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					log.Printf("[worker %d] shutting down", id)
					return
				default:
					task, err := queue.Dequeue(2 * time.Second)
					if err != nil {
						log.Printf("[worker %d] dequeue error: %v", id, err)
						continue
					}
					if task == nil {
						continue
					}

					processTask(task)

					TaskStoreMu.Lock()
					if stored, ok := TaskStore[task.ID]; ok {
						stored.Status = task.Status
						stored.Result = task.Result
					}
					TaskStoreMu.Unlock()

					log.Printf("[worker %d] processed task ID %d", id, task.ID)
				}
			}
		}(i + 1)
	}
}

func processTask(task *model.Task) {
	time.Sleep(100 * time.Millisecond) // Work simulation
	task.Result *= 2
	task.Status = "completed"
}
