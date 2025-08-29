package metrics

import "github.com/prometheus/client_golang/prometheus"

type Counter interface {
	Inc(labels ...string)
}

type Counters struct {
	LogsReceived  Counter
	LogsPublished Counter

	GrpcRequests Counter
}

type PrometheusCounter struct {
	counter *prometheus.CounterVec
}

func NewPrometheusCounter(name, help string, labels []string) *PrometheusCounter {
	c := &PrometheusCounter{
		counter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: name,
			Help: help,
		}, labels),
	}
	prometheus.MustRegister(c.counter)
	return c
}

func (p *PrometheusCounter) Inc(labels ...string) {
	p.counter.WithLabelValues(labels...).Inc()
}

func New() *Counters {
	return &Counters{
		LogsReceived: NewPrometheusCounter(
			"logs_received_total",
			"Количество логов, принятых gRPC",
			[]string{"service", "level"},
		),
		LogsPublished: NewPrometheusCounter(
			"logs_published_total",
			"Количество логов, отправленных в Kafka",
			[]string{"service", "level"},
		),
		GrpcRequests: NewPrometheusCounter(
			"grpc_requests_total",
			"Количество gRPC запросов",
			[]string{"method", "status"},
		),
	}
}
