package worker

import (
	"context"
	"errors"
	"log"
	"math/rand"
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

func updateTaskStore(task *model.Task) {
	TaskStoreMu.Lock()
	if stored, ok := TaskStore[task.ID]; ok {
		stored.Status = task.Status
		stored.Result = task.Result
	}
	TaskStoreMu.Unlock()
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

					err = processTask(task)
					if err != nil {
						if task.Retries < 3 {
							task.Retries++
							delay := time.Second * time.Duration(1<<task.Retries)
							log.Printf("[worker %d] Retrying task %d in %v (attempt %d)", id, task.ID, delay, task.Retries)
							
							time.AfterFunc(delay, func() {
								if err := queue.Enqueue(*task); err != nil {
									log.Printf("[worker %d] Failed to re-enqueue task %d: %v", id, task.ID, err)
								}
							})
						} else {
							task.Status = "failed"
							updateTaskStore(task)
							log.Printf("[worker %d] Task %d failed after %d retries", id, task.ID, task.Retries)
						}
						continue
					}

					updateTaskStore(task)
					log.Printf("[worker %d] processed task ID %d", id, task.ID)
				}
			}
		}(i + 1)
	}
}

func processTask(task *model.Task) error {
	if rand.Intn(4) == 0 {
			return errors.New("simulated failure")
	}
	task.Result *= 2
	task.Status = "completed"
	return nil
}
