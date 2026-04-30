package main

import (
	"log"
	"net/http"

	"go.uber.org/zap"

	"progames/internal/auth"
	"progames/internal/config"
	"progames/internal/events"
	matchsvc "progames/internal/match"
	"progames/internal/obs"
	"progames/internal/store"
	"progames/internal/submission"
	"progames/internal/web"
)

func main() {
	logger, err := obs.Init()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	cfg := config.Load()
	st, err := store.Open(cfg)
	if err != nil {
		zap.L().Fatal("open store", zap.Error(err))
	}
	defer func() {
		if err := st.Close(); err != nil {
			zap.L().Error("close store", zap.Error(err))
		}
	}()

	authSvc := auth.New(st, cfg)
	eventStore := events.New(st)
	submissionSvc := submission.New(st, cfg)
	matchSvc := matchsvc.New(st, eventStore, cfg)
	server := web.New(st, authSvc, submissionSvc, matchSvc)

	zap.L().Info("app.start", zap.String("addr", cfg.Addr), zap.String("db", cfg.DBPath))
	if err := http.ListenAndServe(cfg.Addr, server.Routes()); err != nil {
		zap.L().Fatal("listen", zap.Error(err))
	}
}
