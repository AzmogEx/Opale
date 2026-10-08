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
	"github.com/opale-app/opale/internal/jobs"
	"github.com/opale-app/opale/internal/push"
	"github.com/opale-app/opale/internal/quotes"
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

	ctx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { cancelWorkers(); st.Close() }()

	if err := st.Migrate(ctx); err != nil {
		return err
	}
	log.Info("migrations appliquées")

	// Purge périodique des sessions expirées (au démarrage puis chaque jour).
	go func() {
		purge := func() {
			cleanup, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := st.DeleteExpiredDemos(cleanup); err != nil {
				log.Warn("demo cleanup failed")
			}
			if n, err := st.DeleteExpiredSessions(cleanup); err != nil {
				log.Warn("purge des sessions", "err", err)
			} else if n > 0 {
				log.Info("sessions expirées purgées", "count", n)
			}
		}
		purge()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				purge()
			}
		}
	}()

	// Jobs de fond : cours automatiques (opt-in), snapshot mensuel,
	// alertes poussées (si APNs configuré).
	runner := &jobs.Runner{
		Store:      st,
		Quotes:     quotes.New(&http.Client{Timeout: 30 * time.Second}),
		Log:        log,
		AutoQuotes: cfg.AutoQuotes,
	}
	if cfg.APNSEnabled() {
		apns, err := push.New(cfg.APNSKeyP8, cfg.APNSKeyID, cfg.APNSTeamID, cfg.APNSBundleID, cfg.APNSEnv)
		if err != nil {
			log.Error("apns: configuration invalide — push désactivé", "err", err)
		} else {
			runner.Push = apns
			log.Info("apns: notifications push activées", "env", cfg.APNSEnv)
		}
	}
	runner.Start(ctx)
	if cfg.AutoQuotes {
		log.Info("cours automatiques activés (CoinGecko + BCE)")
	}

	apiServer := api.NewServer(st, cfg, log, runner)
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			apiServer.RunBankSync(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiServer.Routes(),
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
