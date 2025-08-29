package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Egor213/LogiTrack/internal/config"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/service"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/Egor213/LogiTrack/pkg/logger"
	"github.com/Egor213/LogiTrack/pkg/postgres"

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

	// Services
	deps := service.ServicesDependencies{
		Repos: repositories,
	}
	services := service.NewServices(deps)

	// Waiting signal
	log.Info("Configuring graceful shutdown")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		log.Error(errorsUtils.WrapPathErr(err))
	}

	// Graceful shutdown
	log.Info("Shutting down")
	err = httpServer.Shutdown()
	if err != nil {
		log.Error(errorsUtils.WrapPathErr(err))
	}

}
