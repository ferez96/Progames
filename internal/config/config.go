package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr               string
	DBPath             string
	ArtifactDir        string
	PerMoveTimeout     time.Duration
	MaxSourceBytes     int64
	MaxStdoutLineBytes int
	MaxLogBytes        int
	QueueCap           int
	BotMemoryBytes     int64
	BotNanoCPUs        int64
	SessionTTL         time.Duration
	ForceSecureCookie  bool
	DockerImagePrefix  string
	GoBuilderImage     string
}

func Load() Config {
	return Config{
		Addr:               envString("PROGAMES_ADDR", ":8080"),
		DBPath:             envString("PROGAMES_DB", "./progames.db"),
		ArtifactDir:        envString("PROGAMES_ARTIFACTS", "./artifacts"),
		PerMoveTimeout:     envDuration("PROGAMES_PER_MOVE_TIMEOUT", 5*time.Second),
		MaxSourceBytes:     int64(envInt("PROGAMES_MAX_SOURCE_BYTES", 256*1024)),
		MaxStdoutLineBytes: envInt("PROGAMES_MAX_STDOUT_LINE_BYTES", 64),
		MaxLogBytes:        envInt("PROGAMES_MAX_LOG_BYTES", 1024*1024),
		QueueCap:           envInt("PROGAMES_QUEUE_CAP", 4),
		BotMemoryBytes:     int64(envInt("PROGAMES_BOT_MEMORY_BYTES", 64*1024*1024)),
		BotNanoCPUs:        int64(envInt("PROGAMES_BOT_NANO_CPUS", 500_000_000)),
		SessionTTL:         envDuration("PROGAMES_SESSION_TTL", 24*time.Hour),
		ForceSecureCookie:  envBool("PROGAMES_FORCE_SECURE_COOKIE", false),
		DockerImagePrefix:  envString("PROGAMES_DOCKER_IMAGE_PREFIX", "progames/bot"),
		GoBuilderImage:     envString("PROGAMES_GO_BUILDER_IMAGE", "golang:1.26"),
	}
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
