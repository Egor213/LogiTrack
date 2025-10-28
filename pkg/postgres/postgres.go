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
