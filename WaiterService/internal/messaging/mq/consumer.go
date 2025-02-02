package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/kbdering/waiterservice/pkg/message"
	"github.com/kbdering/waiterservice/metrics"
	"github.com/kbdering/waiterservice/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
)

// Convert AMQP table to message headers
func tableToMessageHeaders(table amqp.Table) *message.MessageHeaders {
	headers := message.NewMessageHeaders()
	for key, value := range table {
		if strValue, ok := value.(string); ok {
			headers.Set(key, strValue)
		}
	}
	return headers
}

type MQConsumer struct {
	messages   chan *message.Message
	config     *MQConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

func (c *MQConsumer) updateQueueMetrics() error {
	queue, err := c.channel.QueueInspect(c.config.RequestQueue)
	if err != nil {
		return err
	}

	metrics.QueueDepth.With(prometheus.Labels{
		"queue": c.config.RequestQueue,
	}).Set(float64(queue.Messages))

	return nil
}

func (c *MQConsumer) Poll(ctx context.Context, timeout time.Duration) error {
	_, span := tracing.GetTracer().Start(ctx, "mq-poll")
	defer span.End()

	msgs, err := c.channel.Consume(
		c.config.RequestQueue,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		span.RecordError(err)
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case delivery := <-msgs:
		msg := &message.Message{
			Body:      delivery.Body,
			Headers:   *tableToMessageHeaders(delivery.Headers),
			Timestamp: delivery.Timestamp,
		}

		// Extract tracing context from message headers
		msgCtx := tracing.GetContextFromHeaders(msg)
		_, msgSpan := tracing.GetTracer().Start(msgCtx, "process-mq-message")
		defer msgSpan.End()

		c.messages <- msg
		metrics.ChannelDepth.With(prometheus.Labels{
			"queue" : "request_messages",
		}).Set(float64(len(c.messages)))
		
	case <-time.After(timeout):
		return nil
	}

	return nil
}

func NewMQConsumer(config *MQConfig, messages chan *message.Message) *MQConsumer {
	return &MQConsumer{
		messages: messages,
		config:   config,
	}
}

func (c *MQConsumer) Connect() error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/",
		c.config.Username, c.config.Password, c.config.Host, c.config.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	c.connection = conn
	c.channel = ch
	return nil
}

func (c *MQConsumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}
	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			return err
		}
	}
	return nil
}
