# Opale

Gestion de patrimoine personnelle, auto-hébergée : API Go/PostgreSQL, application iOS SwiftUI et web SvelteKit. Les cinq espaces sont **Accueil, Flux, Patrimoine, Projection, Assistant**. Le moteur réalise les calculs ; l’IA facultative commente des faits contrôlés.

## Démarrer

Prérequis : Docker + Compose v2. Copier `.env.example` en `.env`, choisir un mot de passe de base unique ; configurer et sauvegarder séparément `OPALE_VAULT_KEY` pour activer le coffre.

```bash
cp .env.example .env
# Modifier .env avant le lancement
docker compose up -d --build --wait
curl --fail http://127.0.0.1:3000/readyz
```

Ouvrir [Opale local](http://127.0.0.1:3000), créer un profil et choisir son code personnel. Les ports sont limités à la boucle locale. Pour un accès externe, mettre le web derrière un reverse proxy HTTPS ; voir [exploitation](docs/EXPLOITATION.md).

Pour Coolify : [guide de déploiement](docs/COOLIFY.md). Utiliser uniquement `docker-compose.yml` et associer le domaine au service `web`, port interne `80`. Les ports du poste sont définis séparément dans `docker-compose.override.yml`, chargé automatiquement par les commandes locales sans `-f`.

En développement natif : Go 1.26+, PostgreSQL17, Node22. `make db` démarre PostgreSQL ; charger les variables de `.env` dans l’environnement du terminal avant `make run`. Dans `web/`, `npm ci` puis `npm run dev`. Vite proxifie vers `http://localhost:8080` ; `OPALE_API_URL` permet de cibler une API de test. Pour iOS, générer le projet avec `xcodegen generate --spec ios/project.yml`, puis ouvrir `ios/Opale.xcodeproj` dans Xcode26.

## Fonctionnalités

| Espace | Parcours disponibles |
|---|---|
| Accueil | Patrimoine net et historique, trésorerie, indicateurs, alertes, jalons |
| Flux | Saisie/correction/suppression, virements atomiques, capital remboursé lié à une dette, imports CSV/OFX, pagination/filtres, catégories/règles, budgets, récurrences/calendrier, partage explicite, Wrapped |
| Patrimoine | Actifs/dettes/valorisations, devises, immobilier, investissements et flux, objets, société/CCA, coffre chiffré, contacts/transmission, banque et cotations facultatives |
| Projection | FIRE, inflation, objectifs, scénarios comparés, arbitrages achat/location et crédit/investissement, crédits, fiscalité/PER documentés |
| Assistant | Conversation, états de calcul et sources, bilan mensuel, jumeau patrimonial, cloud avec consentement explicite lorsque autorisé |

Sur iPhone, les nouveaux profils disposent d’un formulaire en six étapes : revenu net, compte principal et solde facultatif, charges fixes, abonnements, budget variable et objectif, puis récapitulatif. Le brouillon se reprend dans **Réglages → Ma situation de départ**. La validation crée les échéances prévisionnelles et l’objectif sans inventer de transactions réalisées. Les profils déjà renseignés peuvent ouvrir ce parcours depuis les Réglages ; la démonstration en est exclue. Le budget variable sert uniquement à l’estimation initiale. Cette fonction nécessite les migrations backend `0022` et `0023`, appliquées automatiquement au démarrage de l’API après redéploiement.

Les intégrations banque, IA externe, cotations et APNs nécessitent configuration et recette avec leur fournisseur. Le code et ses tests ne prouvent pas la réception d’une notification sur appareil réel ou la synchronisation d’un compte bancaire réel. Les limites sont détaillées dans les documents de livraison.

## Données et confidentialité

Les champs API historiques `*_cents` contiennent des **unités mineures entières** : EUR/USD deux décimales, JPY zéro, KWD trois. Les totaux agrégés sont convertis en EUR ; les montants individuels conservent leur devise. Les données sont cloisonnées par profil, avec partage explicite. Le coffre utilise AES-256-GCM. La clé ne réside pas dans le dump de base et doit être conservée séparément.

Le web conserve le jeton uniquement pendant la session d’onglet, verrouille à froid et après inactivité, et retire les données visibles en mode discret. Il n’offre pas d’édition hors ligne. L’IA cloud ne reçoit pas de question libre ou d’historique complet ; elle utilise des intentions et faits minimisés sélectionnés par le serveur. La minimisation ne garantit pas une anonymisation absolue.

## Vérifier

```bash
make test
make vet
cd web
npm ci
npm run check
npm test
npm run build
# Avec le serveur Vite démarré dans un autre terminal :
npx playwright install chromium
npm run test:e2e
```

`npm run test:e2e:integration` nécessite une **API et une base jetables** derrière le proxy Vite ou nginx (coffre configuré) ; ces tests créent puis suppriment leurs profils. Ne jamais utiliser une instance personnelle. Le workflow CI configure Go/race avec PostgreSQL, Svelte/check/build, tests navigateur et pile Compose, sauvegarde/restauration isolée, tests unitaires iOS sur simulateur signé localement. Les résultats distants sont disponibles dans [GitHub Actions](https://github.com/AzmogEx/Opale/actions) ; les vérifications locales et leurs limites figurent dans la recette.

## Documents et organisation

- [Matrice complète de livraison](docs/LIVRAISON.md), [recette reproductible](docs/RECETTE.md), [conventions financières](docs/CONVENTIONS-FINANCIERES.md), [confidentialité IA](docs/CONFIDENTIALITE-IA.md).
- [Cahier des charges](docs/CAHIER-DES-CHARGES.md), [conception](docs/CONCEPTION.md), [modèle de données](docs/DATA-MODEL.md).
- [Livraison web et preuves](web/DELIVERY.md), [livraison iOS et preuves](ios/DELIVERY.md), [livraison backend](docs/DELIVERY-BACKEND.md), [guide web](web/README.md).
- [Installation, migration, sauvegarde, restauration, secrets et supervision](docs/EXPLOITATION.md).
- `backend/` : API et moteur déterministe ; `web/` : SPA ; `ios/` : SwiftUI ; `scripts/` : exploitation ; `.github/workflows/ci.yml` : vérifications.

Sauvegarder avec `./scripts/backup.sh /chemin/protege`. Restaurer exclusivement vers une nouvelle base avec `./scripts/restore.sh fichier.dump nouvelle_base`. Tester périodiquement un téléchargement de document après restauration avec la même clé de coffre.
