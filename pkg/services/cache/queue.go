package cache

import (
	"context"
	"sync"
)

type Queue interface {
	Enqueue(ctx context.Context, request any) error
	Dequeue(ctx context.Context) (any, error)
}

type queueImpl struct {
	data []any
	sync.Mutex
}

func NewQueue(size int) Queue {
	return &queueImpl{
		data: make([]any, 0, size),
	}
}

func (q *queueImpl) Enqueue(ctx context.Context, body any) error {
	q.Lock()
	defer q.Unlock()
	q.data = append(q.data, body)
	return nil
}

func (q *queueImpl) Dequeue(ctx context.Context) (any, error) {
	q.Lock()
	defer q.Unlock()
	if len(q.data) == 0 {
		return nil, nil // or an error if preferred
	}
	body := q.data[0]
	q.data = q.data[1:] // remove the first element
	return body, nil
}

func (q *queueImpl) GetList() []any {
	q.Lock()
	defer q.Unlock()
	return q.data
}
