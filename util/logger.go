package util

import (
	"os"

	"github.com/nitrous-io/nitrous-common/logger"
)

var Logger = &logger.Logger{Level: logger.LogLevelDebug}

func init() {
	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG":
		Logger.Level = logger.LogLevelDebug
	case "INFO":
		Logger.Level = logger.LogLevelInfo
	case "WARN":
		Logger.Level = logger.LogLevelWarn
	case "ERROR":
		Logger.Level = logger.LogLevelError
	case "FATAL":
		Logger.Level = logger.LogLevelFatal
	default:
		Logger.Level = logger.LogLevelDebug
	}
}
