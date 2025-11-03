File: cmd\logitrack\logitrack.go
package main

import "github.com/Egor213/LogiTrack/internal/app"

func main() {
	app.Run()
}

--------------------
File: internal\app\main.go
package app

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	kafkabroker "github.com/Egor213/LogiTrack/internal/broker/kafka"
	"github.com/Egor213/LogiTrack/internal/config"
	grpcv1 "github.com/Egor213/LogiTrack/internal/controller/grpc/v1"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/service"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/Egor213/LogiTrack/pkg/grpcserver"
	"github.com/Egor213/LogiTrack/pkg/httpserver"
	"github.com/Egor213/LogiTrack/pkg/logger"
	"github.com/Egor213/LogiTrack/pkg/postgres"
	"github.com/labstack/echo/v4"

	log "github.com/sirupsen/logrus"
)

func Run() {
	// Config

	cfg, err := config.New()
	if err != nil {
		log.Fatal(errorsUtils.WrapPathErr(err))
	}

	// Logger
	logger.SetupLogger(cfg.Log.Level)
	log.Info("Logger has been set up")

	// Migrations
	Migrate(cfg.PG.URL)

	// DB connecting
	log.Info("Connecting to DB")
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.MaxPoolSize))
	if err != nil {
		log.Fatal(errorsUtils.WrapPathErr(err))
	}
	defer pg.Close()
	log.Info("Connected to DB")

	// Repos
	repositories := repo.NewRepositories(pg)

	// Producer
	brokerProducer := kafkabroker.NewProducer(kafkabroker.ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "notifications",
	})

	// Services
	metricsCnt := metrics.New()
	deps := service.ServicesDependencies{
		Repos:          repositories,
		Counters:       metricsCnt,
		BrokerProducer: brokerProducer,
	}
	services := service.NewServices(deps)

	// gRPC Server
	log.Infof("Starting gRPC server...")
	log.Debugf("gRPC server port: %s", cfg.GRPC.Port)
	registerFun := grpcv1.RegisterServices(services, metricsCnt)
	grpcServer, err := grpcserver.New(registerFun, grpcserver.WithPort(cfg.GRPC.Port))
	if err != nil {
		log.Fatal(errorsUtils.WrapPathErr(err))
	}

	// Prometheus server
	log.Infof("Starting metrics server...")
	log.Debugf("Metrics server port: %s", cfg.Prometheus.Port)
	metricsHandler := echo.New()
	metrics.ConfigureRouter(metricsHandler)
	metricsServer := httpserver.New(metricsHandler, httpserver.Port(cfg.Prometheus.Port))

	log.Info("Configuring graceful shutdown...")

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info(errorsUtils.WrapPathErr(errors.New(s.String())))
	case err := <-metricsServer.Notify():
		log.Info(errorsUtils.WrapPathErr(err))
	case err := <-grpcServer.Notify():
		log.Info(errorsUtils.WrapPathErr(err))
	}

	// Graceful shutdown
	shutdownApp(grpcServer, metricsServer)
}

func shutdownApp(grpcServer *grpcserver.Server, metricsServer *httpserver.Server) {
	log.Info("Shutting down...")
	err := metricsServer.Shutdown()
	if err != nil {
		log.Error(errorsUtils.WrapPathErr(err))
	}
	grpcServer.Shutdown()
}

--------------------
File: internal\app\migrate.go
package app

import (
	"errors"
	"os"

	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"

	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	log "github.com/sirupsen/logrus"
)

const (
	defaultAttempts = 10
	defaultTimeout  = time.Second
)

func Migrate(pgUrl string) {
	pgUrl += "?sslmode=disable"
	log.Infof("Migrate %s", pgUrl)

	var (
		connAttempts = defaultAttempts
		err          error
		mgrt         *migrate.Migrate
	)

	migrationsPath := "migrations"

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		log.Fatalf("migrations directory %q does not exist", migrationsPath)
	}

	for connAttempts > 0 {
		mgrt, err = migrate.New("file://"+migrationsPath, pgUrl)
		if err == nil {
			break
		}

		time.Sleep(defaultTimeout)
		log.Infof("Postgres trying to connect, attempts left: %d", connAttempts)
		connAttempts--
	}

	if err != nil {
		log.Fatal(errorsUtils.WrapPathErr(err))
	}
	defer mgrt.Close()

	if err = mgrt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(errorsUtils.WrapPathErr(err))
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("Migration no change")
		return
	}

	log.Info("Migration successful up")
}

--------------------
File: internal\broker\broker.go
package broker

import "context"

type Producer interface {
	SendMessage(ctx context.Context, value []byte) error
}

--------------------
File: internal\broker\kafka\producer.go
package kafkabroker

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

