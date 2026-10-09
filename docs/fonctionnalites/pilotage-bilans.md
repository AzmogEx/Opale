# Allocation, snapshots et bilans mensuel/annuel

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Pilote automatique → Allocation cible : définir une répartition totalisant 100 % et lire les écarts. Historique mensuel présente les observations enregistrées. Bilan annuel permet de choisir l’année ; Assistant propose aussi bilan mensuel, jumeau financier et radar de risques.

## Données et comportement

`allocation_targets` stocke les pondérations ; les écarts comparent ces cibles au patrimoine enregistré sans passer d’ordres. `monthly_snapshots` enregistre une observation avec `recorded_at` : ce n’est pas nécessairement la clôture du dernier jour du mois.

Wrapped résume les opérations de l’année, catégories/marchands et patrimoines de début/fin lorsqu’ils sont connus. Le bilan mensuel et le jumeau utilisent les agrégats autorisés du profil. Les champs manquants restent manquants, et les explications ne créent pas des faits nouveaux.

## Limites et configuration

Pas de rééquilibrage broker automatique. L’absence d’historique complet limite comparaison et bilan ; aucune capture n’est rétroactivement inventée. IA facultative pour l’explication, règles de consentement identiques à l’assistant.

## Vérification

Tests backend des indicateurs/snapshots, intégration de l’allocation et `DeliveryUITests` pour le bilan annuel. Vérifier une année sans opérations, janvier pour le bilan du mois précédent, somme des cibles à 100 % et affichage discret des textes narratifs.

## API et sources

- `GET /v1/allocation`
- `GET /v1/monthly-review`
- `GET /v1/snapshots`
- `GET /v1/twin`
- `GET /v1/wrapped`
- `PUT /v1/allocation`

- [backend/internal/api/brain.go](../../backend/internal/api/brain.go)
- [backend/internal/api/pilote.go](../../backend/internal/api/pilote.go)
- [backend/internal/store/auto.go](../../backend/internal/store/auto.go)
- [backend/internal/twin/twin.go](../../backend/internal/twin/twin.go)
- [ios/Opale/Features/Assistant/ReviewSheet.swift](../../ios/Opale/Features/Assistant/ReviewSheet.swift)
- [ios/Opale/Features/Pilot/PilotToolsView.swift](../../ios/Opale/Features/Pilot/PilotToolsView.swift)
- [web/src/lib/components/Allocation.svelte](../../web/src/lib/components/Allocation.svelte)
