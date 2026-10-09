# Livraison Web et exploitation — 8 octobre 2026

Travail effectué directement dans `web/`, sans remplacement de l’identité visuelle ni des cinq onglets. Les états ci-dessous distinguent implémentation et validation : une route affichée ne constitue pas une preuve de fonctionnement d’un fournisseur externe.

## Parcours livrés

| Domaine | Réalisation |
|---|---|
| Profils et sécurité | Création, sélection, PIN, rename/changement PIN/confidentialité, export, reset/suppression confirmés ; token dans sessionStorage uniquement ; verrouillage à froid, onglet masqué et inactivité de 5 minutes ; retour login sur 401 ; réponse d’ancien profil ignorée |
| Accueil | Total moteur EUR, alerte total incomplet, historique, évolution, trésorerie, santé, risques, alertes personnalisées CRUD et jalons avec célébration discrète, fonds d’urgence autoritatif |
| Patrimoine | Actifs/dettes CRUD/archive, création+valorisation initiale atomique et identifiant idempotent, historique et correction/suppression de valorisations, unités EUR/JPY/KWD et autres devises |
| Flux | Transactions CRUD, recherche, filtres, pagination 50+1, notes, catégories, règles, nettoyage du libellé via homelab privé avec aperçu et acceptation explicite, splits avec somme exacte, virement deux côtés+frais atomique et suppression groupée, remboursement capital lié à une dette |
| Import | CSV/OFX, UTF-8/Windows-1252, aperçu, compte explicite, récapitulatif réel serveur et doublons, limites 5 Mio / 10 000 opérations |
| Budgets/calendrier | Enveloppes, CRUD règles récurrentes dont trimestrielles, exclusion/confirmation détectée avec montant natif ressaisi, occurrences prévues/exclues/réalisées avec transaction réelle, prévision de trésorerie et analyse des dépenses |
| Objectifs/projection | Objectifs CRUD/rythme affecté, FIRE/inflation, chronologie, comparaison scénarios, achat/location, comptant/crédit, remboursement/investissement avec impacts simultanés à 0/5/10 ans, scénarios prudent/normal/ambitieux, risques et orientation conditionnelle, crédit et capacité, fiscalité 2026/PER avec plafond et source officielle |
| Modules patrimoniaux | Immobilier et crédit, placements/flux/performance conditionnée à couverture, allocation cible, objets, entreprise et créance CCA liée, devises privées, cours CoinGecko avec source/date/erreur |
| Coffre/transmission | Upload/téléchargement, métadonnées et suppression, contacts CRUD, bénéficiaires/quotes-parts, accès d’urgence limité aux actifs/docs choisis, activation/révocation/expiration explicites |
| Partage | Espaces/membres, partage ou retrait explicite d’opérations, vues communes, contrôle final au serveur |
| Assistant/P8 | Conversation en mémoire (20 messages), annulation, faits/état/sources, opt-in cloud par demande autorisée, Wrapped année, bilan mensuel, jumeau, jalons |
| Confort | Thèmes clair/sombre/système, accents, mode discret qui retire le contenu du DOM, navigation mobile à 5 onglets, labels et focus, réduction du mouvement, sélection clavier du graphique |
| Exploitation | Compose trois services et profil Caddy HTTPS configurés, proxy nginx/sans cache API / metrics interne, env exemple, scripts dump/restauration isolée, guide exploitation et README actualisés, CI Go / Web / Compose / iOS |

## Vérifications exécutées

