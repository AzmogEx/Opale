# Livraison iOS — 9 octobre 2026

Ce suivi couvre le client iOS. La matrice interplateforme et les migrations sont tenues dans la documentation de livraison du dépôt. Les modifications locales préexistantes de `APIClient.swift` et `APIPilote.swift` ont été conservées dans la livraison publiée sur GitHub.

Le projet n'impose aucune équipe de signature. Le simulateur utilise une signature ad hoc locale. Le 9 octobre, à la demande de l’utilisateur, l’application a été signée avec son équipe Cleanows et installée sur son iPhone 17 Pro, puis mise à jour avec le formulaire initial. Le choix de l’équipe reste un paramètre de compilation local. L’installation ne prouve pas la réception APNs ou le bon fonctionnement de tous les fournisseurs externes.

## Formulaire initial — 9 octobre

Les profils personnels vides ouvrent automatiquement un formulaire après authentification : revenu net, compte en EUR avec solde facultatif, charges fixes, abonnements, dépenses variables et objectif, puis récapitulatif. Chaque étape reste modifiable. Le brouillon local protégé est isolé par serveur/profil, et les étapes validées sont sauvegardées sur le serveur. « Plus tard » permet de reprendre depuis Réglages → Ma situation de départ ; une panne réseau offre une fermeture conservant le brouillon local.

La confirmation crée les échéances et l’objectif atomiquement, une seule fois, sans comptabiliser de paiements imaginaires. La consultation d’une configuration terminée renvoie vers les comptes, le calendrier et les objectifs actuels. Les abonnements déclarés apparaissent séparément des détections dans les transactions, avec la devise de leur compte actuel. Un compte courant à découvert peut être créé puis revalorisé. Les profils déjà remplis ne sont pas interrompus ; les démos sont exclues.

Le budget variable est une estimation de ce formulaire, pas une nouvelle opération ni une enveloppe automatiquement créée. L’interface reste compatible avec un ancien backend et explique la nécessité du redéploiement depuis les Réglages. Les migrations `0022` et `0023` s’appliquent au démarrage du backend.

Validation : 21 tests unitaires iOS passent, dont 7 pour la mensualisation exacte, les valeurs incomplètes, les dates, la validation par étape et l’isolation du cache. La suite backend avec PostgreSQL réel et `-race` passe ; les tests ciblés couvrent aussi finalisation concurrente, rollback, export/reset, isolation et découvert. L’application et son widget compilent avec signature pour l’iPhone physique.

Les deux recettes UI passent contre une API et une base synthétiques : reprise depuis Réglages puis après relance/réauthentification, et parcours complet avec un salaire, un solde, une charge, un abonnement et un objectif. Le contrôle API final retrouve exactement un compte, trois règles et aucune transaction réalisée. Le récapitulatif affiche bien 1 839,01 € pour 3 200 − 850 − 10,99 − 500. Après deux ajustements de l’automatisation (biométrie enrôlée du simulateur et cible tactile du toggle SwiftUI), la reprise passe à 12:14 et le parcours complet à 12:19 le 09/10/2026. Les captures de compte, reprise et récapitulatif ont été inspectées. Preuves locales temporaires : `/tmp/opale-onboarding-evidence`, `/tmp/opale-onboarding-final-tests.log` et `/tmp/opale-onboarding-completion-tests.log`.

## Contrats, revenus, documents et investissements — 9 octobre

Réglages → Outils donne accès aux contrats, aux revenus variables et à l’explorateur de placements. Les contrats suivent tarif, périodicité, compte, marchand bancaire explicitement renseigné, essai, engagement, renouvellement et préavis. Une hausse possible compare le dernier prélèvement comptabilisé au tarif déclaré sur le même compte, marchand et devise ; elle reste à confirmer ou à ignorer, avec surcoût annuel et historique des tarifs. Les rappels alimentent la boîte d’alertes. Leur livraison push nécessite une configuration APNs fonctionnelle ; le payload ne contient aucun nom de contrat ni montant.

Les revenus variables conservent minimum, habituel et maximum, fréquence et compte. La prévision utilise le minimum en mode prudent, l’habituel sur choix explicite, ou aucun montant hors prévision ; le maximum n’est jamais ajouté automatiquement. Un minimum nul reste nul. Les règles de calendrier gérées se modifient depuis leur contrat ou revenu d’origine. La correspondance explicite compte/marchand/sens évite de substituer une détection récurrente à une déclaration prudente ; sans marchand renseigné, les rapprochements restent à vérifier dans le calendrier. Aucun contrat ni revenu déclaré ne crée un paiement réalisé.