type ProducerConfig struct {
	Brokers []string
	Topic   string
}

type Producer struct {
	writer *kafka.Writer
	topic  string
}

func NewProducer(cfg ProducerConfig) *Producer {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		Balancer: &kafka.RoundRobin{},
	})
	return &Producer{
		writer: w,
		topic:  cfg.Topic,
	}
}

func (p *Producer) SendMessage(ctx context.Context, value []byte) error {
	msg := kafka.Message{
		Value: value,
		Time:  time.Now(),
	}
	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Errorf("Failed to send message: %v", err)
		return err
	}
	log.Debugf("Message sent: value=%s", string(value))
	return nil
}

func (p *Producer) Close() error {
	log.Info("Closing Kafka producer...")
	return p.writer.Close()
}

--------------------
File: internal\config\config.go
package config

import (
	"os"

	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"

	"github.com/ilyakaznacheev/cleanenv"
	log "github.com/sirupsen/logrus"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		App        `yaml:"app"`
		Log        `yaml:"log"`
		PG         `yaml:"postgres"`
		GRPC       `yaml:"grpc"`
		Prometheus `yaml:"prometheus"`
	}

	App struct {
		Name    string `yaml:"name" env-required:"true"`
		Version string `yaml:"version" env-required:"true"`
	}

	Log struct {
		Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
	}

	PG struct {
		MaxPoolSize int    `env-required:"true" env:"MAX_POOL_SIZE" yaml:"max_pool_size"`
		URL         string `env-required:"true" env:"PG_URL"`
	}

	Prometheus struct {
		Port string `env-required:"true" yaml:"port" env:"PROMETHEUS_PORT"`
	}

	GRPC struct {
		Port string `env-required:"true" yaml:"port" env:"GRPC_PORT"`
	}
)

const ENV_PATH = "infra/.env.dev"

