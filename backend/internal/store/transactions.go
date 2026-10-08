package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
)

// Category — catégorie de transaction (EF-022). profile_id NULL = défaut global.
type Category struct {
	ID        string  `json:"id"`
	ProfileID *string `json:"profile_id,omitempty"`
	ParentID  *string `json:"parent_id,omitempty"`
	Name      string  `json:"name"`
	Icon      string  `json:"icon"`
}

// Transaction — un mouvement sur un compte (EF-020). Montant signé en centimes.
type Transaction struct {
	Currency          string      `json:"currency"`
	CurrencyExponent  int         `json:"currency_exponent"`
	FlowKind          string      `json:"flow_kind"`
	BankStatus        string      `json:"bank_status"`
	LinkedLiabilityID *string     `json:"linked_liability_id,omitempty"`
	TransferID        *string     `json:"transfer_id,omitempty"`
	SourceID          string      `json:"source_id,omitempty"`
	ID                string      `json:"id"`
	ProfileID         string      `json:"profile_id"`
	AssetID           string      `json:"asset_id"`
	Amount            money.Cents `json:"amount_cents"`
	OccurredOn        time.Time   `json:"occurred_on"`
	Label             string      `json:"label"`
	RawLabel          string      `json:"raw_label"`
	MerchantKey       string      `json:"-"`
	CategoryID        *string     `json:"category_id,omitempty"`
	CategoryName      *string     `json:"category_name,omitempty"`
	Note              string      `json:"note"`
	SpaceID           *string     `json:"space_id,omitempty"` // dépense commune (EF-007)
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

const txSelect = `
	SELECT t.id, t.profile_id, t.asset_id, t.amount_cents, t.occurred_on,
	       t.label, t.raw_label, t.merchant_key, t.category_id, c.name,
	       t.note, t.space_id, t.created_at, t.updated_at, a.currency, currency_exponent(a.currency),t.flow_kind,t.bank_status,t.source_id,t.transfer_id,t.linked_liability_id
	FROM transactions t JOIN assets a ON a.id=t.asset_id AND a.profile_id=t.profile_id
	LEFT JOIN categories c ON c.id = t.category_id`

func scanTransaction(row pgx.Row) (Transaction, error) {
	var t Transaction
	err := row.Scan(&t.ID, &t.ProfileID, &t.AssetID, &t.Amount, &t.OccurredOn,
		&t.Label, &t.RawLabel, &t.MerchantKey, &t.CategoryID, &t.CategoryName,
		&t.Note, &t.SpaceID, &t.CreatedAt, &t.UpdatedAt, &t.Currency, &t.CurrencyExponent, &t.FlowKind, &t.BankStatus, &t.SourceID, &t.TransferID, &t.LinkedLiabilityID)
	return t, err
}

// ── Catégories ────────────────────────────────────────────────────────────────

// ListCategories renvoie les catégories globales + celles du profil.
func (s *Store) ListCategories(ctx context.Context, profileID string) ([]Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, profile_id, parent_id, name, icon
		FROM categories
		WHERE profile_id IS NULL OR profile_id = $1
		ORDER BY name`, profileID)
	if err != nil {
		return nil, fmt.Errorf("ListCategories: %w", err)
	}
	defer rows.Close()

	cats := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ProfileID, &c.ParentID, &c.Name, &c.Icon); err != nil {
			return nil, fmt.Errorf("ListCategories: scan: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// ── Règles marchands (apprentissage EF-022) ───────────────────────────────────

// MerchantRuleNames renvoie les règles du profil : clé marchand → NOM de catégorie.
func (s *Store) MerchantRuleNames(ctx context.Context, profileID string) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.merchant_key, c.name
		FROM merchant_rules r
		JOIN categories c ON c.id = r.category_id
		WHERE r.profile_id = $1`, profileID)
	if err != nil {
		return nil, fmt.Errorf("MerchantRuleNames: %w", err)
	}
	defer rows.Close()

	m := map[string]string{}
	for rows.Next() {
		var key, name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, fmt.Errorf("MerchantRuleNames: scan: %w", err)
		}
		m[key] = name
	}
	return m, rows.Err()
}

