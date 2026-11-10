package db

import (
	"testing"

	"gorm.io/gorm/logger"
)

func TestGormLogLevel(t *testing.T) {
	cases := map[string]logger.LogLevel{
		"":         logger.Error,
		"error":    logger.Error,
		"nonsense": logger.Error,
		"silent":   logger.Silent,
		" WARN ":   logger.Warn,
		"Info":     logger.Info,
	}
	for in, want := range cases {
		t.Setenv("GORM_LOG_LEVEL", in)
		if got := gormLogLevel(); got != want {
			t.Errorf("GORM_LOG_LEVEL=%q: got %v, want %v", in, got, want)
		}
	}
}

func TestGormLoggerFromEnvNotNil(t *testing.T) {
	t.Setenv("GORM_LOG_LEVEL", "silent")
	if gormLoggerFromEnv() == nil {
		t.Fatal("gormLoggerFromEnv() returned nil")
	}
}
