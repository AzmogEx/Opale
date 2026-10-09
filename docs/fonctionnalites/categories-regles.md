# Catégories, règles et suggestions de libellés

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Flux ou Pilote automatique → Catégories & règles : créer, renommer ou supprimer une catégorie personnalisée, associer une règle marchand et corriger le classement d’une opération. Sur iPhone, demander explicitement une suggestion sur l’appareil ; sur le web, prévisualiser la suggestion de libellé du homelab, accepter ou refuser.

## Données et comportement

Les catégories et `merchant_rules` sont isolées par profil. Les imports appliquent les règles du serveur. Une suggestion reste distincte d’une modification : l’acceptation enregistre les champs prévus, le refus ne modifie rien.

Foundation Models propose une catégorie uniquement parmi les catégories disponibles et peut proposer une règle après confirmation. La suggestion de libellé backend transmet au homelab privé le seul libellé actuel ; elle n’envoie ni montant, note, catégorie, document ou libellé brut importé. Le cloud n’est pas utilisé pour ce nettoyage.

## Limites et configuration

Foundation Models exige un iPhone compatible avec Apple Intelligence disponible. Homelab non configuré ou réponse invalide : état indisponible et correction manuelle. Les règles peuvent être erronées ; elles ne dispensent pas de contrôler les opérations.

## Vérification

`backend/internal/api/label_suggestion_test.go`, `web/tests/e2e/label-suggestion.spec.ts` et `web/tests/integration/decisions-label.spec.ts` couvrent acceptation/refus, indisponibilité et absence d’écriture implicite. Tester les suggestions iPhone réellement après activation d’Apple Intelligence.

## API et sources

- `DELETE /v1/categories/{id}`
- `DELETE /v1/rules/{id}`
- `GET /v1/categories`
- `GET /v1/rules`
- `PATCH /v1/categories/{id}`
- `PATCH /v1/rules/{id}`
- `POST /v1/categories`
- `POST /v1/rules`
- `POST /v1/transactions/{id}/label-suggestion`

- [backend/internal/api/crud.go](../../backend/internal/api/crud.go)
- [backend/internal/api/label_suggestion.go](../../backend/internal/api/label_suggestion.go)
- [backend/internal/api/transactions.go](../../backend/internal/api/transactions.go)
- [backend/internal/store/financial_rules.go](../../backend/internal/store/financial_rules.go)
- [ios/Opale/Core/LocalAI.swift](../../ios/Opale/Core/LocalAI.swift)
- [ios/Opale/Features/Pilot/CategoriesRulesView.swift](../../ios/Opale/Features/Pilot/CategoriesRulesView.swift)
- [web/src/lib/components/Categories.svelte](../../web/src/lib/components/Categories.svelte)