// UpsertMerchantRule mémorise (ou remplace) la règle « clé marchand → catégorie »
// du profil — c'est l'apprentissage des corrections (EF-022).
func (s *Store) UpsertMerchantRule(ctx context.Context, profileID, merchantKey, categoryID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO merchant_rules (profile_id, merchant_key, category_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (profile_id, merchant_key)
		DO UPDATE SET category_id = EXCLUDED.category_id`,
		profileID, merchantKey, categoryID)
	if err != nil {
		return fmt.Errorf("UpsertMerchantRule: %w", err)
	}
	return nil
}

// ── Transactions : CRUD & liste filtrée (EF-020) ─────────────────────────────

// TransactionFilter — filtres de liste (tous optionnels).
type TransactionFilter struct {
	From, To   *time.Time
	Query      string // recherche plein-libellé (ILIKE)
	CategoryID string
	AssetID    string
	Limit      int
	Offset     int
}

// ListTransactions renvoie les mouvements du profil, du plus récent au plus ancien.
func (s *Store) ListTransactions(ctx context.Context, profileID string, f TransactionFilter) ([]Transaction, error) {
	where := []string{"t.profile_id = $1"}
	args := []any{profileID}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.From != nil {
		where = append(where, "t.occurred_on >= "+arg(*f.From))
	}
	if f.To != nil {
		where = append(where, "t.occurred_on <= "+arg(*f.To))
	}
	if f.Query != "" {
		p := arg("%" + f.Query + "%")
		where = append(where, "(t.label ILIKE "+p+" OR t.raw_label ILIKE "+p+")")
	}
	if f.CategoryID != "" {
		where = append(where, "t.category_id = "+arg(f.CategoryID))
	}
	if f.AssetID != "" {
		where = append(where, "t.asset_id = "+arg(f.AssetID))
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	q := txSelect + "\nWHERE " + strings.Join(where, " AND ") +
		"\nORDER BY t.occurred_on DESC, t.created_at DESC, t.id DESC" +
		"\nLIMIT " + arg(limit) + " OFFSET " + arg(max(f.Offset, 0))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTransactions: %w", err)
	}
	defer rows.Close()

	txs := []Transaction{}
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("ListTransactions: scan: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

// NewTransaction — données d'insertion d'un mouvement.
type NewTransaction struct {
	LinkedLiabilityID *string
	FlowKind          string
	SourceID          string
	BankStatus        string
	AssetID           string
	Amount            money.Cents
	OccurredOn        time.Time
	Label             string
	RawLabel          string
	MerchantKey       string
	CategoryID        *string
	Note              string
}

// CreateTransaction insère un mouvement et le renvoie complet.
func (s *Store) CreateTransaction(ctx context.Context, profileID string, n NewTransaction) (Transaction, error) {
	var result Transaction
	e := s.Atomic(ctx, func(st *Store) error {
		var e error
		if n.LinkedLiabilityID != nil {
			var id string
			if e = st.pool.QueryRow(ctx, `SELECT id FROM liabilities WHERE profile_id=$1 AND id=$2 FOR UPDATE`, profileID, *n.LinkedLiabilityID).Scan(&id); errors.Is(e, pgx.ErrNoRows) {
				return ErrNotFound
			} else if e != nil {
				return e
			}
		}
		result, e = st.createTransaction(ctx, profileID, n)
		if e != nil {
			return e
		}
		if n.LinkedLiabilityID != nil {
			return st.validateLiabilityHistory(ctx, profileID, *n.LinkedLiabilityID)
		}
		return nil
	})
	return result, e
}
func (s *Store) createTransaction(ctx context.Context, profileID string, n NewTransaction) (Transaction, error) {
	if n.FlowKind == "" {
		n.FlowKind = "expense_income"
	}
	if n.BankStatus == "" {
		n.BankStatus = "booked"
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO transactions
			(profile_id, asset_id, amount_cents, occurred_on, label, raw_label,
			 merchant_key, category_id, note,flow_kind,source_id,bank_status,linked_liability_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,$10,$11,$12,$13)
		RETURNING id`,
		profileID, n.AssetID, n.Amount, n.OccurredOn, n.Label, n.RawLabel,
		n.MerchantKey, n.CategoryID, n.Note, n.FlowKind, n.SourceID, n.BankStatus, n.LinkedLiabilityID,
	).Scan(&id)
	if err != nil {
		return Transaction{}, fmt.Errorf("CreateTransaction: %w", err)
	}
	return s.GetTransaction(ctx, profileID, id)
}

