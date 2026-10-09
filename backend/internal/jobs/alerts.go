package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/opale-app/opale/internal/push"
)

// TriggeredAlert — un seuil personnalisé franchi (EF-053 étendu).
type TriggeredAlert struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

// EvaluateCustomAlerts confronte les seuils actifs d'un profil aux chiffres
// du moteur. Utilisé par l'API (écran Alertes) ET par le job push.
func (r *Runner) EvaluateCustomAlerts(ctx context.Context, profileID string) ([]TriggeredAlert, error) {
	alerts, err := r.Store.ListCustomAlerts(ctx, profileID, true)
	if err != nil {
		return nil, err
	}
	if len(alerts) == 0 {
		return []TriggeredAlert{}, nil
	}

	out := []TriggeredAlert{}
	loc, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(loc)
	for _, a := range alerts {
		switch a.Kind {
		case "cash_below":
			cash, err := r.Store.CashBalance(ctx, profileID)
			if err != nil {
				continue
			}
			if cash < a.Threshold {
				out = append(out, TriggeredAlert{
					ID: a.ID, Kind: a.Kind, Severity: "critical",
					Title:  "Cash sous ton seuil",
					Detail: "Cash disponible : " + cash.String() + " € (seuil : " + a.Threshold.String() + " €)",
				})
			}
		case "net_worth_below":
			nw, err := r.Store.ComputeNetWorth(ctx, profileID)
			if err != nil {
				continue
			}
			if nw.Net < a.Threshold {
				out = append(out, TriggeredAlert{
					ID: a.ID, Kind: a.Kind, Severity: "warning",
					Title:  "Patrimoine sous ton seuil",
					Detail: "Patrimoine net : " + nw.Net.String() + " € (seuil : " + a.Threshold.String() + " €)",
				})
			}
		case "expenses_month_above":
			summary, err := r.Store.ComputeMonthSummary(ctx, profileID, now.Year(), now.Month())
			if err != nil {
				continue
			}
			if summary.Expenses > a.Threshold {
				out = append(out, TriggeredAlert{
					ID: a.ID, Kind: a.Kind, Severity: "warning",
					Title:  "Dépenses du mois au-dessus de ton seuil",
					Detail: "Déjà " + summary.Expenses.String() + " € dépensés ce mois-ci (seuil : " + a.Threshold.String() + " €)",
				})
			}
		}
	}
	return out, nil
}

// PushTriggeredAlerts pousse les alertes franchies vers les appareils du
// foyer — au plus une notification par alerte et par jour (anti-spam).
func (r *Runner) PushTriggeredAlerts(ctx context.Context) {
	if r.Push == nil {
		return
	}
	r.pushContractAlerts(ctx)
	profiles, err := r.Store.ListProfiles(ctx)
	if err != nil {
		return
	}
	for _, p := range profiles {
		triggered, err := r.EvaluateCustomAlerts(ctx, p.ID)
		if err != nil || len(triggered) == 0 {
			continue
		}
		tokens, err := r.Store.ListPushTokens(ctx, p.ID)
		if err != nil || len(tokens) == 0 {
			continue
		}
		for _, alert := range triggered {
			for _, token := range tokens {
				claimed, e := r.Store.ClaimPush(ctx, p.ID, alert.ID, token)
				if e != nil || !claimed {
					continue
				}
				if err := r.Push.Send(ctx, token, "Opale", "Une alerte nécessite ton attention. Ouvre Opale pour la consulter.", p.ID); err != nil {
					var dead push.ErrBadToken
					if errors.As(err, &dead) {
						// L'appareil n'existe plus : on purge le jeton.
						_ = r.Store.DeleteProfilePushToken(ctx, p.ID, token)
					}
					r.Log.Warn("push delivery failed")
					continue
				}
				_ = r.Store.CompletePush(ctx, p.ID, alert.ID, token)
			}
			r.Log.Info("cycle alerte push terminé", "kind", alert.Kind)
		}
	}
}