func init() {
	if err := godotenv.Load(ENV_PATH); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

func New() (*Config, error) {
	cfg := &Config{}

	pathToConfig, ok := os.LookupEnv("APP_CONFIG_PATH")
	if !ok || pathToConfig == "" {
		log.WithField("env_var", "APP_CONFIG_PATH").
			Info("Config path is not set, using default")
		pathToConfig = "infra/config.yaml"
	}

	if err := cleanenv.ReadConfig(pathToConfig, cfg); err != nil {
		return nil, errorsUtils.WrapPathErr(err)
	}

	if err := cleanenv.UpdateEnv(cfg); err != nil {
		return nil, errorsUtils.WrapPathErr(err)
	}

	return cfg, nil
}

--------------------
File: internal\controller\common\logging\helper.go
package logginghelper

import (
	"github.com/Egor213/LogiTrack/internal/domain"
	log "github.com/sirupsen/logrus"
)

func LogReceived(entry *domain.LogEntry) {
	log.WithFields(log.Fields{
		"service": entry.Service,
		"level":   entry.Level,
		"message": entry.Message,
	}).Info("Received log via gRPC")
}

func LogSaved(entry *domain.LogEntry, id int) {
	log.WithFields(log.Fields{
		"service": entry.Service,
		"level":   entry.Level,
		"id":      id,
	}).Info("Log saved successfully")
}

func LogError(entry *domain.LogEntry, err error) {
	log.WithFields(log.Fields{
		"service": entry.Service,
		"level":   entry.Level,
		"error":   err,
	}).Error("Failed to save log")
}

--------------------
File: internal\controller\grpc\v1\convertors.go
package grpcv1

import (
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewLogEntryFromRequest(req *loggrpc.SendLogRequest) *domain.LogEntry {
	return &domain.LogEntry{
		Service:   req.Service,
		Level:     req.Level.String(),
		Message:   req.Message,
		Timestamp: req.Timestamp.AsTime(),
	}
}

func NewLogFilterFromRequest(req *loggrpc.GetLogsRequest) *repotypes.LogFilter {
	return &repotypes.LogFilter{
		Service: req.Service,
		Level:   req.Level.String(),
		From:    req.From.AsTime(),
		To:      req.To.AsTime(),
		Limit:   int(req.Limit),
	}
}

func ToSendLogRequest(entry domain.LogEntry) *loggrpc.SendLogRequest {
	var level loggrpc.LogLevel
	switch entry.Level {
	case "INFO":
		level = loggrpc.LogLevel_INFO
	case "WARN":
		level = loggrpc.LogLevel_WARN
	case "ERROR":
		level = loggrpc.LogLevel_ERROR
	default:
		level = loggrpc.LogLevel_LOG_LEVEL_UNSPECIFIED
	}

	return &loggrpc.SendLogRequest{
		Service:   entry.Service,
		Level:     level,
		Message:   entry.Message,
		Timestamp: timestamppb.New(entry.Timestamp),
	}
}

func ServiceStatsToGrpc(stats domain.ServiceStats) *loggrpc.GetStatsResponse {
	resp := &loggrpc.GetStatsResponse{
		TotalLogs: int32(stats.TotalLogs),
	}

	for _, ls := range stats.LogsByLevel {
		resp.LogsByLevel = append(resp.LogsByLevel, &loggrpc.LogLevelCount{
			Level: loggrpc.LogLevel(loggrpc.LogLevel_value[ls.Level]),
			Count: int32(ls.Count),
		})
	}

	return resp
}

--------------------
File: internal\controller\grpc\v1\log_controller.go
package grpcv1

import (
	"context"
	"fmt"

	logginghelper "github.com/Egor213/LogiTrack/internal/controller/common/logging"
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/controller/grpc/validators"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/service"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LogController struct {
	logService service.Log
	counters   *metrics.Counters
	loggrpc.UnimplementedLogServiceServer
}

func NewLogController(ls service.Log, cnt *metrics.Counters) *LogController {
	return &LogController{
		logService: ls,
		counters:   cnt,
	}
}

func (c *LogController) SendLog(ctx context.Context, logRequest *loggrpc.SendLogRequest) (*loggrpc.SendLogResponse, error) {
	logEntry := NewLogEntryFromRequest(logRequest)

	c.counters.GrpcRequests.Inc("SendLog", "received")
	if err := validators.Validate(logEntry); err != nil {
		c.counters.GrpcRequests.Inc("SendLog", "failed")
		logginghelper.LogError(logEntry, err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %s", err)
	}

	logginghelper.LogReceived(logEntry)

	id, err := c.logService.SendLog(ctx, logEntry)
	if err != nil {
		c.counters.GrpcRequests.Inc("SendLog", "failed")
		logginghelper.LogError(logEntry, err)
		return nil, status.Errorf(codes.Unknown, "unknown error")
	}

	logginghelper.LogSaved(logEntry, id)

	c.counters.GrpcRequests.Inc("SendLog", "ok")

	return &loggrpc.SendLogResponse{
		LogId:  fmt.Sprint(id),
		Status: loggrpc.SendStatus_STATUS_OK,
	}, nil
}

func (c *LogController) GetLogs(ctx context.Context, req *loggrpc.GetLogsRequest) (*loggrpc.GetLogsResponse, error) {
	lf := NewLogFilterFromRequest(req)
	logs, err := c.logService.GetLogs(ctx, *lf)
	if err != nil {
		c.counters.GrpcRequests.Inc("SendLog", "failed")
		log.Debug(err.Error())
		return nil, status.Errorf(codes.Unknown, "unknown error")
	}
	log.Info(logs)
	var temp []*loggrpc.SendLogRequest
	for _, val := range logs {
		temp = append(temp, ToSendLogRequest(val))
	}
	return &loggrpc.GetLogsResponse{
		Logs: temp,
	}, nil
}

func (c *LogController) GetStats(ctx context.Context, req *loggrpc.GetStatsRequest) (*loggrpc.GetStatsResponse, error) {
	stats, err := c.logService.GetStats(ctx, req.Service, req.From.AsTime(), req.To.AsTime())
	if err != nil {
		log.Debug(err.Error())
		return nil, status.Errorf(codes.Unknown, "unknown error")
	}
	return ServiceStatsToGrpc(stats), nil
}

--------------------
File: internal\controller\grpc\v1\register.go
package grpcv1

import (
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/service"
	"google.golang.org/grpc"
)

func RegisterServices(services *service.Services, counters *metrics.Counters) func(s *grpc.Server) {
	return func(s *grpc.Server) {
		loggrpc.RegisterLogServiceServer(s, NewLogController(services.Log, counters))
	}
}

--------------------
File: internal\controller\grpc\validators\logs.go
package validators

import (
	"errors"

	"github.com/Egor213/LogiTrack/internal/domain"
)

const (
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

var (
	ErrInvalidLogLevel = errors.New("invalid log level")
	ErrEmptyService    = errors.New("service must be specified")
)

func Validate(l *domain.LogEntry) error {
	switch l.Level {
	case LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel
	}

	if l.Service == "" {
		return ErrEmptyService
	}

	return nil
}

--------------------
File: internal\domain\log.go
package domain

import "time"

type LogEntry struct {
	Id        string    `db:"id"`
	Service   string    `db:"service"`
	Level     string    `db:"level"`
	Message   string    `db:"message"`
	Timestamp time.Time `db:"created_at"`
}

type LevelStats struct {
	Level string
	Count int
}

type ServiceStats struct {
	Service     string
	TotalLogs   int
	LogsByLevel []LevelStats
}

--------------------
File: internal\metrics\metrics.go
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

--------------------
File: internal\metrics\router.go
package metrics

import (
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
)

func ConfigureRouter(handler *echo.Echo) {
	handler.GET("/metrics", echoprometheus.NewHandler())
}

--------------------
File: internal\mocks\counters\mock.go
// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/metrics/metrics.go
//
// Generated by this command:
//
//	mockgen -source=./internal/metrics/metrics.go -destination=./internal/mocks/counters/mock.go -package=countermocks
//

// Package countermocks is a generated GoMock package.
package countermocks

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockCounter is a mock of Counter interface.
type MockCounter struct {
	ctrl     *gomock.Controller
	recorder *MockCounterMockRecorder
	isgomock struct{}
}

// MockCounterMockRecorder is the mock recorder for MockCounter.
type MockCounterMockRecorder struct {
	mock *MockCounter
}

// NewMockCounter creates a new mock instance.
func NewMockCounter(ctrl *gomock.Controller) *MockCounter {
	mock := &MockCounter{ctrl: ctrl}
	mock.recorder = &MockCounterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCounter) EXPECT() *MockCounterMockRecorder {
	return m.recorder
}

// Inc mocks base method.
func (m *MockCounter) Inc(labels ...string) {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range labels {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Inc", varargs...)
}

// Inc indicates an expected call of Inc.
func (mr *MockCounterMockRecorder) Inc(labels ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Inc", reflect.TypeOf((*MockCounter)(nil).Inc), labels...)
}

--------------------
File: internal\mocks\repository\mock.go
// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/repo/repo.go
//
// Generated by this command:
//
//	mockgen -source=./internal/repo/repo.go -destination=./internal/mocks/repository/mock.go -package=repomocks
//

// Package repomocks is a generated GoMock package.
package repomocks

import (
	context "context"
	reflect "reflect"
	time "time"

	domain "github.com/Egor213/LogiTrack/internal/domain"
	repotypes "github.com/Egor213/LogiTrack/internal/repo/repotypes"
	gomock "go.uber.org/mock/gomock"
)

// MockLog is a mock of Log interface.
type MockLog struct {
	ctrl     *gomock.Controller
	recorder *MockLogMockRecorder
	isgomock struct{}
}

// MockLogMockRecorder is the mock recorder for MockLog.
type MockLogMockRecorder struct {
	mock *MockLog
}

// NewMockLog creates a new mock instance.
func NewMockLog(ctrl *gomock.Controller) *MockLog {
	mock := &MockLog{ctrl: ctrl}
	mock.recorder = &MockLogMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLog) EXPECT() *MockLogMockRecorder {
	return m.recorder
}

// GetLogs mocks base method.
func (m *MockLog) GetLogs(ctx context.Context, filter repotypes.LogFilter) ([]domain.LogEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLogs", ctx, filter)
	ret0, _ := ret[0].([]domain.LogEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLogs indicates an expected call of GetLogs.
func (mr *MockLogMockRecorder) GetLogs(ctx, filter any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLogs", reflect.TypeOf((*MockLog)(nil).GetLogs), ctx, filter)
}

// GetStatsByService mocks base method.
func (m *MockLog) GetStatsByService(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStatsByService", ctx, service, from, to)
	ret0, _ := ret[0].(domain.ServiceStats)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStatsByService indicates an expected call of GetStatsByService.
func (mr *MockLogMockRecorder) GetStatsByService(ctx, service, from, to any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStatsByService", reflect.TypeOf((*MockLog)(nil).GetStatsByService), ctx, service, from, to)
}

// SendLog mocks base method.
func (m *MockLog) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendLog", ctx, logObj)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendLog indicates an expected call of SendLog.
func (mr *MockLogMockRecorder) SendLog(ctx, logObj any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendLog", reflect.TypeOf((*MockLog)(nil).SendLog), ctx, logObj)
}

--------------------
File: internal\mocks\service\mock.go
// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/service/service.go
//
// Generated by this command:
//
//	mockgen -source=./internal/service/service.go -destination=./internal/mocks/service/mock.go -package=servicemocks
//

// Package servicemocks is a generated GoMock package.
package servicemocks

import (
	context "context"
	reflect "reflect"
	time "time"

	domain "github.com/Egor213/LogiTrack/internal/domain"
	repotypes "github.com/Egor213/LogiTrack/internal/repo/repotypes"
	gomock "go.uber.org/mock/gomock"
)

// MockLog is a mock of Log interface.
type MockLog struct {
	ctrl     *gomock.Controller
	recorder *MockLogMockRecorder
	isgomock struct{}
}

// MockLogMockRecorder is the mock recorder for MockLog.
type MockLogMockRecorder struct {
	mock *MockLog
}

// NewMockLog creates a new mock instance.
func NewMockLog(ctrl *gomock.Controller) *MockLog {
	mock := &MockLog{ctrl: ctrl}
	mock.recorder = &MockLogMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLog) EXPECT() *MockLogMockRecorder {
	return m.recorder
}

// GetLogs mocks base method.
func (m *MockLog) GetLogs(ctx context.Context, lf repotypes.LogFilter) ([]domain.LogEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLogs", ctx, lf)
	ret0, _ := ret[0].([]domain.LogEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLogs indicates an expected call of GetLogs.
func (mr *MockLogMockRecorder) GetLogs(ctx, lf any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLogs", reflect.TypeOf((*MockLog)(nil).GetLogs), ctx, lf)
}

// GetStats mocks base method.
func (m *MockLog) GetStats(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStats", ctx, service, from, to)
	ret0, _ := ret[0].(domain.ServiceStats)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStats indicates an expected call of GetStats.
func (mr *MockLogMockRecorder) GetStats(ctx, service, from, to any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStats", reflect.TypeOf((*MockLog)(nil).GetStats), ctx, service, from, to)
}

// SendLog mocks base method.
func (m *MockLog) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendLog", ctx, logObj)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendLog indicates an expected call of SendLog.
func (mr *MockLogMockRecorder) SendLog(ctx, logObj any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendLog", reflect.TypeOf((*MockLog)(nil).SendLog), ctx, logObj)
}

--------------------
File: internal\repo\pgdb\log.go
package pgdb

import (
	"context"
	"time"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/Egor213/LogiTrack/pkg/postgres"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type LogRepo struct {
	*postgres.Postgres
}

func NewLogRepo(pg *postgres.Postgres) *LogRepo {
	return &LogRepo{pg}
}

func (r *LogRepo) GetLogs(ctx context.Context, filter repotypes.LogFilter) ([]domain.LogEntry, error) {
	//TODO Нормально настроить фильтры, чтобы без указания уровня можно было и время корректное сделать
	conds, limit := BuildLogQueryFilters(filter)

	query := r.Builder.
		Select("id", "service", "level", "message", "created_at").
		From("logs").
		Limit(limit)

	if len(conds) > 0 {
		query = query.Where(sq.And(conds))
	}

	sql, args, _ := query.ToSql()
	rows, err := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool).Query(ctx, sql, args...)

	if err != nil {
		return []domain.LogEntry{}, errorsUtils.WrapPathErr(err)
	}
	defer rows.Close()

	// Рекомендуют использовать курсор, ибо это жрет память
	logs, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.LogEntry])

	if err != nil {
		return []domain.LogEntry{}, errorsUtils.WrapPathErr(err)
	}

	return logs, nil
}

func (r *LogRepo) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	sql, args, _ := r.Builder.
		Insert("logs").
		Columns("service", "level", "message").
		Values(logObj.Service, logObj.Level, logObj.Message).
		Suffix("RETURNING id").
		ToSql()

	var id int
	err := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool).QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		return 0, errorsUtils.WrapPathErr(err)
	}
	return id, nil
}

func (r *LogRepo) GetStatsByService(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error) {
	conds := BuildServiceStatsQueryFilters(service, from, to)
	query := r.Builder.
		Select("level", "COUNT(*) AS count_logs").
		From("logs")

	if len(conds) > 0 {
		query = query.Where(sq.And(conds))
	}
	query = query.GroupBy("level")

	sql, args, err := query.ToSql()
	if err != nil {
		return domain.ServiceStats{}, errorsUtils.WrapPathErr(err)
	}

	rows, err := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool).Query(ctx, sql, args...)
	if err != nil {
		return domain.ServiceStats{}, errorsUtils.WrapPathErr(err)
	}
	defer rows.Close()

	stats := domain.ServiceStats{Service: service}

	for rows.Next() {
		var ls domain.LevelStats
		if err := rows.Scan(&ls.Level, &ls.Count); err != nil {
			return domain.ServiceStats{}, errorsUtils.WrapPathErr(err)
		}
		stats.LogsByLevel = append(stats.LogsByLevel, ls)
		stats.TotalLogs += ls.Count
	}

	if err := rows.Err(); err != nil {
		return domain.ServiceStats{}, errorsUtils.WrapPathErr(err)
	}

	return stats, nil
}

--------------------
File: internal\repo\pgdb\utils.go
package pgdb

import (
	"time"

	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	sq "github.com/Masterminds/squirrel"
)

func BuildLogQueryFilters(filter repotypes.LogFilter) ([]sq.Sqlizer, uint64) {
	conds := []sq.Sqlizer{}

	if filter.Service != "" {
		conds = append(conds, sq.Eq{"service": filter.Service})
	}

	if filter.Level != "" && filter.Level != "LOG_LEVEL_UNSPECIFIED" {
		conds = append(conds, sq.Eq{"level": filter.Level})
	}
	if !filter.From.IsZero() && !filter.From.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.GtOrEq{"created_at": filter.From})
	}
	if !filter.To.IsZero() && !filter.To.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.LtOrEq{"created_at": filter.To})
	}

	limit := uint64(100)
	if filter.Limit > 0 {
		limit = uint64(filter.Limit)
	}

	return conds, limit
}

func BuildServiceStatsQueryFilters(service string, from, to time.Time) []sq.Sqlizer {
	conds := []sq.Sqlizer{}
	if service != "" {
		conds = append(conds, sq.Eq{"service": service})
	}

	if !from.IsZero() && !from.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.GtOrEq{"created_at": from})
	}
	if !to.IsZero() && !from.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.LtOrEq{"created_at": to})
	}
	return conds
}

