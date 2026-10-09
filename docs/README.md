# Documentation Opale

Référence du produit au **9 octobre 2026**. Le backend de l’iPhone et du web est imposé : **https://opale.vaycode.com**. Les fiches ci-dessous décrivent ce que fait le code, ses limites et comment le vérifier. Les anciens documents de vision restent des archives, pas une preuve de livraison.

## Commencer

Sur iPhone : **Mon parcours** → comptes → revenus → charges fixes → abonnements → biens/crédits → budget/projets → bilan. Chaque étape explique quoi saisir et permet de revenir aux informations existantes. Sur le web, les cinq espaces restent organisés par module ; le parcours guidé complet est propre à iOS.

## Catalogue fonctionnel

Chaque fiche décrit l’utilisation, les données/calculs, les limites/configurations, les vérifications et ses sources/API. Les fonctions propres à iOS ou dépendantes d’un fournisseur sont signalées dans la fiche.

| Fonctionnalité | Documentation |
|---|---|
| Connexion, profils et verrouillage | [Lire la fiche](fonctionnalites/connexion-profils.md) |
| Parcours manuel et situation de départ | [Lire la fiche](fonctionnalites/parcours-guide.md) |
| Patrimoine net, trésorerie et santé financière | [Lire la fiche](fonctionnalites/tableau-de-bord.md) |
| Transactions, recherche et import de relevés | [Lire la fiche](fonctionnalites/transactions-imports.md) |
| Catégories, règles et suggestions de libellés | [Lire la fiche](fonctionnalites/categories-regles.md) |
| Virements, ventilation et espaces partagés | [Lire la fiche](fonctionnalites/virements-partage.md) |
| Calendrier, récurrences et prévision de trésorerie | [Lire la fiche](fonctionnalites/calendrier-cashflow.md) |
| Enveloppes budgétaires et objectifs | [Lire la fiche](fonctionnalites/budget-objectifs.md) |
| Contrats, abonnements et engagements | [Lire la fiche](fonctionnalites/contrats-abonnements.md) |
| Alertes de hausse et rappels de contrats | [Lire la fiche](fonctionnalites/hausses-abonnements.md) |
| Salaires et revenus variables | [Lire la fiche](fonctionnalites/revenus-variables.md) |
| Actifs, dettes, valorisations et devises | [Lire la fiche](fonctionnalites/patrimoine-devises.md) |
| Immobilier et crédit lié | [Lire la fiche](fonctionnalites/immobilier.md) |
| Placements, performance et cotations | [Lire la fiche](fonctionnalites/investissements-cours.md) |
| Objets de valeur et assurance | [Lire la fiche](fonctionnalites/objets.md) |
| Sociétés, participations et compte courant d’associé | [Lire la fiche](fonctionnalites/societes-cca.md) |
| Coffre chiffré et documents | [Lire la fiche](fonctionnalites/coffre-documents.md) |
| Préremplissage depuis une facture ou fiche de paie | [Lire la fiche](fonctionnalites/prefill-factures-paie.md) |
| Contacts, bénéficiaires et accès d’urgence | [Lire la fiche](fonctionnalites/transmission-urgence.md) |
| Projections, indépendance financière et chronologie | [Lire la fiche](fonctionnalites/projections-scenarios.md) |
| Décisions, crédit, capacité et fiscalité | [Lire la fiche](fonctionnalites/decisions-credit-fiscalite.md) |
| Assistant, guide intégré et fournisseurs IA | [Lire la fiche](fonctionnalites/assistant-ia.md) |
| Explorer les placements par risque, pays et support | [Lire la fiche](fonctionnalites/explorateur-placements.md) |
| Alertes, notifications locales et APNs | [Lire la fiche](fonctionnalites/alertes-notifications.md) |
| Allocation, snapshots et bilans mensuel/annuel | [Lire la fiche](fonctionnalites/pilotage-bilans.md) |
| Personnalisation, accessibilité et widget iPhone | [Lire la fiche](fonctionnalites/reglages-widget.md) |
| Export, remise à zéro et suppression du profil | [Lire la fiche](fonctionnalites/export-effacement.md) |
| Connexion bancaire et association des comptes | [Lire la fiche](fonctionnalites/synchronisation-bancaire.md) |
| Backend Vaycode imposé et migration des anciennes adresses | [Lire la fiche](fonctionnalites/serveur-vaycode.md) |
| Architecture, démarrage, migrations et exploitation | [Lire la fiche](fonctionnalites/architecture-exploitation.md) |

## Contrats et guides techniques

- [Référence de toutes les routes HTTP](API.md), [modèle de données](DATA-MODEL.md), [conventions financières](CONVENTIONS-FINANCIERES.md).
- [Confidentialité et localisation des données](CONFIDENTIALITE.md), [contrat IA et consentement](CONFIDENTIALITE-IA.md).
- [Configuration Coolify, PC Windows et cloud](COOLIFY.md), [déploiement Vaycode](DEPLOIEMENT.md), [secrets, sauvegardes, restauration et supervision](EXPLOITATION.md).
- [Recette et commandes](RECETTE.md), [livraison iOS et preuves datées](DELIVERY-IOS.md), [livraison web et preuves datées](DELIVERY-WEB.md), [livraison backend](DELIVERY-BACKEND.md), [matrice historique par exigence](LIVRAISON.md).
- [Cahier des charges initial](CAHIER-DES-CHARGES.md), [vision/conception initiale](CONCEPTION.md) : historique des intentions, à confronter aux fiches actuelles.

## Maintenir la documentation

[AGENTS.md](../AGENTS.md) impose une mise à jour documentaire pour **chaque ajout, modification ou suppression de fonctionnalité**, dans le même changement que le code. `catalogue.json` associe explicitement les routes et les écrans à leurs fiches ; un fichier source peut servir plusieurs modules.

```bash
python3 scripts/check_docs.py --write  # après mise à jour du catalogue
python3 scripts/check_docs.py          # contrôle exécuté aussi en CI
```

Le contrôle vérifie couverture, existence des sources/fiches, rubriques et liens locaux, ainsi que l’actualité de la référence API. Une nouvelle route ou un nouvel écran non associé fait échouer le contrôle. Il ne prouve pas que le texte reflète toute modification interne : cette revue reste obligatoire, même sans nouveau fichier.

Une fiche doit distinguer calcul et hypothèse, données enregistrées et prévision, tests synthétiques et fournisseurs réellement éprouvés. Ne jamais y mettre de secret ou de donnée personnelle. Les essais qui écrivent utilisent une base jetable, pas `opale.vaycode.com`.
