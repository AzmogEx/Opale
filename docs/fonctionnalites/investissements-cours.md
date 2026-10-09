# Placements, performance et cotations

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → Investissements : sélectionner une position, ajouter ses valorisations et enregistrer apports, retraits, distributions et frais externes. Confirmer la période de couverture des flux avant d’interpréter une performance. Pilote automatique → Cours automatiques permet d’associer une position à une source puis actualiser.

## Données et comportement

`investment_flows` et `investment_coverage` distinguent variation de valeur et gain : passer de 1 000 à 1 500 avec un apport de 500 ne constitue pas un gain de marché. Sans couverture suffisante, la performance est indisponible/partielle, pas inventée.

Les métadonnées de cotation conservent source, date et erreur. Une association valorise la position entière selon ses paramètres ; séparer les positions si plusieurs titres sont détenus. Les devises sont traitées par les conventions communes.

## Limites et configuration

Suivi patrimonial, sans ordre d’achat/vente ni connexion broker universelle. Les sources actuellement implémentées ne couvrent pas tous les ETF/actions ; vérifier le fournisseur et l’identifiant réellement supportés. Cotations automatiques seulement si activées côté serveur, données retardées possibles.

Sur le web, les détails sont chargés avant ouverture du formulaire : une réponse tardive ne peut pas effacer une saisie en cours. Un échec propose une nouvelle tentative.

## Vérification

`engine/delivery_investments_test.go`, tests API/store et intégration `advanced.spec.ts` : apport 500, gain exactement zéro et couverture relue. La validation réelle BCE/CoinGecko est consignée dans la livraison backend ; elle ne valide pas tous les instruments.

## API et sources

- `DELETE /v1/assets/{id}/investment/flows/{flowID}`
- `GET /v1/assets/{id}/investment`
- `GET /v1/assets/{id}/quote`
- `GET /v1/investments`
- `PATCH /v1/assets/{id}/investment/flows/{flowID}`
- `POST /v1/assets/{id}/investment/flows`
- `POST /v1/quotes/refresh`
- `PUT /v1/assets/{id}/investment/coverage`
- `PUT /v1/assets/{id}/quote`

- [backend/internal/api/crud.go](../../backend/internal/api/crud.go)
- [backend/internal/api/delivery_investments.go](../../backend/internal/api/delivery_investments.go)
- [backend/internal/api/pilote.go](../../backend/internal/api/pilote.go)
- [backend/internal/api/profondeur.go](../../backend/internal/api/profondeur.go)
- [ios/Opale/Features/Wealth/Centers/InvestmentDetailView.swift](../../ios/Opale/Features/Wealth/Centers/InvestmentDetailView.swift)
- [ios/Opale/Features/Wealth/Centers/InvestmentsView.swift](../../ios/Opale/Features/Wealth/Centers/InvestmentsView.swift)
- [web/src/lib/components/Investments.svelte](../../web/src/lib/components/Investments.svelte)
