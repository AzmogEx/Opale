# Recette reproductible de livraison

Recette du 8 octobre 2026, exclusivement sur données synthétiques. L’état par exigence est dans [LIVRAISON.md](LIVRAISON.md). Les preuves détaillées des clients se trouvent dans [iOS](../ios/DELIVERY.md) et [web](../web/DELIVERY.md).

## Environnement utilisé

Go 1.26.4, PostgreSQL 17.11, Node 22.16, Svelte 5/SvelteKit, Xcode 26.6 (17F113), XcodeGen 2.45.4, simulateur iOS 26. Aucun iPhone physique utilisé. Aucun fournisseur bancaire ou IA payant n’a reçu de données personnelles.

La première base jetable était dans Docker. Après l’indisponibilité de Docker Desktop, la recette a continué avec un cluster PostgreSQL natif distinct dans `/tmp/opale-delivery-postgres`, port 61547, base `opale_test`, utilisateur `opale_test`. L’API de recette écoute uniquement `127.0.0.1:58088` ; le web de développement est sur 5173. Ces chemins et ports décrivent cette session ; créer un autre environnement jetable pour une nouvelle exécution. Ne jamais employer une URL de base personnelle avec les tests. Les tests de migration créent et suppriment leurs propres bases auxiliaires et demandent le droit CREATEDB.

Incident environnemental observé : la saturation disque a interrompu une création de démo (PostgreSQL `53100`, HTTP 500) et l’écriture de résultats navigateur/Xcode. Seuls les caches et artefacts de cette recette ont été nettoyés. Les contrôles concernés ont été repris après libération d’espace ; cet incident ne constitue pas une validation du fonctionnement en disque plein. Prévoir une alerte d’espace libre avant déploiement.

## Backend et intégrité

Depuis `backend/`, avec `OPALE_TEST_DATABASE_URL` pointant exclusivement sur la base jetable :

```sh
go mod tidy -diff
go vet ./...
go build ./...
go test -race ./...
```

Sans cette variable, les tests SQL sont ignorés : un résultat vert ne prouve alors que les tests unitaires. Les migrations sont appliquées automatiquement par les tests SQL. Elles sont sérialisées par verrou et exécutées dans une transaction ; ne pas réinitialiser un schéma pour les faire passer.

| Risque / parcours | Preuve exécutable |
|---|---|
| Références A/B, absence de mutation après refus | `api.TestCrossProfileReferencesRejectAtomically` |
| Partage, retrait et détachement des opérations | `api.TestSharedSpaceRemovalRevokesAndDetaches` |
| Urgence, document exact, périmètre, expiration et révocation | `api.TestDeliveryEmergencyLifecycleAndBeneficiaryIsolation` |
| Démo homonyme et identités concurrentes | `api.TestDemoDoesNotReplaceNamesake`, `TestConcurrentDemosNeverShareIdentity` |
| 505 puis 12001 opérations, sommes indépendantes, export et Wrapped | `api.TestExhaustive505AndLargeExport` |
| Import/split/réimport, huit imports concurrents, changement fournisseur | `store.TestImportsConcurrencySplitAndProviderUpdates` |
| Ancienne ventilation ambiguë, refus de tout le fichier | `store.TestLegacySplitImportRefusesAmbiguousParentAtomically` |
| Clôture, opérations futures, virements, frais et principal | `store.TestBalancesTransfersAndValuationBoundaries`, `TestLinkedPrincipalPreservesNetWorthAndRefusesOverpayment` |
| JPY/KWD, change daté privé et référence manquante | `store.TestCurrencyMinorUnitsHistoryAndIsolation` |
| Calculs aux limites sans débordement | tests `money`, `engine`, dont `TestDeliveryExtreme*` et `TestProjectionExtremesAndRatePrecision` |
| Décisions à 0/5/10 ans, trois scénarios et orientation conditionnelle | `engine.TestDecisionTimelineConservesWealthAndInitialFees`, `TestDecisionScenariosAndConditionalRecommendation`, `TestDecisionRecommendationNeverChoosesUnfundedAlternative` |
| Objectifs et double affectation de l’épargne | `store.TestDeliveryGoalAllocationsSerialize`, `engine.TestGoalProjectionCivilAndBoundaries` |
| Calendrier, fins de mois, réalisation unique | `store.TestDeliveryCalendarPersistenceAndRealizations`, `TestCalendarDatesMonthEnds` |
| Trimestre conservé, fins de mois et retour arrière sans perte | `store.TestCalendarDatesQuarterlyPreservesOriginalDay`, `TestDeliveryQuarterlyCalendarPersistsAndDownRefusesDataLoss`, `api.TestDeliveryQuarterlyCalendarAPI` |
| Performance sans gain fictif après apport ; archive | `engine.TestContributionIsNotMarketGain`, `store.TestDeliveryInvestmentFlowsAndArchiveHistory` |
| CCA compté une fois | `store.TestDeliveryCompanyCCAIsCountedOnceAndKeepsEdits` |
| API banque, dates/identifiants, reprises et délai429 | `api.TestDeliveryBankMockSyncReconciliationIdempotencyAndBackoff`, tests `bank` |
| Consentement et minimisation des requêtes réellement émises par le SDK | `ai.TestOutboundHTTPIsAllowlistedAndOptIn`, `api.TestCloudPayloadNeverContainsFreeText` |
| Réponses du modèle : chiffres inventés, champs ou identifiants non autorisés refusés | `ai.TestProviderCannotInventDisplayedNumbers` |
| Nettoyage de libellé privé, prévisualisation sans mutation et refus croisés | `api.TestLabelSuggestionPrivatePreviewAndOwnership` |
| Questions avec mois répétés, années distinctes et bissextiles ; ambiguïtés non tronquées | `nlq.TestExplicitIndependentPeriods`, `TestAmbiguousOrExcessPeriodsNeverConfident` |
| Total incomplet visible et déconnexion effective | `api.TestMissingValuationAndSessionRevocation` |
| APNs : signature/payload sans montants, bon profil, déduplication persistante | tests `push`, `jobs.TestPushPrivacyAndPersistentDeduplication` |
| Migrations depuis version 10, conservation et refus atomiques | `store.TestMigrationPreservesExistingDataAndRefusesCrossOwner`, `TestMigrationRefusesAmbiguousLegacyCurrencyUnits` |
| Suppression/réinitialisation avec banque et CCA | `store.TestProfileCleanupWithLinkedDomains` |
| Suppression de compte lié et valorisations de dette historiques/futures | `store.TestDeliveryAccountDeletionPreservesTransferAndPrincipal`, `TestDeliveryLiabilityValuationCorrectionRollsBack`, `TestDeliveryLaterSnapshotCannotHideNegativeDebtHistory` |
| Journaux et métriques sans chemin utilisateur libre | `api.TestMetricsAndLogsNeverRecordUserPaths`, `TestMetricsConcurrentCollection` |

