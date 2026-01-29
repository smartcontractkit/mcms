package sdk

import (
	"context"

	"go.uber.org/zap"
)

type Logger interface {
	Infof(template string, args ...any)
}

type contextLoggerValueT string

const ContextLoggerValue = contextLoggerValueT("mcms-logger")

func LoggerFrom(ctx context.Context) Logger {
	value := ctx.Value(ContextLoggerValue)
	logger, ok := value.(Logger)
	if !ok {
		logger = zap.Must(zap.NewProduction()).Sugar()
	}

	return logger
}
