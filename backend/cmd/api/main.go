// Commande api : point d'entrée du backend Opale.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opale-app/opale/internal/api"
	"github.com/opale-app/opale/internal/config"
	"github.com/opale-app/opale/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("démarrage impossible", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := newLogger(cfg.LogLevel)
	log.Info("Opale — démarrage du backend", "env", cfg.Env, "addr", cfg.HTTPAddr)

	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		return err
	}
	log.Info("migrations appliquées")

	// Purge périodique des sessions expirées (au démarrage puis chaque jour).
	go func() {
		purge := func() {
			if n, err := st.DeleteExpiredSessions(context.Background()); err != nil {
				log.Warn("purge des sessions", "err", err)
			} else if n > 0 {
				log.Info("sessions expirées purgées", "count", n)
			}
		}
		purge()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			purge()
		}
	}()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewServer(st, cfg, log).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Démarrage + arrêt gracieux sur SIGINT/SIGTERM.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		log.Info("arrêt demandé", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
