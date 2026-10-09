# Connexion, profils et verrouillage

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Au lancement, choisir un profil ou « Nouveau profil », saisir son nom et un code personnel, puis se connecter. La démonstration crée un profil synthétique identifié comme tel. Réglages permet de renommer le profil, changer le code et choisir sa politique IA.

Sur iPhone, le verrouillage utilise l’authentification de l’appareil. À froid, au retour d’arrière-plan et après inactivité, le contenu privé est protégé. Sur le web, le code du profil est demandé à chaque réouverture/rechargement, retour dans l’onglet et après cinq minutes d’inactivité.

## Données et comportement

`profiles` et `sessions` séparent les propriétaires. Le serveur hache le PIN et les jetons de session ; les handlers privés déduisent le profil du bearer, pas d’un identifiant fourni par l’écran. iOS conserve le jeton dans Keychain, lié à l’URL du backend, et le cache par profil/serveur. Le web garde la session dans sessionStorage, sans jeton durable dans localStorage.

Un 401 authentifié révoque l’état local. Une panne réseau ou un 503 ne détruit pas une session iOS connue : la lecture du cache peut reprendre après authentification locale. Les requêtes tardives d’un ancien profil ne doivent pas remplacer les données du nouveau.

## Limites et configuration

Le serveur est [Vaycode](serveur-vaycode.md), sans champ de sélection. Le web exige une connexion pour déverrouiller et écrire. iOS n’a pas de file d’écritures hors ligne ; sans authentification appareil disponible, le PIN du profil exige le serveur. La liste publique des profils expose leur nom : ce modèle familial n’est pas une inscription publique multi-tenant anonyme.

## Vérification

`ios/OpaleTests/SessionIsolationTests.swift` couvre réseau indisponible, 503, 401, jeton capturé et cache étranger. `web/tests/api.test.mjs` et `web/tests/e2e/privacy.spec.ts` couvrent isolation, verrouillage et déconnexion. Les contrôles réels Face ID/VoiceOver restent une recette sur appareil ; voir [livraison iOS](../DELIVERY-IOS.md).

## API et sources

- `GET /v1/access-log`
- `GET /v1/me`
- `GET /v1/profiles`
- `PATCH /v1/me`
- `POST /v1/auth/login`
- `POST /v1/auth/logout`
- `POST /v1/profiles`
- `POST /v1/profiles/demo`

- [backend/internal/api/crud.go](../../backend/internal/api/crud.go)
- [backend/internal/api/export.go](../../backend/internal/api/export.go)
- [backend/internal/api/pilote.go](../../backend/internal/api/pilote.go)
- [backend/internal/api/profiles.go](../../backend/internal/api/profiles.go)
- [ios/Opale/Core/AppLock.swift](../../ios/Opale/Core/AppLock.swift)
- [ios/Opale/Core/DiskCache.swift](../../ios/Opale/Core/DiskCache.swift)
- [ios/Opale/Core/Keychain.swift](../../ios/Opale/Core/Keychain.swift)
- [ios/Opale/Features/Assistant/SecuritySheets.swift](../../ios/Opale/Features/Assistant/SecuritySheets.swift)
- [ios/Opale/Features/Auth/ProfileGateView.swift](../../ios/Opale/Features/Auth/ProfileGateView.swift)
- [ios/Opale/Features/Settings/ProfileSettingsView.swift](../../ios/Opale/Features/Settings/ProfileSettingsView.swift)
- [web/src/lib/session.svelte.ts](../../web/src/lib/session.svelte.ts)
- [web/src/routes/login/+page.svelte](../../web/src/routes/login/+page.svelte)
