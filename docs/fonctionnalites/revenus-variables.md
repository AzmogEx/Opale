# Salaires et revenus variables

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Mon parcours → Revenus : ajouter un salaire fixe ou ouvrir les revenus variables. Pour primes, missions, loyers/dividendes : nom, type, devise, minimum/habituel/maximum, fréquence et date ; choisir hors prévision, minimum prudent ou montant habituel. Un revenu fixe utilise le net réellement versé après impôt, éventuellement prérempli depuis la paie.

## Données et comportement

`variable_incomes` porte la fourchette, le compte, la révision et la règle liée facultative. Le salaire fixe utilise le même modèle avec minimum = habituel = maximum. Les équivalents mensuels sont annualisés exactement et regroupés par devise ; les revenus ponctuels sont exclus de la moyenne récurrente.

L’hypothèse retenue alimente le calendrier et le mois type. Elle ne crée aucune transaction reçue et n’augmente pas le solde courant. Changer la devise d’un compte implique une ressaisie, pas une conversion supposée.

## Limites et configuration

UI dédiée sur iPhone ; web sans formulaire de fourchette dédié. Aucune prévision ne garantit le revenu. Le net fiscal, le brut et le net social d’une fiche de paie ne remplacent pas automatiquement le net payé. Les changements concurrents doivent recharger la révision serveur.

## Vérification

`FinancialToolsTests`, `FinancialJourneyTests`, tests Go `financial_tools_test.go` et store : fourchettes, annualisation, devises, prévisions et conflits. Le parcours UI vérifie salaire à 3 200 EUR avec zéro transaction réalisée créée par la déclaration.

## API et sources

- `DELETE /v1/incomes/variable/{id}`
- `GET /v1/incomes/variable`
- `PUT /v1/incomes/variable/{id}`

- [backend/internal/api/financial_tools.go](../../backend/internal/api/financial_tools.go)
- [backend/internal/store/variable_incomes.go](../../backend/internal/store/variable_incomes.go)
- [ios/Opale/Core/FinancialTools.swift](../../ios/Opale/Core/FinancialTools.swift)
- [ios/Opale/Features/Income/VariableIncomeEditSheet.swift](../../ios/Opale/Features/Income/VariableIncomeEditSheet.swift)
- [ios/Opale/Features/Income/VariableIncomesView.swift](../../ios/Opale/Features/Income/VariableIncomesView.swift)
- [ios/Opale/Features/Journey/FixedIncomeSheet.swift](../../ios/Opale/Features/Journey/FixedIncomeSheet.swift)
