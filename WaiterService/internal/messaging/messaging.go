package messaging

import (
	"context"
	"time"
	"github.com/kbdering/waiterservice/pkg/message"
)

type Producer interface {
	Connect() error
	Send(ctx context.Context, message *message.Message, destination string) error
	Close() error
}

type Consumer interface {
	Connect() error
	Poll(ctx context.Context, timeout time.Duration) error
	Close() error
}

type MessagingFactory interface {
	CreateProducer(config interface{}, responses chan *message.Message) (Producer, error)
	CreateConsumer(config interface{}, requests chan *message.Message) (Consumer, error)
}