Depuis le formulaire initial, un contrat ou un revenu variable, le préremplissage accepte une image ou un PDF depuis Fichiers. Extraction PDF ou OCR Vision sur l’appareil, limite de 10 Mio et des cinq premières pages, puis choix du montant/devise et vérification explicite avant application. Le net payé après impôt est privilégié pour la paie et le total à payer pour une facture ; les champs incertains restent à corriger. Aucun document brut n’est envoyé à l’API ou conservé par ce parcours. Une facture passée ne fixe pas silencieusement une prochaine échéance future.

L’explorateur contient 19 pistes documentées au 09/10/2026, filtrées par risque indicatif, résidence fiscale, zone, support, enveloppe et horizon. Il comprend livrets, comptes à terme, fonds euros, obligations, ETF, actions, crypto et immobilier. Les fiches distinguent risques, frais, liquidité, accès à vérifier et sources officielles ; les favoris locaux sont isolés par serveur/profil. Les filtres ne constituent pas une recommandation personnalisée, aucun cours ou rendement en direct n’est affiché, et la fiscalité hors France n’est pas modélisée.

Le backend doit recevoir la migration `0024`. Elle reprend les abonnements d’un formulaire déjà terminé sans ajouter de règle en double. Les mutations contrôlent profil, compte/devise et révision ; export et réinitialisation incluent les nouvelles données. La migration depuis `0023`, son rejeu, les transactions atomiques, les hausses, les prévisions, les comptes distincts et la déduplication push ont été testés sur PostgreSQL réel de recette avec `-race`. `go vet` et la compilation Go passent.

Validation iOS finale : **27 tests unitaires passent à 14:08 le 09/10/2026**, dont 6 nouveaux couvrant les montants, devises, ambiguïtés de documents, filtres d’investissement et une vraie extraction d’image/PDF. **Le parcours UI complet passe à 13:59** : contrat 10,99 €, prélèvement 12,99 €, hausse confirmée, revenu 0/1 000/2 000 €, fiche WPEA et favori. L’API de recette retrouve deux tarifs et une seule transaction, celle du prélèvement synthétique. Les captures de hausse, revenus et fiche d’investissement ont été inspectées. Preuves locales temporaires : `/tmp/opale-financial-tools-evidence`, `/tmp/opale-financial-tools-ui-tests.log`, `/tmp/opale-financial-tools-final-swift-tests.log`, `/tmp/opale-financial-tools-go-final.log`. La compilation signée Cleanows et l’installation sur l’iPhone physique réussissent ; la livraison réelle APNs reste à vérifier.

## État et parcours

« Vérifié » ci-dessous précise la preuve : compilation, test de logique, contrat contre une vraie API/PostgreSQL synthétique, ou interface du simulateur. Une compilation ne constitue pas une recette physique de toutes les interfaces.

