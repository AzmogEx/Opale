# Décisions, crédit, capacité et fiscalité

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Assistant → Décision/Comparer : renseigner achat/location, comptant/crédit ou remboursement/investissement, puis lire les deux alternatives aux mêmes horizons. Pilote automatique propose Crédit & capacité et Fiscalité & PER ; saisir les hypothèses et consulter résultats, limites et source du barème.

## Données et comportement

Le comparateur calcule immédiatement puis à 5/10 ans les patrimoines nominaux/réels, flux, dettes, risques et scénarios prudent/normal/ambitieux. L’orientation est conditionnelle : une alternative non finançable ne devient pas recommandée. Le moteur évalue les chiffres ; l’IA peut expliquer à partir de faits autorisés.

Le simulateur de crédit produit mensualité, intérêts/coût et amortissement ; la capacité est une estimation paramétrée. Le module fiscal code le barème français 2026 sur revenus 2025 et une estimation PER avec plafond explicite, source affichée et refus des années non supportées.

## Limites et configuration

Pas d’offre de prêt, déclaration fiscale ou prise en compte universelle des particularités du foyer. Le module fiscal ne modélise pas les autres pays ni tous les crédits/réductions d’impôt. Relire la source actuelle pour un usage ultérieur ; les hypothèses ne sont pas des faits bancaires.

## Vérification

Tests `engine/delivery_decisions_test.go`, `loan_test.go`, `internal/tax`, API et intégration `decisions-label.spec.ts`/`modules.spec.ts` : trois décisions, dates 0/60/120 mois, scénarios, risques et source fiscale. Aucune validation d’un dossier de crédit réel n’est revendiquée.

## API et sources

- `GET /v1/tax/deadlines`
- `GET /v1/tax/estimate`
- `POST /v1/decision`
- `POST /v1/decisions/compare`
- `POST /v1/loan/capacity`
- `POST /v1/loan/simulate`

- [backend/internal/api/brain.go](../../backend/internal/api/brain.go)
- [backend/internal/api/delivery_decisions.go](../../backend/internal/api/delivery_decisions.go)
- [backend/internal/api/delivery_tax.go](../../backend/internal/api/delivery_tax.go)
- [backend/internal/api/pilote.go](../../backend/internal/api/pilote.go)
- [backend/internal/engine/delivery_decisions.go](../../backend/internal/engine/delivery_decisions.go)
- [backend/internal/engine/loan.go](../../backend/internal/engine/loan.go)
- [backend/internal/tax/tax.go](../../backend/internal/tax/tax.go)
- [ios/Opale/Features/Assistant/DecisionComparisonView.swift](../../ios/Opale/Features/Assistant/DecisionComparisonView.swift)
- [ios/Opale/Features/Assistant/DecisionSheet.swift](../../ios/Opale/Features/Assistant/DecisionSheet.swift)
- [ios/Opale/Features/Pilot/PilotToolsView.swift](../../ios/Opale/Features/Pilot/PilotToolsView.swift)
- [web/src/lib/components/Calculator.svelte](../../web/src/lib/components/Calculator.svelte)
- [web/src/lib/components/Decisions.svelte](../../web/src/lib/components/Decisions.svelte)
