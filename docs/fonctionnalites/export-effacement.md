# Export, remise à zéro et suppression du profil

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Réglages → Données : télécharger l’export ZIP, remettre à zéro les données ou supprimer le profil. Les actions destructrices demandent la confirmation du nom. Conserver l’export dans un endroit protégé ; il contient des données lisibles et les documents autorisés déchiffrés.

## Données et comportement

L’export utilise un instantané SQL cohérent, une liste explicite des tables privées et un manifeste. Contrats, tarifs, revenus, progression du parcours et documents font partie des données exportables ; PIN, sessions, tokens secrets et tables techniques de livraison push sont exclus.

La remise à zéro conserve le profil mais efface ses données financières et états associés suivant les relations serveur. La suppression retire aussi le profil. Les clients nettoient état/caches et empêchent une réponse tardive de rouvrir un contenu de l’ancienne session. Un téléchargement en cours est également lié à son profil d’origine.

## Limites et configuration

Export utilisateur et sauvegarde PostgreSQL ne sont pas interchangeables : la procédure de restauration serveur utilise dump + clé du coffre. Les favoris/scénarios locaux et conversation en mémoire ne sont pas tous dans l’export API. La remise à zéro/suppression est irréversible sans sauvegarde exploitable.

## Vérification

Tests API d’export/reset/parcours et unitaires web de téléchargement tardif. Intégration quotidienne télécharge un ZIP réel ; les profils synthétiques des tests sont supprimés en cleanup. Voir [conventions d’export](../CONVENTIONS-FINANCIERES.md#export) et [restauration](../EXPLOITATION.md).

## API et sources

- `DELETE /v1/me`
- `DELETE /v1/me/data`
- `GET /v1/export`

- [backend/internal/api/export.go](../../backend/internal/api/export.go)
