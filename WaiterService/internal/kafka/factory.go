package kafka

import (
	"fmt"

	"github.com/kbdering/waiterservice/internal/messaging"
	"github.com/kbdering/waiterservice/pkg/message"
)

type KafkaFactory struct{}

func NewKafkaFactory() *KafkaFactory {
	return &KafkaFactory{}
}

func (f *KafkaFactory) CreateProducer(configer interface{}, responses chan *message.Message) (messaging.Producer, error) {
	kafkaConfig, ok := configer.(*KafkaConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for Kafka producer")
	}

	// Convert config.KafkaConfig to internal kafka.KafkaConfig
	cfg := &KafkaConfig{
		BootstrapServers: kafkaConfig.BootstrapServers,
		ClientID:         kafkaConfig.ClientID,
		GroupID:          kafkaConfig.GroupID,
		RequestTopic:     kafkaConfig.RequestTopic,
		ResponseTopic:    kafkaConfig.ResponseTopic,
		MaxRetries:       kafkaConfig.MaxRetries,
		RetryBackoff:     kafkaConfig.RetryBackoff,
	}

	return NewKafkaProducer(cfg, responses), nil
}

func (f *KafkaFactory) CreateConsumer(config interface{}, requests chan *message.Message) (messaging.Consumer, error) {
	kafkaConfig, ok := config.(*KafkaConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for Kafka consumer")
	}

	// Convert config.KafkaConfig to internal kafka.KafkaConfig
	cfg := &KafkaConfig{
		BootstrapServers: kafkaConfig.BootstrapServers,
		ClientID:         kafkaConfig.ClientID,
		GroupID:          kafkaConfig.GroupID,
		RequestTopic:     kafkaConfig.RequestTopic,
		ResponseTopic:    kafkaConfig.ResponseTopic,
		MaxRetries:       kafkaConfig.MaxRetries,
		RetryBackoff:     kafkaConfig.RetryBackoff,
	}

	return NewKafkaConsumer(cfg, requests)
}