--------------------
File: internal\repo\repo.go
package repo

import (
	"context"
	"time"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/pgdb"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	"github.com/Egor213/LogiTrack/pkg/postgres"
)

type Log interface {
	GetLogs(ctx context.Context, filter repotypes.LogFilter) ([]domain.LogEntry, error)
	SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error)
	GetStatsByService(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error)
}

type Repositories struct {
	Log
}

func NewRepositories(pg *postgres.Postgres) *Repositories {
	return &Repositories{
		Log: pgdb.NewLogRepo(pg),
	}
}

--------------------
File: internal\repo\repoerrs\errors.go
package repoerrs

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")

	ErrNotEnoughBalance = errors.New("not enough balance")
)

--------------------
File: internal\repo\repotypes\log.go
package repotypes

import (
	"time"
)

// кажется, что это все таки бизнес сущность
type LogFilter struct {
	Service string
	Level   string
	From    time.Time
	To      time.Time
	Limit   int
}

--------------------
File: internal\service\errors.go
package service

import "fmt"

var (
	ErrLogAlreadyExists = fmt.Errorf("log already exists")
	ErrCannotCreateLog  = fmt.Errorf("cannot create log")
)

--------------------
File: internal\service\log.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Egor213/LogiTrack/internal/broker"
	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/labstack/gommon/log"
)

