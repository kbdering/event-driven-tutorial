package mq

import (
	"context"
	"fmt"

	"github.com/kbdering/waiterservice/pkg/message"
	"github.com/kbdering/waiterservice/pkg/tracing"
	"github.com/streadway/amqp"
)

type MQProducer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	responses  chan *message.Message
	config     *MQConfig
}

func NewMQProducer(config *MQConfig, responses chan *message.Message) *MQProducer {
	return &MQProducer{
		responses: responses,
		config:    config,
	}
}

func (p *MQProducer) Connect() error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/",
		p.config.Username, p.config.Password, p.config.Host, p.config.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	p.connection = conn
	p.channel = ch
	return nil
}

// Convert message headers to AMQP table
func messageHeadersToTable(headers message.MessageHeaders) amqp.Table {
	table := make(amqp.Table)
	for _, key := range headers.Keys() {
		table[key] = headers.Get(key)
	}
	return table
}

func (p *MQProducer) Send(ctx context.Context, msg *message.Message, defaultDestination string) error {
	_, span := tracing.GetTracer().Start(ctx, "mq-send")
	defer span.End()

	// Check for destination in message headers, fallback to default
	destination := msg.Headers.Get("destination")
	if destination == "" {
		destination = defaultDestination
	}

	// Inject tracing context into message headers
	tracing.InjectContextToHeaders(msg, ctx)

	return p.channel.Publish(
		"",          // exchange
		destination, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg.Body,
			Headers:     messageHeadersToTable(msg.Headers),
			Timestamp:   msg.Timestamp,
		},
	)
}

func (p *MQProducer) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			return err
		}
	}
	if p.connection != nil {
		if err := p.connection.Close(); err != nil {
			return err
		}
	}
	return nil
}
