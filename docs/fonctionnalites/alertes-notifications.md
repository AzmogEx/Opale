# Alertes, notifications locales et APNs

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Accueil → Alertes pour consulter les points à vérifier. Pilote automatique → Alertes personnalisées pour ajouter/corriger un seuil et l’activer/désactiver. Réglages → Notifications sur iPhone : autoriser le système et activer pour le profil. Toucher une notification ouvre la destination autorisée après déverrouillage.

## Données et comportement

Le moteur produit les alertes de santé/budget/cash et les contrats leurs rappels/hausses. `custom_alerts` stocke les seuils. Les tokens APNs sont liés au profil et à l’environnement ; les jobs ont bail et déduplication persistants. Les payloads sont minimisés, sans montants/libellés sensibles.

Le rafraîchissement iOS en arrière-plan utilise exactement le même backend Vaycode et un jeton lié à cette URL. Une destination inconnue ou destinée à un autre profil est ignorée. Déconnexion et désactivation retirent les éléments locaux/inscriptions selon leur flux de révocation.

## Limites et configuration

APNs exige clé .p8, Key ID, Team ID, topic, environnement cohérent et entitlement Push Notifications signé ; variables dans `.env.example`/Coolify. Le système décide quand exécuter un rafraîchissement et peut refuser l’autorisation. Une alerte visible dans l’app ne prouve pas la réception d’un push. Pas de web push livré.

## Vérification

Tests jobs/store/API des jetons, déduplication et destinations ; `SessionIsolationTests` refuse un autre profil et une destination arbitraire. La livraison APNs sur iPhone doit être testée avec la configuration réelle ; ne pas la déduire du build ou de l’installation réussie.

## API et sources

- `DELETE /v1/alerts/custom/{id}`
- `DELETE /v1/push/register`
- `GET /v1/alerts`
- `GET /v1/alerts/custom`
- `PATCH /v1/alerts/custom/{id}`
- `POST /v1/alerts/custom`
- `POST /v1/push/register`

- [backend/internal/api/crud.go](../../backend/internal/api/crud.go)
- [backend/internal/api/pilotage.go](../../backend/internal/api/pilotage.go)
- [backend/internal/api/pilote.go](../../backend/internal/api/pilote.go)
- [ios/Opale/Features/Home/AlertsInboxView.swift](../../ios/Opale/Features/Home/AlertsInboxView.swift)
- [ios/Opale/Features/Settings/PushSettingsView.swift](../../ios/Opale/Features/Settings/PushSettingsView.swift)
