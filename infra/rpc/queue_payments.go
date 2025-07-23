package rpc

import (
	"context"
	"github.com/valyala/gorpc"
	"niltonkummer/rinha-2025/pkg/adapters"
	"niltonkummer/rinha-2025/pkg/models"
)

type queueRPCPayments struct {
	rpcClient *gorpc.Client
}

func NewPayments(client *gorpc.Client) adapters.QueueAdapter {
	return &queueRPCPayments{
		rpcClient: client,
	}
}

// Enqueue adds a message to the queue
func (q queueRPCPayments) Enqueue(ctx context.Context, message any) error {
	_, err := q.rpcClient.Call(&models.EnqueueRPC{
		Request: message.(*models.PaymentRequest),
	})
	if err != nil {
		return err
	}
	return nil
}

// Dequeue retrieves and removes a message from the queue
func (q queueRPCPayments) Dequeue(ctx context.Context) (any, error) {
	var req models.DequeueRPC
	response, err := q.rpcClient.Call(&req)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (q queueRPCPayments) DequeueBatch(ctx context.Context, batchSize int) (any, error) {
	var req = models.DequeueBatchRPC{
		BatchSize: batchSize,
	}
	response, err := q.rpcClient.Call(&req)
	if err != nil {
		return nil, err
	}
	return response, nil
}
