package internal

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger() *zap.Logger {
	var config zap.Config
	config = zap.NewDevelopmentConfig()
	config.Development = false
	config.OutputPaths = []string{"stdout"}
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)

	zapLogger, err := config.Build()
	if err != nil {
		panic(err)
	}

	// Return our wrapped zap logger
	return zapLogger
}
