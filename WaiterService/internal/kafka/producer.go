package kafka

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kbdering/waiterservice/pkg/message"
	"github.com/kbdering/waiterservice/pkg/tracing"
)

type KafkaProducer struct {
	producer    *kafka.Producer
	chResponses chan *message.Message
	config      *KafkaConfig
	chDelivery  chan kafka.Event
}

func (p *KafkaProducer) Send(ctx context.Context, message *message.Message, defaultTopic string) error {
	ctx = tracing.GetContextFromHeaders(message)
	_, span := tracing.GetTracer().Start(ctx, "send-response")
	defer span.End()

	// Check for destination in message headers, fallback to default
	topic := message.Headers.Get("destination")
	if topic == "" {
		topic = defaultTopic
	}

	m := kafkaMessageFromMessage(message, topic)
	p.producer.Produce(m, p.chDelivery)

	<-p.chDelivery
	span.AddEvent("response-sent")

	return nil
}

func (p *KafkaProducer) Connect() error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": p.config.BootstrapServers,
		"client.id":         p.config.ClientID,
		"acks":              "all"})
	if err != nil {
		return err
	}
	p.producer = producer
	return nil
}

func (p *KafkaProducer) Close() error {
	p.producer.Close()
	return nil
}

func NewKafkaProducer(config *KafkaConfig, chMessages chan *message.Message) *KafkaProducer {
	return &KafkaProducer{
		chResponses: chMessages,
		config:      config,
		chDelivery:  make(chan kafka.Event, 100),
	}
}

func kafkaMessageFromMessage(message *message.Message, defaultTopic string) (m *kafka.Message) {
	m = new(kafka.Message)
	for _, header := range message.Headers.Keys() {
		m.Headers = append(m.Headers, kafka.Header{Key: header, Value: []byte(message.Headers.Get(header))})
	}
	m.Value = message.Body
	m.Key = []byte(message.Headers.Get("correlation"))
	var responseTopic string
	if message.Headers.Get("responseTopic") != "" {
		responseTopic = string(message.Headers.Get("responseTopic"))
	} else {
		responseTopic = defaultTopic
	}
	m.TopicPartition = kafka.TopicPartition{Topic: &responseTopic, Partition: kafka.PartitionAny}
	return m
}
