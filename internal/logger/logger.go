package logger

import (
	"encoding/json"
	"log"
	"os"

	"go.uber.org/zap"
)

var Logger *zap.Logger

func InitLogger(logpath string) *zap.Logger {
	configFile, err := os.ReadFile(logpath)
	if err != nil {
		log.Fatal("reading log config file error ", err)
	}

	var cfg zap.Config
	if err = json.Unmarshal(configFile, &cfg); err != nil {
		log.Fatal("json read error ", err)
	}

	localLogger, err := cfg.Build()
	if err != nil {
		log.Fatal("logger init ", err)
	}

	Logger = localLogger

	return Logger
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}
