# Patrimoine net, trésorerie et santé financière

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Accueil présente patrimoine net, évolution, historique, trésorerie, santé financière, risques et alertes. Sur iPhone, le parcours est la première entrée ; l’accueil chiffré reste accessible. Sélectionner un point du graphique pour sa date/valeur et ouvrir les analyses pour détailler les dépenses.

## Données et comportement

Le patrimoine net est la somme des actifs courants moins les dettes courantes, convertie en EUR par le moteur. Une position sans valeur ou taux exploitable rend le total incomplet ; elle n’est pas silencieusement valorisée à zéro. Le cash ne confond pas un bien immobilier avec de la liquidité.

Le score /100 et ses composantes, les risques de concentration/liquidité/endettement et les jalons sont calculés à partir des données enregistrées. Les revenus/dépenses réalisés excluent les virements internes et les opérations provisoires. Le fonds d’urgence ne peut être affirmé sans dépenses connues.

## Limites et configuration

Un score est un indicateur du moteur, sans garantie financière. L’historique dépend des valorisations et écritures datées disponibles. Un profil peu renseigné donne une couverture partielle. Les graphiques utilisent des valeurs de tracé approchées ; les montants affichés et calculés gardent les unités exactes.

## Vérification

Tests Go `engine/p4_test.go`, `p5_test.go`, `delivery_extremes_test.go`, tests d’intégrité API ; `FinancialPresentationTests` et navigateur `privacy.spec.ts`. Vérifier un compte avec/sans valorisation, un taux absent, une dette et un virement : aucun revenu fictif ni total complet abusif.

## API et sources

- `GET /v1/analytics`
- `GET /v1/health-score`
- `GET /v1/net-worth`
- `GET /v1/net-worth/history`
- `GET /v1/risks`

- [backend/internal/api/brain.go](../../backend/internal/api/brain.go)
- [backend/internal/api/networth.go](../../backend/internal/api/networth.go)
- [backend/internal/api/pilotage.go](../../backend/internal/api/pilotage.go)
- [ios/Opale/Features/Home/AnalyticsView.swift](../../ios/Opale/Features/Home/AnalyticsView.swift)
- [ios/Opale/Features/Home/HomeView.swift](../../ios/Opale/Features/Home/HomeView.swift)
- [web/src/lib/components/HealthRing.svelte](../../web/src/lib/components/HealthRing.svelte)
- [web/src/lib/components/Milestones.svelte](../../web/src/lib/components/Milestones.svelte)
- [web/src/routes/+page.svelte](../../web/src/routes/+page.svelte)
