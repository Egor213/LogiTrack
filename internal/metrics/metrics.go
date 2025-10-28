package metrics

import "github.com/prometheus/client_golang/prometheus"

type Counter interface {
	Inc(labels ...string)
}

type Counters struct {
	LogsReceived Counter

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
		GrpcRequests: NewPrometheusCounter(
			"grpc_requests_total",
			"Количество gRPC запросов",
			[]string{"method", "status"},
		),
	}
}

func NewTestCounters() *Counters {
	reg := prometheus.NewRegistry()

	logsReceived := &PrometheusCounter{
		counter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "logs_received_total",
			Help: "Количество логов, принятых gRPC",
		}, []string{"service", "level"}),
	}

	grpcRequests := &PrometheusCounter{
		prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Количество gRPC запросов",
		}, []string{"method", "status"}),
	}

	reg.MustRegister(logsReceived.counter)
	reg.MustRegister(grpcRequests.counter)

	return &Counters{
		LogsReceived: logsReceived,
		GrpcRequests: grpcRequests,
	}
}
