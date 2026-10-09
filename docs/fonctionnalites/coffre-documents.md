# Coffre chiffré et documents

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → Coffre : importer un document, saisir son titre/type et ses associations proposées, puis télécharger, corriger les métadonnées ou supprimer. Les factures/actes/pièces déposés ici sont conservés ; cela se distingue du préremplissage local d’une facture, qui ne dépose pas le fichier.

## Données et comportement

Le backend chiffre le contenu en AES-256-GCM avec `OPALE_VAULT_KEY`. Les métadonnées autorisées sont séparées du contenu. L’accès privé, l’export et les droits d’urgence sélectionnés sont vérifiés au serveur et journalisés. Une clé incorrecte ou une altération empêche le déchiffrement.

Téléchargement et partage produisent une copie lisible sur l’appareil ; iOS protège puis nettoie ses fichiers temporaires après partage. Le coffre n’envoie pas ses documents aux fournisseurs IA.

## Limites et configuration

Clé serveur obligatoire : si absente, le coffre affiche son indisponibilité. Sauvegarder la clé séparément de PostgreSQL ; dump seul insuffisant pour récupérer les fichiers. Les copies exportées/téléchargées ne sont plus protégées par le chiffrement du coffre.

## Vérification

Tests `internal/vault`, export/API et intégration `modules.spec.ts` : contenu relu identique. Restauration isolée éprouvée avec SHA-256 identique dans [EXPLOITATION.md](../EXPLOITATION.md). Ne pas tester une rotation destructrice sur les données personnelles.

## API et sources

- `DELETE /v1/documents/{id}`
- `GET /v1/documents`
- `GET /v1/documents/{id}/content`
- `PATCH /v1/documents/{id}`
- `POST /v1/documents`

- [backend/internal/api/delivery_metadata.go](../../backend/internal/api/delivery_metadata.go)
- [backend/internal/api/profondeur.go](../../backend/internal/api/profondeur.go)
- [backend/internal/vault/vault.go](../../backend/internal/vault/vault.go)
- [ios/Opale/Features/Wealth/Centers/VaultView.swift](../../ios/Opale/Features/Wealth/Centers/VaultView.swift)
- [web/src/lib/components/Vault.svelte](../../web/src/lib/components/Vault.svelte)