// GetTransaction renvoie un mouvement du profil.
func (s *Store) GetTransaction(ctx context.Context, profileID, id string) (Transaction, error) {
	t, err := scanTransaction(s.pool.QueryRow(ctx,
		txSelect+" WHERE t.id = $1 AND t.profile_id = $2", id, profileID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("GetTransaction: %w", err)
	}
	return t, nil
}

// TransactionPatch — champs modifiables (nil = inchangé).
type TransactionPatch struct {
	AssetID    *string
	FlowKind   *string
	Label      *string
	Note       *string
	CategoryID *string // pointeur vers "" pour effacer la catégorie
	OccurredOn *time.Time
	Amount     *money.Cents
}

// UpdateTransaction applique un patch partiel.
func (s *Store) UpdateTransaction(ctx context.Context, profileID, id string, p TransactionPatch) (Transaction, error) {
	linked, e := s.isLinkedMovement(ctx, profileID, id)
	if e != nil {
		return Transaction{}, e
	}
	if !linked {
		var loan *string
		if e := s.pool.QueryRow(ctx, `SELECT linked_liability_id FROM transactions WHERE profile_id=$1 AND id=$2`, profileID, id).Scan(&loan); e != nil {
			return Transaction{}, e
		}
		linked = loan != nil
	}
	if linked && (p.Amount != nil || p.AssetID != nil || p.OccurredOn != nil || p.FlowKind != nil) {
		return Transaction{}, fmt.Errorf("%w: supprimer puis recréer le mouvement lié", ErrInvalid)
	}
	sets := []string{}
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if p.AssetID != nil {
		sets = append(sets, "asset_id = "+arg(*p.AssetID))
	}
	if p.FlowKind != nil {
		sets = append(sets, "flow_kind = "+arg(*p.FlowKind))
	}
	if p.Label != nil {
		sets = append(sets, "label = "+arg(*p.Label))
	}
	if p.Note != nil {
		sets = append(sets, "note = "+arg(*p.Note))
	}
	if p.CategoryID != nil {
		if *p.CategoryID == "" {
			sets = append(sets, "category_id = NULL")
		} else {
			sets = append(sets, "category_id = "+arg(*p.CategoryID))
		}
	}
	if p.OccurredOn != nil {
		sets = append(sets, "occurred_on = "+arg(*p.OccurredOn))
	}
	if p.Amount != nil {
		sets = append(sets, "amount_cents = "+arg(*p.Amount))
	}
	if len(sets) == 0 {
		return s.GetTransaction(ctx, profileID, id)
	}

	args = append(args, id, profileID)
	q := fmt.Sprintf("UPDATE transactions SET %s WHERE id = $%d AND profile_id = $%d",
		strings.Join(sets, ", "), len(args)-1, len(args))
	tag, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return Transaction{}, fmt.Errorf("UpdateTransaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Transaction{}, ErrNotFound
	}
	return s.GetTransaction(ctx, profileID, id)
}

// ApplyCategoryToMerchant applique une catégorie à tous les mouvements du
// profil partageant la même clé marchand (correction « en masse », EF-022).
func (s *Store) ApplyCategoryToMerchant(ctx context.Context, profileID, merchantKey, categoryID string) (int64, error) {
	if merchantKey == "" {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE transactions SET category_id = $3
		WHERE profile_id = $1 AND merchant_key = $2`,
		profileID, merchantKey, categoryID)
	if err != nil {
		return 0, fmt.Errorf("ApplyCategoryToMerchant: %w", err)
	}
	return tag.RowsAffected(), nil
}

// DeleteTransaction supprime un mouvement du profil.
func (s *Store) DeleteTransaction(ctx context.Context, profileID, id string) error {
	linked, e := s.isLinkedMovement(ctx, profileID, id)
	if e != nil {
		return e
	}
	if linked {
		return fmt.Errorf("%w: supprimer le virement complet depuis son identifiant", ErrInvalid)
	}
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM transactions WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteTransaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Résumé mensuel (EF-020) ───────────────────────────────────────────────────

// MonthSummary — totaux d'un mois (centimes ; calcul SQL déterministe).
type MonthSummary struct {
	Income   money.Cents `json:"income_cents"`
	Expenses money.Cents `json:"expenses_cents"` // valeur positive
	Net      money.Cents `json:"net_cents"`
}

// ComputeMonthSummary agrège revenus/dépenses d'un mois calendaire.
func (s *Store) ComputeMonthSummary(ctx context.Context, profileID string, year int, month time.Month) (MonthSummary, error) {
	var income, expenses int64
	err := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(eur_cents) FILTER (WHERE amount_cents > 0), 0),
			COALESCE(-SUM(eur_cents) FILTER (WHERE amount_cents < 0), 0)
		FROM financial_transactions
		WHERE profile_id = $1
		  AND occurred_on >= make_date($2, $3, 1)
		  AND occurred_on < make_date($2, $3, 1) + interval '1 month'`,
		profileID, year, int(month),
	).Scan(&income, &expenses)
	if err != nil {
		return MonthSummary{}, fmt.Errorf("ComputeMonthSummary: %w", err)
	}
	net, err := money.Sub(money.Cents(income), money.Cents(expenses))
	if err != nil {
		return MonthSummary{}, err
	}
	return MonthSummary{
		Income:   money.Cents(income),
		Expenses: money.Cents(expenses),
		Net:      net,
	}, nil
}

// ── Import (EF-021) ───────────────────────────────────────────────────────────

var ErrImportAmbiguous = errors.New("import: possible legacy split requires reconciliation")

// ImportResult — bilan d'un import CSV.
type ImportResult struct {
	Imported    int `json:"imported"`
	Duplicates  int `json:"duplicates"`
	Categorized int `json:"categorized"`
}

// ImportTransactions insère un lot de mouvements préparés, en ignorant les
// doublons exacts (même compte, date, montant, libellé brut).
func importKey(n NewTransaction, occurrence int) string {
	if n.SourceID != "" {
		return "provider:" + n.SourceID
	}
	return fmt.Sprintf("csv:%x:%d", sha256.Sum256([]byte(n.OccurredOn.Format("2006-01-02")+fmt.Sprintf("|%d|", n.Amount)+n.RawLabel)), occurrence)
}
func (s *Store) ImportTransactions(ctx context.Context, profileID string, rows []NewTransaction) (ImportResult, error) {
	result := ImportResult{}
	err := s.Atomic(ctx, func(st *Store) error {
		// Serialise each profile's import, so two overlapping files cannot create duplicates.
		if _, err := st.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", profileID); err != nil {
			return err
		}
		occurrences := map[string]int{}
		for _, n := range rows {
			fingerprint := n.AssetID + importKey(n, 0)
			occurrences[fingerprint]++
			key := importKey(n, occurrences[fingerprint])
			var previousID *string
			var previousAmount int64
			err := st.pool.QueryRow(ctx, `SELECT transaction_id,original_amount FROM imported_operations WHERE profile_id=$1 AND asset_id=$2 AND source_key=$3`, profileID, n.AssetID, key).Scan(&previousID, &previousAmount)
			if err == nil {
				if previousID == nil && n.SourceID != "" && previousAmount != int64(n.Amount) {
					return fmt.Errorf("opération fournisseur modifiée après ventilation : réconciliation manuelle requise")
				}
				if previousID != nil && n.SourceID != "" {
					status := n.BankStatus
					if status == "" {
						status = "booked"
					}
					if _, err = st.pool.Exec(ctx, `UPDATE transactions SET amount_cents=$3,occurred_on=$4,bank_status=$5 WHERE id=$1 AND profile_id=$2 AND (bank_status='pending' OR $5='booked')`, *previousID, profileID, n.Amount, n.OccurredOn, status); err != nil {
						return err
					}
				}
				if n.SourceID != "" {
					if _, e := st.pool.Exec(ctx, `UPDATE imported_operations SET original_amount=$4,occurred_on=$5 WHERE profile_id=$1 AND asset_id=$2 AND source_key=$3`, profileID, n.AssetID, key, n.Amount, n.OccurredOn); e != nil {
						return e
					}
				}
				result.Duplicates++
				continue
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			// Recognise legacy rows before recording their durable source identity.
			var id string
			err = st.pool.QueryRow(ctx, `SELECT id FROM transactions t WHERE profile_id=$1 AND asset_id=$2 AND occurred_on=$3 AND amount_cents=$4 AND raw_label=$5 AND NOT EXISTS(SELECT 1 FROM imported_operations o WHERE o.transaction_id=t.id) ORDER BY id LIMIT 1`, profileID, n.AssetID, n.OccurredOn, n.Amount, n.RawLabel).Scan(&id)
			if errors.Is(err, pgx.ErrNoRows) {
				if n.SourceID == "" {
					var suspect bool
					if e := st.pool.QueryRow(ctx, `SELECT count(*)>1 AND COALESCE(SUM(amount_cents),0)=$5 FROM transactions t WHERE profile_id=$1 AND asset_id=$2 AND occurred_on=$3 AND raw_label=$4 AND NOT EXISTS(SELECT 1 FROM imported_operations o WHERE o.transaction_id=t.id)`, profileID, n.AssetID, n.OccurredOn, n.RawLabel, n.Amount).Scan(&suspect); e != nil {
						return e
					}
					if suspect {
						return ErrImportAmbiguous
					}
				}
				t, e := st.CreateTransaction(ctx, profileID, n)
				if e != nil {
					return e
				}
				id = t.ID
				result.Imported++
				if n.CategoryID != nil {
					result.Categorized++
				}
			} else if err != nil {
				return err
			} else {
				result.Duplicates++
			}
			if _, err = st.pool.Exec(ctx, `INSERT INTO imported_operations(profile_id,asset_id,source_key,transaction_id,original_amount,occurred_on,raw_label) VALUES($1,$2,$3,$4,$5,$6,$7)`, profileID, n.AssetID, key, id, n.Amount, n.OccurredOn, n.RawLabel); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

// SplitPart — une part d'un mouvement scindé (EF-024).
type SplitPart struct {
	Amount     money.Cents
	CategoryID *string
	Label      string
}

// SplitTransaction remplace un mouvement par plusieurs parts (split
// multi-catégories, EF-024). La somme des parts doit valoir exactement le
// montant d'origine (au centime — ENF-007) ; tout se joue dans une
// transaction SQL : jamais d'état intermédiaire visible.
func (s *Store) SplitTransaction(ctx context.Context, profileID, id string, parts []SplitPart) ([]Transaction, error) {
	if len(parts) < 2 || len(parts) > 100 {
		return nil, fmt.Errorf("SplitTransaction: au moins 2 parts requises")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	st := &Store{pool: tx}
	var lockedID string
	if err = tx.QueryRow(ctx, "SELECT id FROM transactions WHERE profile_id=$1 AND id=$2 FOR UPDATE", profileID, id).Scan(&lockedID); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	original, err := st.GetTransaction(ctx, profileID, id)
	if err != nil {
		return nil, err
	}
	if original.TransferID != nil || original.LinkedLiabilityID != nil {
		return nil, fmt.Errorf("ventilation interdite sur un virement lié")
	}
	var sum int64
	for _, p := range parts {
		if int64(p.Amount) == 0 {
			return nil, fmt.Errorf("SplitTransaction: une part ne peut pas être nulle")
		}
		next, e := money.Add(money.Cents(sum), p.Amount)
		if e != nil {
			return nil, e
		}
		sum = int64(next)
	}
	if sum != int64(original.Amount) {
		return nil, fmt.Errorf("SplitTransaction: la somme des parts (%d) doit égaler le montant d'origine (%d)",
			sum, int64(original.Amount))
	}

	n := NewTransaction{AssetID: original.AssetID, Amount: original.Amount, OccurredOn: original.OccurredOn, RawLabel: original.RawLabel, SourceID: original.SourceID}
	if _, err = tx.Exec(ctx, `INSERT INTO imported_operations(profile_id,asset_id,source_key,transaction_id,original_amount,occurred_on,raw_label) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, profileID, original.AssetID, importKey(n, 1), id, original.Amount, original.OccurredOn, original.RawLabel); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM transactions WHERE id = $1 AND profile_id = $2`, id, profileID); err != nil {
		return nil, fmt.Errorf("SplitTransaction: suppression : %w", err)
	}

	created := make([]Transaction, 0, len(parts))
	for i, p := range parts {
		label := p.Label
		if label == "" {
			label = fmt.Sprintf("%s (%d/%d)", original.Label, i+1, len(parts))
		}
		var t Transaction
		err := tx.QueryRow(ctx, `
			INSERT INTO transactions (profile_id, asset_id, amount_cents, occurred_on,
				label, raw_label, merchant_key, category_id, note, space_id,flow_kind,bank_status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,$11,$12)
			RETURNING id, created_at, updated_at`,
			profileID, original.AssetID, int64(p.Amount), original.OccurredOn,
			label, original.RawLabel, original.MerchantKey, p.CategoryID,
			original.Note, original.SpaceID, original.FlowKind, original.BankStatus,
		).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("SplitTransaction: part %d : %w", i+1, err)
		}
		t.Currency, t.CurrencyExponent, t.FlowKind, t.BankStatus = original.Currency, original.CurrencyExponent, original.FlowKind, original.BankStatus
		t.ProfileID, t.AssetID = profileID, original.AssetID
		t.Amount, t.OccurredOn = p.Amount, original.OccurredOn
		t.Label, t.RawLabel = label, original.RawLabel
		t.CategoryID, t.Note, t.SpaceID = p.CategoryID, original.Note, original.SpaceID
		created = append(created, t)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("SplitTransaction: %w", err)
	}
	return created, nil
}
