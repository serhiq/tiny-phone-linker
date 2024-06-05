package logger

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
)

var log *Logger

func init() {
	if log == nil {
		var isProduction = false
		if strings.ToLower(strings.TrimSpace(os.Getenv("IS_PRODUCTION"))) == "true" {
			isProduction = true
		}

		newLogger, err := initLogger(isProduction)
		if err != nil {
			panic(err)
		}

		log = &Logger{logger: newLogger}
	}
}

func GetLogger() *zap.Logger {
	return log.logger
}

func Debug(msg string, fields ...zap.Field) {
	log.logger.Debug(msg, fields...)

}

func Info(msg string, fields ...zap.Field) {
	log.logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	log.logger.Warn(msg, fields...)
}
func WarnWith(message string, obj interface{}, fields ...zap.Field) {
	log.logger.Error(fmt.Sprintf("%s : %v", message, obj), fields...)
}
func Error(err error, fields ...zap.Field) {
	log.logger.Error(err.Error(), fields...)
}

func ErrorWith(obj interface{}, fields ...zap.Field) {
	log.logger.Error(fmt.Sprintf("%v", obj), fields...)
}

func Sync() {
	log.logger.Sync()
}

func Fatal(s string) {
	log.logger.Fatal(s)
}
