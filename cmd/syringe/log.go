package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func setupLogging(debugMode bool) (err error) {
	var cfg zap.Config

	if debugMode {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(zapcore.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.Development = false
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zapcore.InfoLevel)
	}

	cfg.OutputPaths = []string{
		"stdout",
	}

	var logger *zap.Logger
	if logger, err = cfg.Build(); err != nil {
		return err
	}

	_ = zap.ReplaceGlobals(logger)
	return
}
