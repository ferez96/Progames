package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
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

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		zap.L().Warn("docker.unavailable", zap.Error(err))
		dockerCli = nil
	}

	authSvc := auth.New(st, cfg)
	eventStore := events.New(st)
	submissionSvc := submission.New(st, cfg, dockerCli)
	matchSvc := matchsvc.New(st, eventStore, cfg, dockerCli)
	matchQueue := matchsvc.NewQueue(matchSvc, cfg.QueueCap)
	server := web.New(st, authSvc, submissionSvc, matchQueue)

	srv := &http.Server{Addr: cfg.Addr, Handler: server.Routes()}

	go func() {
		zap.L().Info("app.start", zap.String("addr", cfg.Addr), zap.String("db", cfg.DBPath))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	zap.L().Info("app.shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Error("http.shutdown", zap.Error(err))
	}

	matchQueue.Shutdown()
	zap.L().Info("app.stopped")
}
