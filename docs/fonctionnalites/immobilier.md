# Immobilier et crédit lié

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → Immobilier : choisir/créer l’actif, renseigner achat, estimation, loyers/charges et informations du bien ; associer explicitement sa dette si nécessaire. Consulter rendement et cashflow avec les hypothèses affichées, puis corriger les informations ou valorisations.

## Données et comportement

`property_details` enrichit un actif immobilier et lie un crédit du même propriétaire. Sa valeur courante vient des valorisations de l’actif, le capital restant de la dette liée. Les calculs du centre utilisent les informations déclarées ; un loyer prévisionnel ne devient pas une transaction bancaire.

## Limites et configuration

Pas d’estimation automatique par adresse, de signature de bail ni de gestion d’encaissement locataire. Les charges/fiscalité réelles non saisies ne peuvent être devinées. Ne pas créer une deuxième dette pour le crédit déjà présent.

Sur le web, les détails sont chargés avant ouverture du formulaire : une réponse tardive ne peut pas effacer une saisie en cours. Un échec propose une nouvelle tentative.

## Vérification

Contrats API des centres patrimoniaux et tests d’intégrité des liens actif/dette. Recette : bien avec crédit du même profil, modification des données puis relecture ; un crédit d’un autre profil doit être refusé.

## API et sources

- `GET /v1/real-estate`
- `PUT /v1/assets/{id}/property`

- [backend/internal/api/profondeur.go](../../backend/internal/api/profondeur.go)
- [ios/Opale/Features/Wealth/Centers/RealEstateView.swift](../../ios/Opale/Features/Wealth/Centers/RealEstateView.swift)