Les sources fiscales officielles, millésime et limites sont dans [DELIVERY-BACKEND.md](DELIVERY-BACKEND.md). Les tests de banque/APNs utilisent des serveurs HTTP locaux : ils ne prouvent pas un consentement bancaire ou une réception Apple réelle.

La suite globale avec `-race -count=1` a réussi après les ajouts de comparaisons, de NLQ multi-années et de suggestion de libellé. Après les deux derniers changements (libellé affiché seul transmis au homelab et migration 0021 trimestrielle), une passe ciblée avec `-race` a réussi sur la suggestion, le trimestre et les migrations depuis la version 10, suivie de `go vet ./...`, `go build ./...` et `go mod tidy -diff` sans erreur ni écart.

## Web et iOS

Web depuis `web/` :

```sh
npm ci
npm run check
npm test
npm run build
npx playwright install chromium
npm run dev -- --host 127.0.0.1
# Dans un second terminal, serveur web actif :
npm run test:e2e
# Uniquement avec l’API et la base de recette isolées :
OPALE_INTEGRATION=1 npm run test:e2e:integration
```

`OPALE_API_URL` configure la cible du proxy Vite ; `OPALE_WEB_URL` configure l’origine Playwright. Pour cette session, le proxy utilise `http://127.0.0.1:58088`. `OPALE_CHROMIUM_PATH` permet de réutiliser Chrome installé quand le téléchargement Playwright n’est pas disponible. Les tests créent des profils synthétiques avec leur propre nom et les suppriment avec la confirmation requise.

Recette iOS : générer le projet depuis `ios/project.yml` avec `xcodegen generate --spec ios/project.yml`, choisir un simulateur iPhone iOS 26 disponible avec `xcrun simctl list devices available`, puis compiler et tester. La commande exacte et les parcours couverts figurent dans [ios/DELIVERY.md](../ios/DELIVERY.md). Les tests Keychain nécessitent une signature ad hoc de simulateur : `CODE_SIGN_IDENTITY=- CODE_SIGNING_ALLOWED=YES CODE_SIGNING_REQUIRED=NO DEVELOPMENT_TEAM=`. Désactiver toute signature fait échouer le Keychain ; cela n’autorise pas à supprimer les tests. La CI macOS inclut les tests de présentation et d’isolation/session.

