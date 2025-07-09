package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go-taskqueue/model"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var redisClient *redis.Client

const queueKey = "taskqueue:tasks"

func InitRedis() error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})

	err := redisClient.Ping(ctx).Err()
	if err != nil {
		return err
	}
	return nil
}

func Enqueue(task model.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return redisClient.LPush(ctx, queueKey, data).Err()
}

func Dequeue(blockFor time.Duration) (*model.Task, error) {
	result, err := redisClient.BRPop(ctx, blockFor, queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if len(result) != 2 {
		return nil, fmt.Errorf("unexpected BRPOP result: %v", result)
	}

	var task model.Task

	err = json.Unmarshal([]byte(result[1]), &task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func Remove(taskID int) error {
	entries, err := redisClient.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to fetch queue entries: %w", err)
	}

	for _, entry := range entries {
		var t model.Task
		if err := json.Unmarshal([]byte(entry), &t); err != nil {
			continue
		}

		if t.ID == taskID {
			return redisClient.LRem(ctx, queueKey, 1, entry).Err()
		}
	}

	return nil
}
