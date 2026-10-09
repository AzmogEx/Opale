# Parcours manuel et situation de départ

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Sur iPhone, « Mon parcours » propose sept étapes : comptes → revenus → charges fixes → abonnements → biens/crédits → budget/projets → bilan. Ouvrir une étape, ajouter ou corriger les informations, puis la marquer vérifiée. « Passer » laisse la vérification à faire ; revenir plus tard ne supprime pas les données.

Le formulaire initial des profils personnels vides propose aussi six pages : revenu, compte, charges, abonnements, budget/objectif et récapitulatif. « Plus tard » permet une reprise depuis Réglages → Ma situation de départ. Un profil démo n’impose pas cette saisie.

## Données et comportement

`profile_journey` (migration 0025) mémorise étape, listes vérifiées/passées, budget variable et révision. Le brouillon local de budget est protégé et lié au profil, serveur et version. Les ajouts du parcours sont enregistrés immédiatement dans leurs modules ; vérifier une étape ne recrée pas ces éléments.

Le formulaire initial utilise `profile_onboarding` et valide son ensemble atomiquement à la fin. Ses écritures créent des hypothèses et règles de calendrier, jamais un salaire ou prélèvement fictivement reçu. Le parcours peut reprendre son budget déclaré sans déclarer les étapes déjà vérifiées.

Le mois type annualise les fréquences avant l’arrondi final et sépare les devises. Le reste disponible exige revenus, charges, abonnements et budget vérifiés ; une étape passée ou une donnée invalide donne « À compléter », pas zéro.

## Limites et configuration

Interfaces guidées propres à iOS ; le web conserve les modules et formulaires séparés. Une mise à jour concurrente produit un conflit de révision, à recharger explicitement. Les prévisions ne prouvent pas un versement. En cache, la validation attend le réseau.

## Vérification

`FinancialJourneyTests`, `FinancialSetupTests`, `FinancialSetupUITests` et les tests Go `journey_test.go`/`onboarding_test.go` vérifient reprise, conflits, isolation et absence de transactions inventées. Recette synthétique : 3 200 − 850 − 10,99 − 500 = 1 839,01 EUR, sept étapes vérifiées et zéro transaction réalisée ; preuves dans [livraison iOS](../DELIVERY-IOS.md).

## API et sources

- `GET /v1/journey`
- `GET /v1/onboarding`
- `POST /v1/onboarding/complete`
- `POST /v1/onboarding/skip`
- `PUT /v1/journey`
- `PUT /v1/onboarding`

- [backend/internal/api/journey.go](../../backend/internal/api/journey.go)
- [backend/internal/api/onboarding.go](../../backend/internal/api/onboarding.go)
- [backend/internal/migrations/0025_guided_journey.up.sql](../../backend/internal/migrations/0025_guided_journey.up.sql)
- [backend/internal/store/journey.go](../../backend/internal/store/journey.go)
- [backend/internal/store/onboarding.go](../../backend/internal/store/onboarding.go)
- [ios/Opale/Core/FinancialJourney.swift](../../ios/Opale/Core/FinancialJourney.swift)
- [ios/Opale/Core/FinancialSetup.swift](../../ios/Opale/Core/FinancialSetup.swift)
- [ios/Opale/Features/Auth/FinancialSetupForm.swift](../../ios/Opale/Features/Auth/FinancialSetupForm.swift)
- [ios/Opale/Features/Auth/FinancialSetupGate.swift](../../ios/Opale/Features/Auth/FinancialSetupGate.swift)
- [ios/Opale/Features/Auth/FinancialSetupPages.swift](../../ios/Opale/Features/Auth/FinancialSetupPages.swift)
- [ios/Opale/Features/Auth/OnboardingView.swift](../../ios/Opale/Features/Auth/OnboardingView.swift)
- [ios/Opale/Features/Journey/FinancialJourneyView.swift](../../ios/Opale/Features/Journey/FinancialJourneyView.swift)
