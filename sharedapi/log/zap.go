package log

import (
	"fmt"
	"os"

	"github.com/cresta/zapctx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(zapLogger *zap.Logger, dynamicFields ...zapctx.DynamicFields) *zapctx.Logger {
	ret := zapctx.New(zapLogger)
	for _, df := range dynamicFields {
		ret = ret.DynamicFields(df)
	}
	return ret
}

func ZapLoggerFromConfig(c zap.Config) (*zap.Logger, error) {
	l, err := c.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}
	return l, nil
}

func ZapLoggerConfigFromEnv() (zap.Config, error) {
	ret := zap.NewDevelopmentConfig()
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	l, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return ret, fmt.Errorf("failed to parse log_level %s: %w", logLevel, err)
	}
	ret.Level = zap.NewAtomicLevelAt(l)
	return ret, nil
}
