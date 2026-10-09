# Projections, indépendance financière et chronologie

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Projection : saisir épargne mensuelle, rendement, inflation et horizon ; comparer nominal et euros constants. Consulter l’objectif d’indépendance financière et sa date estimée. Créer/comparer des scénarios et ouvrir la chronologie patrimoniale pour les jalons.

## Données et comportement

Le moteur applique les hypothèses explicites aux valeurs disponibles. La cible FIRE utilise sa convention de retrait et les dépenses renseignées ; le calendrier d’atteinte respecte les dates civiles. Rendement nul, inflation supérieure au rendement, horizon non atteignable et données manquantes ont des résultats distincts.

Les scénarios comparent des futurs hypothétiques sans modifier les actifs, contrats ou transactions réels. Les scénarios sauvegardés restent dans le client avec séparation par profil ; ce n’est pas une table de transactions exportée. Les objectifs affectent leur rythme suivant les contrôles de capacité.

## Limites et configuration

Une projection n’est ni une promesse de rendement ni une date de retraite garantie. Elle dépend de la couverture des données, des hypothèses et des conventions ; elle ne prédit pas les marchés. La chronologie n’est pas un calendrier d’ordres automatiques.

## Vérification

Tests `engine/projection_test.go`, `delivery_goal_test.go`, tests de scénarios et contrats API ; tests web des dates FIRE et de la présentation. Vérifier rendement zéro, contributions seules, inflation forte, objectif déjà atteint et horizon insuffisant.

## API et sources

- `GET /v1/projection`
- `GET /v1/timeline`
- `POST /v1/scenarios/compare`

- [backend/internal/api/profondeur.go](../../backend/internal/api/profondeur.go)
- [backend/internal/api/projection.go](../../backend/internal/api/projection.go)
- [backend/internal/api/scenarios.go](../../backend/internal/api/scenarios.go)
- [ios/Opale/Features/Projection/ComparisonView.swift](../../ios/Opale/Features/Projection/ComparisonView.swift)
- [ios/Opale/Features/Projection/ProjectionView.swift](../../ios/Opale/Features/Projection/ProjectionView.swift)
- [ios/Opale/Features/Wealth/Centers/TimelineView.swift](../../ios/Opale/Features/Wealth/Centers/TimelineView.swift)
- [web/src/lib/components/ProjectionSimulator.svelte](../../web/src/lib/components/ProjectionSimulator.svelte)
- [web/src/lib/components/ScenarioComparison.svelte](../../web/src/lib/components/ScenarioComparison.svelte)
- [web/src/routes/projection/+page.svelte](../../web/src/routes/projection/+page.svelte)
