package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"log/slog"
	"niltonkummer/rinha-2025/infra/pubsub"
	"niltonkummer/rinha-2025/pkg/adapters"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/cache"
	"niltonkummer/rinha-2025/pkg/services/orch"
	"niltonkummer/rinha-2025/pkg/services/payment"
	"time"
)

type Consumer struct {
	pubsub        pubsub.Publisher
	queue         adapters.QueueAdapter
	jobProcessor  orch.JobProcessorInterface
	paymentClient *payment.Service
	log           *slog.Logger
}

func NewConsumer(publisher pubsub.Publisher,
	queue cache.Queue,
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

func (c *Consumer) ConsumerQueue(queueName string) {
	for {
		msg, err := c.queue.Dequeue(context.TODO())
		if err != nil {
			c.log.Error("Error consuming message from queue", "queue", queueName, "error", err.Error())
			time.Sleep(1 * time.Second) // Wait before retrying
			continue
		}

		dequeue, ok := msg.(*models.DequeueRPC)
		if !ok {
			time.Sleep(1 * time.Second) // Wait before checking again
			c.log.Error("Received message is not a PaymentRequest", "message", fmt.Sprintf("%T", msg))
			continue
		}
		if dequeue.Request == nil {
			c.log.Debug("Received nil PaymentRequest in dequeue", "message", fmt.Sprintf("%T", msg))
			time.Sleep(1 * time.Second) // Wait before checking again
			continue
		}

		paymentRequest := *dequeue.Request

		//err = json.Unmarshal([]byte(msg), &paymentRequest)

		job := orch.NewJob(paymentRequest.CorrelationID, func(ctx context.Context) error {
			c.log.Debug("Processing payment request", "correlation_id", paymentRequest.CorrelationID)

			ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			paymentRequest.RequestedAt = time.Now().UTC()
			// Call the payment service to process the payment
			_, err := c.paymentClient.PaymentRequest(ctxWithTimeout, paymentRequest)
			if err != nil {
				c.log.Error("Failed to process payment", "correlation_id", paymentRequest.CorrelationID, "error", err.Error())

				err := c.queue.Enqueue(ctx, &paymentRequest)
				if err != nil {
					c.log.Error("Failed to publish message back to queue", "correlation_id", paymentRequest.CorrelationID, "error", err.Error())
				}
				return err
			}

			c.log.Debug("Payment processed successfully", "correlation_id", paymentRequest.CorrelationID)
			return nil
		})

		err = c.jobProcessor.AddJob(job)
		if err != nil {
			err := c.queue.Enqueue(context.TODO(), &paymentRequest)
			if err != nil {
				c.log.Error("Failed to publish message back to queue", "correlation_id", paymentRequest.CorrelationID, "error", err.Error())
			}
		}

	}
}

// Start starts the consumer to listen for messages on the specified queue
func (c *Consumer) Start(queueName string, handler func([]byte)) error {
	return c.pubsub.ConsumeMessages(queueName, func(delivery amqp091.Delivery) {

		paymentRequest := models.PaymentRequest{}
		err := json.Unmarshal(delivery.Body, &paymentRequest)
		if err != nil {
			// Handle the error, e.g., log it or return it
			// For now, we just print the error to the console
			c.log.Error("Error unmarshalling payment request:", err.Error())
			return
		}
		job := orch.NewJob(paymentRequest.CorrelationID, func(ctx context.Context) error {
			c.log.Debug("Processing payment request", "correlation_id", paymentRequest.CorrelationID)

			ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			// Call the payment service to process the payment
			_, err := c.paymentClient.PaymentRequest(ctxWithTimeout, paymentRequest)
			if err != nil {
				c.log.Error("Failed to process payment", "correlation_id", paymentRequest.CorrelationID, "error", err.Error())

				err := c.pubsub.PublishMessage(queueName, "payments", delivery.Body)
				if err != nil {
					c.log.Error("Failed to publish message back to queue", "correlation_id", paymentRequest.CorrelationID, "error", err.Error())
				}
				return err
			}

			c.log.Debug("Payment processed successfully", "correlation_id", paymentRequest.CorrelationID)
			return nil
		})

		err = c.jobProcessor.AddJob(job)
		if err != nil {
			err := c.pubsub.PublishMessage(queueName, "payments", delivery.Body)
			if err != nil {
				c.log.Error("Failed to publish message back to queue", "correlation_id", paymentRequest.CorrelationID, "error", err.Error())
			}
		}
	})
}