type LogService struct {
	logRepo        repo.Log
	counters       *metrics.Counters
	brokerProducer broker.Producer
}

func NewLogService(lr repo.Log, cnt *metrics.Counters, p broker.Producer) *LogService {
	return &LogService{
		logRepo:        lr,
		counters:       cnt,
		brokerProducer: p,
	}
}

func (s *LogService) GetLogs(ctx context.Context, lf repotypes.LogFilter) ([]domain.LogEntry, error) {
	logs, err := s.logRepo.GetLogs(ctx, lf)
	log.Info(logs, err)
	if err != nil {
		return []domain.LogEntry{}, err
	}
	return logs, nil
}

func (s *LogService) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	s.brokerProducer.SendMessage(
		ctx,
		[]byte(fmt.Sprintf("service=%s, level=%s, Message=%s", logObj.Service, logObj.Level, logObj.Message)),
	)
	s.counters.LogsReceived.Inc(logObj.Service, logObj.Level)
	id, err := s.logRepo.SendLog(ctx, logObj)
	if err != nil {
		return 0, errorsUtils.WrapPathErr(ErrCannotCreateLog)
	}
	return id, nil
}

func (s *LogService) GetStats(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error) {
	stats, err := s.logRepo.GetStatsByService(ctx, service, from, to)
	if err != nil {
		return domain.ServiceStats{}, errorsUtils.WrapPathErr(ErrCannotCreateLog)
	}
	return stats, nil
}

