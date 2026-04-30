package obs

import (
	"expvar"

	"go.uber.org/zap"
)

var (
	MatchesCompleted    = expvar.NewInt("matches_completed")
	MatchesFailed       = expvar.NewInt("matches_failed")
	SubmissionsCompiled = expvar.NewInt("submissions_compiled")
	SubmissionsInvalid  = expvar.NewInt("submissions_invalid")
	LoginsSuccess       = expvar.NewInt("logins_success")
	LoginsFailure       = expvar.NewInt("logins_failure")
)

func Init() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	zap.ReplaceGlobals(logger)
	return logger, nil
}
