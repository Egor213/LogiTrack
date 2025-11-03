package broker

import "context"

type Producer interface {
	SendMessage(ctx context.Context, value []byte) error
	Close() error
}