--------------------
File: internal\service\log_test.go
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	repository_mock "github.com/Egor213/LogiTrack/internal/mocks/repository"
	"github.com/Egor213/LogiTrack/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLogService_GetStats(t *testing.T) {
	type args struct {
		ctx     context.Context
		service string
		from    time.Time
		to      time.Time
	}

	type mockBehavior func(r *repository_mock.MockLog, args args)

	now := time.Now()
	testCases := []struct {
		name         string
		args         args
		mockBehavior mockBehavior
		want         domain.ServiceStats
		wantErr      bool
	}{
		{
			name: "success",
			args: args{
				ctx:     context.Background(),
				service: "auth",
				from:    now.Add(-time.Hour),
				to:      now,
			},
			mockBehavior: func(r *repository_mock.MockLog, args args) {
				r.EXPECT().
					GetStatsByService(args.ctx, args.service, args.from, args.to).
					Return(domain.ServiceStats{
						Service:   "auth",
						TotalLogs: 100,
						LogsByLevel: []domain.LevelStats{
							{"INFO", 80},
							{"ERROR", 20},
						},
					}, nil)
			},
			want: domain.ServiceStats{
				Service:   "auth",
				TotalLogs: 100,
				LogsByLevel: []domain.LevelStats{
					{"INFO", 80},
					{"ERROR", 20},
				},
			},
			wantErr: false,
		},
		{
			name: "repository error",
			args: args{
				ctx:     context.Background(),
				service: "billing",
				from:    now.Add(-2 * time.Hour),
				to:      now,
			},
			mockBehavior: func(r *repository_mock.MockLog, args args) {
				r.EXPECT().
					GetStatsByService(args.ctx, args.service, args.from, args.to).
					Return(domain.ServiceStats{}, errors.New("db error"))
			},
			want:    domain.ServiceStats{},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository_mock.NewMockLog(ctrl)
			tc.mockBehavior(mockRepo, tc.args)

			cnt := metrics.NewTestCounters()
			s := service.NewLogService(mockRepo, cnt)

			got, err := s.GetStats(tc.args.ctx, tc.args.service, tc.args.from, tc.args.to)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLogService_SendLog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository_mock.NewMockLog(ctrl)
	mockCounters := metrics.NewTestCounters()

	svc := service.NewLogService(mockRepo, mockCounters)

	ctx := context.Background()
	logEntry := &domain.LogEntry{
		Service:   "auth",
		Level:     "WARN",
		Message:   "something happened",
		Timestamp: time.Now(),
	}

	tcs := []struct {
		name         string
		mockBehavior func()
		wantID       int
		wantErr      bool
	}{
		{
			name: "success",
			mockBehavior: func() {
				mockRepo.EXPECT().
					SendLog(ctx, logEntry).
					Return(42, nil)
			},
			wantID:  42,
			wantErr: false,
		},
		{
			name: "repository error",
			mockBehavior: func() {
				mockRepo.EXPECT().
					SendLog(ctx, logEntry).
					Return(0, errors.New("db error"))
			},
			wantID:  0,
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockBehavior()

			gotID, err := svc.SendLog(ctx, logEntry)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantID, gotID)
		})
	}
}

