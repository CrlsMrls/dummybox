package main

import (
	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/logger"
	"github.com/crlsmrls/dummybox/metrics"
	"github.com/crlsmrls/dummybox/server"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	logger.InitLogger(cfg.LogLevel, nil)

	log.Info().Interface("config", cfg).Msg("configuration loaded")

	reg := metrics.InitMetrics()

	srv := server.New(cfg, nil, reg)
	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("server stopped with error")
	}
}
