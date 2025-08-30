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
