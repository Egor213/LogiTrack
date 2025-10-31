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
