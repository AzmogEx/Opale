# Enveloppes budgétaires et objectifs

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Flux → Enveloppes : fixer le budget d’une catégorie et comparer consommé/reste. Projection → Objectifs : nommer le projet, saisir sa cible, son avancement et son rythme d’épargne ; corriger ou supprimer ensuite. Dans Mon parcours, déclarer aussi le budget mensuel des dépenses variables.

## Données et comportement

`envelopes` compare les opérations réalisées du mois à la limite de catégorie. `goals` conserve cible, avancement et contribution affectée ; le serveur sérialise les affectations pour ne pas promettre plusieurs fois la même capacité d’épargne observée.

Le budget variable du parcours est une hypothèse de mois type, distincte des enveloppes par catégorie et des dépenses réellement enregistrées. Une date d’atteinte d’objectif vient d’hypothèses explicites et du moteur ; les objectifs non finançables ne sont pas déclarés réalisés.

## Limites et configuration

Affecter une somme n’effectue aucun virement et ne bloque pas d’argent en banque. Un historique insuffisant peut empêcher de justifier une capacité d’épargne. Ne pas additionner mécaniquement le budget global du parcours et toutes les enveloppes comme des charges supplémentaires.

## Vérification

Tests Go `delivery_goal_test.go`, store des objectifs et API de pilotage ; intégration `modules.spec.ts` crée puis corrige une cible à 6 000 EUR. Tests du parcours couvrent brouillon exact, étapes passées et calcul du reste mensuel.

## API et sources

- `DELETE /v1/envelopes/{id}`
- `DELETE /v1/goals/{id}`
- `GET /v1/envelopes`
- `GET /v1/goals`
- `PATCH /v1/goals/{id}`
- `POST /v1/goals`
- `PUT /v1/envelopes`

- [backend/internal/api/pilotage.go](../../backend/internal/api/pilotage.go)
- [ios/Opale/Features/Flows/PilotageViews.swift](../../ios/Opale/Features/Flows/PilotageViews.swift)
- [ios/Opale/Features/Projection/GoalsSection.swift](../../ios/Opale/Features/Projection/GoalsSection.swift)
- [web/src/lib/components/Budgets.svelte](../../web/src/lib/components/Budgets.svelte)
- [web/src/lib/components/Goals.svelte](../../web/src/lib/components/Goals.svelte)
