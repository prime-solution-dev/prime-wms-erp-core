package db

import (
	"os"
	"strings"

	"gorm.io/gorm/logger"
)

// gormLogLevel maps the GORM_LOG_LEVEL env var (silent|error|warn|info) to a
// GORM log level. It defaults to Error so GORM's built-in "SLOW SQL" warnings
// stay out of the logs while real SQL errors are still reported.
func gormLogLevel() logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GORM_LOG_LEVEL"))) {
	case "silent":
		return logger.Silent
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Error
	}
}

// gormLoggerFromEnv builds the GORM logger configured by GORM_LOG_LEVEL.
func gormLoggerFromEnv() logger.Interface {
	return logger.Default.LogMode(gormLogLevel())
}
