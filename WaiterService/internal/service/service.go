package service

import (
	"context"
	"sync"

	"github.com/kbdering/waiterservice/health"
	"github.com/kbdering/waiterservice/internal/config"
	"github.com/kbdering/waiterservice/metrics"
	"github.com/kbdering/waiterservice/pkg/logging"
	"github.com/kbdering/waiterservice/pkg/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Servicer interface {
	ProcessMessage(ctx context.Context, message *message.Message) (*message.Message, error)
	Setup(ctx context.Context) error
	TearDown(ctx context.Context) error
	Run(ctx context.Context) error
}

type Service struct {
	chRequests  chan *message.Message
	chResponses chan *message.Message
	threads     int
	rateLimiter *rate.Limiter
	wg          sync.WaitGroup
	logger      *zap.Logger
	healthCheck *health.HealthCheck
	processor   Servicer
}

func NewService(config *config.Config, processor Servicer) *Service {
	return &Service{
		chRequests:  make(chan *message.Message, config.Service.RequestBuffer),
		chResponses: make(chan *message.Message, config.Service.ResponseBuffer),
		threads:     config.Service.NumThreads,
		rateLimiter: rate.NewLimiter(rate.Limit(config.Service.RateLimit), 1),
		logger:      logging.GetLogger(),
		healthCheck: health.NewHealthCheck(),
		processor:   processor,
	}
}

func (s *Service) Run(ctx context.Context) error {
	s.healthCheck.Start()
	defer s.healthCheck.Stop()

	for i := 0; i < s.threads; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.processLoop(ctx)
		}()
	}

	s.healthCheck.SetReady(true)
	s.wg.Wait()
	return nil
}

func (s *Service) processLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.chRequests:
			if err := s.rateLimiter.Wait(ctx); err != nil {
				s.logger.Error("rate limit error", zap.Error(err))
				continue
			}

			response, err := s.processor.ProcessMessage(ctx, msg)
			if err != nil {
				s.logger.Error("processing error", zap.Error(err))
				continue
			}

			select {
			case s.chResponses <- response:
				metrics.ChannelDepth.With(prometheus.Labels{
					"channel" : "response_messages",
				}).Set(float64(len(s.chResponses)))
				
			case <-ctx.Done():
				return
			}
		}
	}
}
