// Package jobs — tâches de fond du serveur : rafraîchissement des cours
// (opt-in) et snapshot mensuel du patrimoine. Le serveur travaille pour le
// foyer même quand personne n'ouvre l'app.
package jobs

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/push"
	"github.com/opale-app/opale/internal/quotes"
	"github.com/opale-app/opale/internal/store"
)

// Runner regroupe les dépendances des jobs.
type Runner struct {
	Store      *store.Store
	Quotes     *quotes.Client // nil = cours automatiques coupés
	Push       *push.Client   // nil = notifications push coupées
	Log        *slog.Logger
	AutoQuotes bool
}

// Start refreshes quotes/snapshots every twelve hours, and evaluates push
// thresholds every fifteen minutes. The persistent delivery ledger deduplicates
// overlapping cycles and restarts.
func (r *Runner) Start(ctx context.Context) {
	go func() {
		r.RunOnce(ctx)
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.RunOnce(ctx)
			}
		}
	}()
	if r.Push != nil {
		go func() {
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					cycle, cancel := context.WithTimeout(ctx, 2*time.Minute)
					r.PushTriggeredAlerts(cycle)
					cancel()
				}
			}
		}()
	}
}

// RunOnce exécute un cycle complet : cours puis snapshots.
func (r *Runner) RunOnce(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if r.AutoQuotes && r.Quotes != nil {
		if err := r.RefreshQuotes(ctx); err != nil {
			r.Log.Warn("rafraîchissement des cours", "err", err)
		}
	}
	if err := r.MonthlySnapshots(ctx); err != nil {
		r.Log.Warn("snapshot mensuel", "err", err)
	}
	r.PushTriggeredAlerts(ctx)
}

// RefreshQuotes met à jour les taux BCE (devises déjà utilisées) et pose la
// valorisation du jour des cryptos suivies (quantité × cours, entiers).
func (r *Runner) RefreshQuotes(ctx context.Context) error { return r.RefreshQuotesForProfile(ctx, "") }
func (r *Runner) RefreshQuotesForProfile(ctx context.Context, owner string) error {
	// Devises : seulement celles déjà présentes dans fx_rates (on ne remplit
	// pas la table avec 30 devises inutiles).
	existing, missing, err := r.Store.ListFXRates(ctx, owner)
	if err != nil {
		return err
	}
	for _, currency := range missing {
		existing = append(existing, store.FXRate{Currency: currency})
	}
	if len(existing) > 0 {
		ecb, asOf, err := r.Quotes.FetchECBRatesDated(ctx)
		if err != nil {
			r.Log.Warn("taux BCE indisponibles", "err", err)
		} else {
			updated := 0
			for _, fx := range existing {
				if micro, ok := ecb[fx.Currency]; ok && micro > 0 {
					if err := r.Store.UpsertReferenceFX(ctx, fx.Currency, micro, asOf); err == nil {
						updated++
					}
				}
			}
			r.Log.Info("taux BCE mis à jour", "count", updated)
		}
	}

	// Cryptos suivies : un appel CoinGecko groupé pour tous les profils.
	assets, err := r.Store.ListQuotedAssets(ctx, owner)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		return nil
	}
	symbolSet := map[string]bool{}
	ids := []string{}
	for _, a := range assets {
		if !symbolSet[a.QuoteSymbol] {
			symbolSet[a.QuoteSymbol] = true
			ids = append(ids, a.QuoteSymbol)
		}
	}
	prices, err := r.Quotes.FetchCryptoPricesMicroEUR(ctx, ids)
	if err != nil {
		for _, a := range assets {
			_ = r.Store.QuoteResult(ctx, a.ProfileID, a.ID, false)
		}
		return err
	}
	today := civilToday()
	updated := 0
	for _, a := range assets {
		price, ok := prices[a.QuoteSymbol]
		if !ok {
			_ = r.Store.QuoteResult(ctx, a.ProfileID, a.ID, false)
			continue
		}
		// Quantity and price retain six decimals; round only the final EUR cents.
		product := new(big.Int).Mul(big.NewInt(a.QuantityMicro), big.NewInt(int64(price)))
		product.Add(product, big.NewInt(5_000_000_000))
		product.Quo(product, big.NewInt(10_000_000_000))
		if !product.IsInt64() {
			_ = r.Store.QuoteResult(ctx, a.ProfileID, a.ID, false)
			continue
		}
		value := money.Cents(product.Int64())
		if err := r.Store.ReplaceAutoValuation(ctx, a.ProfileID, a.ID, value, today); err != nil {
			_ = r.Store.QuoteResult(ctx, a.ProfileID, a.ID, false)
			continue
		}
		_ = r.Store.QuoteResult(ctx, a.ProfileID, a.ID, true)
		updated++
	}
	r.Log.Info("cours crypto appliqués", "assets", updated)
	return nil
}

// MonthlySnapshots pose le point du mois courant pour chaque profil qui ne
// l'a pas encore (idempotent : au plus un par profil et par mois).
func (r *Runner) MonthlySnapshots(ctx context.Context) error {
	profiles, err := r.Store.ListProfiles(ctx)
	if err != nil {
		return err
	}
	now := civilToday()
	for _, p := range profiles {
		exists, err := r.Store.HasMonthlySnapshot(ctx, p.ID, now)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		nw, err := r.Store.ComputeNetWorth(ctx, p.ID)
		if err != nil {
			r.Log.Warn("patrimoine snapshot", "profile_id", p.ID, "err", err)
			continue
		}
		if !nw.Complete {
			continue
		}
		if err := r.Store.UpsertMonthlySnapshot(ctx, p.ID, now,
			nw.AssetsTotal, nw.LiabilitiesTotal, nw.Net); err != nil {
			r.Log.Warn("écriture snapshot", "profile_id", p.ID, "err", err)
			continue
		}
		r.Log.Info("snapshot mensuel pris", "profile_id", p.ID, "month", now.Format("2006-01"))
	}
	return nil
}

func civilToday() time.Time {
	loc, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
