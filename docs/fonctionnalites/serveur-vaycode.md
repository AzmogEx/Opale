# Backend Vaycode imposé et migration des anciennes adresses

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Sur iPhone et web, l’application se connecte automatiquement à `https://opale.vaycode.com`. Aucun formulaire de serveur n’est proposé, y compris lors d’une panne. « Réessayer » relance le même service ; Réglages iOS affiche seulement son adresse en lecture seule.

## Données et comportement

iOS définit la destination dans `AppBackend`, la session la conserve immuable et les notifications de fond utilisent cette même source. L’ancienne préférence `opale.baseURL` est supprimée sans être utilisée. Aucun fallback vers localhost n’existe sur iPhone. Un jeton Keychain lié à un autre serveur reste inutilisable : se reconnecter à un profil Vaycode, sans transférer de données privées d’une instance étrangère.

Le web définit une adresse constante dans `session.svelte.ts`. Les anciennes préférences sont retirées ; une ancienne adresse étrangère invalide la session d’onglet avant toute requête. Les nouvelles sessions web portent aussi une liaison explicite `opale.server` ; une session ancienne sans liaison n’est conservée que si son origine effective était Vaycode. Une session Vaycode reste soumise au verrouillage habituel. Il n’existe ni paramètre public d’URL, ni sélection via query string ou variable de build.

Les endpoints des fournisseurs IA/banque sont des paramètres d’exploitation du backend et ne changent pas le serveur utilisé par l’app.

## Limites et configuration

Seule exception compilée en Debug **sur simulateur iOS** : `--base-url http://localhost:58088` (ou boucle locale équivalente, port explicite, sans chemin/identifiants/query). Elle n’est pas compilée sur iPhone ni en Release. Le web livré reste constant même en recette ; les fixtures Playwright interceptent le domaine et relaient uniquement vers l’API jetable en boucle locale.

Une modification d’hébergement/domaine exige un changement de code et une livraison. Le DNS/TLS et le proxy Coolify du domaine doivent fonctionner. Ce verrou fixe la destination choisie par l’app ; il ne protège pas contre un appareil, navigateur ou DNS administré de manière hostile.

## Vérification

`SessionIsolationTests` vérifie préférence étrangère ignorée, destination HTTPS Vaycode des requêtes publiques/privées, refus d’un jeton étranger et garde-fou simulateur. `web/tests/e2e/backend.spec.ts` vérifie URL forcée, effacement de session étrangère et absence d’input serveur. Les quatre parcours API réels passent par `tests/integration/fixtures.ts`, jamais par la production.

## API et sources

Pas de route HTTP propre à ce module ; les opérations passent par les modules associés ou restent locales.

- [ios/Opale/Core/AppBackend.swift](../../ios/Opale/Core/AppBackend.swift)
- [ios/Opale/Core/NotificationManager.swift](../../ios/Opale/Core/NotificationManager.swift)
- [ios/Opale/Core/SessionStore.swift](../../ios/Opale/Core/SessionStore.swift)
- [ios/OpaleTests/SessionIsolationTests.swift](../../ios/OpaleTests/SessionIsolationTests.swift)
- [web/src/lib/session.svelte.ts](../../web/src/lib/session.svelte.ts)
- [web/tests/e2e/backend.spec.ts](../../web/tests/e2e/backend.spec.ts)
- [web/tests/integration/fixtures.ts](../../web/tests/integration/fixtures.ts)
