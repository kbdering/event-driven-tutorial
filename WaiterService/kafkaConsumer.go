package main

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/trace"
)

type KafkaConsumer struct {
	chMessages      chan *Message
	config          map[string]string
	messageConsumer *kafka.Consumer
}

func newKafkaConsumer(config map[string]string, chMessages chan *Message) (*KafkaConsumer, error) {
	consumer := new(KafkaConsumer)
	consumer.config = config
	consumer.chMessages = chMessages
	return consumer, nil
}

func (c *KafkaConsumer) connect() error {

	messageConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": c.config["bootstrap.servers"],
		"group.id":          c.config["group.id"],
		"auto.offset.reset": "smallest"})
	if err != nil {
		panic(err)
	}

	err = messageConsumer.Subscribe(c.config["topic"], nil)
	if err != nil {
		fmt.Println(err)
	}
	c.messageConsumer = messageConsumer
	return nil

}
func (c *KafkaConsumer) poll(timeout int) error {
	ev := c.messageConsumer.Poll(timeout)
	switch ev.(type) {
	case *kafka.Message:
		message := messageFromKafka(ev.(*kafka.Message))
		ctx := getContextFromHeaders(message)

		_, span := getTracer().Start(ctx, "poll-kafka-message")
		defer span.End()

		span.AddEvent("Kafka Message Time", trace.WithTimestamp(message.timestamp))
		fmt.Println(message)
		span.AddEvent("Forward message to service")
		c.chMessages <- message
		span.AddEvent("Message forwarded to service ")
		return nil
	case kafka.Error:
		fmt.Printf("Error polling from kafka topic!")
		fmt.Printf(ev.String())
		return ev.(error)
	default:
		return nil

	}

}

func messageFromKafka(kafkaMessage *kafka.Message) *Message {
	message := NewMessage()
	for _, header := range kafkaMessage.Headers {
		message.headers.Set(header.Key, string(header.Value))
	}
	message.headers.Set("correlation", string(kafkaMessage.Key))
	message.timestamp = kafkaMessage.Timestamp
	message.body = kafkaMessage.Value
	return message
}

func (c *KafkaConsumer) close() error {
	return c.messageConsumer.Close()
}
