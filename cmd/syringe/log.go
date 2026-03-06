package main

import (
	slog "github.com/ZentriaMC/syringe/internal/log"
)

func setupLogging(debugMode bool) error {
	return slog.SetupLogging(debugMode)
}
