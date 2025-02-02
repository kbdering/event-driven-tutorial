package logging

import (
	"go.uber.org/zap"
)

var logger *zap.Logger

func InitLogger() error {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		return err
	}
	return nil
}

func GetLogger() *zap.Logger {
	return logger
}

func Sync() error {
	return logger.Sync()
}
