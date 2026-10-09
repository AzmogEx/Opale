# Contacts, bénéficiaires et accès d’urgence

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → Transmission : gérer contacts, bénéficiaires/quotes-parts et contrats associés. Dans les accès d’urgence, choisir un autre profil destinataire et une sélection explicite d’actifs/documents, créer le droit, puis l’activer ou le révoquer. Le destinataire ouvre seulement ce qui lui a été accordé.

## Données et comportement

`contacts` n’accorde aucun accès par sa seule présence. Les quotes-parts sont exactes en points de base : 33,33 % = 3333. `emergency_grants` conserve propriétaire, destinataire, sélection, activation, expiration et révocation ; l’accès au document passe par une route contrôlée dédiée.

Les contrôles sont réévalués au serveur. Retirer/expirer un droit bloque les lectures suivantes, même si un identifiant est encore connu. L’export privé reste celui du propriétaire, pas un export complet offert au bénéficiaire.

## Limites et configuration

Ce suivi ne remplace pas un testament, une clause bénéficiaire contractuelle ni un acte notarié. Aucun accès automatique après décès n’est déduit. Les copies déjà téléchargées ne peuvent pas être rappelées après révocation.

## Vérification

Intégration `advanced.spec.ts` et tests `delivery_emergency` : un actif/un document sélectionnés, activation, lecture par le destinataire, révocation puis refus (404). Tester un troisième profil et un document hors sélection : aucun accès.

## API et sources

- `DELETE /v1/beneficiaries/{id}`
- `DELETE /v1/contacts/{id}`
- `DELETE /v1/emergency-grants/{id}`
- `GET /v1/beneficiaries`
- `GET /v1/contacts`
- `GET /v1/emergency-grants`
- `GET /v1/emergency/{id}`
- `GET /v1/emergency/{id}/documents/{documentID}`
- `GET /v1/transmission`
- `PATCH /v1/contacts/{id}`
- `PATCH /v1/emergency-grants/{id}`
- `POST /v1/contacts`
- `POST /v1/emergency-grants`
- `PUT /v1/beneficiaries`

- [backend/internal/api/delivery_emergency.go](../../backend/internal/api/delivery_emergency.go)
- [backend/internal/api/delivery_metadata.go](../../backend/internal/api/delivery_metadata.go)
- [backend/internal/api/profondeur.go](../../backend/internal/api/profondeur.go)
- [ios/Opale/Features/Wealth/Centers/TransmissionAccessView.swift](../../ios/Opale/Features/Wealth/Centers/TransmissionAccessView.swift)
- [ios/Opale/Features/Wealth/Centers/TransmissionView.swift](../../ios/Opale/Features/Wealth/Centers/TransmissionView.swift)
- [web/src/lib/components/Transmission.svelte](../../web/src/lib/components/Transmission.svelte)
