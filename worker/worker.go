package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"go-taskqueue/model"
	"go-taskqueue/queue"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func Init(pool *pgxpool.Pool) {
	dbPool = pool
}

func updateTaskStore(ctx context.Context, task *model.Task) error {
	_, err := dbPool.Exec(ctx, `
		UPDATE tasks
		SET status = $1, result = $2, retries = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4`,
		task.Status, task.Result, task.Retries, task.ID,
	)
	return err
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

					err = processTask(ctx, task, id)
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
							if err := updateTaskStore(ctx, task); err != nil {
								log.Printf("[worker %d] Failed to update task %d: %v", id, task.ID, err)
							}
							log.Printf("[worker %d] Task %d failed after %d retries", id, task.ID, task.Retries)
						}
						continue
					}

					if err := updateTaskStore(ctx, task); err != nil {
						log.Printf("[worker %d] Failed to update task %d: %v", id, task.ID, err)
						continue
					}
					log.Printf("[worker %d] processed task ID %d", id, task.ID)
				}
			}
		}(i + 1)
	}
}

func processTask(ctx context.Context, task *model.Task, id int) error {
	// this block might be shady
	task.Status = "processing"
	if err := updateTaskStore(ctx, task); err != nil {
		log.Printf("[worker %d] Failed to update task status %d: %v", id, task.ID, err)
	}
	// ABOVE

	if rand.Intn(4) == 0 {
		return errors.New("simulated failure")
	}
	var n int
	if err := json.Unmarshal(task.Payload, &n); err != nil {
		return err
	}

	n *= 2

	b, err := json.Marshal(n)
	if err != nil {
		return err
	}

	task.Result = json.RawMessage(b)
	task.Status = "completed"
	return nil
}
