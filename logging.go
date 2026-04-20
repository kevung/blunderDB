package main

import (
	"log/slog"
	"os"
)

func initLogging(mode string) {
	level := slog.LevelInfo
	if mode == "gui" {
		level = slog.LevelWarn
	}
	if os.Getenv("BLUNDERDB_DEBUG") == "1" {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