| Exigence | Priorité | Parcours livré | État / preuve |
|---|---|---|---|
| EF-001/002, ENF-004 | Bloquant | PIN profil ; Keychain ; verrouillage par défaut au démarrage, arrière-plan et après cinq minutes d’inactivité ; écran opaque ; contenu financier démonté pendant le verrouillage ; Face ID/code appareil ou PIN profil en ligne | Vérifié : tests Keychain, inactivité, cache hors ligne, 401 et 503 ; recette UI à froid passée ; authentification physique restante |
| EF-003/004/005 | Haute | Cinq onglets ; navigation reconstruite au changement de profil ; masquage des montants et des données accessibles ; thèmes clair/sombre/système et couleurs d’accent ; widget discret | Interface compilée ; navigation, mode discret et arbre accessible Accueil/P8 testés ; thèmes/widget revus et compilés |
| EF-006/007/008 | Haute | Export protégé et nettoyé après partage ; choix/création de plusieurs espaces, ajout/retrait de membres ; EUR/USD/GBP/CHF/JPY/KWD/CAD/AUD ; taux datés/source | Vérifié : parsing exact JPY/EUR/KWD et dates ; droits et exports vérifiés côté serveur ; UI compilée |
| EF-010–016 | Haute | Patrimoine et cash courants du serveur ; avertissement total incomplet ; compteur animé ; variation et pourcentage ; graphe tactile ; score ; paliers et fonds d’urgence avec célébration | Vérifié : contrats solde/cash/total incomplet et affichage accueil ; sélection de graphique et jalons revus/compilés, sans test tactile de chaque palier |
| EF-020–024 | Haute | Saisie, détail/édition montant/date/compte/catégorie/note, suppression, ventilation ; filtres compte/catégorie/mois/recherche et pagination ; CSV/OFX 5 Mio/10 000 opérations ; suggestions IA locales vérifiables et règles apprises | Vérifié : mutations API et validation des propositions IA ; moteur Foundation Models à vérifier sur iPhone compatible |
| Intégrité des flux | Bloquant | Nature du mouvement ; virement entre comptes atomique avec frais et devises ; principal rattaché à une dette ; mouvements liés non éditables/ventilables individuellement ; suppression du virement complet | Vérifié : décodage et mutation réelle du principal/solde ; moteur et atomicité vérifiés par tests backend |
| EF-025–028/042 | Haute | Échéances/séries CRUD, fin de récurrence, confirmation/exclusion détection, occurrence prévu/exclu/réalisé liée à transaction recherchable/paginée ; horizon ; enveloppes modifiables ; objectifs éditables, rythme, date estimée et explication | Vérifié : contrat calendrier/goals, création/suppression réelle et fréquence trimestrielle conservée ; montant détecté EUR à ressaisir quand le compte change de devise ; interfaces compilées |
| EF-030–032 | Haute | Actif/dette avec valeur/date initiales atomiques et clé d’idempotence ; édition du nom/note ; archives accessibles ; ajout/correction/suppression confirmée des valorisations | Vérifié : création idempotente et solde après édition/suppression de transaction ; recette UI création à 1 234,56 € puis historique |
| EF-033–036 | Moyenne | Immobilier, crédits de même devise ; placements, apports/retraits/distributions/frais et complétude ; objets et pièces/photos ; société/parts/CCA liée explicitement | Vérifié : investissement 1 000 + apport 500 → valeur 1 500, gain 0 ; monnaies natives et couverture du rendement ; autres écrans compilés |
| EF-040–045/052 | Haute | FIRE daté, inflation, objectifs, scénarios enregistrés par profil, timeline ; comparaisons acheter/louer, cash/crédit, rembourser/investir avec frais, liquidité, nominal/réel, impacts 0/5/10 ans, scénarios prudent/normal/ambitieux, orientation conditionnelle et risques | Vérifié : contrat réel des impacts, trois scénarios, recommandation, risques et sensibilités ; vues compilées ; revue des entrées Projection/Assistant/Patrimoine |
| EF-050/051/060–062 | Haute | IA locale pour intentions/définitions, faits chiffrés rendus depuis le moteur ; états de capacité ; historique 20 messages isolé et effaçable ; annulation ; cloud N2 avec consentement explicite par demande ; bilan/Twin/risques | Vérifié : compilation et contraintes locales ; fournisseurs réels restant à valider |
| EF-063/064 | Haute | Contacts modifiables ; bénéficiaires liés aux documents avec quotes-parts précises à 0,01 % ; désignation du profil de confiance, périmètre, activation/expiration/révocation ; accès reçu limité ; coffre ajout/édition/téléchargement/suppression | Vérifié : mutations contact/document/bénéficiaire, 33,33 % conservés au rechargement et contenu téléchargé identique contre API avec coffre activé ; droits testés backend |
| Banque | Haute | Connexion, renouvellement/révocation, état et dernière synchro, erreurs, découverte des comptes et association distincte, soldes/provisoires séparés | Implémenté, contrat de liste vérifié ; consentement et synchronisation fournisseur réels restants |
| P8 | Haute | Réglages → Pilote automatique : fiscalité/PER 2026 avec plafond/source/limites, crédit/capacité/amortissement, allocations cibles, alertes CRUD, Wrapped/période, cours/source/fraîcheur, historique mensuel observé | Vérifié : contrats fiscalité/allocation/alertes/Wrapped/cours/instantanés ; test UI huit entrées et bilan ; crédit/capacité compilés, sans simulation UI exhaustive |
| Démo | Bloquant | Bouton connexion immédiate par jeton d’un profil synthétique isolé/expirable ; introduction que l’on peut réellement passer | Vérifié : API et recette UI ; aucun PIN fixe, aucune suppression par nom |
| Push | Haute | Permission explicite, enregistrement/rotation/révocation ; environnement développement/production ; payload sans montants ; profil contrôlé ; destination `alerts` dans boîte dédiée présentée après déverrouillage | Vérifié : tests mauvais profil/destination inconnue/alerts ; livraison APNs réelle restante |