Pour la recette visuelle, vérifier les cinq onglets, formulaires et erreurs ; petit écran web et grand écran ; navigation au clavier ; textes longs ; mode clair/sombre/discret ; montants absents de l’arbre d’accessibilité en mode discret ; verrouillage à froid et inactivité ; réponse tardive après changement de profil. Tester perte réseau et401 séparément : une panne réseau n’est pas une révocation confirmée. Aucun test simulateur ne constitue une mesure 60–120 fps sur iPhone ni une recette VoiceOver physique complète.

Résultats clients obtenus par lots : web **13 tests unitaires, 8 tests navigateur avec API simulée et 4 parcours avec API/PostgreSQL réels**, check/build réussis ; iOS **15 tests de logique/contrats et 4 parcours UI distincts**, compilation signée ad hoc réussie. Trois parcours UI iOS ont réussi dans la dernière suite intégrée, et le parcours très grande taille de texte a été relancé avec succès après l’incident disque puis après la dernière correction graphique. La recette UI est ciblée, elle ne prétend pas avoir parcouru chaque formulaire. Les rapports clients détaillent les preuves et les captures inspectées.

## Sauvegarde et restauration

Suivre [EXPLOITATION.md](EXPLOITATION.md) : dump custom, somme SHA-256, restauration dans une **nouvelle** base, même clé de coffre, API clone avec toutes intégrations désactivées. Reconnexion par PIN, comparaison des totaux et téléchargement déchiffré avec SHA-256 identique sont obligatoires. Une simple création d’archive ne suffit pas. Ce parcours a été réalisé sur PostgreSQL natif 17. Docker Compose est validé syntaxiquement ; son démarrage complet reste à exécuter sur un moteur Docker fonctionnel ou dans la CI préparée.

## Vérifications extérieures restantes

| Fonctionnalité | Code et tests locaux | Prérequis / action précise | Résultat attendu |
|---|---|---|---|
| Banque GoCardless | Association par compte, booked/pending, reprise/renouvellement/révocation ; mock HTTP + SQL | Identifiants Bank Account Data autorisés, compte sandbox puis consentement réel explicite. Configurer `OPALE_GC_*`, connecter, découvrir, associer séparément, synchroniser deux fois, renouveler et révoquer | Solde daté rapproché, aucune duplication ; mapping conservé ; erreur/consentement visibles ; absence de nouvel accès après révocation |
| IA homelab | Routeur, temps limites, intention validée, fallback et contrats testés | Ollama accessible et modèle installé, `OPALE_OLLAMA_URL/MODEL`, profil synthétique | État disponible, sélection de faits valide ; aperçu du nettoyage de libellé puis acceptation explicite ; arrêt fournisseur → réponse déterministe ou indisponibilité identifiée |
| IA cloud | SDK HTTP intercepté, allowlist, consentement à chaque appel, refus logiciel d’affirmations inventées | Clé autorisée, `OPALE_CLOUD_AI=on`, profil N2 et consentement explicite sur des données synthétiques | Payload minimisé conforme ; données libres absentes ; off/N1/N3 → aucun appel |
| IA Apple locale | App compilée, classification typée et validation des sorties | iPhone compatible Foundation Models, Apple Intelligence activée et modèle téléchargé | Proposition locale valide ; indisponibilité explicite ; montants du backend uniquement |
| APNs | JWT ES256, payload générique lié au profil, évaluation toutes les 15 minutes, jeton et déduplication persistante | Clé P8, équipe/bundle/provisioning, entitlement, iPhone signé et environnement APNs correspondant | Réception générique, destination du bon profil, rotation et déconnexion effectives, aucune donnée financière dans l’aperçu |
| Distribution et accessibilité iPhone | Compilation/tests simulateur et recette visuelle documentés | iPhone physique signé ; VoiceOver/Dynamic Type et Instruments | Parcours atteignables, montants masqués accessibles, verrouillage/apercu protégés, mesure de fluidité réelle |
| CI distante | Workflow Go/PostgreSQL, web/Compose/restauration et iOS prêt ; mêmes contrôles exécutés localement selon les limites ci-dessus | Publier les changements revus sur le dépôt autorisé et déclencher le workflow avec des runners Linux et macOS 26 disponibles | Jobs distants réussis, journaux conservés, aucun contrôle désactivé |
| Déploiement HTTPS/Compose | Fichiers, CI et procédure de récupération prêts | Docker fonctionnel ; domaine/certificat/stockage chiffré ; lancement sur environnement autorisé | Trois services prêts, données persistantes après redémarrage, export fonctionnel en lecture seule, API/DB non exposées, sauvegarde restaurable |

Les cours publics BCE et CoinGecko ont été vérifiés réellement le 8 octobre 2026 sans données de portefeuille, via `OPALE_TEST_PUBLIC_QUOTES=1 go test ./internal/quotes -run TestPublicProvidersLive -v`. La BCE retournait une publication du7octobre2026. Ce test est opt-in car dépendant du réseau public et de ses limites.
