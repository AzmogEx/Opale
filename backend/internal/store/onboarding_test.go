package store

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opale-app/opale/internal/migrations"
	"github.com/opale-app/opale/internal/money"
)

func TestOnboardingOverdraftSerializesWithAccountKind(t *testing.T) {
	// Fresh isolated DB ensures this exercises the migration itself, including
	// both orders of the racing statements rather than an already installed trigger.
	s := legacyTestStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	p := newTestProfile(t, s, "Overdraft race")
	for _, kindFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(kindFirst), func(t *testing.T) {
			a, err := s.CreateAsset(ctx, p.ID, "Race", "checking", "EUR", "")
			if err != nil {
				t.Fatal(err)
			}
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if kindFirst {
				_, err = tx.Exec(ctx, "UPDATE assets SET kind='savings' WHERE id=$1", a.ID)
			} else {
				_, err = tx.Exec(ctx, "INSERT INTO valuations(profile_id,asset_id,value_cents,as_of) VALUES($1,$2,-100,CURRENT_DATE)", p.ID, a.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			racing := make(chan error, 1)
			go func() {
				if kindFirst {
					_, e := s.AddAssetValuation(ctx, p.ID, a.ID, -100, onboardingToday(), "")
					racing <- e
				} else {
					_, e := s.pool.Exec(ctx, "UPDATE assets SET kind='savings' WHERE id=$1", a.ID)
					racing <- e
				}
			}()
			select {
			case e := <-racing:
				t.Fatalf("racing statement failed to wait for account lock: %v", e)
			case <-time.After(100 * time.Millisecond):
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case e := <-racing:
				if e == nil {
					t.Fatal("both incompatible operations committed")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("racing statement did not finish")
			}
			var invalid bool
			if err = s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM valuations v JOIN assets a ON a.id=v.asset_id WHERE a.id=$1 AND v.value_cents<0 AND a.kind<>'checking')", a.ID).Scan(&invalid); err != nil || invalid {
				t.Fatalf("invalid balance %t %v", invalid, err)
			}
		})
	}
	// A downgrade must refuse to drop the capability while its data exists.
	down, err := migrations.FS.ReadFile("0023_checking_overdraft.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, string(down)); err == nil {
		t.Fatal("downgrade accepted existing overdraft")
	}
}

func completeSetupDraft() OnboardingDraft {
	today := onboardingToday()
	return OnboardingDraft{
		Income:        OnboardingIncome{Enabled: true, Amount: "3200,15", NextDate: today.AddDate(0, 0, 10).Format("2006-01-02")},
		Account:       OnboardingAccount{Enabled: true, Name: "Compte initial", Balance: "-50,25", BalanceDate: today.AddDate(0, 0, -1).Format("2006-01-02")},
		Expenses:      []OnboardingExpense{{ID: "66e08a02-6d48-44dc-89c2-6b2b7a3dd4cb", Label: "Loyer", Amount: "800", Date: today.AddDate(0, 0, 1).Format("2006-01-02"), Frequency: "monthly"}},
		Subscriptions: []OnboardingExpense{{ID: "178cdf74-7c85-43e2-8d23-fc8b12e36d0b", Label: "Musique", Amount: "35,97", Date: today.AddDate(0, 0, 2).Format("2006-01-02"), Frequency: "quarterly"}},
		Goal:          OnboardingGoal{Enabled: true, Name: "Sécurité", Target: "5000"}, VariableBudget: "450,50",
	}
}

