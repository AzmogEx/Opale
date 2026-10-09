package jobs

import (
	"context"
	"errors"
	"github.com/opale-app/opale/internal/push"
	"github.com/opale-app/opale/internal/store"
	"time"
)

func (r *Runner) pushContractAlerts(ctx context.Context) {
	profiles, e := r.Store.ListProfiles(ctx)
	if e != nil {
		return
	}
	loc, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, p := range profiles {
		contracts, e := r.Store.ListFinancialContracts(ctx, p.ID)
		if e != nil {
			continue
		}
		alerts := store.FinancialContractAlerts(contracts, today)
		if len(alerts) == 0 {
			continue
		}
		tokens, e := r.Store.ListPushTokens(ctx, p.ID)
		if e != nil {
			continue
		}
		for _, a := range alerts {
			for _, token := range tokens {
				claimed, e := r.Store.ClaimContractPush(ctx, p.ID, a.ContractID, a.ID, token)
				if e != nil || !claimed {
					continue
				}
				// Never put the merchant, amount, document, or contract name on the lock screen.
				if e = r.Push.Send(ctx, token, "Opale", "Une alerte de contrat nécessite ton attention. Ouvre Opale pour la consulter.", p.ID); e != nil {
					var dead push.ErrBadToken
					if errors.As(e, &dead) {
						_ = r.Store.DeleteProfilePushToken(ctx, p.ID, token)
					}
					r.Log.Warn("contract push delivery failed")
					continue
				}
				_ = r.Store.CompleteContractPush(ctx, p.ID, a.ContractID, a.ID, token)
			}
		}
	}
}