--------------------
File: internal\service\service.go
package service

import (
	"context"
	"time"

	"github.com/Egor213/LogiTrack/internal/broker"
	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
)

type Log interface {
	GetLogs(ctx context.Context, lf repotypes.LogFilter) ([]domain.LogEntry, error)
	SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error)
	GetStats(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error)
}

type Services struct {
	Log
}

type ServicesDependencies struct {
	Repos          *repo.Repositories
	Counters       *metrics.Counters
	BrokerProducer broker.Producer
}

func NewServices(deps ServicesDependencies) *Services {
	return &Services{
		Log: NewLogService(deps.Repos.Log, deps.Counters, deps.BrokerProducer),
	}
}

--------------------
File: pkg\errors\errors.go
package errorsUtils

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	CodeUniqueViolation     = "23505"
	CodeForeignKeyViolation = "23503"
	CodeNotNullViolation    = "23502"
)

func Is(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}

func IsUniqueViolation(err error) bool {
	return Is(err, CodeUniqueViolation)
}

func IsForeignKeyViolation(err error) bool {
	return Is(err, CodeForeignKeyViolation)
}

func IsNotNullViolation(err error) bool {
	return Is(err, CodeNotNullViolation)
}

func WrapPathErr(err error) error {
	pc, _, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc).Name()
	return fmt.Errorf("[%s:%d] %w", fn, line, err)
}

--------------------
File: pkg\grpcserver\options.go
package grpcserver

import (
	"net"
	"time"
)

type Option func(*Server)

func WithPort(port string) Option {
	return func(s *Server) {
		listener, err := net.Listen("tcp", ":"+port)
		if err == nil {
			s.listener = listener
		}
	}
}

func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = d
	}
}

--------------------
File: pkg\grpcserver\server.go
package grpcserver

import (
	"net"
	"time"

	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"

	"google.golang.org/grpc"
)

const (
	defaultAddr            = ":50051"
	defaultShutdownTimeout = 3 * time.Second
)

type Server struct {
	Server          *grpc.Server
	listener        net.Listener
	notify          chan error
	shutdownTimeout time.Duration
}

