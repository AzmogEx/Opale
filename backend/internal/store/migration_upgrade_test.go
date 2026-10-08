package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/migrations"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func legacyTestStore(t *testing.T) *Store {
	t.Helper()
	raw := os.Getenv("OPALE_TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("isolated PostgreSQL required")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, raw)
	if e != nil {
		t.Fatal(e)
	}
	name := fmt.Sprintf("opale_migration_%d", time.Now().UnixNano())
	if _, e = admin.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()); e != nil {
		admin.Close(ctx)
		t.Fatal(e)
	}
	t.Cleanup(func() {
		admin.Exec(ctx, `DROP DATABASE `+pgx.Identifier{name}.Sanitize()+` WITH (FORCE)`)
		admin.Close(ctx)
	})
	u, e := url.Parse(raw)
	if e != nil {
		t.Fatal(e)
	}
	u.Path = "/" + name
	s, e := New(ctx, u.String())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	if _, e = s.pool.Exec(ctx, `CREATE TABLE schema_migrations(version TEXT PRIMARY KEY,applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); e != nil {
		t.Fatal(e)
	}
	entries, e := migrations.FS.ReadDir(".")
	if e != nil {
		t.Fatal(e)
	}
	names := []string{}
	for _, x := range entries {
		if strings.HasSuffix(x.Name(), ".up.sql") && x.Name() < "0011" {
			names = append(names, x.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		b, _ := migrations.FS.ReadFile(name)
		if _, e = s.pool.Exec(ctx, string(b)); e != nil {
			t.Fatal(name, e)
		}
		if _, e = s.pool.Exec(ctx, `INSERT INTO schema_migrations(version)VALUES($1)`, strings.TrimSuffix(name, ".up.sql")); e != nil {
			t.Fatal(e)
		}
	}
	return s
}
func TestMigrationPreservesExistingDataAndRefusesCrossOwner(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(fmt.Sprint(broken), func(t *testing.T) {
			s := legacyTestStore(t)
			ctx := context.Background()
			p, e := s.CreateProfile(ctx, "legacy", "hash", "N1")
			if e != nil {
				t.Fatal(e)
			}
			q, e := s.CreateProfile(ctx, "other", "hash", "N1")
			if e != nil {
				t.Fatal(e)
			}
			var a string
			if e = s.pool.QueryRow(ctx, `INSERT INTO assets(profile_id,name,kind,currency)VALUES($1,'legacy cash','checking','EUR') RETURNING id`, p.ID).Scan(&a); e != nil {
				t.Fatal(e)
			}
			owner := p.ID
			if broken {
				owner = q.ID
			}
			if _, e = s.pool.Exec(ctx, `INSERT INTO transactions(profile_id,asset_id,amount_cents,occurred_on,label,raw_label)VALUES($1,$2,-123,'2026-03-01','legacy','legacy')`, owner, a); e != nil {
				t.Fatal(e)
			}
			if _, e = s.pool.Exec(ctx, `INSERT INTO valuations(profile_id,asset_id,value_cents,as_of)VALUES($1,$2,10000,'2026-01-01')`, p.ID, a); e != nil {
				t.Fatal(e)
			}
			e = s.Migrate(ctx)
			if broken {
				if e == nil {
					t.Fatal("cross owner historical row was silently accepted")
				}
				var n int
				s.pool.QueryRow(ctx, `SELECT count(*)FROM schema_migrations WHERE version>='0011'`).Scan(&n)
				if n != 0 {
					t.Fatal("partial migration committed")
				}
			} else {
				if e != nil {
					t.Fatal(e)
				}
				nw, e := s.ComputeNetWorth(ctx, p.ID)
				if e != nil || nw.Net != 9877 {
					t.Fatal(nw, e)
				}
				if e = s.Migrate(ctx); e != nil {
					t.Fatal("restart migration", e)
				}
			}
			var count int
			var amount int64
			if e = s.pool.QueryRow(ctx, `SELECT count(*),sum(amount_cents) FROM transactions`).Scan(&count, &amount); e != nil || count != 1 || amount != -123 {
				t.Fatal("historical data changed", count, amount, e)
			}
		})
	}
}

func TestMigrationRefusesAmbiguousLegacyCurrencyUnits(t *testing.T) {
	s := legacyTestStore(t)
	ctx := context.Background()
	p, e := s.CreateProfile(ctx, "legacy XOF", "hash", "N1")
	if e != nil {
		t.Fatal(e)
	}
	var id string
	if e = s.pool.QueryRow(ctx, `INSERT INTO assets(profile_id,name,kind,currency)VALUES($1,'legacy cash','checking','XOF') RETURNING id`, p.ID).Scan(&id); e != nil {
		t.Fatal(e)
	}
	if _, e = s.pool.Exec(ctx, `INSERT INTO valuations(profile_id,asset_id,value_cents,as_of)VALUES($1,$2,10000,'2026-01-01')`, p.ID, id); e != nil {
		t.Fatal(e)
	}
	if e = s.Migrate(ctx); e == nil || !strings.Contains(e.Error(), "Legacy currency units") {
		t.Fatalf("ambiguous legacy scale accepted %v", e)
	}
	var v int64
	if e = s.pool.QueryRow(ctx, `SELECT value_cents FROM valuations WHERE asset_id=$1`, id).Scan(&v); e != nil || v != 10000 {
		t.Fatal("historical amount changed", v, e)
	}
	var n int
	if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version>='0011'`).Scan(&n); e != nil || n != 0 {
		t.Fatal("partial migration", n, e)
	}
}
