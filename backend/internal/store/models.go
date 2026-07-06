package store

import (
	"time"

	"github.com/opale-app/opale/internal/money"
)

// Profile — un profil (utilisateur, parent…). Cloison de données privée (EF-001).
type Profile struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	PrivacyDefault string    `json:"privacy_default"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Asset — actif (compte, livret, placement, immobilier, objet…).
type Asset struct {
	ID          string       `json:"id"`
	ProfileID   string       `json:"profile_id"`
	Name        string       `json:"name"`
	Kind        string       `json:"kind"`
	Currency    string       `json:"currency"`
	Note        string       `json:"note"`
	Archived    bool         `json:"archived"`
	LatestValue *money.Cents `json:"latest_value_cents"` // dernière valorisation, nil si aucune
	// TheoreticalValue (comptes de flux) : dernière valorisation + mouvements
	// postérieurs — signale la dérive quand la valorisation n'a pas suivi.
	TheoreticalValue *money.Cents `json:"theoretical_cents,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// Liability — passif (crédit immo/auto/conso).
type Liability struct {
	ID          string       `json:"id"`
	ProfileID   string       `json:"profile_id"`
	Name        string       `json:"name"`
	Kind        string       `json:"kind"`
	Currency    string       `json:"currency"`
	Note        string       `json:"note"`
	Archived    bool         `json:"archived"`
	LatestValue *money.Cents `json:"latest_value_cents"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Valuation — snapshot daté de la valeur d'un actif ou d'un passif (EF-032).
type Valuation struct {
	ID          string      `json:"id"`
	ProfileID   string      `json:"profile_id"`
	AssetID     *string     `json:"asset_id,omitempty"`
	LiabilityID *string     `json:"liability_id,omitempty"`
	Value       money.Cents `json:"value_cents"`
	AsOf        time.Time   `json:"as_of"`
	Note        string      `json:"note"`
	CreatedAt   time.Time   `json:"created_at"`
}

// NetWorth — résultat du calcul de patrimoine net (CA-1). Tous les montants en
// centimes ; calcul déterministe (jamais produit par l'IA — EIA-040/041).
type NetWorth struct {
	AssetsTotal      money.Cents `json:"assets_total_cents"`
	LiabilitiesTotal money.Cents `json:"liabilities_total_cents"`
	Net              money.Cents `json:"net_cents"`
	Currency         string      `json:"currency"`
}

// AssetKinds et LiabilityKinds : valeurs autorisées (doivent rester alignées
// avec les CHECK SQL de la migration 0001).
var AssetKinds = map[string]bool{
	"checking": true, "savings": true, "life_insurance": true, "pea": true,
	"cto": true, "crypto": true, "real_estate": true, "precious_metal": true,
	"vehicle": true, "valuable": true, "company_share": true, "other": true,
}

var LiabilityKinds = map[string]bool{
	"mortgage": true, "auto_loan": true, "consumer_loan": true, "other": true,
}
