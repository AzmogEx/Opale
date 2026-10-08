# Livraison Opale — matrice de référence

Mise à jour du 8 octobre 2026. Périmètre : cahier des charges, six défauts prioritaires, gestion quotidienne sur les deux clients, domaines approfondis, P8 et exploitation. Le dépôt existant et ses modifications locales ont été conservés ; aucun reset ni commit automatique. Le relevé initial est dans `/tmp/opale-delivery-baseline-20261008`.

## État final et reprise

- Backend intégré, migrations 0011–0021 appliquées uniquement aux bases synthétiques. Suite finale `go test -race -count=1 ./...` réussie après intégration des correctifs comparateur, NLQ et libellés, compilation et dépendances propres ; analyse statique corrigée et réussie. Les vérifications ciblées postérieures sont consignées dans le guide de recette.
- Web terminé : check/build, 13 tests unitaires, 8 tests navigateur et 4 parcours intégrés API réels réussis. iOS : compilation signée ad hoc, 15 tests de logique et contrats passés, 4 parcours UI distincts validés. Le parcours très grande taille de texte a été repris après saturation disque et a réussi après la dernière correction graphique ; captures inspectées. Les rapports détaillés font foi pour le niveau de preuve : test de logique, contrat API/PostgreSQL, revue de raccordement et compilation, ou parcours UI effectivement exécuté. Tous les formulaires iOS ne disposent pas d’un test UI automatisé ; une compilation seule ne vaut pas preuve d’un parcours complet.
- API de recette 58088 et PostgreSQL natif 61547 exclusivement synthétiques. Les bases personnelles n’ont pas été migrées. Docker Desktop indisponible : aucun redémarrage global ni suppression de volume personnel pour contourner le problème.
- Aucun manque fonctionnel connu laissé dans le code du périmètre traité. Les réserves de validation restent explicites : recette UI ciblée, appareil physique, fournisseurs réels, Docker/HTTPS et exécution distante de la CI. Prochaine action : choisir l’environnement de déploiement autorisé et exécuter les procédures externes de [RECETTE.md](RECETTE.md), après sauvegarde. Le dossier [EXPLOITATION.md](EXPLOITATION.md) donne les commandes et prérequis.
- Preuves iOS conservées dans `/tmp/opale-ios-evidence-20261008` ; journaux Go finaux `/tmp/opale-final-integrated-race.log` et `/tmp/opale-final-additive-tests.log`. Ces artefacts locaux sont temporaires ; les résultats et commandes de reproduction sont consignés dans les rapports du dépôt. Les caches de compilation de cette recette ont été nettoyés, les sources et données personnelles conservées.

## Lecture des statuts

« Vérifié » précise une validation locale adaptée dans la dernière colonne et dans les rapports de recette. « Validation externe restante » signifie que le code et ses tests locaux sont présents, avec un prérequis extérieur documenté. « En cours » indique une recette/finition encore active ; « implémenté à vérifier » ne vaut pas livraison. « — » signifie que l’exigence ne nécessite pas de réalisation sur cette surface.

Les références EF/EIA/ENF conservent les priorités M/S/C du cahier des charges ; les références SEC/DATA/OFF/WEB/P8/OPS couvrent les exigences supplémentaires de la mission. Les montants sont des unités mineures natives, pas toujours des centimes EUR. Aucune transmission personnelle cloud n’a été utilisée pour les tests.

## Matrice

