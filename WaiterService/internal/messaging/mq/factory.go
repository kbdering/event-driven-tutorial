package mq

import (
	"fmt"

	"github.com/kbdering/waiterservice/internal/messaging"
	"github.com/kbdering/waiterservice/pkg/message"
)

type MQFactory struct{}

func NewMQFactory() *MQFactory {
	return &MQFactory{}
}

func (f *MQFactory) CreateProducer(config interface{}, responses chan *message.Message) (messaging.Producer, error) {
	mqConfig, ok := config.(*MQConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for MQ producer")
	}

	return NewMQProducer(&MQConfig{
		Host:          mqConfig.Host,
		Port:          mqConfig.Port,
		Username:      mqConfig.Username,
		Password:      mqConfig.Password,
		RequestQueue:  mqConfig.RequestQueue,
		ResponseQueue: mqConfig.ResponseQueue,
		MaxRetries:    mqConfig.MaxRetries,
		RetryBackoff:  mqConfig.RetryBackoff,
	}, responses), nil
}

func (f *MQFactory) CreateConsumer(config interface{}, requests chan *message.Message) (messaging.Consumer, error) {
	mqConfig, ok := config.(*MQConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for MQ consumer")
	}

	return NewMQConsumer(&MQConfig{
		Host:          mqConfig.Host,
		Port:          mqConfig.Port,
		Username:      mqConfig.Username,
		Password:      mqConfig.Password,
		RequestQueue:  mqConfig.RequestQueue,
		ResponseQueue: mqConfig.ResponseQueue,
		MaxRetries:    mqConfig.MaxRetries,
		RetryBackoff:  mqConfig.RetryBackoff,
	}, requests), nil
}
