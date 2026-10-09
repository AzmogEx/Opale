# Transactions, recherche et import de relevés

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Flux → ajouter une opération : compte, date, libellé, montant dans sa devise et catégorie ; ajouter une note si utile. Rechercher, filtrer, changer de page, corriger ou supprimer. Pour un relevé, choisir explicitement son compte, ouvrir CSV/OFX, vérifier l’aperçu puis confirmer l’import.

## Données et comportement

Les opérations conservent libellé actuel et source brute, date civile, devise, catégorie et état comptabilisé/provisoire. Les imports détectent leurs identités durables dans `imported_operations`, y compris après ventilation. CSV UTF-8/Windows-1252 et OFX sont traités par le backend ; le résultat indique importés/doublons/rejets réels.

La pagination web utilise 50 lignes avec détection de page suivante. Une dépense est un montant négatif ; le solde d’un compte part de sa clôture datée et ajoute uniquement les écritures ultérieures comptabilisées. Une opération future n’augmente pas le solde actuel.

## Limites et configuration

Imports limités à 5 Mio et 10 000 opérations. Ne pas changer arbitrairement de compte/devise pour contourner un doublon. Les écritures liées aux virements ou au principal d’une dette ont des restrictions d’édition ; corriger l’ensemble concerné. Aucun rapprochement bancaire universel n’est promis.

## Vérification

Parcours réel jetable `web/tests/integration/daily.spec.ts` : compte à 1 000,25 EUR, 55 lignes importées, pagination, recherche, correction, export. Tests store/API d’import et d’intégrité. Tester le même fichier deux fois : pas de multiplication des dépenses.

## API et sources

- `DELETE /v1/transactions/{id}`
- `GET /v1/transactions`
- `GET /v1/transactions/summary`
- `PATCH /v1/transactions/{id}`
- `POST /v1/transactions`
- `POST /v1/transactions/import`

- [backend/internal/api/transactions.go](../../backend/internal/api/transactions.go)
- [ios/Opale/Features/Flows/FlowsView.swift](../../ios/Opale/Features/Flows/FlowsView.swift)
- [ios/Opale/Features/Flows/TransactionSheets.swift](../../ios/Opale/Features/Flows/TransactionSheets.swift)
- [web/src/lib/components/Transactions.svelte](../../web/src/lib/components/Transactions.svelte)
- [web/src/routes/flux/+page.svelte](../../web/src/routes/flux/+page.svelte)