- `npm run check` : zéro erreur, zéro warning lors de la dernière passe validée.
- `npm test` : 13 tests passés : dates civiles France été/hiver/bissextile, année FIRE, décimales exactes et dépassements, devises à 0/3 décimales, pagination/encodage, 401/403/réseau, réponse tardive d’ancien profil, cloud explicite, rejet de montants JSON non représentables, formatage exact à la limite entière JavaScript et téléchargement tardif après changement de profil.
- `npm run build` : build production SvelteKit statique réussi. Dernière passe après intégration des contrats réussie.
- `npm audit` : zéro vulnérabilité après mises à jour compatibles et override cookie 0.7.2.
- Navigateur Chrome : 6/6 tests initiaux avec API simulée, confidentialité du DOM/texte/accessibilité, verrouillage au reload, expiration, inactivité immédiate après login, navigation 390×844 sans débordement, déconnexion immédiate malgré 503 et jalon fonds d’urgence avec réduction du mouvement. Capture mobile inspectée visuellement ; correction d’un contexte de backdrop-filter qui bloquait la toolbar.
- API Go/PostgreSQL réels jetables : parcours quotidien passé (profil UI, compte + 1 000,25 EUR atomique, CSV de 55 lignes, pagination/recherche/édition, export ZIP, reload PIN, révocation serveur 401). Aucun test personnel ni fournisseur simulé dans l’application.
- **3/3 parcours intégrés initiaux réels passés** (`daily`, `modules`, `advanced`), complétés par le quatrième parcours ciblé ci-dessous :
  - Virement 125,55 EUR et frais 1,20 : 3 lignes liées et somme −120 unités mineures ; objectif créé puis cible corrigée à 6 000 EUR et relue ; revenu synthétique fourni pour la capacité d’épargne observée.
  - Alerte 500 EUR créée puis relue ; calendrier −12,50 EUR créé puis occurrence exclue et relue ; comparaison comptant/crédit avec résultat patrimonial nominal/réel affiché.
  - Placement 1 000→1 500 EUR : versement 500 saisi via UI puis historique confirmé ; gain moteur exactement 0. Société CCA 500 EUR enregistrée via UI, créance dédiée créée et valorisation relue.
  - Contact/bénéficiaire contrat saisis ; droit d’urgence créé inactif, activé, puis lecture par destinataire limitée à 1 actif/1 document ; révocation via UI puis 404 côté destinataire.
  - Fiscalité 50 000 EUR calculée avec source officielle affichée ; document uploadé puis téléchargement égal à l’original ; ouverture des 10 sections patrimoine sans erreur.
- Ajouts finaux ciblés : 2/2 tests navigateur supplémentaires passés. Le libellé reste inchangé avant acceptation, le refus ne sauvegarde rien, l’acceptation ne transmet que `label`, et l’indisponibilité n’offre aucune acceptation. La détection trimestrielle conserve sa fréquence ; son montant EUR n’est jamais repris comme un montant JPY, et une quote-part de 33,33 % conserve exactement 3333 points de base.
- Dernière recette réelle ciblée `decisions-label.spec.ts` passée : les trois types de décision affichent chacun les dates 0/60/120 mois, les trois scénarios, les risques et l’orientation conditionnelle. Contrôle à 390 px sans débordement et capture inspectée. Homelab réellement absent : réponse indisponible affichée, aucun bouton d’acceptation, libellé relu inchangé. Profil de recette supprimé. Les contrôles et le build ont été repassés après les corrections.
- La recette a reproduit un défaut de formulaire : une liste d’options chargée tardivement réinitialisait l’état d’occurrence saisi. Le formulaire conserve maintenant les valeurs en cours lors d’un changement d’options ; la mutation exclue est vérifiée. Corrections complémentaires : confirmation du nom pour reset/delete, déconnexion immédiate même hors réseau, mois du bilan précédent correct en janvier.
- Capture 390×844 et page Décisions 1 280 px inspectées visuellement. Correction du débordement des tableaux de décision et des noms de profil longs sur mobile. Correction du survol de bouton primaire qui perdait son contraste ; libellés des résultats traduits et chiffres rendus sans approximation silencieuse.
- Sauvegarde/restauration **réelle** PostgreSQL 17 native : nouveau clone, sessions révoquées, API clone 58089, reconnexion PIN, relations actif/transaction/document préservées, objectif retrouvé, patrimoine attendu 97 500 unités mineures EUR, téléchargement déchiffré du document avec SHA-256 identique. Test de refus d’une base cible déjà existante passé. API clone arrêtée et bases clones supprimées après vérification. Détails dans `docs/EXPLOITATION.md`.

## Limites de validation et décisions explicites

