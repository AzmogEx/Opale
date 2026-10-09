# Virements, ventilation et espaces partagés

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Dans Flux, saisir un virement entre deux comptes avec frais éventuels. Ventiler une opération en plusieurs catégories en gardant exactement sa somme. Ouvrir les espaces partagés, créer/choisir un espace, ajouter/retirer un membre et partager explicitement une opération ; retirer son partage si nécessaire.

## Données et comportement

Un virement crée ses deux côtés et ses frais dans une transaction SQL ; les mouvements internes ne gonflent pas les revenus/dépenses. Sa suppression est groupée. Une ventilation conserve l’identité d’import et doit équilibrer le montant original.

`spaces`/`space_members` n’accordent que la vue prévue sur les opérations explicitement partagées. Appartenir à un espace ne donne pas accès au patrimoine privé, au coffre ni à l’ensemble du profil. Le serveur contrôle les droits et détache les opérations du membre retiré.

## Limites et configuration

Ne pas éditer un côté isolé d’un virement : supprimer/recréer l’ensemble. Les comptes et devises doivent satisfaire les contrôles du serveur ; ce module ne réalise pas de transfert auprès d’une banque. Le partage n’est pas une fusion des profils.

## Vérification

`web/tests/integration/modules.spec.ts` vérifie un virement de 125,55 EUR avec 1,20 EUR de frais : trois écritures, somme −120 unités mineures. Tests Go d’intégrité/partage : ventilation équilibrée, droits entre profils et révocation.

## API et sources

- `DELETE /v1/spaces/{id}/members/{profileID}`
- `DELETE /v1/transfers/{id}`
- `GET /v1/spaces`
- `GET /v1/spaces/{id}`
- `POST /v1/spaces`
- `POST /v1/spaces/{id}/members`
- `POST /v1/transactions/{id}/split`
- `POST /v1/transfers`
- `PUT /v1/transactions/{id}/space`

- [backend/internal/api/movements.go](../../backend/internal/api/movements.go)
- [backend/internal/api/partage.go](../../backend/internal/api/partage.go)
- [backend/internal/api/transactions.go](../../backend/internal/api/transactions.go)
- [ios/Opale/Features/Flows/SharedSpaceView.swift](../../ios/Opale/Features/Flows/SharedSpaceView.swift)
- [ios/Opale/Features/Flows/SplitSheet.swift](../../ios/Opale/Features/Flows/SplitSheet.swift)
- [ios/Opale/Features/Flows/TransferSheet.swift](../../ios/Opale/Features/Flows/TransferSheet.swift)
- [web/src/lib/components/Sharing.svelte](../../web/src/lib/components/Sharing.svelte)