| Identifiant | Priorité | Comportement attendu | Backend | iOS | Web | Migration | Validation / dépendance externe |
|---|---|---|---|---|---|---|---|
| EF-001 | M | Multi-profil : profils séparés, données privées par profil | vérifié | vérifié | vérifié | 0011, 0013 | Références A/B, révocation/retrait ; tests session et confidentialité des clients |
| EF-002 | M | Authentification par profil (Face ID / code), verrouillage auto | vérifié | validation externe restante | vérifié | 0011, 0013 | Tests session/cache/401/503/Keychain et verrouillage à froid ; Face ID/code appareil physique restant |
| EF-003 | M | Navigation à 5 onglets (Accueil / Flux / Patrimoine / Projection / Assistant) | — | vérifié | vérifié | — | Recette navigation, graphique/clavier, thèmes, masquage DOM/VoiceOver ; rapports clients |
| EF-004 | S | Mode discret : flouter les montants d'un geste | — | vérifié | vérifié | — | Recette navigation, graphique/clavier, thèmes, masquage DOM/VoiceOver ; rapports clients |
| EF-005 | S | Système de thèmes : couleurs d'accent + clair/sombre/auto | — | vérifié | vérifié | — | Recette navigation, graphique/clavier, thèmes, masquage DOM/VoiceOver ; rapports clients |
| EF-006 | S | Export complet des données du profil (portabilité) | vérifié | vérifié | vérifié | 0011–0021 | 505/12001 opérations : sommes, IDs, manifestes/documents ; téléchargement web/iOS |
| EF-007 | C | Espace partagé optionnel (dépenses communes famille) | vérifié | vérifié | vérifié | 0011 | TestSharedSpaceRemovalRevokesAndDetaches ; balance à somme nulle ; seuls mouvements partagés |
| EF-008 | C | Multi-devises | vérifié | vérifié | vérifié | 0011, 0013, 0015, 0017, 0018, 0020 | JPY/KWD, change historique privé, principal/virement, limites entières |
| EF-010 | M | Afficher le **patrimoine net total** (actifs − dettes) | vérifié | vérifié | vérifié | 0011, 0015, 0018 | Soldes/histoire exacts ; warning valorisation absente ; snapshot cohérent du Twin |
| EF-011 | S | Compteur animé du patrimoine net à l'ouverture | — | vérifié | vérifié | — | Recette navigation, graphique/clavier, thèmes, masquage DOM/VoiceOver ; rapports clients |
| EF-012 | M | Évolution mensuelle du patrimoine (variation + %) | vérifié | vérifié | vérifié | 0011, 0015, 0018 | Soldes/histoire exacts ; warning valorisation absente ; snapshot cohérent du Twin |
| EF-013 | M | Graphe d'évolution du patrimoine net dans le temps, scrubable | — | vérifié | vérifié | — | Recette navigation, graphique/clavier, thèmes, masquage DOM/VoiceOver ; rapports clients |
| EF-014 | M | Afficher le **cash disponible réel** | vérifié | vérifié | vérifié | 0011, 0015, 0018 | Soldes/histoire exacts ; warning valorisation absente ; snapshot cohérent du Twin |
| EF-015 | S | **Score de santé financière** (/100) avec points forts/faibles | vérifié | vérifié | vérifié | — | Tests santé/risques, seuil fonds urgence et limites extrêmes ; présentation clients |
| EF-016 | C | Jalons & paliers de patrimoine (10k, 100k, fonds d'urgence) + célébrations | vérifié | vérifié | vérifié | — | Tests santé/risques, seuil fonds urgence et limites extrêmes ; présentation clients |
| EF-020 | M | Lister revenus/dépenses, recherche, filtres | vérifié | vérifié | vérifié | 0011, 0017, 0018 | CRUD, pagination stable, filtre, split/réimport, transferts et principal liés |
| EF-021 | M | Import de transactions CSV/OFX | vérifié | vérifié | vérifié | 0011 | Fichiers invalides atomiques, Windows1252/OFX/FITID, 8 imports concurrents, ancienne ventilation ambiguë |
| EF-022 | M | Catégorisation automatique (IA) + correction manuelle (apprentissage) | vérifié | validation externe restante | vérifié | — | Règles serveur et corrections apprises vérifiées ; proposition Apple typée et confirmation branchées, modèle réel restant |
| EF-023 | S | Nettoyage des libellés bancaires (IA) | validation externe restante | validation externe restante | validation externe restante | — | Proposition privée prévisualisée/confirmée ; HTTP homelab intercepté, aucun cloud ni mutation avant acceptation ; modèles Apple/Ollama réels restants |
| EF-024 | S | Détail/édition d'une transaction, split multi-catégories | vérifié | vérifié | vérifié | 0011, 0017, 0018 | CRUD, pagination stable, filtre, split/réimport, transferts et principal liés |
| EF-025 | S | Calendrier financier (flux futurs datés) | vérifié | vérifié | vérifié | 0012, 0016, 0021 | Fins de mois/bissextiles/trimestres, réalisation unique, exclusion ; montant natif confirmé, projection sans doublon |
| EF-026 | S | Détection d'abonnements récurrents | vérifié | vérifié | vérifié | 0012, 0016, 0021 | Fins de mois/bissextiles/trimestres, réalisation unique, exclusion ; montant natif confirmé, projection sans doublon |
| EF-027 | S | **Cashflow futur** : cash dispo projeté à une date donnée | vérifié | vérifié | vérifié | 0012, 0016 | Fins de mois/bissextile, réalisation unique, exclusion détection, projection sans doublon |
| EF-028 | S | Enveloppes budgétaires (allocation à l'avance, optionnel) | vérifié | vérifié | vérifié | 0011 | Catégories/règles/enveloppes par propriétaire ; formulaires des deux clients |
| EF-030 | M | Saisie manuelle d'actifs (comptes, livrets, AV, PEA, CTO, crypto, immo, or, parts) | vérifié | vérifié | vérifié | 0011, 0018 | Création initiale atomique/idempotente, CRUD/archives, correction historique et recalcul |
| EF-031 | M | Saisie manuelle de passifs (crédits immo/auto/conso) | vérifié | vérifié | vérifié | 0011, 0018 | Création initiale atomique/idempotente, CRUD/archives, correction historique et recalcul |
| EF-032 | M | Historique de valorisations par actif/passif | vérifié | vérifié | vérifié | 0011, 0018 | Création initiale atomique/idempotente, CRUD/archives, correction historique et recalcul |
| EF-033 | C | Centre immobilier (valeur achat/estimée, rendement, crédit restant, cashflow, taxe) | vérifié | vérifié | vérifié | 0011, 0018 | Liens bien/crédit/documents contrôlés, montants natifs/agrégats EUR, modules clients |
| EF-034 | C | Centre investissement (suivi PEA/CTO/AV/crypto + répartition) | vérifié | vérifié | vérifié | 0012, 0016 | Apport1000+500→1500 gain0 ; couverture requise ; retrait/frais/distribution et archive |
| EF-035 | C | Valeur des objets (montres, or, voitures, œuvres… + photo/facture/assurance) | vérifié | vérifié | vérifié | 0011, 0018 | Liens bien/crédit/documents contrôlés, montants natifs/agrégats EUR, modules clients |
| EF-036 | C | Module entrepreneur (valeur société, parts, CCA, dividendes, fiscalité) | vérifié | vérifié | vérifié | 0012, 0016 | CCA explicitement lié, valeur de ma part ; test répété100→200→100 sans double créance |
| EF-040 | M | **Date d'indépendance financière** (FIRE) calculée et affichée | vérifié | vérifié | vérifié | — | Projection déterministe et extrêmes ; date FIRE réelle ; hypothèse inflation affichée |
| EF-041 | M | Simulateur d'épargne (« si j'épargne X €/mois → patrimoine à N ans ») | vérifié | vérifié | vérifié | — | Projection déterministe et extrêmes ; date FIRE réelle ; hypothèse inflation affichée |
| EF-042 | S | Objectifs intelligents (date prévue, retard/avance, montant restant) | vérifié | vérifié | vérifié | 0012 | Objectifs atteints/sans épargne/retard ; affectations concurrentes plafonnées |
| EF-043 | C | Projections en euros constants (corrigées de l'inflation) | vérifié | vérifié | vérifié | — | Projection déterministe et extrêmes ; date FIRE réelle ; hypothèse inflation affichée |
| EF-044 | C | Comparateur de scénarios (deux futurs côte à côte) | vérifié | vérifié | vérifié | — | Projection déterministe et extrêmes ; date FIRE réelle ; hypothèse inflation affichée |
| EF-045 | C | Timeline patrimoniale (chronologie de la vie financière) | vérifié | vérifié | vérifié | 0011 | Chronologie d’acquisitions/valorisations/jalons et projections raccordée |
| EF-050 | S | Recherche en langage naturel sur les données (« combien en courses en mars ? ») | vérifié | vérifié | vérifié | 0011, 0015 | NLQ exhaustif : mois répétés, deux années distinctes, années entières et bissextiles ; ambiguïtés refusées |
| EF-051 | S | IA patrimoniale contextuelle (connaît la situation complète, raisonne) | validation externe restante | validation externe restante | validation externe restante | — | Contrats, fallback et états testés ; Ollama/Anthropic réels non configurés |
| EF-052 | S | **Mode Décision** (acheter/louer, cash/crédit, rembourser/investir…) avec impacts 0/5/10 ans, risque, reco, scénarios prudent/normal/ambitieux | vérifié | vérifié | vérifié | — | Trois comparaisons, budgets communs, jalons 0/5/10 ans, trois scénarios, faisabilité, risques et orientation conditionnelle |
| EF-053 | S | Alertes intelligentes (dépassement enveloppe, solde bas prévu…) | vérifié | vérifié | vérifié | 0010, 0017 | Seuils personnalisés CRUD ; déclenchement déterministe ; notifications génériques |
| EF-060 | S | **Financial Twin** : double financier (revenus, charges, actifs, dettes, objectifs, habitudes, risques) servant de moteur de simulation | vérifié | vérifié | vérifié | 0011, 0015, 0018 | Soldes/histoire exacts ; warning valorisation absente ; snapshot cohérent du Twin |
| EF-061 | S | **Radar de risques** (cash dormant, surendettement, fonds d'urgence, dépendance revenu, illiquidité…) | vérifié | vérifié | vérifié | — | Tests santé/risques, seuil fonds urgence et limites extrêmes ; présentation clients |
| EF-062 | S | **Bilan mensuel intelligent** (résumé humain rédigé par l'IA) | validation externe restante | validation externe restante | validation externe restante | — | Contrats, fallback et états testés ; Ollama/Anthropic réels non configurés |
| EF-063 | C | **Plan de transmission** (contrats, bénéficiaires, contacts, accès d'urgence) | vérifié | vérifié | vérifié | 0012, 0016 | Bénéficiaires/quotités et accès urgence limité : activation, document, expiration, révocation |
| EF-064 | C | **Coffre-fort** patrimonial (stockage chiffré de documents) | vérifié | vérifié | vérifié | 0011 | AES-GCM, contrôles propriétaire, contenu et métadonnées ; téléchargement/restauration SHA256 |
| EIA-001 | S | Niveau 1 — IA locale iPhone (Apple Foundation Models) pour tâches simples/fréquentes/sensibles | — | validation externe restante | — | — | Classification et proposition typées iOS ; modèle Apple nécessite iPhone compatible |
| EIA-002 | S | Niveau 2 — IA homelab GPU pour analyses privées lourdes | validation externe restante | validation externe restante | validation externe restante | — | Contrats, fallback et états testés ; Ollama/Anthropic réels non configurés |
| EIA-003 | S | Niveau 3 — IA cloud premium (Fable 5) pour décisions complexes, données anonymisées | validation externe restante | validation externe restante | validation externe restante | — | Contrats, fallback et états testés ; Ollama/Anthropic réels non configurés |
| EIA-010 | S | Module **AI Router** décidant du niveau cible pour chaque demande | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-011 | S | Critères de routage : complexité, sensibilité, confidentialité requise, disponibilité homelab, coût cloud, temps de réponse attendu, taille du contexte, niveau de raisonnement | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-012 | C | Journalisation des décisions de routage (quel niveau, pourquoi) | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-020 | S | Fallback fluide : si niveau 1 insuffisant → « Je vais faire une analyse plus poussée. » → homelab | validation externe restante | validation externe restante | validation externe restante | — | Contrats, fallback et états testés ; Ollama/Anthropic réels non configurés |
| EIA-021 | S | Si homelab indisponible : proposer le cloud — « Analyse avancée indisponible en local. Utiliser le modèle cloud ? » | validation externe restante | validation externe restante | validation externe restante | — | Contrats, fallback et états testés ; Ollama/Anthropic réels non configurés |
| EIA-022 | M | Pour données sensibles : confirmation avant cloud — « Cette analyse nécessite un modèle externe. Les données seront anonymisées avant envoi. Continuer ? » | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-030 | M | **N1 — Local only** | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-031 | M | **N2 — Cloud anonymisé possible** | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-032 | M | **N3 — Cloud interdit par défaut** | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-033 | M | Moteur d'anonymisation appliqué automatiquement avant tout envoi cloud (N2) | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-034 | M | Classification de chaque donnée selon son niveau (N1/N2/N3) | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| EIA-040 | M | Tous les calculs financiers passent par un moteur déterministe (jamais l'IA seule) | vérifié | vérifié | vérifié | — | Calculs moteur et sélection de faits validée ; numéro inventé/texte supplémentaire refusé |
| EIA-041 | M | L'IA ne produit jamais de chiffre financier non issu du moteur | vérifié | vérifié | vérifié | — | Calculs moteur et sélection de faits validée ; numéro inventé/texte supplémentaire refusé |
| EIA-042 | M | Le moteur expose des fonctions de calcul testables unitairement (impact patrimoine, projection, fonds d'urgence, date d'indépendance…) | vérifié | vérifié | vérifié | — | Calculs moteur et sélection de faits validée ; numéro inventé/texte supplémentaire refusé |
| ENF-001 | M | UI iOS fluide (cible 60–120 fps), animations sans à-coups | — | validation externe restante | — | — | Compilation/simulateur ; mesure 60–120 fps physique nécessite iPhone/Instruments |
| ENF-002 | S | Réponse API < 300 ms pour les lectures courantes | vérifié | — | — | — | 1000 mouvements, HTTP réel local :20 lectures/route, p95≤10,84ms sur 6 routes ; pas une garantie serveur distant |
| ENF-003 | M | Données chiffrées au repos et en transit (TLS) | validation externe restante | validation externe restante | validation externe restante | 0011 | Coffre/cache protégés ; TLS et chiffrement volume dépendent de l’hôte, procédure prête |
| ENF-004 | M | Auth par profil (Face ID/code), verrouillage auto, journal d'accès | vérifié | validation externe restante | vérifié | 0011, 0013 | Tests session/cache/401/503/Keychain et verrouillage à froid ; Face ID/code appareil physique restant |
| ENF-005 | M | Aucune revente de données ; IA locale d'abord ; cloud anonymisé/opt-in | vérifié | vérifié | vérifié | — | Contrat sortie minimisé, absence traceurs, tests off/N1/N3 et historique ; pas de promesse anonymat |
| ENF-006 | M | Respect strict des niveaux N1/N2/N3 (§6.5) | vérifié | vérifié | vérifié | — | Contrat sortie minimisé, absence traceurs, tests off/N1/N3 et historique ; pas de promesse anonymat |
| ENF-007 | M | Montants en **centimes (entiers)** ou DECIMAL — jamais de float | vérifié | vérifié | vérifié | 0011, 0013, 0015, 0017, 0018, 0020 | JPY/KWD, change historique privé, principal/virement, limites entières |
| ENF-008 | S | Fonctionnement dégradé si homelab/cloud indisponibles (fallback) | vérifié | vérifié | vérifié | — | Fournisseurs absents explicites ; données moteur disponibles ; panne réseau distincte401 |
| ENF-009 | S | Export complet des données utilisateur | vérifié | vérifié | vérifié | 0011–0021 | 505/12001 opérations : sommes, IDs, manifestes/documents ; téléchargement web/iOS |
| ENF-010 | M | Code testé (unitaire sur le moteur financier), CI/CD | vérifié | vérifié | vérifié | — | Go race/vet/build/tidy, web check/unit/build, iOS signé ad hoc ; CI écrite, exécution distante restante |
| ENF-011 | C | Logs structurés, métriques, suivi des routages IA | vérifié | — | — | — | Logs structurés/routage, métriques histogrammes par route normalisée ; test fuite et concurrence |
| ENF-012 | S | Contraste, Dynamic Type, VoiceOver (iOS) | — | validation externe restante | vérifié | — | Dynamic Type/masquage AX et captures simulateur vérifiés ; VoiceOver physique complet restant |
| ENF-013 | S | RGPD léger (usage familial), pas de tiers traceurs | vérifié | vérifié | vérifié | — | Export, suppression, consentement et absence traceurs ; pas une certification juridique |
| EF-070 | M | Import manuel CSV/OFX | vérifié | vérifié | vérifié | 0011 | Fichiers invalides atomiques, Windows1252/OFX/FITID, 8 imports concurrents, ancienne ventilation ambiguë |
| EF-071 | C | Synchro bancaire automatique via GoCardless (DSP2) | validation externe restante | validation externe restante | validation externe restante | 0014, 0019 | Fournisseur HTTP simulé + SQL ; banque réelle/consentement/continuitéIDs à vérifier |
| EF-072 | M | Saisie/maj manuelle des actifs non bancaires (immo, objets, crypto, entreprise) | vérifié | vérifié | vérifié | 0011, 0018 | Création initiale atomique/idempotente, CRUD/archives, correction historique et recalcul |
| SEC-01 | M | Isolation de toutes les relations et refus atomiques | vérifié | vérifié | vérifié | 0011, 0013 | Références A/B, révocation/retrait ; tests session et confidentialité des clients |
| SEC-02 | M | Contrat cloud structuré, consentement profil, zéro texte libre | vérifié | vérifié | vérifié | — | SDK HTTP réellement intercepté ; allowlist, consentement par appel/profil, zéro texte libre/historique cloud |
| DATA-01 | M | Export cohérent complet et bilans exhaustifs 505/volume | vérifié | vérifié | vérifié | 0011–0021 | 505/12001 opérations : sommes, IDs, manifestes/documents ; téléchargement web/iOS |
| DATA-02 | M | Identité import persistante après split et concurrence | vérifié | vérifié | vérifié | 0011 | Fichiers invalides atomiques, Windows1252/OFX/FITID, 8 imports concurrents, ancienne ventilation ambiguë |
| DATA-03 | M | Solde courant, virements, principal, frais et bornes civiles | vérifié | vérifié | vérifié | 0011, 0013, 0015, 0017, 0018, 0020 | JPY/KWD, change historique privé, principal/virement, limites entières |
| SEC-03 | M | Démo isolée sans suppression par nom | vérifié | vérifié | vérifié | 0011 | Démo neuve/TTL/PIN aléatoire/nom homonyme et concurrence testés |
| OFF-01 | M | Démarrage verrouillé et lecture hors ligne autorisée | vérifié | vérifié | vérifié | 0011, 0013 | Références A/B, révocation/retrait ; tests session et confidentialité des clients |
| WEB-01 | M | Parité des tâches web avec iOS et navigation accessible | — | vérifié | vérifié | — | Recette navigation, graphique/clavier, thèmes, masquage DOM/VoiceOver ; rapports clients |
| P8-01 | M | Fiscalité et PER millésimés sources officielles | vérifié | vérifié | vérifié | 0010 | Sources officielles2026 vérifiées, barème brut/PER saisi/PFU explicites et tests exemples |
| P8-02 | M | Crédit et capacité emprunt accessibles | vérifié | vérifié | vérifié | — | Crédit, capacité et amortissement testés ; formulaires/résultats accessibles |
| P8-03 | M | Allocation cible et écarts | vérifié | vérifié | vérifié | 0010 | Allocation actuelle/objectif/écart et saisie100% ; valeurs serveur en EUR |
| P8-04 | M | Alertes personnalisées | vérifié | vérifié | vérifié | 0010, 0017 | Seuils personnalisés CRUD ; déclenchement déterministe ; notifications génériques |
| P8-05 | M | Wrapped exhaustif | vérifié | vérifié | vérifié | 0010, 0011 | Wrapped505/12001 exhaustif, sélection année et chiffres moteur |
| P8-06 | M | Cours auto et fraîcheur | vérifié | vérifié | vérifié | 0013, 0020 | BCE réelle datée du 7 octobre et CoinGecko BTC ; sous-centime, source/erreur/rafraîchissement testés |
| P8-07 | M | APNs autorisation rotation révocation destinations | validation externe restante | validation externe restante | — | 0010, 0017 | JWT/payloadHTTP/générique/profil/dédoublonnage testés ; APNs signé sur appareil reste externe |
| OPS-01 | M | Migrations neuves/existantes, sauvegarde restaurée et documents | vérifié | — | — | 0011–0021 | Migrations neuves et anciennes, refus atomiques ; restauration clone avec document déchiffré SHA-256 ; Docker restant |
| OPS-02 | M | CI Go/web/iOS et documentation reproductible | vérifié | vérifié | vérifié | — | Go race/vet/build/tidy, web check/unit/build, iOS signé ad hoc ; CI écrite, exécution distante restante |

## Migrations et conservation

| Migration | Effet |
|---|---|
|0011_integrity|Contraintes propriétaire et droits espace, dates archives, devises exactes, registre durable d’import, démo isolée, identité création atomique|
|0012_domain|Épargne affectée, calendrier, flux investissement, CCA, bénéficiaires et droits d’urgence|
|0013_security|État/fraîcheur cours, conversion native inverse, liens calendrier|
|0014_bank|Comptes fournisseur distincts, observations provisoires, état/bail/reprise persistants|
|0015_profile_fx|Taux privés par profil/date et conversions historiques autoritatives|
|0016_domain_constraints|Contraintes et index complémentaires idempotents de domaine|
|0017_movements_push|Principal lié à une dette ; registre persistant de livraisons push|
|0018_loan_principal|Capital restant dû après mouvements de principal et contrôle des références|
|0019_bank_binding|Mapping fournisseur→actif conservé après déconnexion|
|0020_quote_sources|Date de publication/source FX et vue financière actualisée|
|0021_quarterly_calendar|Fréquence trimestrielle conservée ; retour arrière refusé si des règles trimestrielles existent|

Les données historiques incohérentes entre propriétaires ou avec une échelle monétaire ambiguë bloquent la migration sans réparation silencieuse. Les migrations sont transactionnelles. La recette teste une version 10 représentative et sa migration, les mêmes totaux après migration, la répétition au redémarrage et les refus sans changement partiel. Sauvegarde/restauration restent la procédure de retour arrière documentée ; les fichiers down qui effacent des domaines ne promettent pas de retrouver leurs données.

## Décisions maintenables

- Architecture conservée : Go/PostgreSQL, SwiftUI iOS 26 et Svelte5/SvelteKit.
- Autorité des calculs côté backend, dates civiles France, mutations liées atomiques, export sur instantané SQL complet.
- iOS : lecture hors ligne d’instantanés isolés après déverrouillage local ; pas de file d’écritures hors ligne. Web : session d’onglet verrouillée à froid, serveur requis, aucune persistance de données financières.
- Cloud : schéma de sortie explicitement autorisé, agrégats minimisés toujours sensibles, consentement par demande et profil N2. Le modèle sélectionne des faits dont le serveur rend les chiffres ; aucun texte libre fournisseur n’est affiché comme fait financier validé.
- Banque : chaque compte associé explicitement, import durable, état provisoire distinct ; les variations de chaque établissement demandent une recette fournisseur réelle.
- Performance, fiscalité et projections ont des conventions et limites explicites, décrites dans [CONVENTIONS-FINANCIERES.md](CONVENTIONS-FINANCIERES.md).

## Preuves et procédures

[Guide de recette](RECETTE.md) · [Backend et sources fiscales](DELIVERY-BACKEND.md) · [iOS](../ios/DELIVERY.md) · [Web](../web/DELIVERY.md) · [Confidentialité IA](CONFIDENTIALITE-IA.md) · [Exploitation/restauration](EXPLOITATION.md).

Mesure locale complémentaire :1 000 transactions synthétiques, 20 requêtes HTTP par route après chauffe. p95 : patrimoine0,66ms ; historique12mois2,15ms ; actifs0,76ms ; transactions paginées2,55ms ; résumé mensuel2,83ms ; twin10,84ms. Cela valide la cible 300 ms sur ce poste et cette charge, sans garantir une latence réseau ou un grand serveur concurrent. Le profil de mesure a été supprimé.