- `docker compose config --quiet` et `bash -n` des scripts passent. Docker Desktop est devenu indisponible après saturation disque durant la recette. Build/image/Compose complet et certificatHTTPS non validés localement ; job CI ajouté mais pas exécuté sur GitHub pendant cette session. Les scripts ont été éprouvés en mode PostgreSQL natif isolé.
- Banque GoCardless, homelab Ollama, cloud Anthropic et APNs ne disposent pas ici de service/compte/appareil/secret de recette réel. Contrats et UI sont branchés, état indisponible visible ; leur bon fonctionnement réel reste à valider avec les accès autorisés. La recette backend a vérifié les cours publics BCE et CoinGecko réellement.
- La suggestion de libellé n’envoie que le libellé actuel affiché au homelab privé, jamais de montant, note, catégorie ni libellé brut importé. L’acceptation enregistre uniquement le libellé. Le fournisseur homelab réel reste à éprouver ; les réponses disponibles ont été contrôlées avec interception API en test et côté serveur avec un fournisseur HTTP simulé.
- Le web exige un serveur pour le déverrouillage et les écritures, et ne persiste pas de données financières pour un mode hors ligne. Aucun service worker ni file d’écritures cachée.
- Les écritures liées à un virement ou à un remboursement de principal se corrigent en supprimant puis recréant l’ensemble concerné ; l’édition d’un côté seul est bloquée.
- Les restitutions avancées utilisent des cartes/listes/détails repliables accessibles. Ce n’est pas une refonte graphique ni une certification WCAG/lecteur d’écran complète ; les données longues restent consultables et les listes de transactions sont paginées.
- Ne pas lancer les tests d’intégration sur une instance personnelle. Les profils synthétiques propres au test sont supprimés avec confirmation du nom ; les échecs de cleanup sont à surveiller.

Les détails des scripts/variables et procédures externes sont centralisés dans `docs/EXPLOITATION.md` ; la matrice globale par exigence est dans `docs/LIVRAISON.md`. Aucun fournisseur manquant n’a été remplacé par de fausses données de production.

## Backend Vaycode, formulaires et documentation — 9 octobre 2026

Le client web utilise désormais une adresse constante `https://opale.vaycode.com`. Le panneau de sélection du serveur est supprimé. Une ancienne préférence étrangère invalide sa session d’onglet ; une session nouvelle est explicitement liée à Vaycode. Les anciennes sessions sans liaison sont conservées seulement lorsque leur ancienne origine effective était Vaycode. La politique de verrouillage reste inchangée.

Le web de recette n’a pas de mode de build permettant de changer de backend. `tests/integration/fixtures.ts` intercepte les requêtes dans le seul navigateur automatisé et les relaie vers une API jetable en boucle locale, avec refus d’une origine de recette distante.

Cette recette a révélé un défaut préexistant : sélectionner une société puis saisir son CCA avant l’arrivée des détails pouvait faire enregistrer zéro. `AssetDetails.svelte` attend maintenant le chargement initial avant de monter le formulaire, ignore les réponses obsolètes et propose de réessayer après erreur. La protection s’applique aussi aux détails immobilier/objet/cotation. Le test navigateur maintient volontairement une réponse en attente et vérifie un CCA exact à 500,25 EUR. Sa première assertion de présentation attendait une virgule alors que l’éditeur normalise avec un point ; l’attente a été corrigée sans changer le calcul.

Validations finales réelles, données exclusivement synthétiques :

- `npm run check` : zéro erreur, zéro warning ; `npm test` : **13 tests réussis** ; `npm run build` production réussi.
- `npm run test:e2e` : **10/10 tests réussis**, préférence étrangère, session, absence d’input serveur, chargement lent du CCA, confidentialité, navigation mobile, verrouillage, catégories et devises.
- `npm run test:e2e:integration` : **4/4 parcours réussis** après correction, contre API Go/PostgreSQL natifs jetables via Vite. Création/édition/import/export, calendrier/décisions, performance hors apports, CCA, droits d’urgence/révocation, virements/objectifs/coffre et fiscalité. Les profils créés sont supprimés en fin de test.
- `python3 scripts/check_docs.py` : **30 fiches, 158 routes et 90 écrans/composants**, sources et liens locaux valides. Les fiches et les preuves de livraison sont centralisées dans `docs/` ; `AGENTS.md` impose leur mise à jour à chaque évolution.

La première passe intégrée n’était pas entièrement verte (CCA remis à zéro) ; les résultats 4/4 proviennent de la relance après correction. Banque, PC Windows, clé cloud et réception APNs ne sont pas certifiés par ces tests locaux.

Preuves locales temporaires : `/tmp/opale-fixed-web-build-final.log`, `/tmp/opale-fixed-web-browser-final.log`, `/tmp/opale-fixed-real-web-final.log`, `/tmp/opale-fixed-test-api.log`. Les commandes reproductibles sont dans [RECETTE.md](RECETTE.md) et le [guide web](../web/README.md).