func TestOnboardingValidationExactAmountsAndDates(t *testing.T) {
	valid := completeSetupDraft()
	p, err := validateOnboarding(valid, onboardingToday())
	if err != nil || p.income != 320015 || *p.balance != -5025 || p.expenses[0][0] != 80000 || p.expenses[1][0] != 3597 || p.goal != 500000 {
		t.Fatalf("parse %+v %v", p, err)
	}
	for input, want := range map[string]money.Cents{"1 234,56 €": 123456, "+1\u00a0234.5": 123450, ".5": 50, "-,50": -50, "1.": 100} {
		if got, err := setupAmount(input, "test", false, true); err != nil || got != want {
			t.Fatalf("parse %q: %d %v", input, got, err)
		}
	}
	for _, input := range []string{"-+1", "1e3", "NaN", "1,234", "1.2.3"} {
		if _, err := setupAmount(input, "test", false, true); err == nil {
			t.Fatalf("malformed accepted: %s", input)
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*OnboardingDraft)
	}{
		{"extra decimals", func(d *OnboardingDraft) { d.Income.Amount = "3200,159" }},
		{"overflow", func(d *OnboardingDraft) { d.Income.Amount = "9223372036854775808" }},
		{"negative income", func(d *OnboardingDraft) { d.Income.Amount = "-3" }},
		{"zero expense", func(d *OnboardingDraft) { d.Expenses[0].Amount = "0" }},
		{"negative expense", func(d *OnboardingDraft) { d.Expenses[0].Amount = "-8" }},
		{"negative variable", func(d *OnboardingDraft) { d.VariableBudget = "-8" }},
		{"missing account", func(d *OnboardingDraft) { d.Account.Enabled = false }},
		{"invalid existing id", func(d *OnboardingDraft) { d.Account.ExistingAssetID = "not-a-uuid" }},
		{"duplicate row ids", func(d *OnboardingDraft) { d.Subscriptions[0].ID = d.Expenses[0].ID }},
		{"invalid row id", func(d *OnboardingDraft) { d.Expenses[0].ID = "" }},
		{"old salary", func(d *OnboardingDraft) { d.Income.NextDate = onboardingToday().AddDate(0, 0, -1).Format("2006-01-02") }},
		{"invalid date", func(d *OnboardingDraft) { d.Expenses[0].Date = "2026-02-30" }},
		{"future valuation", func(d *OnboardingDraft) {
			d.Account.BalanceDate = onboardingToday().AddDate(0, 0, 1).Format("2006-01-02")
		}},
		{"frequency", func(d *OnboardingDraft) { d.Expenses[0].Frequency = "daily" }},
		{"empty goal", func(d *OnboardingDraft) { d.Goal.Name = " " }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := completeSetupDraft()
			tc.mutate(&d)
			if _, err := validateOnboarding(d, onboardingToday()); err == nil {
				t.Fatal("invalid draft accepted")
			}
		})
	}
	// All pages may be declined. Unused incomplete input must not block completion.
	optional := OnboardingDraft{Income: OnboardingIncome{Amount: "incomplete"}, Account: OnboardingAccount{Balance: "-"}, Goal: OnboardingGoal{Target: "."}}
	if _, err = validateOnboarding(optional, onboardingToday()); err != nil {
		t.Fatal(err)
	}
	noBalance := completeSetupDraft()
	noBalance.Account.Balance = ""
	noBalance.Account.BalanceDate = ""
	if p, err = validateOnboarding(noBalance, onboardingToday()); err != nil || p.balance != nil {
		t.Fatalf("optional balance %+v %v", p, err)
	}
}

func TestOnboardingOverdraftIsOnlyAllowedForChecking(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Overdraft integrity")
	for _, kind := range []string{"checking", "savings", "real_estate", "other"} {
		a, err := s.CreateAsset(ctx, p.ID, kind, kind, "EUR", "")
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.AddAssetValuation(ctx, p.ID, a.ID, -1, onboardingToday(), "")
		if kind == "checking" && err != nil {
			t.Fatalf("valid overdraft %v", err)
		}
		if kind != "checking" && err == nil {
			t.Fatalf("negative %s accepted", kind)
		}
		if kind == "checking" {
			if _, err = s.pool.Exec(ctx, "UPDATE assets SET kind='other' WHERE id=$1", a.ID); err == nil {
				t.Fatal("negative account changed kind")
			}
		}
	}
	loan, err := s.CreateLiability(ctx, p.ID, "Loan", "consumer_loan", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddLiabilityValuation(ctx, p.ID, loan.ID, -1, onboardingToday(), ""); err == nil {
		t.Fatal("negative debt accepted")
	}
}

func TestOnboardingResumeSkipRevisionAndPrompt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Setup resume")
	state, err := s.GetOnboarding(ctx, p.ID)
	if err != nil || state.Status != "not_started" || state.Revision != 0 || !state.CanStart || !state.ShouldPrompt || state.Result != nil || state.Draft.Expenses == nil {
		t.Fatalf("initial %+v %v", state, err)
	}
	d := state.Draft
	d.Income.Enabled = true
	d.Income.Amount = "12,"
	state, err = s.SaveOnboarding(ctx, p.ID, 0, 1, d)
	if err != nil || state.Status != "draft" || state.Revision != 1 || state.Step != 1 {
		t.Fatalf("save %+v %v", state, err)
	}
	if _, err = s.SaveOnboarding(ctx, p.ID, 0, 2, d); !errors.Is(err, ErrOnboardingConflict) {
		t.Fatalf("stale save %v", err)
	}
	state, err = s.GetOnboarding(ctx, p.ID)
	if err != nil || state.Draft.Income.Amount != "12," || state.Step != 1 {
		t.Fatalf("resume %+v %v", state, err)
	}
	state, err = s.SkipOnboarding(ctx, p.ID, 1)
	if err != nil || state.Status != "skipped" || state.ShouldPrompt || !state.CanStart || state.Draft.Income.Amount != "12," {
		t.Fatalf("skip %+v %v", state, err)
	}
	state, err = s.SaveOnboarding(ctx, p.ID, state.Revision, 2, d)
	if err != nil || state.Status != "draft" || !state.ShouldPrompt {
		t.Fatalf("resume skip %+v %v", state, err)
	}
	if _, err = s.CreateAsset(ctx, p.ID, "Existing", "checking", "EUR", ""); err != nil {
		t.Fatal(err)
	}
	state, err = s.GetOnboarding(ctx, p.ID)
	if err != nil || state.ShouldPrompt || !state.CanStart {
		t.Fatalf("existing profile prompt %+v %v", state, err)
	}
}

