package pubsub

import (
	"context"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

type Publisher interface {
	PublishMessage(routingKey string, queueName string, body []byte) error
	ConsumeMessages(queueName string, handler func(amqp.Delivery)) error
}

type publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQPublisher(ctx context.Context, url string, exchangeName string) (*publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		exchangeName, // name
		"direct",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				ch.Close()
				conn.Close()
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return &publisher{
		conn: conn,
		ch:   ch,
	}, nil
}

var channelPool *amqp.Channel

func (c *publisher) ChannelPool() (*amqp.Channel, error) {

	var once sync.Once
	once.Do(func() {
		ch, err := c.conn.Channel()
		if err != nil {
			panic(err)
		}
		channelPool = ch
	})

	return channelPool, nil
}

func (c *publisher) PublishMessage(routingKey string, queueName string, body []byte) error {

	err := c.ch.Publish(
		queueName,  // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *publisher) ConsumeMessages(queueName string, handler func(amqp.Delivery)) error {
	ch, err := c.conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// Declare the same exchange
	err = ch.ExchangeDeclare(
		"payments",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare exchange")

	// Single queue for all consumers
	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare queue")

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,
		"payment.pending",
		"payments",
		false,
		nil,
	)
	failOnError(err, "Failed to bind queue")

	// Prefetch (QoS) - ensures fair dispatch (1 message per worker)
	err = ch.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	failOnError(err, "Failed to register consumer")

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			handler(d)
			d.Ack(false)
			// fmt.Println("[✓] Done processing message")
		}
	}()
	log.Println("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
	return nil
}

/*func (c *publisher) ConsumeMessages(queueName string, handler func(amqp.Delivery)) error {
	for {
		ch, err := c.conn.Channel()
		if err != nil {
			return err
		}

		// Declare exchange
		err = ch.ExchangeDeclare(
			"payments", // exchange name
			"fanout",   // type
			true,       // durable
			false,      // auto-deleted
			false,      // internal
			false,      // no-wait
			nil,        // arguments
		)

		// Declare queue
		q, err := ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)

		// Bind queue to exchange with routing key
		err = ch.QueueBind(
			q.Name,            // queue name
			"payment.pending", // routing key
			"payments",        // exchange
			false,
			nil,
		)

		failOnError(err, "Failed to declare a queue")

		// Bind queue to exchange with routing key
		err = ch.QueueBind(
			q.Name,            // queue name
			"payment.pending", // routing key
			"payments",        // exchange
			false,
			nil,
		)

		msgs, err := ch.Consume(
			queueName,  // queue
			"payments", // consumer
			true,       // auto-ack
			false,      // exclusive
			false,      // no-local
			false,      // no-wait
			nil,        // args
		)
		if err != nil {
			return err
		}

		for d := range msgs {
			handler(d)
		}
		err = ch.Close()
		if err != nil {
			return err
		}
	}

	return nil
}*/
