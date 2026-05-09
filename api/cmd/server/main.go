package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trade-buddy/api/internal/config"
	"trade-buddy/api/internal/db"
	httpapi "trade-buddy/api/internal/http"
	"trade-buddy/api/internal/marketdata"
	"trade-buddy/api/internal/monitor"
	"trade-buddy/api/internal/stream"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	gormDB, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database connect: %w", err)
	}
	slog.Info("database connected")

	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("database migrate: %w", err)
	}
	slog.Info("database migrations applied")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	monitorCfg := monitor.NewConfigService(gormDB, 60*time.Second)
	monitorDefaults, _ := monitorCfg.Load(ctx)
	monitorSource, err := marketdata.NewSourceFromName(cfg.DefaultSource, cfg.FinnhubAPIKey)
	if err != nil {
		slog.Warn("monitor source unavailable, defaulting to yahoo", "error", err)
		monitorSource, _ = marketdata.NewSourceFromName("yahoo", "")
	}
	signalSvc := monitor.NewSignalService(gormDB, 5*time.Minute)
	lineToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	lineToID := os.Getenv("LINE_TO_ID")
	notifier := monitor.NewNotifier(lineToken, lineToID)
	mon := monitor.New(gormDB, monitorSource, monitorCfg, signalSvc, slog.Default(), notifier)
	go func() {
		if err := mon.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("monitor stopped", "error", err)
		}
	}()
	_ = monitorDefaults
	slog.Info("monitor started")

	hub := stream.NewHub()
	builder := stream.NewBuilder(hub)
	if cfg.FinnhubAPIKey != "" {
		go stream.RunFinnhub(ctx, cfg.FinnhubAPIKey, builder)
		slog.Info("finnhub stream started")
	} else {
		slog.Info("finnhub stream disabled — set FINNHUB_API_KEY to enable")
	}

	router := httpapi.NewRouter(httpapi.Dependencies{
		Config:  cfg,
		DB:      gormDB,
		Hub:     hub,
		Monitor: mon,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0, // SSE connections need no write timeout
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", server.Addr, "data_dir", cfg.DataDir)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		slog.Info("shutting down api")
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
