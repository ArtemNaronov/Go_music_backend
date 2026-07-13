package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/temic/go-music/internal/app"
	"github.com/temic/go-music/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := app.New(cfg, log.Logger)
	if err := server.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

	<-ctx.Done()

	if err := server.Stop(); err != nil {
		log.Error().Err(err).Msg("http server shutdown failed")
	}
}