## Conventions importantes

- Les montants saisis sont convertis par `Decimal` vers les unités mineures de leur devise, avec contrôle des fractions et dépassements. `Double` sert seulement au tracé et aux animations. Les agrégats du serveur sont en EUR ; les fiches individuelles gardent leur devise native.
- `theoretical_cents` est la valeur courante d’un actif. La dernière valorisation reste distincte. Une absence de valeur n’est pas affichée comme zéro. Les opérations de la date d’une valorisation sont déjà comprises dans celle-ci ; les opérations ultérieures sont ajoutées par le serveur.
- Dates civiles et bornes françaises : calendrier grégorien Europe/Paris, validation stricte des dates et présentation française indépendante du fuseau du téléphone.
- Les instantanés mensuels sont des observations prises à la première exécution du mois (`recorded_at` affiché), pas des bilans garantis de fin de mois. Le graphe d’accueil est l’évolution recalculée.
- Hors ligne : consultation du cache du bon serveur/profil après authentification locale. Pas de file de mutations. Une absence réseau ou erreur 503 conserve le jeton/cache ; un 401 confirmé les invalide. Sans code appareil disponible, le PIN profil demande une connexion au serveur.
- Un jeton Keychain est lié à son URL serveur. Changer l’URL ne transmet pas l’ancien jeton au nouveau serveur, y compris pour le rafraîchissement en arrière-plan. Le cache et l’historique restent séparés par serveur et profil.
- Les fichiers de coffre/export sont temporaires et protégés, puis supprimés après fermeture. L’export explicitement demandé contient les données complètes ; le mode discret ne chiffre ni n’occulte le contenu exporté.
- Les règles serveur assurent le premier classement/nettoyage des imports. La suggestion Foundation Models est explicitement nommée « sur l’appareil », contrôlée contre les catégories existantes, proposée à l’utilisateur puis enregistrée après acceptation. Aucune sortie locale arbitraire n’est présentée comme un calcul financier.

## Validation reproductible

Xcode 26, SDK/simulateur iOS 26.5, XcodeGen ; cible iOS 26.0. Simulateur utilisé : iPhone 17 Pro `14EA674C-FDDA-4A61-A988-64D9551D73AD`. Projet généré depuis `project.yml`, tous les nouveaux fichiers inclus. Les données de test viennent uniquement de l’API synthétique `127.0.0.1:58088` et de profils démo créés pour les tests.

```sh
xcodegen generate --spec ios/project.yml
TEST_RUNNER_OPALE_TEST_API_URL=http://127.0.0.1:58088 \
xcodebuild -project ios/Opale.xcodeproj -scheme Opale -configuration Debug \
  -destination 'platform=iOS Simulator,id=14EA674C-FDDA-4A61-A988-64D9551D73AD' \
  -derivedDataPath /tmp/opale-ios-delivery-build \
  CODE_SIGN_IDENTITY=- CODE_SIGNING_ALLOWED=YES CODE_SIGNING_REQUIRED=NO \
  DEVELOPMENT_TEAM= ARCHS=arm64 COMPILER_INDEX_STORE_ENABLE=NO \
  test -collect-test-diagnostics never \
  -only-testing:OpaleTests -only-testing:OpaleUITests/DeliveryTests
```

La signature ad hoc est nécessaire aux tests Keychain : `CODE_SIGNING_ALLOWED=NO` compile mais ne valide pas le fonctionnement réel du trousseau. La CI peut exécuter `FinancialPresentationTests` et `SessionIsolationTests` sans API ; `DeliveryContractTests` signale explicitement un skip si `OPALE_TEST_API_URL` manque. Les tests UI Delivery nécessitent une API synthétique sur le port 58088.

Résultats réels du 08/10/2026 :