func TestOnboardingCompleteConcurrentRetryAndNoRealizedFlows(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Setup concurrent")
	state, err := s.SaveOnboarding(ctx, p.ID, 0, 5, completeSetupDraft())
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan OnboardingState, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, e := s.CompleteOnboarding(ctx, p.ID, state.Revision)
			results <- out
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var first string
	for out := range results {
		if out.Status != "completed" || out.CanStart || out.ShouldPrompt || out.Result == nil || len(out.Result.ExpenseRuleIDs) != 1 || len(out.Result.SubscriptionRuleIDs) != 1 {
			t.Fatalf("complete %+v", out)
		}
		if first == "" {
			first = out.Result.AssetID
		} else if first != out.Result.AssetID {
			t.Fatal("different retry resources")
		}
	}
	for table, want := range map[string]int{"assets": 1, "valuations": 1, "calendar_rules": 3, "goals": 1, "transactions": 0, "calendar_occurrences": 0} {
		var count int
		if e := s.pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE profile_id=$1", p.ID).Scan(&count); e != nil || count != want {
			t.Fatalf("%s %d %v", table, count, e)
		}
	}
	inc, exp, err := s.FlowTotals3M(ctx, p.ID)
	if err != nil || inc != 0 || exp != 0 {
		t.Fatalf("invented actuals %d %d %v", inc, exp, err)
	}
	goals, err := s.ListGoals(ctx, p.ID)
	if err != nil || len(goals) != 1 || goals[0].MonthlySavings != 0 || goals[0].AssetID != nil {
		t.Fatalf("goal %+v %v", goals, err)
	}
	occ, err := s.CalendarOccurrences(ctx, p.ID, onboardingToday(), onboardingToday().AddDate(0, 1, 0))
	if err != nil || len(occ) != 3 {
		t.Fatalf("calendar %+v %v", occ, err)
	}
	for _, o := range occ {
		if o.Status != "planned" || o.TransactionID != nil {
			t.Fatalf("invented transaction %+v", o)
		}
	}
	state, err = s.CompleteOnboarding(ctx, p.ID, 0)
	if err != nil || state.Result.AssetID != first {
		t.Fatalf("late retry %+v %v", state, err)
	}
	if _, err = s.SaveOnboarding(ctx, p.ID, state.Revision, 0, defaultOnboardingDraft()); !errors.Is(err, ErrOnboardingConflict) {
		t.Fatalf("overwrite completed %v", err)
	}
	// Portability includes the private draft, and reset removes it alongside its resources.
	var exported bytes.Buffer
	if err = s.ExportZIP(ctx, p.ID, &exported, nil); err != nil {
		t.Fatal(err)
	}
	zipReader, err := zip.NewReader(bytes.NewReader(exported.Bytes()), int64(exported.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range zipReader.File {
		if f.Name == "export.json" {
			reader, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			raw, e := io.ReadAll(reader)
			reader.Close()
			if e != nil {
				t.Fatal(e)
			}
			var data map[string]json.RawMessage
			if e = json.Unmarshal(raw, &data); e != nil {
				t.Fatal(e)
			}
			found = bytes.Contains(data["profile_onboarding"], []byte(`"completed"`))
		}
	}
	if !found {
		t.Fatal("onboarding missing from private export")
	}
	if err = s.ResetProfileData(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	state, err = s.GetOnboarding(ctx, p.ID)
	if err != nil || state.Status != "not_started" || !state.ShouldPrompt {
		t.Fatalf("reset %+v %v", state, err)
	}
}

func TestOnboardingCompleteRollsBackLateFailure(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Setup rollback")
	state, err := s.SaveOnboarding(ctx, p.ID, 0, 5, completeSetupDraft())
	if err != nil {
		t.Fatal(err)
	}
	// Fail at goal insertion, after account, valuation and all calendar rules
	// have been written. The trigger only concerns this synthetic test profile.
	name := "test_setup_" + strings.ReplaceAll(p.ID, "-", "")
	_, err = s.pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.profile_id='%s'::uuid THEN RAISE EXCEPTION 'synthetic setup failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON goals FOR EACH ROW EXECUTE FUNCTION %s()`, name, p.ID, name, name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = s.pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON goals; DROP FUNCTION IF EXISTS %s()", name, name))
	})
	if _, err = s.CompleteOnboarding(ctx, p.ID, state.Revision); err == nil {
		t.Fatal("expected late failure")
	}
	for _, table := range []string{"assets", "valuations", "calendar_rules", "goals", "transactions"} {
		var count int
		if err = s.pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE profile_id=$1", p.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial write %s=%d %v", table, count, err)
		}
	}
	after, err := s.GetOnboarding(ctx, p.ID)
	if err != nil || after.Status != "draft" || after.Revision != state.Revision || after.Result != nil {
		t.Fatalf("lost draft %+v %v", after, err)
	}
	_, err = s.pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON goals", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompleteOnboarding(ctx, p.ID, state.Revision); err != nil {
		t.Fatalf("retry after failure %v", err)
	}
}

