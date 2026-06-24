package testhelper

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/moby/moby/client"

	"progames/internal/artifact"
	"progames/internal/config"
	"progames/internal/store"
)

func NewDockerClient(t *testing.T) *client.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("requires Docker")
	}
	cli, err := client.New(client.FromEnv)
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	if _, err := cli.Ping(context.Background(), client.PingOptions{}); err != nil {
		_ = cli.Close()
		t.Skipf("docker daemon unreachable: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func NewStore(t *testing.T, cfg config.Config) *store.Store {
	t.Helper()
	st, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return st
}

func NewArtifactRepo(t *testing.T) *artifact.LocalRepository {
	t.Helper()
	if testing.Short() {
		t.Skip("requires local filesystem")
	}
	return artifact.NewLocalRepository(t.TempDir())
}

func TestConfig(t *testing.T) config.Config {
	t.Helper()
	base := t.TempDir()
	return config.Config{
		DBPath:             filepath.Join(base, "progames.db"),
		ArtifactDir:        filepath.Join(base, "artifacts"),
		MaxSourceBytes:     256 * 1024,
		PerMoveTimeout:     time.Second,
		MaxStdoutLineBytes: 64 * 1024,
		MaxLogBytes:        1024 * 1024,
		SessionTTL:         time.Hour,
		GoBuilderImage:     "golang:1.26",
		DockerImagePrefix:  "progames/bot",
	}
}