- **15 tests unitaires/API passés à 14:58:49** : 8 `FinancialPresentationTests`, 6 `SessionIsolationTests`, 1 `DeliveryContractTests`. Ce dernier s’exécute contre l’API et PostgreSQL réels de recette, avec un nouveau profil démo, et n’a pas été ignoré. Il vérifie notamment le total incomplet, le solde courant après mutations, l’idempotence, le principal de dette, la performance hors apports, le coffre, une part bénéficiaire de 33,33 %, la récurrence trimestrielle et les impacts/scénarios/recommandation/risques du comparateur.
- **Trois parcours UI passés dans la même exécution**, achevée à 15:03:58 : connexion démo et huit entrées P8 avec bilan annuel ; lancement à froid verrouillé sans éléments financiers accessibles ; création d’un compte à 1 234,56 €, historique et mode discret.
- **Le quatrième parcours, `testLargeTextAndDiscreetPilot`, est passé lors de la relance ciblée à 15:06:17**. La première exécution avait échoué avant la connexion : le serveur avait renvoyé HTTP 500 à 15:01:01, PostgreSQL `53100` (disque plein). Après récupération d’espace, la création démo fonctionne. Cet incident d’environnement reste consigné ; la suite intégrée n’est pas présentée comme entièrement verte en une seule exécution.
- **Dernière compilation et recette ciblée réussies à 15:09:21** après correction de la largeur des graduations du graphique. La capture d’accueil XXXL a été inspectée : variation et pourcentage lisibles, graduations complètes `150 k€`, `100 k€`, `50 k€`. Cette retouche d’affichage n’a pas changé les contrats ou la logique testés à 14:58.
- Le test XXXL contrôle l’accueil, la disparition du graphique en mode discret, le libellé accessible « Montant masqué », l’accès au bilan P8 et l’absence de texte contenant `€` dans son arbre accessible. L’inspection visuelle a conduit à empiler la variation mensuelle et les lignes de bilan ; le texte financier est remplacé par des points, car un simple flou ne suffit pas à protéger les annonces accessibles.

Les preuves sont conservées dans `/tmp/opale-ios-evidence-20261008` : journaux `opale-ios-final-tests.log`, `opale-ios-final-accessibility.log` et `opale-ios-final-chart-accessibility.log`, bundles XCTest horodatés, captures `cold-start-locked.png`, `asset-history.png`, `wrapped.png`, `home-xxxl.png` et `wrapped-discreet-xxxl.png`. Les données sont exclusivement synthétiques. Ce dossier de preuves est local et temporaire ; les commandes ci-dessus permettent de reproduire la validation. Le DerivedData temporaire de cette mission a été supprimé après conservation des preuves.

La couverture UI est ciblée. Les autres formulaires et écrans sont revus et compilés, avec contrats réels lorsque cités dans le tableau ; ils ne sont pas déclarés tous parcourus manuellement. En particulier, la saisie complète de chaque simulation crédit/FIRE, tous les filtres/imports depuis Fichiers, chaque centre métier, chaque palier et chaque geste de graphique n’ont pas une recette UI exhaustive. Aucun manque fonctionnel connu n’est laissé volontairement derrière un bouton factice.

## Recette externe restante

| Dépendance | Code et vérification locale | Action nécessaire / résultat attendu |
|---|---|---|
| Face ID/Touch ID et PIN appareil | Écran explicite, verrouillage/inactivité testés ; UI à froid passée | Installer avec signature valide sur iPhone ; tester réussite/annulation, arrière-plan, cinq minutes, démarrage sans réseau, reprise. Aucune donnée accessible avant authentification |
| Foundation Models | Routes et validation structurée testées ; aucun cloud implicite | iPhone iOS 26 compatible, Apple Intelligence activé/téléchargé ; ouvrir transaction, demander suggestion, refuser/accepter, vérifier catégorie et règle après rechargement ; désactiver la capacité et contrôler le message honnête |
| Banque réelle | Mapping/status/erreurs raccordés ; contrats décodés | Fournisseur configuré côté serveur ; consentir sur compte de test, découvrir deux comptes, associer distinctement, resynchroniser, renouveler/révoquer ; vérifier absence de doublons et mise à jour sans app ouverte |
| APNs | Jetons/profils/destination/environnements et contenu minimisé testés localement | Équipe/provisioning avec Push Notifications et App Group, clé/certificat serveur et topic ; autoriser sur appareil, recevoir, toucher avec app verrouillée, déverrouiller → boîte d’alertes du bon profil ; tester profil différent, rotation et déconnexion |
| Fournisseurs IA/cours | Contrats et erreurs raccordés, consentement explicite | Activer les fournisseurs retenus sur le serveur ; vérifier date/échec des cours et états fournisseur IA ; N1 aucun cloud ; N2 sans consentement aucun cloud |
| Accessibilité physique | Libellés/montants masqués et layouts adaptatifs ; échantillon simulateur XXXL/AX testé | VoiceOver, tailles maximales et Réduire les animations sur iPhone : ordre de lecture, saisie au clavier, navigation de graphique, aucune annonce des montants masqués |
| Performance et déploiement | App/widget/tests compilés et app installée en simulateur | Mesurer 60–120 fps avec Instruments sur appareil ; configurer signature, App Group et TLS de l’hôte cible ; les tests locaux utilisent HTTP de boucle locale |

