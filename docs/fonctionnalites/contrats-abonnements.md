# Contrats, abonnements et engagements

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Sur iPhone : Mon parcours → Charges/Abonnements ou Réglages → Contrats et abonnements. Saisir nom, montant/devise, fréquence, compte et marchand surveillé, puis éventuellement essai, engagement, renouvellement, reconduction, préavis et rappel. La fiche montre tarif, prochaines dates et historique ; modifier, arrêter/réactiver le suivi ou supprimer.

## Données et comportement

`financial_contracts` stocke l’engagement et sa révision ; `contract_prices` conserve les tarifs datés. Une prévision calendrier est facultative et liée au contrat. Le backend reprend les abonnements de l’ancien formulaire terminé via migration 0024 sans dupliquer leurs règles.

Une modification de tarif clôture la série précédente et construit la suite, sans réécrire les paiements passés. Le serveur refuse les révisions périmées. La fiche distingue suivi uniquement et prévision reliée ; rapprocher les paiements constatés évite le double comptage.

## Limites et configuration

L’interface dédiée est iOS ; les endpoints existent mais le web ne fournit pas encore son gestionnaire d’engagements complet. Arrêter le suivi ou supprimer ne résilie rien auprès du fournisseur. Le préavis est celui déclaré par l’utilisateur, sans vérification juridique automatique.

## Vérification

`backend/internal/api/financial_tools_test.go`, tests store des contrats et `FinancialToolsTests` : isolation, conflit, règle gérée et fréquence. Recette : créer un contrat mensuel, corriger son tarif, relire l’historique et contrôler les anciennes occurrences.

## API et sources

- `DELETE /v1/contracts/{id}`
- `GET /v1/contracts`
- `PUT /v1/contracts/{id}`

- [backend/internal/api/financial_tools.go](../../backend/internal/api/financial_tools.go)
- [backend/internal/migrations/0024_contracts_income.up.sql](../../backend/internal/migrations/0024_contracts_income.up.sql)
- [backend/internal/store/contracts.go](../../backend/internal/store/contracts.go)
- [ios/Opale/Features/Contracts/ContractDetailView.swift](../../ios/Opale/Features/Contracts/ContractDetailView.swift)
- [ios/Opale/Features/Contracts/ContractEditSheet.swift](../../ios/Opale/Features/Contracts/ContractEditSheet.swift)
- [ios/Opale/Features/Contracts/ContractsView.swift](../../ios/Opale/Features/Contracts/ContractsView.swift)
- [ios/Opale/Features/Contracts/FinancialFormFields.swift](../../ios/Opale/Features/Contracts/FinancialFormFields.swift)
