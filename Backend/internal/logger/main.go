package logger

import (
	"log/slog"
	"os"
)

// Global logger will be initialized in main
var Logger *slog.Logger

func InitLogger(env string) {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}
