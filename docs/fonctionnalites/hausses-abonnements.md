# Alertes de hausse et rappels de contrats

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Ouvrir les alertes ou la fiche d’un contrat. Une hausse à vérifier affiche ancien tarif, dernier prélèvement et surcoût annuel conditionnel. Comparer à la facture puis « Confirmer le nouveau tarif » ou « Ignorer ce prélèvement ». Les rappels affichent aussi fin d’essai/engagement et renouvellement.

## Données et comportement

La détection se fonde sur les opérations correspondant au compte/marchand surveillé et au contrat déclaré, pas sur une recherche automatique du tarif commercial sur Internet. Une observation n’actualise pas le contrat sans acceptation. Les observations ignorées sont mémorisées dans `contract_price_dismissals`.

Accepter une hausse historise son tarif et actualise les prochaines prévisions. Le paiement du jour est rapproché ou l’occurrence est exclue s’il est déjà rapproché, afin de ne pas le compter deux fois. Les livraisons push ont une déduplication durable par événement/appareil.

## Limites et configuration

Un paiement exceptionnel peut ressembler à une hausse : vérifier avant de confirmer. L’absence d’alerte ne garantit pas un prix stable. Les alertes dans l’app fonctionnent avec les données disponibles ; réception APNs et rafraîchissement iOS exigent leur configuration et l’autorisation de l’appareil.

## Vérification

Tests API/store de `financial_tools`, hausse acceptée/ignorée, rapprochement et isolation. En recette jetable, enregistrer un contrat, ajouter un prélèvement supérieur puis vérifier confirmation, nouvelle prévision et absence de deuxième débit. Les notifications physiques restent à éprouver avec APNs.

## API et sources

- `GET /v1/contracts/{id}/prices`
- `POST /v1/contracts/{id}/price-observation`

- [backend/internal/api/financial_tools.go](../../backend/internal/api/financial_tools.go)
- [backend/internal/store/contract_push.go](../../backend/internal/store/contract_push.go)
- [backend/internal/store/contracts.go](../../backend/internal/store/contracts.go)
- [ios/Opale/Features/Contracts/ContractDetailView.swift](../../ios/Opale/Features/Contracts/ContractDetailView.swift)
