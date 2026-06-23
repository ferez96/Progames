package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"progames/internal/auth"
	"progames/internal/config"
	"progames/internal/events"
	"progames/internal/matchexec"
	"progames/internal/obs"
	"progames/internal/service"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	authSvc := auth.New(st, cfg)
	eventStore := events.New(st)
	submissionSvc := submission.New(st, cfg, dockerCli)
	matchProcessor := matchexec.NewProcessor(st, eventStore, cfg, dockerCli)
	matchQueue := matchexec.NewQueue(ctx, matchProcessor, cfg.QueueCap)
	practiceSvc := service.NewPractice(st, submissionSvc, matchQueue)
	matchSvc := service.NewMatch(st)
	gameSvc := service.NewGame(st)
	server := web.New(authSvc, practiceSvc, matchSvc, gameSvc)

	srv := &http.Server{Addr: cfg.Addr, Handler: server.Routes()}

	go func() {
		zap.L().Info("app.start", zap.String("addr", cfg.Addr), zap.String("db", cfg.DBPath))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen", zap.Error(err))
		}
	}()

	<-ctx.Done()
	stop() // release signal resources

	zap.L().Info("app.shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("http.shutdown", zap.Error(err))
	}

	matchQueue.Wait()
	zap.L().Info("app.stopped")
}
