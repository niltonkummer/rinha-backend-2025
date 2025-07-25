package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"niltonkummer/rinha-2025/infra/pubsub"
	"niltonkummer/rinha-2025/pkg/adapters"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/orch"
	"niltonkummer/rinha-2025/pkg/services/payment"
)

type Consumer struct {
	pubsub        pubsub.Publisher
	queue         adapters.QueueAdapter
	jobProcessor  orch.JobProcessorInterface
	paymentClient *payment.Service
	log           *slog.Logger
}

func NewConsumer(publisher pubsub.Publisher,
	queue adapters.QueueAdapter,
	paymentService *payment.Service,
	jobProcessor orch.JobProcessorInterface,
	log *slog.Logger) *Consumer {
	return &Consumer{
		pubsub:        publisher,
		queue:         queue,
		jobProcessor:  jobProcessor,
		log:           log,
		paymentClient: paymentService,
	}
}

func (c *Consumer) ConsumerQueue() {
	for {
		msgs, err := c.queue.DequeueBatch(context.TODO(), 30)
		if err != nil {
			c.log.Error("Error consuming message from queue", "error", err.Error())
			time.Sleep(1 * time.Second) // Wait before retrying
			continue
		}

		dequeue, ok := msgs.(*models.DequeueBatchRPC)
		if !ok {
			time.Sleep(1 * time.Second) // Wait before checking again
			c.log.Error("Received message is not a PaymentRequest", "message", fmt.Sprintf("%T", msgs))
			continue
		}

		if dequeue.Requests == nil {
			c.log.Debug("Received nil PaymentRequest in dequeue", "message", fmt.Sprintf("%T", msgs))
			time.Sleep(250 * time.Millisecond) // Wait before checking again
			continue
		}

		for _, paymentRequestPtr := range dequeue.Requests {
			paymentRequest := *paymentRequestPtr

			job := orch.NewJob(paymentRequest.CorrelationID, func(ctx context.Context) error {
				c.log.Debug("Processing payment request", "correlation_id", paymentRequest.CorrelationID)

				ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Minute)
				defer cancel()
				paymentRequest.RequestedAt = time.Now().UTC()
				// Call the payment service to process the payment
				_, err := c.paymentClient.PaymentRequest(ctxWithTimeout, paymentRequest)
				if err != nil {
					errEnqueue := c.queue.Enqueue(ctx, paymentRequestPtr)
					if errEnqueue != nil {
						c.log.Error("Failed to publish message back to queue", "correlation_id", paymentRequest.CorrelationID, "error", errEnqueue.Error())
					}
					return err
				}

				c.log.Debug("Payment processed successfully", "correlation_id", paymentRequest.CorrelationID)
				return nil
			})

			c.jobProcessor.AddJob(job)

		}
	}
}