func New(register func(*grpc.Server), opts ...Option) (*Server, error) {
	listener, err := net.Listen("tcp", defaultAddr)

	if err != nil {
		return nil, errorsUtils.WrapPathErr(err)
	}

	grpcServer := grpc.NewServer()

	register(grpcServer)

	s := &Server{
		Server:          grpcServer,
		listener:        listener,
		notify:          make(chan error, 1),
		shutdownTimeout: defaultShutdownTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.start()

	return s, nil
}

func (s *Server) start() {
	go func() {
		s.notify <- s.Server.Serve(s.listener)
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() {
	done := make(chan any)
	go func() {
		s.Server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(s.shutdownTimeout):
		s.Server.Stop()
	}
}

--------------------
File: pkg\httpserver\options.go
package httpserver

import (
	"net"
	"time"
)

type Option func(*Server)

func Port(port string) Option {
	return func(s *Server) {
		s.server.Addr = net.JoinHostPort("", port)
	}
}

func ReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.server.ReadTimeout = timeout
	}
}

func WriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.server.WriteTimeout = timeout
	}
}

func ShutdownTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}

--------------------
File: pkg\httpserver\server.go
package httpserver

import (
	"context"
	"net/http"
	"time"
)

const (
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 5 * time.Second
	defaultAddr            = ":80"
	defaultShutdownTimeout = 3 * time.Second
)

type Server struct {
	server          *http.Server
	notify          chan error
	shutdownTimeout time.Duration
}

func New(handler http.Handler, opts ...Option) *Server {
	httpServer := &http.Server{
		Handler:      handler,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		Addr:         defaultAddr,
	}

	s := &Server{
		server:          httpServer,
		notify:          make(chan error, 1),
		shutdownTimeout: defaultShutdownTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.start()

	return s
}

func (s *Server) start() {
	go func() {
		s.notify <- s.server.ListenAndServe()
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	return s.server.Shutdown(ctx)
}

--------------------
File: pkg\logger\logger.go
package logger

import (
	"fmt"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"
)

func SetupLogger(level string) {
	loggerLevel, err := log.ParseLevel(level)
	log.SetReportCaller(true)

	log.SetFormatter(&log.JSONFormatter{
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return "", fmt.Sprintf("%s:%d", path.Base(frame.File), frame.Line)
		},
		TimestampFormat: "2006-01-02 15:04:05",
	})

	if err != nil {
		log.Infof("Level setup default INFO, err: %v", err)
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(loggerLevel)
	}

}

--------------------
File: pkg\postgres\options.go
package postgres

import "time"

type Option func(*Postgres)

func MaxPoolSize(size int) Option {
	return func(p *Postgres) {
		p.maxPoolSize = size
	}
}

func ConnAttempts(attempts int) Option {
	return func(p *Postgres) {
		p.connAttempts = attempts
	}
}

func ConnTimeout(timeout time.Duration) Option {
	return func(p *Postgres) {
		p.connTimeout = timeout
	}
}

--------------------
File: pkg\postgres\postgres.go
package postgres

import (
	"context"
	"time"

	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"

	"github.com/Masterminds/squirrel"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultMaxPoolSize  = 1
	DefaultConnAttempts = 10
	DefaultConnTimeout  = time.Second
)

type PgxPool interface {
	Close()
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	Ping(ctx context.Context) error
}

type Postgres struct {
	maxPoolSize  int
	connAttempts int
	connTimeout  time.Duration

	Builder   squirrel.StatementBuilderType
	CtxGetter *trmpgx.CtxGetter
	Pool      PgxPool
}

func New(pgUrl string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  DefaultMaxPoolSize,
		connAttempts: DefaultConnAttempts,
		connTimeout:  DefaultConnTimeout,
		CtxGetter:    trmpgx.DefaultCtxGetter,
		Builder:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(pgUrl)

	if err != nil {
		return nil, errorsUtils.WrapPathErr(err)
	}

	poolConfig.MaxConns = int32(pg.maxPoolSize)

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)

		if err != nil {
			return nil, errorsUtils.WrapPathErr(err)
		}

		if err = pg.Pool.Ping(context.Background()); err == nil {
			break
		}

		pg.connAttempts--
		log.Infof("Postgres trying to connect, attempts left: %d", pg.connAttempts)
		time.Sleep(pg.connTimeout)
	}

	// Походу еще это надо заюзать, но https://github.com/avito-tech/go-transaction-manager/blob/main/drivers/pgxv5/example_test.go
	// trManager := manager.Must(trmpgx.NewDefaultFactory(pg.Pool))
	// pg.trManager

	if err != nil {
		return nil, errorsUtils.WrapPathErr(err)
	}

	return pg, nil
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

--------------------
