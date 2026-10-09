# Explorer les placements par risque, pays et support

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Sur iPhone : Réglages/Patrimoine → Explorer les investissements. Choisir risque faible/moyen/élevé, résidence fiscale, zone d’exposition, famille (épargne, obligations, ETF, actions, crypto, immobilier), enveloppe et horizon. Rechercher un nom/ISIN et ajouter une fiche à « Ma liste à étudier ».

## Données et comportement

Le catalogue embarqué contient 19 pistes documentées le 9 octobre 2026, avec risques, frais/liquidité, accès à vérifier et sources officielles. Pays d’exposition, résidence fiscale et domicile du fonds sont des notions distinctes. Une enveloppe (PEA/CTO/assurance-vie) n’est pas un niveau de risque.

Les favoris sont stockés dans un cache local protégé lié au serveur/profil, sans ordre de marché. Le filtre d’horizon peut être désactivé explicitement. L’app avertit après 90 jours sans révision des sources du catalogue.

## Limites et configuration

iOS uniquement ; aucun classement de performance en direct, cours temps réel ni recommandation personnalisée. Les repères Opale diffèrent du score réglementaire du DIC. Hors France, la fiscalité et l’éligibilité détaillée ne sont pas modélisées. Vérifier les documents actuels du produit avant d’investir.

## Vérification

`FinancialToolsTests.testInvestmentFiltersDistinguishCountryResidenceRiskAndHorizon` contrôle les filtres. Les sources se trouvent dans `InvestmentIdeas.swift` et les fiches détaillées. Une modification du catalogue exige de revoir date, liens, risques, disponibilité et documentation ; les tests ne garantissent pas une éligibilité fiscale réelle.

## API et sources

Pas de route HTTP propre à ce module ; les opérations passent par les modules associés ou restent locales.

- [ios/Opale/Core/InvestmentIdeas.swift](../../ios/Opale/Core/InvestmentIdeas.swift)
- [ios/Opale/Features/Investing/InvestmentExplorerView.swift](../../ios/Opale/Features/Investing/InvestmentExplorerView.swift)
- [ios/Opale/Features/Investing/InvestmentIdeaDetail.swift](../../ios/Opale/Features/Investing/InvestmentIdeaDetail.swift)
