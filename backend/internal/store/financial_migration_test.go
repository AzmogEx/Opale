package store

import (
	"context"
	"encoding/json"
	"github.com/opale-app/opale/internal/migrations"
	"strings"
	"testing"
)

func TestFinancialMigrationBackfillsOnboardingWithoutDuplicatingForecasts(t *testing.T) {
	s := legacyTestStore(t)
	ctx := context.Background()
	entries, e := migrations.FS.ReadDir(".")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range entries {
		name := file.Name()
		if !strings.HasSuffix(name, ".up.sql") || name < "0011" || name >= "0024" {
			continue
		}
		sql, _ := migrations.FS.ReadFile(name)
		if _, e = s.pool.Exec(ctx, string(sql)); e != nil {
			t.Fatal(name, e)
		}
		if _, e = s.pool.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, strings.TrimSuffix(name, ".up.sql")); e != nil {
			t.Fatal(e)
		}
	}
	p := newTestProfile(t, s, "Before migration")
	a, e := s.CreateAsset(ctx, p.ID, "Existing", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	rule, e := s.SaveCalendarRule(ctx, p.ID, CalendarRule{AssetID: a.ID, Label: "Existing subscription", Amount: -1199, Date: onboardingToday().Format("2006-01-02"), Frequency: "monthly", Active: true})
	if e != nil {
		t.Fatal(e)
	}
	result, _ := json.Marshal(map[string]any{"subscription_rule_ids": []string{rule.ID}})
	if _, e = s.pool.Exec(ctx, `INSERT INTO profile_onboarding(profile_id,status,draft,result,completed_at) VALUES($1,'completed','{}',$2,now())`, p.ID, result); e != nil {
		t.Fatal(e)
	}
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	items, e := s.ListFinancialContracts(ctx, p.ID)
	if e != nil || len(items) != 1 || items[0].Amount != 1199 || items[0].CalendarRuleID != rule.ID {
		t.Fatalf("backfill %+v %v", items, e)
	}
	for table, want := range map[string]int{"financial_contracts": 1, "contract_prices": 1, "calendar_rules": 1, "transactions": 0} {
		var n int
		if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE profile_id=$1`, p.ID).Scan(&n); e != nil || n != want {
			t.Fatalf("%s %d %v", table, n, e)
		}
	}
	// Rebooting is safe and cannot duplicate resources.
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	down, _ := migrations.FS.ReadFile("0024_contracts_income.down.sql")
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, string(down)); e == nil {
		t.Fatal("downgrade silently erased contracts")
	}
}
