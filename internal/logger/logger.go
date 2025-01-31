package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

func Setup(logPath string, maxAge int) (func(), error) {
	if err := cleanup(maxAge); err != nil {
		return nil, fmt.Errorf("failed to cleanup old logs: %v", err)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create log file: %v", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, file)
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(multiWriter, nil)))

	return func() {
		slog.SetDefault(old)
		if err := file.Close(); err != nil {
			slog.Error("failed to close log file: %v", err)
		}
	}, nil
}

func cleanup(maxAge int) error {
	files, err := os.ReadDir("log")
	if err != nil {
		return fmt.Errorf("failed to read log directory: %v", err)
	}
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return fmt.Errorf("failed to get log file info: %v", err)
		}

		if time.Since(info.ModTime()) > time.Duration(maxAge)*24*time.Hour {
			if err = os.Remove("log/" + file.Name()); err != nil {
				return fmt.Errorf("failed to remove old log file: %v", err)
			}
		}
	}
	return nil
}
