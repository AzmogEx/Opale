# Calendrier, récurrences et prévision de trésorerie

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Flux → Calendrier : créer un revenu ou une dépense ponctuelle/récurrente, choisir compte, montant natif, première date et fréquence. Une occurrence peut être exclue ou rapprochée d’une transaction réellement constatée. Vérifier la prévision de trésorerie et les abonnements détectés avant de confirmer une nouvelle règle.

## Données et comportement

Les séries supportent ponctuel, hebdomadaire, mensuel, trimestriel et annuel. `calendar_occurrences` conserve exceptions et rapprochements. Les détections de récurrence issues des opérations restent des propositions ; `recurring_exclusions` évite leur réapparition indésirable.

Le cashflow combine solde courant, échéances prévues et hypothèses de dépenses. Une transaction rapprochée ne doit pas être comptée une deuxième fois comme échéance. Un contrat/revenu géré possède sa règle liée : le modifier depuis son module conserve l’historique et remplace ses prévisions futures.

## Limites et configuration

Une prévision n’est pas un débit bancaire ni un encaissement réalisé. Le montant d’une détection converti en EUR ne doit pas être repris dans une autre devise native sans ressaisie. Les données manquantes et prévisions obsolètes peuvent rendre la trajectoire incorrecte.

## Vérification

Tests moteur `p4_test.go`, API `delivery_quarterly_test.go`, intégration `advanced.spec.ts`, navigateur `financial-inputs.spec.ts` : récurrence trimestrielle conservée, exclusion relue, devise contrôlée et rapprochement sans doublon.

## API et sources

- `DELETE /v1/calendar/{id}`
- `GET /v1/calendar`
- `GET /v1/cashflow`
- `GET /v1/recurring`
- `GET /v1/subscriptions`
- `PATCH /v1/calendar/{id}`
- `POST /v1/calendar`
- `PUT /v1/calendar/{id}/occurrences`
- `PUT /v1/recurring/exclusion`

- [backend/internal/api/delivery_calendar.go](../../backend/internal/api/delivery_calendar.go)
- [backend/internal/api/pilotage.go](../../backend/internal/api/pilotage.go)
- [ios/Opale/Features/Flows/CalendarView.swift](../../ios/Opale/Features/Flows/CalendarView.swift)
- [ios/Opale/Features/Home/SubscriptionsView.swift](../../ios/Opale/Features/Home/SubscriptionsView.swift)
- [web/src/lib/components/Calendar.svelte](../../web/src/lib/components/Calendar.svelte)
