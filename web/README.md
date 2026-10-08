# Opale Web

Svelte5/SvelteKit, TypeScript, application statique servie par nginx. Cinq onglets ; formulaires, données et calculs utilisent la véritable API Go.

```bash
npm ci
# API par défaut localhost:8080 ; fournir une autre URL seulement pour le proxy de développement.
OPALE_API_URL=http://127.0.0.1:8080 npm run dev
npm run check
npm test
npm run build
```

`build/` est généré par adapter-static ; le serveur doit renvoyer `index.html` pour les routes SPA, et proxifier `/v1` vers Go sans cache. Le Dockerfile et `nginx.conf` assurent ces règles.

## Recette navigateur

Lancer Vite avant les tests. Installer Chromium par `npx playwright install chromium`, puis `npm run test:e2e`. Une installation Chrome existante peut être utilisée via `OPALE_CHROMIUM_PATH=/chemin/vers/chrome`. `OPALE_WEB_URL` change l’origine cible (défaut `http://127.0.0.1:5173`).

- `npm test` : unités monétaires entières, JPY/KWD, mois civils/été-hiver, dates FIRE, paramètres de pagination, expiration et isolation des sessions.
- `npm run test:e2e` : API simulée pour confidentialité du DOM, écran mobile, verrouillage après inactivité et expiration confirmée.
- `npm run test:e2e:integration` : API/PostgreSQL réels **jetables** avec coffre actif. Création de profils de test, saisie atomique, import55 opérations/pagination/édition/export, virement lié, objectifs, coffre et fiscalité. Les profils propres au test sont supprimés à la fin ; ne jamais utiliser une base personnelle.

## Choix de confidentialité

Pas de données financières persistées dans localStorage, pas de service worker ni d’édition hors ligne. Session d’onglet dans sessionStorage ; rechargement, onglet masqué ou cinq minutes sans activité verrouillent l’interface. Le serveur est requis pour déverrouiller. Les réponses d’un ancien profil sont ignorées. Le mode discret retire le contenu privé du DOM, y compris les textes narratifs. Historique assistant limité en mémoire et vidé lors du changement de profil.

Les formulaires d’argent utilisent des chaînes décimales puis des entiers, avec l’exposant de la devise. Les résultats agrégés viennent du moteur en EUR. Une fonctionnalité externe indisponible affiche un état explicite ; aucune réponse bancaire ou IA n’est simulée dans l’application.

Voir [DELIVERY.md](DELIVERY.md) et [guide d’exploitation](../docs/EXPLOITATION.md).
