# Actifs, dettes, valorisations et devises

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → ajouter un actif ou une dette ; préciser type, devise et éventuellement valeur de clôture datée. Ouvrir sa fiche, compléter l’historique, corriger/supprimer une valorisation ou archiver. Réglages/Devises permet un taux privé daté avec source. Les centres spécialisés enrichissent les actifs existants.

## Données et comportement

Actif/dette et première valorisation sont créés atomiquement avec identifiant idempotent. Le solde courant d’un compte = dernière clôture + opérations comptabilisées strictement postérieures ; les écritures du même jour sont déjà incluses. Pour une dette, seul le principal remboursé diminue le capital ; intérêts/frais sont des dépenses.

L’archive retire la position du courant selon sa date et conserve l’historique. Les taux privés datés du profil ont priorité sur les références publiques ; un taux manquant n’est jamais remplacé par 1. EUR/USD à deux décimales, JPY zéro et KWD trois ; les totaux du moteur sont en EUR.

## Limites et configuration

Un compte avec mouvements ne se supprime pas librement : archiver ou corriger explicitement les dépendances. Une valeur inconnue n’est pas zéro. Une clôture ultérieure ne doit pas masquer un capital négatif historique. Les cours et taux automatiques doivent être configurés et datés.

## Vérification

Tests store/API d’intégrité, `FinancialPresentationTests`, `web/tests/domain.test.mjs` et intégration quotidienne : exactitude, devises, idempotence, clôture, remboursement et taux privés. Détails normatifs : [conventions financières](../CONVENTIONS-FINANCIERES.md).

## API et sources

- `DELETE /v1/assets/{id}`
- `DELETE /v1/fx/{currency}`
- `DELETE /v1/liabilities/{id}`
- `DELETE /v1/valuations/{id}`
- `GET /v1/assets`
- `GET /v1/assets/{id}`
- `GET /v1/assets/{id}/valuations`
- `GET /v1/fx`
- `GET /v1/liabilities`
- `GET /v1/liabilities/{id}`
- `GET /v1/liabilities/{id}/valuations`
- `PATCH /v1/assets/{id}`
- `PATCH /v1/liabilities/{id}`
- `PATCH /v1/valuations/{id}`
- `POST /v1/assets`
- `POST /v1/assets/{id}/valuations`
- `POST /v1/liabilities`
- `POST /v1/liabilities/{id}/valuations`
- `PUT /v1/fx/{currency}`

- [backend/internal/api/assets.go](../../backend/internal/api/assets.go)
- [backend/internal/api/crud.go](../../backend/internal/api/crud.go)
- [backend/internal/api/liabilities.go](../../backend/internal/api/liabilities.go)
- [backend/internal/api/partage.go](../../backend/internal/api/partage.go)
- [backend/internal/store/fx.go](../../backend/internal/store/fx.go)
- [backend/internal/store/networth.go](../../backend/internal/store/networth.go)
- [backend/internal/store/patrimoine.go](../../backend/internal/store/patrimoine.go)
- [ios/Opale/Core/Money.swift](../../ios/Opale/Core/Money.swift)
- [ios/Opale/Features/Wealth/FXRatesSheet.swift](../../ios/Opale/Features/Wealth/FXRatesSheet.swift)
- [ios/Opale/Features/Wealth/HoldingDetailView.swift](../../ios/Opale/Features/Wealth/HoldingDetailView.swift)
- [ios/Opale/Features/Wealth/WealthForms.swift](../../ios/Opale/Features/Wealth/WealthForms.swift)
- [ios/Opale/Features/Wealth/WealthView.swift](../../ios/Opale/Features/Wealth/WealthView.swift)
- [web/src/lib/components/AssetDetails.svelte](../../web/src/lib/components/AssetDetails.svelte)
- [web/src/lib/components/Currencies.svelte](../../web/src/lib/components/Currencies.svelte)
- [web/src/lib/components/Holdings.svelte](../../web/src/lib/components/Holdings.svelte)
- [web/src/lib/domain.ts](../../web/src/lib/domain.ts)
- [web/src/routes/patrimoine/+page.svelte](../../web/src/routes/patrimoine/+page.svelte)
