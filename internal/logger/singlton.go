package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
}

func initLogger(isProduction bool) (*zap.Logger, error) {
	var config zap.Config
	if isProduction {
		config = zap.NewProductionConfig()
	} else {
		config = createDevConfig()
	}

	return config.Build()
}

func createDevConfig() zap.Config {
	return zap.Config{
		Encoding:      "console",
		OutputPaths:   []string{"stdout"},
		EncoderConfig: devEncoderConfig(),
		Level:         zap.NewAtomicLevelAt(zapcore.DebugLevel),
	}

}

func devEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    shortLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func shortLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(level.CapitalString()[0:1])
}
