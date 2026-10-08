package store

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

// ExportZIP streams a repeatable-read snapshot into a caller-owned private file.
// The completed manifest is only written after every document was decrypted and verified.
func (s *Store) ExportZIP(ctx context.Context, profileID string, out io.Writer, decrypt func([]byte) ([]byte, error)) error {
	zw := zip.NewWriter(out)
	err := s.Snapshot(ctx, func(st *Store) error {
		counts := map[string]int64{}
		checksums := map[string]string{}
		entry, err := zw.Create("export.json")
		if err != nil {
			return err
		}
		hash := sha256.New()
		w := io.MultiWriter(entry, hash)
		if _, err = io.WriteString(w, `{"format":"opale-export/2","reference_currency":"EUR","amounts":"integer native currency minor units; currency_exponent defines scale","valuation_boundary":"closing balance inclusive of civil date","exported_at":`); err != nil {
			return err
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(time.Now().UTC()); err != nil {
			return err
		}
		tables := []struct{ name, query string }{
			{"profile", `SELECT jsonb_build_object('id',id,'name',name,'privacy_default',privacy_default,'created_at',created_at,'updated_at',updated_at) FROM profiles WHERE id=$1`},
			{"assets", `SELECT to_jsonb(t)||jsonb_build_object('currency_exponent',currency_exponent(currency)) FROM assets t WHERE profile_id=$1 ORDER BY id`},
			{"liabilities", `SELECT to_jsonb(t)||jsonb_build_object('currency_exponent',currency_exponent(currency)) FROM liabilities t WHERE profile_id=$1 ORDER BY id`},
			{"transactions", `SELECT to_jsonb(t) FROM transactions t WHERE profile_id=$1 ORDER BY occurred_on,id`},
			{"valuations", `SELECT to_jsonb(t) FROM valuations t WHERE profile_id=$1 ORDER BY as_of,id`},
			{"categories", `SELECT to_jsonb(t) FROM categories t WHERE profile_id=$1 OR profile_id IS NULL ORDER BY id`},
			{"documents", `SELECT to_jsonb(t)-'content' FROM documents t WHERE profile_id=$1 ORDER BY id`},
			{"spaces", `SELECT to_jsonb(t) FROM spaces t WHERE EXISTS(SELECT 1 FROM space_members m WHERE m.space_id=t.id AND m.profile_id=$1) ORDER BY id`},
			{"space_members", `SELECT to_jsonb(t) FROM space_members t WHERE EXISTS(SELECT 1 FROM space_members m WHERE m.space_id=t.space_id AND m.profile_id=$1) ORDER BY space_id,profile_id`},
			{"fx_history", `SELECT to_jsonb(t) FROM fx_history t WHERE currency IN(SELECT currency FROM assets WHERE profile_id=$1 UNION SELECT currency FROM liabilities WHERE profile_id=$1) ORDER BY currency,as_of`},
			{"emergency_grants", `SELECT to_jsonb(t) FROM emergency_grants t WHERE owner_profile_id=$1 ORDER BY id`},
			{"bank_links", `SELECT to_jsonb(t)-'requisition_id' FROM bank_links t WHERE profile_id=$1 ORDER BY id`},
		}
		// Explicit allowlist: new tables are never silently exported with secrets.
		for _, name := range []string{"bank_account_bindings", "profile_fx_history", "bank_accounts", "bank_pending", "envelopes", "merchant_rules", "goals", "contacts", "property_details", "object_details", "company_details", "monthly_snapshots", "allocation_targets", "custom_alerts", "imported_operations", "calendar_rules", "calendar_occurrences", "recurring_exclusions", "investment_flows", "investment_coverage", "beneficiaries"} {
			var exists bool
			if err = st.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND column_name='profile_id')`, name).Scan(&exists); err != nil {
				return err
			}
			if exists {
				tables = append(tables, struct{ name, query string }{name, `SELECT to_jsonb(t) FROM ` + name + ` t WHERE profile_id=$1`})
			}
		}
		for _, table := range tables {
			if _, err = fmt.Fprintf(w, ",%q:[", table.name); err != nil {
				return err
			}
			rows, e := st.pool.Query(ctx, table.query, profileID)
			if e != nil {
				return e
			}
			var count int64
			for rows.Next() {
				var raw []byte
				if err = rows.Scan(&raw); err != nil {
					rows.Close()
					return err
				}
				if count > 0 {
					if _, err = w.Write([]byte(",")); err != nil {
						rows.Close()
						return err
					}
				}
				if _, err = w.Write(raw); err != nil {
					rows.Close()
					return err
				}
				count++
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			counts[table.name] = count
			if _, err = io.WriteString(w, "]"); err != nil {
				return err
			}
		}
		if _, err = io.WriteString(w, "}"); err != nil {
			return err
		}
		checksums["export.json"] = hex.EncodeToString(hash.Sum(nil))
		rows, err := st.pool.Query(ctx, `SELECT id,name,content,sha256 FROM documents WHERE profile_id=$1 ORDER BY id`, profileID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, name, digest string
			var sealed []byte
			if err = rows.Scan(&id, &name, &sealed, &digest); err != nil {
				return err
			}
			if decrypt == nil {
				return fmt.Errorf("vault unavailable: documents cannot be exported")
			}
			plain, e := decrypt(sealed)
			if e != nil {
				return fmt.Errorf("document integrity verification failed")
			}
			h := sha256.Sum256(plain)
			if hex.EncodeToString(h[:]) != digest {
				return fmt.Errorf("document checksum mismatch")
			}
			file := "documents/" + id + "-" + path.Base(strings.ReplaceAll(name, "\\", "/"))
			f, e := zw.Create(file)
			if e != nil {
				return e
			}
			if _, e = f.Write(plain); e != nil {
				return e
			}
			checksums[file] = digest
		}
		if err = rows.Err(); err != nil {
			return err
		}
		f, err := zw.Create("manifest.json")
		if err != nil {
			return err
		}
		return json.NewEncoder(f).Encode(map[string]any{"format": "opale-export/2", "complete": true, "counts": counts, "sha256": checksums, "exclusions": []string{"PIN hashes", "sessions", "push tokens", "provider credentials", "other profiles' private data"}})
	})
	closeErr := zw.Close()
	if err != nil {
		return err
	}
	return closeErr
}
