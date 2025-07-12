package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/kgrsutos/cw-railspathmetrics/internal/cli"
)

// App represents the main application with its dependencies
type App struct {
	executeFunc func() error
	exitFunc    func(int)
	logger      *slog.Logger
}

// NewApp creates a new App with default dependencies
func NewApp() *App {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return &App{
		executeFunc: cli.Execute,
		exitFunc:    os.Exit,
		logger:      logger,
	}
}

// NewAppWithDeps creates a new App with custom dependencies for testing
func NewAppWithDeps(executeFunc func() error, exitFunc func(int), logWriter io.Writer) *App {
	logger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return &App{
		executeFunc: executeFunc,
		exitFunc:    exitFunc,
		logger:      logger,
	}
}

// Run executes the application
func (a *App) Run() {
	slog.SetDefault(a.logger)

	if err := a.executeFunc(); err != nil {
		slog.Error("Failed to execute command", "error", err)
		a.exitFunc(1)
	}
}

func main() {
	app := NewApp()
	app.Run()
}
