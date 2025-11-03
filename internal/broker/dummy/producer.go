package dummybroker

import (
	"context"
)

type Producer struct {
}

func NewProducer() *Producer {
	return &Producer{}
}

func (p *Producer) SendMessage(ctx context.Context, value []byte) error {
	return nil
}

func (p *Producer) Close() error {
	return nil
}
