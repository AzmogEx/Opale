# Objets de valeur et assurance

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → Objets : suivre montre, voiture, bijou, œuvre ou matériel en créant son actif et ses informations spécialisées. Renseigner achat, estimation et informations d’assurance/identification disponibles ; conserver une facture dans le coffre si souhaité.

## Données et comportement

`object_details` complète l’actif sans remplacer son historique de valorisations. Le patrimoine utilise les valeurs datées, pas un prix de revente deviné. Les documents du coffre restent une gestion distincte ; les champs de description ne constituent pas un dépôt de document.

## Limites et configuration

Aucune expertise automatique, cote garantie, assurance souscrite ni place de marché. Toute estimation est déclarative. Un document téléchargé est une copie lisible à protéger sur l’appareil.

Sur le web, les détails sont chargés avant ouverture du formulaire : une réponse tardive ne peut pas effacer une saisie en cours. Un échec propose une nouvelle tentative.

## Vérification

Tests des handlers de profondeur et relations propriétaires ; recette manuelle création/édition/relecture et contrôle de la valeur dans le patrimoine. Les captures historiques ne constituent pas une validation de chaque champ.

## API et sources

- `GET /v1/objects`
- `PUT /v1/assets/{id}/object`

- [backend/internal/api/profondeur.go](../../backend/internal/api/profondeur.go)
- [ios/Opale/Features/Wealth/Centers/ObjectsView.swift](../../ios/Opale/Features/Wealth/Centers/ObjectsView.swift)
