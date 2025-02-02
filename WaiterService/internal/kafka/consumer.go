package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kbdering/waiterservice/metrics"
	"github.com/kbdering/waiterservice/pkg/logging"
	"github.com/kbdering/waiterservice/pkg/message"
	"github.com/kbdering/waiterservice/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type KafkaConsumer struct {
	chMessages      chan *message.Message
	config          *KafkaConfig
	messageConsumer *kafka.Consumer
	logger          *zap.Logger
}

func NewKafkaConsumer(config *KafkaConfig, chMessages chan *message.Message) (*KafkaConsumer, error) {
	return &KafkaConsumer{
		config:     config,
		chMessages: chMessages,
		logger:     logging.GetLogger(),
	}, nil
}

func (c *KafkaConsumer) Connect() error {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": c.config.BootstrapServers,
		"group.id":          c.config.GroupID,
		"auto.offset.reset": "smallest",
	})
	if err != nil {
		return err
	}

	err = consumer.Subscribe(c.config.RequestTopic, nil)
	if err != nil {
		consumer.Close()
		return err
	}

	c.messageConsumer = consumer
	return nil

}

func (c *KafkaConsumer) updateLagMetrics() error {
	// Get assigned partitions
	tpos, err := c.messageConsumer.Assignment()
	if err != nil {
		return fmt.Errorf("failed to get assignments: %w", err)
	}

	for _, tp := range tpos {
		// Get the last committed offset
		committed, err := c.messageConsumer.Committed([]kafka.TopicPartition{tp}, 5000)
		if err != nil {
			c.logger.Warn("failed to get committed offset",
				zap.String("topic", *tp.Topic),
				zap.Int32("partition", tp.Partition),
				zap.Error(err))
			continue
		}

		// Get low and high watermarks
		low, high, err := c.messageConsumer.QueryWatermarkOffsets(
			*tp.Topic, tp.Partition, 5000)
		if err != nil {
			c.logger.Warn("failed to get watermark offsets",
				zap.String("topic", *tp.Topic),
				zap.Int32("partition", tp.Partition),
				zap.Error(err))
			continue
		}

		var lag int64
		if committed[0].Offset < 0 {
			// No committed offset yet, use low watermark
			lag = high - low
		} else {
			// Calculate lag based on committed offset
			lag = high - int64(committed[0].Offset)
		}

		// Ensure lag is not negative
		if lag < 0 {
			lag = 0
		}

		metrics.ConsumerLag.With(prometheus.Labels{
			"topic":     *tp.Topic,
			"partition": fmt.Sprintf("%d", tp.Partition),
		}).Set(float64(lag))
	}
	return nil
}

func (c *KafkaConsumer) Poll(ctx context.Context, timeout time.Duration) error {
	// Update lag metrics every poll
	if err := c.updateLagMetrics(); err != nil {
		logging.GetLogger().Warn("Failed to update lag metrics", zap.Error(err))
	}

	ev := c.messageConsumer.Poll(int(timeout.Milliseconds()))
	switch ev.(type) {
	case *kafka.Message:
		message := messageFromKafka(ev.(*kafka.Message))
		ctx := tracing.GetContextFromHeaders(message)

		_, span := tracing.GetTracer().Start(ctx, "poll-kafka-message")
		defer span.End()

		span.AddEvent("Kafka Message Time", trace.WithTimestamp(message.Timestamp))
		fmt.Println(message)
		span.AddEvent("Forward message to service")
		c.chMessages <- message
		metrics.ChannelDepth.With(prometheus.Labels{
			"channel" : "request_messages",
		}).Set(float64(len(c.chMessages)))

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

func messageFromKafka(kafkaMessage *kafka.Message) *message.Message {
	message := message.NewMessage()
	for _, header := range kafkaMessage.Headers {
		message.Headers.Set(header.Key, string(header.Value))
	}
	message.Headers.Set("correlation", string(kafkaMessage.Key))
	message.Timestamp = kafkaMessage.Timestamp
	message.Body = kafkaMessage.Value
	return message
}

func (c *KafkaConsumer) Close() error {
	return c.messageConsumer.Close()
}