Aucune banque réelle, livraison APNs, session Face ID physique ou inférence Foundation Models sur téléphone n’est revendiquée comme validée par ces tests.

## Parcours guidé et choix IA — 9 octobre 2026

L’accueil iOS devient « Mon parcours » : comptes, revenus, charges fixes, abonnements, biens/crédits, budget/projets, puis bilan expliqué. Chaque étape présente les données existantes et leur édition. Les ajouts déclenchent le rechargement même quand une étape est ouverte. La progression est conservée sur le serveur avec révision ; le budget non validé dispose d’un brouillon local lié au serveur, au profil et à la révision. Les étapes passées restent distinctes des étapes vérifiées. Le bilan n’affiche aucun reste disponible tant que revenus, charges, abonnements et budget n’ont pas été vérifiés.

L’assistant propose Automatique, Mon PC et Cloud. Foundation Models n’intercepte plus le chat. Ollama vérifie le modèle installé ; Claude est configurable. Les sorties financières utilisent des références de faits, d’explications et d’actions contrôlées. Le guide intégré fonctionne sans fournisseur. Les deux configurations sont décrites dans [le guide Coolify](../docs/COOLIFY.md#choisir-lia--pc-windows-ou-cloud).

Validations réelles de cette mise à jour :

- Backend : tests avec détection de concurrence des packages `ai`, `api` et `store`, vérifications ciblées du report de budget et du consentement, `go vet ./...` et `go build ./...` réussis. PostgreSQL de recette distinct de la base personnelle. La migration `0025`, les révisions concurrentes, l’isolation des profils, l’export et la remise à zéro du parcours sont couverts.
- iOS : 32 tests unitaires réussis à 15:24:11, dont les cinq nouveaux tests du parcours. Après correction du rafraîchissement des étapes, les cinq tests du parcours sont à nouveau passés. Les autres changements depuis les 32 tests concernent la navigation et les contrôles de recette UI.
- Recette UI complète réussie à 15:33:36 : compte à 2 000 €, salaire à 3 200 €, charge à 850 €, abonnement à 10,99 €, budget à 500 €, reste à 1 839,01 €, sept étapes vérifiées, trois règles de calendrier et **zéro transaction réalisée**. Ouverture des réglages IA, réponse intégrée « Par où commencer ? » et retour au parcours également vérifiés.
- Recette UI de reprise réussie à 15:36:16 : cinq étapes passées, brouillon à 735,42 € conservé après fermeture et reconnexion, validation sur le serveur et bilan « À compléter » pour les étapes non vérifiées.
- Compilation signée pour l’iPhone physique et installation de la version corrigée réussies. Le conteneur de données de l’app est conservé.

Les premières recettes ont révélé un rafraîchissement attaché à l’accueil masqué pendant la navigation, corrigé en portant la tâche sur le `NavigationStack`, ainsi que des sélecteurs UI qui cherchaient des textes isolés alors qu’iOS regroupait les libellés. Les réussites ci-dessus proviennent des relances après correction ; la première exécution n’est pas déclarée verte.

Preuves locales temporaires : `/tmp/opale-journey-backend-final.log`, `/tmp/opale-journey-targeted.log`, `/tmp/opale-journey-recette-verified.log`, `/tmp/opale-journey-manual-ui.log`, `/tmp/opale-journey-resume-ui.log` et les bundles XCTest dans `/tmp/opale-local-simulator-build/Logs/Test`. Pour rejouer les deux parcours sur l’API synthétique `127.0.0.1:58088`, utiliser la commande de validation précédente avec `-only-testing:OpaleUITests/FinancialSetupUITests` ; les profils créés par ces tests sont supprimés en fin de test.

Les fournisseurs distants ont été testés par serveurs HTTP synthétiques. Aucune clé cloud personnelle ni connexion au PC Windows n’a été utilisée ; leur accessibilité, leur coût réel et leurs performances restent à vérifier après le choix du fournisseur.
