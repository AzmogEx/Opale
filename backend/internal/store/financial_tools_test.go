package store

import (
	"context"
	"errors"
	"github.com/opale-app/opale/internal/money"
	"testing"
	"time"
)

func financialTestID(t *testing.T, s *Store) string {
	t.Helper()
	var id string
	if e := s.pool.QueryRow(context.Background(), `SELECT gen_random_uuid()::text`).Scan(&id); e != nil {
		t.Fatal(e)
	}
	return id
}
func TestFinancialContractObservedPriceAndHistory(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Contracts")
	a, e := s.CreateAsset(ctx, p.ID, "Debit", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := s.CreateAsset(ctx, p.ID, "Other", "checking", "EUR", "")
	today := onboardingToday()
	tomorrow := today.AddDate(0, 0, 1).Format("2006-01-02")
	c, e := s.SaveFinancialContract(ctx, p.ID, FinancialContract{ID: financialTestID(t, s), Name: "Music", Category: "subscription", Amount: 1000, Currency: "EUR", Frequency: "monthly", NextDueDate: tomorrow, AssetID: a.ID, MerchantKey: "MUSIC", Active: true, ReminderDays: 7, TrialEnd: tomorrow})
	if e != nil || c.Revision != 1 || c.CalendarRuleID == "" {
		t.Fatalf("create %+v %v", c, e)
	}
	if _, e = s.SaveUserCalendarRule(ctx, p.ID, CalendarRule{ID: c.CalendarRuleID}); !errors.Is(e, ErrManagedCalendar) {
		t.Fatalf("managed edit %v", e)
	}
	if e = s.DeleteUserCalendarRule(ctx, p.ID, c.CalendarRuleID); !errors.Is(e, ErrManagedCalendar) {
		t.Fatalf("managed delete %v", e)
	}
	add := func(asset string, amount money.Cents, bank string, date time.Time, merchant string) string {
		t.Helper()
		tx, e := s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: asset, Amount: amount, OccurredOn: date, Label: merchant, RawLabel: merchant, MerchantKey: merchant, BankStatus: bank, FlowKind: "expense_income"})
		if e != nil {
			t.Fatal(e)
		}
		return tx.ID
	}
	add(a.ID, -3000, "pending", today, "MUSIC")
	add(b.ID, -6000, "booked", today, "MUSIC")
	add(a.ID, -7000, "booked", today, "OTHER")
	add(a.ID, -9000, "booked", today.AddDate(0, 0, -1), "MUSIC")
	add(a.ID, -8000, "booked", today.AddDate(0, 0, 1), "MUSIC")
	list, e := s.ListFinancialContracts(ctx, p.ID)
	if e != nil || len(list) != 1 || list[0].PendingPrice != nil {
		t.Fatalf("spurious observation %+v %v", list, e)
	}
	txID := add(a.ID, -1250, "booked", today, "MUSIC")
	list, e = s.ListFinancialContracts(ctx, p.ID)
	if e != nil || list[0].PendingPrice == nil || list[0].PendingPrice.AnnualDelta != 3000 || list[0].Amount != 1000 {
		t.Fatalf("price %+v %v", list, e)
	}
	if e = s.ResolveContractPrice(ctx, p.ID, c.ID, txID, c.Revision, false); e != nil {
		t.Fatal(e)
	}
	list, _ = s.ListFinancialContracts(ctx, p.ID)
	if list[0].PendingPrice != nil {
		t.Fatal("dismiss not respected")
	}
	txID = add(a.ID, -1350, "booked", today, "MUSIC")
	oldRule := c.CalendarRuleID
	if e = s.ResolveContractPrice(ctx, p.ID, c.ID, txID, c.Revision, true); e != nil {
		t.Fatal(e)
	}
	list, e = s.ListFinancialContracts(ctx, p.ID)
	c = list[0]
	if e != nil || c.Amount != 1350 || c.PendingPrice != nil || c.Revision != 2 || c.CalendarRuleID == oldRule {
		t.Fatalf("accept %+v %v", c, e)
	}
	prices, e := s.ContractPrices(ctx, p.ID, c.ID)
	if e != nil || len(prices) != 2 || prices[0].Source != "observed" || prices[0].Amount != 1350 {
		t.Fatalf("history %+v %v", prices, e)
	}
	var oldActive bool
	if e = s.pool.QueryRow(ctx, `SELECT active FROM calendar_rules WHERE id=$1`, oldRule).Scan(&oldActive); e != nil || oldActive {
		t.Fatalf("old forecast active=%t %v", oldActive, e)
	}
	occ, e := s.CalendarOccurrences(ctx, p.ID, today, today.AddDate(0, 0, 2))
	if e != nil || len(occ) != 1 || occ[0].Amount != -1350 {
		t.Fatalf("forecast duplicate %+v %v", occ, e)
	}
	add(a.ID, -1600, "booked", today, "MUSIC")
	add(a.ID, -1350, "booked", today, "MUSIC")
	list, _ = s.ListFinancialContracts(ctx, p.ID)
	if list[0].PendingPrice != nil {
		t.Fatal("older high payment used")
	}
	stale := c
	stale.Revision = 1
	if _, e = s.SaveFinancialContract(ctx, p.ID, stale); !errors.Is(e, ErrFinancialConflict) {
		t.Fatalf("revision %v", e)
	}
	if e = s.DeleteFinancialContract(ctx, p.ID, c.ID, c.Revision); e != nil {
		t.Fatal(e)
	}
	occ, e = s.CalendarOccurrences(ctx, p.ID, today, today.AddDate(0, 0, 2))
	if e != nil || len(occ) != 0 {
		t.Fatalf("deleted forecast %+v %v", occ, e)
	}
}
func TestVariableIncomePrudentForecastAtomicityAndIsolation(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Income")
	other := newTestProfile(t, s, "Other")
	a, _ := s.CreateAsset(ctx, p.ID, "Account", "checking", "EUR", "")
	foreign, _ := s.CreateAsset(ctx, other.ID, "Foreign", "checking", "EUR", "")
	v := VariableIncome{ID: financialTestID(t, s), Name: "Prime", Kind: "bonus", Currency: "EUR", Low: 0, Usual: 15000, High: 25000, Frequency: "monthly", NextDate: onboardingToday().Format("2006-01-02"), Forecast: "prudent", AssetID: a.ID, Active: true}
	saved, e := s.SaveVariableIncome(ctx, p.ID, v)
	if e != nil || saved.CalendarRuleID != "" {
		t.Fatalf("zero minimum %+v %v", saved, e)
	}
	v = saved
	v.Forecast = "usual"
	saved, e = s.SaveVariableIncome(ctx, p.ID, v)
	if e != nil || saved.CalendarRuleID == "" {
		t.Fatalf("usual %+v %v", saved, e)
	}
	occ, e := s.CalendarOccurrences(ctx, p.ID, onboardingToday(), onboardingToday())
	if e != nil || len(occ) != 1 || occ[0].Amount != 15000 || occ[0].Status != "planned" {
		t.Fatalf("income forecast %+v %v", occ, e)
	}
	v = saved
	v.AssetID = foreign.ID
	if _, e = s.SaveVariableIncome(ctx, p.ID, v); !errors.Is(e, ErrNotFound) {
		t.Fatalf("foreign %v", e)
	}
	v = saved
	v.Currency = "USD"
	if _, e = s.SaveVariableIncome(ctx, p.ID, v); !errors.Is(e, ErrNotFound) {
		t.Fatalf("currency %v", e)
	}
	v = saved
	v.Low = 16000
	if _, e = s.SaveVariableIncome(ctx, p.ID, v); e == nil {
		t.Fatal("invalid range accepted")
	}
	list, e := s.ListVariableIncomes(ctx, p.ID)
	if e != nil || len(list) != 1 || list[0] != saved {
		t.Fatalf("failed save mutated %+v %v", list, e)
	}
	v = saved
	v.Forecast = "off"
	saved, e = s.SaveVariableIncome(ctx, p.ID, v)
	if e != nil || saved.CalendarRuleID != "" {
		t.Fatalf("off %+v %v", saved, e)
	}
	if _, e = s.SaveVariableIncome(ctx, p.ID, v); !errors.Is(e, ErrFinancialConflict) {
		t.Fatalf("stale income %v", e)
	}
	occ, e = s.CalendarOccurrences(ctx, p.ID, onboardingToday(), onboardingToday())
	if e != nil || len(occ) != 0 {
		t.Fatalf("off forecast %+v %v", occ, e)
	}
	var count int
	if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM transactions WHERE profile_id=$1`, p.ID).Scan(&count); e != nil || count != 0 {
		t.Fatal("invented realized income")
	}
	if e = s.ResetProfileData(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	list, e = s.ListVariableIncomes(ctx, p.ID)
	if e != nil || len(list) != 0 {
		t.Fatal("reset retained income")
	}
}
func TestContractAlertWindowsAndPushOwnership(t *testing.T) {
	today := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	c := FinancialContract{ID: "test", Name: "Trial", Active: true, TrialEnd: "2026-10-10", CommitmentEnd: "2026-11-01", RenewalDate: "2026-11-10", AutoRenew: true, NoticeDays: 30, ReminderDays: 7}
	alerts := FinancialContractAlerts([]FinancialContract{c}, today)
	if len(alerts) != 2 || alerts[0].Kind != "contract_trial" || alerts[1].Kind != "contract_notice" {
		t.Fatalf("alert windows %+v", alerts)
	}
	c.Active = false
	if len(FinancialContractAlerts([]FinancialContract{c}, today)) != 0 {
		t.Fatal("archived reminder")
	}
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Push owner")
	foreign := newTestProfile(t, s, "Push foreign")
	contract, e := s.SaveFinancialContract(ctx, p.ID, FinancialContract{ID: financialTestID(t, s), Name: "Push", Category: "other", Amount: 100, Currency: "EUR", Frequency: "yearly", NextDueDate: onboardingToday().Format("2006-01-02"), Active: true})
	if e != nil {
		t.Fatal(e)
	}
	token := "financial-tools-test-token"
	if e = s.UpsertPushToken(ctx, foreign.ID, token, "ios"); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ClaimContractPush(ctx, p.ID, contract.ID, "trial", token); e != nil || ok {
		t.Fatalf("cross-profile push %v %v", ok, e)
	}
	if e = s.UpsertPushToken(ctx, p.ID, token, "ios"); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ClaimContractPush(ctx, p.ID, contract.ID, "trial", token); e != nil || !ok {
		t.Fatalf("claim %v %v", ok, e)
	}
	if e = s.CompleteContractPush(ctx, p.ID, contract.ID, "trial", token); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ClaimContractPush(ctx, p.ID, contract.ID, "trial", token); e != nil || ok {
		t.Fatalf("duplicate push %v %v", ok, e)
	}
}

func TestConfirmedSameDayPaymentIsNotForecastAgain(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Same day")
	a, _ := s.CreateAsset(ctx, p.ID, "Account", "checking", "EUR", "")
	c, e := s.SaveFinancialContract(ctx, p.ID, FinancialContract{ID: financialTestID(t, s), Name: "Music", Category: "subscription", Amount: 1000, Currency: "EUR", Frequency: "monthly", NextDueDate: onboardingToday().Format("2006-01-02"), AssetID: a.ID, MerchantKey: "MUSIC", Active: true})
	if e != nil {
		t.Fatal(e)
	}
	tx, e := s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: -1200, OccurredOn: onboardingToday(), Label: "MUSIC", MerchantKey: "MUSIC"})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ResolveContractPrice(ctx, p.ID, c.ID, tx.ID, c.Revision, true); e != nil {
		t.Fatal(e)
	}
	occ, e := s.CalendarOccurrences(ctx, p.ID, onboardingToday(), onboardingToday())
	if e != nil || len(occ) != 1 || occ[0].Status != "realized" || occ[0].TransactionID == nil || *occ[0].TransactionID != tx.ID {
		t.Fatalf("double forecast %+v %v", occ, e)
	}
	list, _ := s.ListFinancialContracts(ctx, p.ID)
	c = list[0]
	if e = s.DeleteFinancialContract(ctx, p.ID, c.ID, c.Revision); e != nil {
		t.Fatal(e)
	}
	keys, e := s.CalendarMerchantKeys(ctx, p.ID)
	if e != nil || len(keys) != 0 {
		t.Fatalf("stopped merchant excluded forever %v %v", keys, e)
	}
}

func TestDeclaredMerchantDoesNotHideAnotherAccountOrOptimisticIncome(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Scoped forecast")
	a, _ := s.CreateAsset(ctx, p.ID, "Managed", "checking", "EUR", "")
	b, _ := s.CreateAsset(ctx, p.ID, "Other", "checking", "EUR", "")
	today := onboardingToday()
	for _, account := range []string{a.ID, b.ID} {
		for i := 0; i < 3; i++ {
			if _, e := s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: account, Amount: -900, OccurredOn: today.AddDate(0, -i, 0), Label: "MUSIC", MerchantKey: "MUSIC"}); e != nil {
				t.Fatal(e)
			}
		}
	}
	c, e := s.SaveFinancialContract(ctx, p.ID, FinancialContract{ID: financialTestID(t, s), Name: "Music", Category: "subscription", Amount: 900, Currency: "EUR", Frequency: "monthly", NextDueDate: today.Format("2006-01-02"), AssetID: a.ID, MerchantKey: "MUSIC", Active: true})
	if e != nil {
		t.Fatal(e)
	}
	_ = c
	obs, e := s.RecurringObservations(ctx, p.ID)
	if e != nil || len(obs) != 3 {
		t.Fatalf("other account lost %v %v", obs, e)
	}
	daily, e := s.AvgDailyVariableSpend(ctx, p.ID, []string{})
	if e != nil || daily != 30 {
		t.Fatalf("other account variable spending %d %v", daily, e)
	}
	for i := 0; i < 3; i++ {
		if _, e := s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: 20000, OccurredOn: today.AddDate(0, -i, 0), Label: "BONUS", MerchantKey: "BONUS"}); e != nil {
			t.Fatal(e)
		}
	}
	v := VariableIncome{ID: financialTestID(t, s), Name: "Bonus", Kind: "bonus", Currency: "EUR", Low: 0, Usual: 20000, High: 30000, Frequency: "monthly", NextDate: today.Format("2006-01-02"), AssetID: a.ID, MerchantKey: "BONUS", Forecast: "prudent", Active: true}
	if _, e = s.SaveVariableIncome(ctx, p.ID, v); e != nil {
		t.Fatal(e)
	}
	obs, e = s.RecurringObservations(ctx, p.ID)
	if e != nil || len(obs) != 3 {
		t.Fatalf("optimistic observed income replaces zero %v %v", obs, e)
	}
}
