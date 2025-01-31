package main

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log/slog"
	"os/signal"
	"sap_segmentation/internal/config"
	"sap_segmentation/internal/logger"
	"sap_segmentation/internal/service"
	"syscall"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	onDone, err := logger.Setup("log/segmentation_import.log", cfg.LogCleanupMaxAge)
	if err != nil {
		panic(err)
	}
	defer onDone()

	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName))
	if err != nil {
		panic(err)
	}
	defer func(db *sqlx.DB) {
		if err := db.Close(); err != nil {
			slog.Error("db close error", "err", err)
		}
	}(db)

	importService := service.NewImportService(db, cfg, ctx)
	if err = importService.ImportData(); err != nil {
		slog.Error("failed to import data", "err", err)
	}
}
