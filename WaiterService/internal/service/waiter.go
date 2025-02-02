package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kbdering/waiterservice/internal/config"
	"github.com/kbdering/waiterservice/internal/kafka"
	"github.com/kbdering/waiterservice/internal/messaging"
	"github.com/kbdering/waiterservice/internal/messaging/mq"
	"github.com/kbdering/waiterservice/pkg/message"
	"github.com/kbdering/waiterservice/pkg/tracing"
	"go.uber.org/zap"
)

type WaiterService struct {
	*Service
	waitTime     int
	pollDuration time.Duration
	producers    []messaging.Producer
	consumers    []messaging.Consumer
	factory      messaging.MessagingFactory
	config       interface{}
}

func NewWaiterService(cfg *config.Config) *WaiterService {
	var factory messaging.MessagingFactory
	var msgConfig interface{}

	switch cfg.Service.MessagingSystem {
	case "kafka":
		factory = kafka.NewKafkaFactory()
		// Convert to internal Kafka config
		msgConfig = &kafka.KafkaConfig{
			BootstrapServers: cfg.Kafka.BootstrapServers,
			ClientID:         cfg.Kafka.ClientID,
			GroupID:          cfg.Kafka.GroupID,
			RequestTopic:     cfg.Kafka.RequestTopic,
			ResponseTopic:    cfg.Kafka.ResponseTopic,
			MaxRetries:       cfg.Kafka.MaxRetries,
			RetryBackoff:     cfg.Kafka.RetryBackoff,
		}
	case "mq":
		factory = mq.NewMQFactory()
		msgConfig = &mq.MQConfig{
			Host:          cfg.MQ.Host,
			Port:          cfg.MQ.Port,
			Username:      cfg.MQ.Username,
			Password:      cfg.MQ.Password,
			RequestQueue:  cfg.MQ.RequestQueue,
			ResponseQueue: cfg.MQ.ResponseQueue,
			MaxRetries:    cfg.MQ.MaxRetries,
			RetryBackoff:  cfg.MQ.RetryBackoff,
		}
	default:
		panic("unsupported messaging system")
	}

	ws := &WaiterService{
		waitTime:     cfg.Service.WaitTime,
		pollDuration: time.Duration(cfg.Service.PollDuration) * time.Millisecond,
		producers:    make([]messaging.Producer, cfg.Service.NumProducers),
		consumers:    make([]messaging.Consumer, cfg.Service.NumConsumers),
		factory:      factory,
		config:       msgConfig,
	}
	ws.Service = NewService(cfg, ws)
	return ws
}

func (ws *WaiterService) Setup(ctx context.Context) error {
	// Initialize producers
	for i := 0; i < len(ws.producers); i++ {
		producer, err := ws.factory.CreateProducer(ws.config, ws.chResponses)
		if err != nil {
			return fmt.Errorf("failed to create producer %d: %w", i, err)
		}

		if err := producer.Connect(); err != nil {
			return fmt.Errorf("failed to connect producer %d: %w", i, err)
		}
		ws.producers[i] = producer
	}

	// Initialize consumers
	for i := 0; i < len(ws.consumers); i++ {
		consumer, err := ws.factory.CreateConsumer(ws.config, ws.chRequests)
		if err != nil {
			return fmt.Errorf("failed to create consumer %d: %w", i, err)
		}
		if err := consumer.Connect(); err != nil {
			return fmt.Errorf("failed to connect consumer %d: %w", i, err)
		}
		ws.consumers[i] = consumer
	}

	return nil
}

func (ws *WaiterService) Run(ctx context.Context) error {
	// Start consumer goroutines
	var wg sync.WaitGroup
	for _, consumer := range ws.consumers {
		wg.Add(1)
		go func(c messaging.Consumer) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := c.Poll(ctx, ws.pollDuration); err != nil {
						ws.logger.Error("poll error", zap.Error(err))
					}
				}
			}
		}(consumer)
	}

	// Start producer goroutines
	for _, producer := range ws.producers {
		wg.Add(1)
		go func(p messaging.Producer) {
			defer wg.Done()
			var defaultDestination string

			// Set default destination based on messaging system
			switch ws.config.(type) {
			case *kafka.KafkaConfig:
				defaultDestination = ws.config.(*kafka.KafkaConfig).ResponseTopic
			case *mq.MQConfig:
				defaultDestination = ws.config.(*mq.MQConfig).ResponseQueue
			default:
				ws.logger.Error("unknown config type")
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-ws.chResponses:
					if err := p.Send(ctx, msg, defaultDestination); err != nil {
						ws.logger.Error("send error", zap.Error(err))
					}
				}
			}
		}(producer)
	}

	// Call parent Run method to start message processing
	return ws.Service.Run(ctx)
}

func (ws *WaiterService) TearDown(ctx context.Context) error {
	for _, producer := range ws.producers {
		if err := producer.Close(); err != nil {
			ws.logger.Error("producer close error", zap.Error(err))
		}
	}
	for _, consumer := range ws.consumers {
		if err := consumer.Close(); err != nil {
			ws.logger.Error("consumer close error", zap.Error(err))
		}
	}
	return nil
}

func (ws *WaiterService) ProcessMessage(context context.Context, message *message.Message) (*message.Message, error) {
	_, span := tracing.GetTracer().Start(tracing.GetContextFromHeaders(message), "wait")
	defer span.End()
	time.Sleep(time.Duration(ws.waitTime) * time.Millisecond)

	resp := message
	resp.Body = []byte(fmt.Sprintf("Hello %s, i've waited for you for %d milliseconds", string(message.Body), ws.waitTime))
	fmt.Println(string(resp.Body))
	return resp, nil
}