func TestOnboardingAccountIsolationAndDemo(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Setup owner")
	other := newTestProfile(t, s, "Setup outsider")
	a, err := s.CreateAsset(ctx, other.ID, "Existing", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	d := completeSetupDraft()
	d.Account.ExistingAssetID = a.ID
	state, err := s.SaveOnboarding(ctx, p.ID, 0, 5, d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompleteOnboarding(ctx, p.ID, state.Revision); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-profile account %v", err)
	}
	// Reusing our own account never writes the stale creation name/balance.
	a, err = s.CreateAsset(ctx, p.ID, "Preserved", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddAssetValuation(ctx, p.ID, a.ID, money.Cents(123456), onboardingToday().AddDate(0, 0, -1), ""); err != nil {
		t.Fatal(err)
	}
	d.Account.ExistingAssetID = a.ID
	state, err = s.SaveOnboarding(ctx, p.ID, state.Revision, 5, d)
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.CompleteOnboarding(ctx, p.ID, state.Revision)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := s.ListAssets(ctx, p.ID)
	if err != nil || len(assets) != 1 || assets[0].Name != "Preserved" || *assets[0].LatestValue != 123456 {
		t.Fatalf("existing account modified %+v %v", assets, err)
	}
	// GET is safe for a demo, but all changes are refused server-side.
	demo := newTestProfile(t, s, "Setup demo")
	_, err = s.pool.Exec(ctx, "UPDATE profiles SET is_demo=true,demo_expires_at=$2 WHERE id=$1", demo.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.GetOnboarding(ctx, demo.ID)
	if err != nil || state.CanStart || state.ShouldPrompt {
		t.Fatalf("demo prompt %+v %v", state, err)
	}
	if _, err = s.SaveOnboarding(ctx, demo.ID, 0, 0, d); !errors.Is(err, ErrOnboardingUnavailable) {
		t.Fatalf("demo save %v", err)
	}
	if _, err = s.SkipOnboarding(ctx, demo.ID, 0); !errors.Is(err, ErrOnboardingUnavailable) {
		t.Fatalf("demo skip %v", err)
	}
	if _, err = s.CompleteOnboarding(ctx, demo.ID, 0); !errors.Is(err, ErrOnboardingUnavailable) {
		t.Fatalf("demo complete %v", err)
	}
}
