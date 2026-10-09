# Connexion bancaire et association des comptes

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Flux → Banque : vérifier la disponibilité, choisir l’institution, donner son consentement chez le fournisseur/banque, puis associer chaque compte découvert à un actif Opale distinct. Lancer une synchronisation, consulter état/expiration, renouveler ou déconnecter. CSV/OFX et saisie manuelle restent disponibles.

## Données et comportement

L’intégration GoCardless conserve consentement, comptes fournisseur, mapping, observations provisoires et état de synchronisation. Le mapping est par compte, pas un seul actif pour toute une banque. Les imports sont dédupliqués durablement ; un changement fournisseur incompatible est refusé plutôt que recréé silencieusement.

Les tâches utilisent un bail/reprise pour éviter les synchronisations concurrentes. Les opérations provisoires sont distinguées des mouvements comptabilisés. La déconnexion retire le lien fournisseur sans supprimer les écritures déjà suivies ni faire disparaître implicitement les associations utiles.

## Limites et configuration

Variables `OPALE_GC_SECRET_ID`, `OPALE_GC_SECRET_KEY` et éventuel `OPALE_GC_BASE_URL` côté serveur. Sans configuration, état indisponible honnête. L’app ne reçoit pas les identifiants de connexion bancaire ; l’accès dépend du fournisseur, du consentement et de la banque. Aucune banque personnelle réelle n’a été validée par la recette synthétique.

## Vérification

Tests `internal/bank/delivery_gocardless_test.go`, store/API `delivery_bank` couvrent contrats, mapping, erreurs et isolation. Recette externe : deux comptes distincts, resync sans doublon, provisoire→comptabilisé, renouvellement puis révocation ; contrôler les soldes et les périodes couvertes.

## API et sources

- `DELETE /v1/bank/links/{id}`
- `GET /v1/bank/accounts`
- `GET /v1/bank/institutions`
- `GET /v1/bank/status`
- `POST /v1/bank/connect`
- `POST /v1/bank/links/{id}/renew`
- `POST /v1/bank/sync`
- `PUT /v1/bank/accounts/{id}`

- [backend/internal/api/confort.go](../../backend/internal/api/confort.go)
- [backend/internal/api/delivery_bank.go](../../backend/internal/api/delivery_bank.go)
- [ios/Opale/Features/Flows/BankAccountsView.swift](../../ios/Opale/Features/Flows/BankAccountsView.swift)
- [ios/Opale/Features/Flows/BankSheet.swift](../../ios/Opale/Features/Flows/BankSheet.swift)
- [web/src/lib/components/Bank.svelte](../../web/src/lib/components/Bank.svelte)
