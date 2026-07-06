package store

// Tests d'intégration du store (audit) : le SQL réel contre un vrai
// PostgreSQL. Ignorés sans OPALE_TEST_DATABASE_URL (CI : service postgres ;
// local : createdb opale_test puis
//   OPALE_TEST_DATABASE_URL=postgres://opale:opale@localhost:5433/opale_test?sslmode=disable go test ./internal/store/).

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// newTestStore ouvre le store de test et applique les migrations.
// Chaque exécution repart d'un schéma propre (down implicite : les tests
// utilisent des profils dédiés, l'isolation se fait par profile_id).
func newTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("OPALE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("OPALE_TEST_DATABASE_URL absente — test d'intégration ignoré")
	}
	ctx := context.Background()
	s, err := New(ctx, url)
	if err != nil {
		t.Fatalf("connexion : %v", err)
	}
	t.Cleanup(s.Close)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	return s
}

func newTestProfile(t *testing.T, s *Store, name string) Profile {
	t.Helper()
	p, err := s.CreateProfile(context.Background(), name, "hash-bidon", "N1")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestIntegrationNetWorthWithFX(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "IT-networth")

	// 40 000 € de compte courant + 10 000 USD + un crédit de 5 000 €.
	eur, err := s.CreateAsset(ctx, p.ID, "Compte EUR", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	usd, err := s.CreateAsset(ctx, p.ID, "Compte USD", "savings", "USD", "")
	if err != nil {
		t.Fatal(err)
	}
	loan, err := s.CreateLiability(ctx, p.ID, "Crédit", "consumer_loan", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	asOf := day(t, "2026-07-01")
	if _, err := s.AddAssetValuation(ctx, p.ID, eur.ID, money.Cents(4_000_000), asOf, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddAssetValuation(ctx, p.ID, usd.ID, money.Cents(1_000_000), asOf, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddLiabilityValuation(ctx, p.ID, loan.ID, money.Cents(500_000), asOf, ""); err != nil {
		t.Fatal(err)
	}

	// Taux : 1 USD = 0,92 € → 10 000 USD = 9 200 €.
	if _, err := s.UpsertFXRate(ctx, "USD", 920_000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DeleteFXRate(context.Background(), "USD") })

	nw, err := s.ComputeNetWorth(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 40 000 + 9 200 − 5 000 = 44 200 € — au centime (CA-1 + EF-008).
	if nw.Net != money.Cents(4_420_000) {
		t.Fatalf("patrimoine net attendu 4420000, obtenu %d", nw.Net)
	}
}

func TestIntegrationSpaceBalance(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	adam := newTestProfile(t, s, "IT-adam")
	lea := newTestProfile(t, s, "IT-lea")

	space, err := s.CreateSpace(ctx, adam.ID, "IT-foyer")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddSpaceMember(ctx, space.ID, lea.ID); err != nil {
		t.Fatal(err)
	}

	compte, err := s.CreateAsset(ctx, adam.ID, "Compte", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.CreateTransaction(ctx, adam.ID, NewTransaction{
		AssetID: compte.ID, Amount: money.Cents(-9_000),
		OccurredOn: day(t, "2026-07-04"), Label: "Courses", RawLabel: "COURSES",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTransactionSpace(ctx, adam.ID, tx.ID, &space.ID); err != nil {
		t.Fatal(err)
	}

	// Cloisonnement : un non-membre ne peut pas marquer dans cet espace.
	intrus := newTestProfile(t, s, "IT-intrus")
	compteIntrus, _ := s.CreateAsset(ctx, intrus.ID, "C", "checking", "EUR", "")
	txIntrus, _ := s.CreateTransaction(ctx, intrus.ID, NewTransaction{
		AssetID: compteIntrus.ID, Amount: money.Cents(-100),
		OccurredOn: day(t, "2026-07-04"), Label: "x", RawLabel: "x",
	})
	if err := s.SetTransactionSpace(ctx, intrus.ID, txIntrus.ID, &space.ID); err == nil {
		t.Fatal("un non-membre ne doit pas pouvoir marquer une dépense commune")
	}

	members, total, err := s.SpaceBalance(ctx, space.ID)
	if err != nil {
		t.Fatal(err)
	}
	if total != money.Cents(9_000) || len(members) != 2 {
		t.Fatalf("total %d / %d membres — attendu 9000 / 2", total, len(members))
	}
	// Adam a payé 90 €, quote-part 45 € chacun → +45 / −45.
	for _, m := range members {
		want := money.Cents(-4_500)
		if m.ProfileID == adam.ID {
			want = money.Cents(4_500)
		}
		if m.Balance != want {
			t.Fatalf("balance de %s : %d, attendu %d", m.Name, m.Balance, want)
		}
	}
}

func TestIntegrationSplitTransaction(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "IT-split")

	compte, err := s.CreateAsset(ctx, p.ID, "Compte", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.CreateTransaction(ctx, p.ID, NewTransaction{
		AssetID: compte.ID, Amount: money.Cents(-10_000),
		OccurredOn: day(t, "2026-07-05"), Label: "Hypermarché", RawLabel: "HYPER",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Somme fausse → refus, l'original doit survivre.
	if _, err := s.SplitTransaction(ctx, p.ID, tx.ID, []SplitPart{
		{Amount: money.Cents(-6_000)}, {Amount: money.Cents(-3_000)},
	}); err == nil {
		t.Fatal("somme incorrecte : erreur attendue")
	}
	if _, err := s.GetTransaction(ctx, p.ID, tx.ID); err != nil {
		t.Fatal("l'original doit survivre à un split refusé")
	}

	// Split exact : 100 € → 60 + 40.
	created, err := s.SplitTransaction(ctx, p.ID, tx.ID, []SplitPart{
		{Amount: money.Cents(-6_000), Label: "Courses"},
		{Amount: money.Cents(-4_000), Label: "Loisirs"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("2 parts attendues, obtenu %d", len(created))
	}
	if _, err := s.GetTransaction(ctx, p.ID, tx.ID); err == nil {
		t.Fatal("l'original doit disparaître après un split réussi")
	}
	var sum int64
	for _, c := range created {
		sum += int64(c.Amount)
	}
	if sum != -10_000 {
		t.Fatalf("la somme des parts doit rester -10000, obtenu %d", sum)
	}
}

func TestIntegrationTheoreticalBalance(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "IT-theo")

	compte, err := s.CreateAsset(ctx, p.ID, "Compte", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	// Valorisé 1 000 € au 1er juillet, puis −50 € le 3 juillet.
	if _, err := s.AddAssetValuation(ctx, p.ID, compte.ID, money.Cents(100_000), day(t, "2026-07-01"), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTransaction(ctx, p.ID, NewTransaction{
		AssetID: compte.ID, Amount: money.Cents(-5_000),
		OccurredOn: day(t, "2026-07-03"), Label: "x", RawLabel: "x",
	}); err != nil {
		t.Fatal(err)
	}

	assets, err := s.ListAssets(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 || assets[0].TheoreticalValue == nil {
		t.Fatal("solde théorique attendu")
	}
	// 1 000 − 50 = 950 € — la dérive est visible.
	if *assets[0].TheoreticalValue != money.Cents(95_000) {
		t.Fatalf("solde théorique attendu 95000, obtenu %d", *assets[0].TheoreticalValue)
	}
}
