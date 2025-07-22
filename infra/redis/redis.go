package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

// Use LPUSH and BRPOP to implement a job queue

type RedisQueue struct {
	client *redis.Client
}

type Queue interface {
	Enqueue(ctx context.Context, queueName string, message string) error
	Dequeue(ctx context.Context, queueName string, timeout time.Duration) (string, error)
}

func NewRedisQueue(addr string) *RedisQueue {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	return &RedisQueue{client: client}
}

func (q *RedisQueue) Enqueue(ctx context.Context, queueName string, message string) error {
	// Use LPUSH to add a message to the queue
	err := q.client.LPush(ctx, queueName, message).Err()
	if err != nil {
		return err
	}
	return nil
}

func (q *RedisQueue) Dequeue(ctx context.Context, queueName string, timeout time.Duration) (string, error) {
	// Use BLPOP to remove and return the first message from the queue
	result, err := q.client.RPopLPush(ctx, "payments.pending", queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // No message available
		}
		return "", err
	}

	return result, nil // Return the message part of the result
}
