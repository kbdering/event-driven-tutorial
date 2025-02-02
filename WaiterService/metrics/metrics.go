package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "The current lag of the Kafka consumer per topic-partition",
		},
		[]string{"topic", "partition"},
	)

	QueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mq_queue_depth",
			Help: "The current depth of RabbitMQ queues",
		},
		[]string{"queue"},
	)

	ChannelDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "channel_depth",
			Help: "Depth of Request Channel, from consumer to service processing",
		},
		[]string{"channel"},
	)
)

type MetricsServer struct {
	server *http.Server
}

func NewMetricsServer(port int) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &MetricsServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

func (m *MetricsServer) Start() error {
	return m.server.ListenAndServe()
}

func (m *MetricsServer) Stop(ctx context.Context) error {
	return m.server.Shutdown(ctx)
}
