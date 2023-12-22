package main

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaProducer struct {
	producer    *kafka.Producer
	chResponses chan *Message
	config      map[string]string
	chDelivery  chan kafka.Event
}

func (p *KafkaProducer) send(message *Message, destination string) error {
	ctx := getContextFromHeaders(message)
	_, span := getTracer().Start(ctx, "send-response")
	defer span.End()
	m := kafkaMessageFromMessage(message, "Waiter-Responses")
	p.producer.Produce(m, p.chDelivery)

	event := <-p.chDelivery
	span.AddEvent("response-sent")
	fmt.Print(event.(*kafka.Message))

	return nil
}

func (p *KafkaProducer) connect() error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": p.config["bootstrap.servers"],
		"client.id":         p.config["client.id"],
		"acks":              "all"})
	if err != nil {
		return err
	}
	p.producer = producer
	return nil
}

func (p *KafkaProducer) close() error {
	p.producer.Close()
	return nil
}

func newKafkaProducer(config map[string]string, chMessages chan *Message) *KafkaProducer {
	p := new(KafkaProducer)
	p.chResponses = chMessages
	p.config = config
	p.chDelivery = make(chan kafka.Event, 100)
	return p
}

func kafkaMessageFromMessage(message *Message, defaultTopic string) (m *kafka.Message) {
	m = new(kafka.Message)
	for _, header := range message.headers.Keys() {
		m.Headers = append(m.Headers, kafka.Header{Key: header, Value: []byte(message.headers.Get(header))})
	}
	m.Value = message.body
	m.Key = []byte(message.headers.Get("correlation"))
	var responseTopic string
	if message.headers.Get("responseTopic") != "" {
		responseTopic = string(message.headers.Get("responseTopic"))
	} else {
		responseTopic = defaultTopic
	}
	m.TopicPartition = kafka.TopicPartition{Topic: &responseTopic, Partition: kafka.PartitionAny}
	return m
}
