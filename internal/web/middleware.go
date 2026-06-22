package web

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		zap.L().Info("http",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", sr.status),
			zap.Int64("dur_ms", time.Since(start).Milliseconds()),
		)
	})
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
