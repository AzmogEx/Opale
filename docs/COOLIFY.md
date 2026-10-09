# Déployer Opale dans Coolify

Le dépôt fournit une pile API Go + PostgreSQL + web nginx. Coolify construit les deux images et termine HTTPS. Configuration fondée sur la [documentation Compose](https://coolify.io/docs/applications/builds/docker-compose) et les [domaines Coolify](https://coolify.io/docs/core/networking/domains).

Le backend des clients est imposé sur **https://opale.vaycode.com** ; aucun champ de serveur dans l’app. Un push sur `main` déclenche automatiquement le déploiement. Les fournisseurs IA sont joints par l’API Vaycode, sans changer cette destination.

## Configuration

1. Créer une application depuis le dépôt Git public `https://github.com/AzmogEx/Opale`, branche `main`, avec le build pack **Docker Compose**.
2. Base Directory : `/`. Docker Compose Location : `/docker-compose.yml`. Charger/recharger le fichier depuis Git. Utiliser le déploiement Compose normal, qui génère les routes du proxy.
3. Dans les domaines, sélectionner **web**, protocole **HTTPS**, domaine **opale.vaycode.com**, port interne **80**, sans préfixe de chemin. Si l'interface attend une URL complète, saisir `https://opale.vaycode.com:80` ; `80` désigne le port du conteneur, le navigateur accède toujours en HTTPS sur 443.
4. Renseigner les variables d'environnement ci-dessous, puis déployer. Aucun argument de build n'est nécessaire ; les secrets doivent être des variables d'exécution.

| Variable | Valeur |
|---|---|
| `OPALE_DB_PASSWORD` | Mot de passe PostgreSQL unique, conservé dans Coolify |
| `OPALE_DB_USER` | `opale` (défaut) |
| `OPALE_DB_NAME` | `opale` (défaut) |
| `OPALE_ENV` | `prod` |
| `OPALE_VAULT_KEY` | Pour activer le coffre : 64 caractères hexadécimaux, générés avec `openssl rand -hex 32`, sauvegardés séparément |
| `OPALE_CLOUD_AI` | `off` par défaut |
| `OPALE_AUTO_QUOTES` | `off` par défaut |

Les autres fournisseurs restent facultatifs ; voir [exploitation](EXPLOITATION.md). Conserver le mot de passe PostgreSQL déjà utilisé si la base existe : modifier la variable ne change pas le mot de passe du rôle dans un volume initialisé. Conserver également la clé du coffre pour relire les documents existants.

## Corriger « port is already allocated »

La pile `docker-compose.yml` ne contient plus de `ports:`. PostgreSQL et l'API communiquent sur le réseau interne ; nginx transmet `/v1/` à `api:8080`. Seul le service `web:80` reçoit le trafic du proxy Coolify.

Pour mettre à jour une ressource existante : recharger le Compose depuis `main`, vérifier l'absence de correspondances de ports hôte dans la configuration générée et redéployer. **Ne pas ajouter `docker-compose.override.yml`** dans Coolify : il est réservé au poste local. Les variables `OPALE_API_PORT`, `OPALE_WEB_PORT` et `OPALE_DB_PORT` n'ont aucun effet sur la pile serveur. Le volume nommé `opale_db` reste inchangé ; conserver la même ressource Coolify pour retrouver sa base.

## Vérifier

Le DNS doit viser le serveur Coolify ; ses ports 80 et 443 doivent être accessibles pour HTTPS. Après déploiement, les trois services `db`, `api` et `web` doivent être sains et les journaux ne doivent montrer aucune erreur de migration.

Les healthchecks sont définis dans le Compose : PostgreSQL utilise `pg_isready`, l'API interroge son `/readyz` avec un binaire Go embarqué compatible avec l'image distroless, et le web interroge `/readyz` via nginx. L'API attend PostgreSQL sain ; le web attend l'API saine. La sonde API s'exécute toutes les 10 secondes, avec 4 secondes de délai HTTP, 5 secondes de limite Docker, 3 échecs consécutifs et 60 secondes de tolérance initiale pour les migrations. Recharger le Compose depuis Git et reconstruire/redéployer pour obtenir le nouveau binaire et ses sondes. Aucun champ Healthcheck supplémentaire n'est nécessaire dans l'interface Coolify pour cette pile Compose.

```sh
curl --fail https://opale.vaycode.com/healthz
curl --fail https://opale.vaycode.com/readyz
```

Ouvrir le site, créer un profil puis vérifier verrouillage, reconnexion, saisie et export. Sur iPhone, ouvrir cette URL dans Safari ; ce parcours web ne nécessite aucune équipe Apple. L'installation de l'application SwiftUI sur un iPhone physique nécessite une signature Apple valide. Le projet iOS n'impose aucune équipe ; le simulateur fonctionne avec la signature locale indiquée dans [la livraison iOS](../ios/DELIVERY.md).

Si le domaine renvoie 503, vérifier en priorité le service `web`, sa sonde `/readyz`, le domaine associé à `web` et son port interne 80. Un certificat `TRAEFIK DEFAULT CERT` indique que le certificat du domaine n'est pas encore correctement servi : contrôler DNS, route HTTPS et journaux ACME du proxy.

Sauvegarder PostgreSQL et la clé du coffre avant les mises à jour ; voir [les scripts et la restauration isolée](EXPLOITATION.md#sauvegarder-et-restaurer). Depuis un terminal qui dispose du dépôt et de Docker sur le serveur, les scripts peuvent cibler le conteneur PostgreSQL Coolify via `OPALE_DB_CONTAINER` et les noms de rôle/base correspondants. Aucun accès SSH depuis Codex n'est nécessaire pour déployer dans l'interface Coolify.

## Choisir l’IA : PC Windows ou cloud

L’application iOS propose **Assistant → Choisir ou connecter mon IA** (également dans Réglages). Le choix est propre au profil : Automatique essaie le PC privé puis revient au moteur ; Mon PC ne cascade jamais vers le cloud ; Cloud demande un accord à chaque envoi. L’iPhone conserve la lecture locale de documents et les fonctions locales, sans intercepter le chat avec son petit modèle. Une réponse du moteur/guide reste disponible sans fournisseur. Le statut Ollama exige que **le modèle demandé figure dans `/api/tags`**, pas seulement un serveur HTTP qui répond.

### Option PC Windows avec NVIDIA

Sur Windows, installer [Ollama](https://docs.ollama.com/windows), mettre à jour le pilote NVIDIA et vérifier la mémoire disponible avec `nvidia-smi`. Un premier modèle à essayer est [Qwen3.5 9B](https://ollama.com/library/qwen3.5:9b) ; ce point de départ n’est pas un benchmark de la carte personnelle. Dans PowerShell :

```powershell
ollama pull qwen3.5:9b
ollama run qwen3.5:9b
ollama ps
Invoke-RestMethod http://127.0.0.1:11434/api/tags
```

Vérifier dans `ollama ps` que le modèle utilise le GPU, puis sa vitesse sur une demande synthétique. Conserver l’écoute Ollama sur localhost. Le PC doit rester allumé ; sa mise en veille rend l’IA indisponible.

Coolify étant sur un serveur distant, une adresse `192.168.*` du domicile ne suffit pas. Installer Tailscale sur le PC et le serveur distant, dans le même réseau privé, et limiter les droits d’accès au serveur API. Dans un terminal Windows administrateur, [Tailscale Serve](https://tailscale.com/docs/reference/tailscale-cli/serve) peut publier Ollama **uniquement dans le réseau privé** :

```powershell
tailscale serve --bg http://127.0.0.1:11434
tailscale serve status
```

Reporter l’URL HTTPS réellement affichée dans les variables **d’exécution du service API** de Coolify :

```dotenv
OPALE_OLLAMA_URL=https://nom-reel-du-pc.nom-reel-du-tailnet.ts.net
OPALE_OLLAMA_MODEL=qwen3.5:9b
OPALE_CLOUD_AI=off
```

Ne pas utiliser Tailscale Funnel ni ouvrir le port 11434 sur Internet. Un proxy privé qui exige un jeton Bearer peut recevoir `OPALE_OLLAMA_API_KEY` ; cette variable est facultative avec un accès privé correctement limité. Ne pas la confondre avec une clé de cloud Ollama.

**La connexion doit fonctionner depuis le conteneur API**, pas seulement depuis l’hôte Coolify. Vérifier routage Docker vers le réseau Tailscale et résolution du nom privé. L’image API distroless ne contient pas de shell/curl : un outil de diagnostic temporaire dans le même réseau Docker permet de lire `/api/tags`, puis le bouton « Vérifier la connexion » dans l’app contrôle le vrai client backend. Aucun port hôte supplémentaire n’est nécessaire dans le Compose Opale.

### Option cloud Claude

Dans les secrets/variables d’exécution Coolify du service API :

```dotenv
OPALE_ANTHROPIC_API_KEY=<cle-secrete-du-compte>
OPALE_ANTHROPIC_MODEL=claude-sonnet-5-5
OPALE_CLOUD_AI=on
```

Le modèle est configurable ; vérifier l’accès et le coût dans la [documentation officielle Claude](https://platform.claude.com/docs/en/models/overview) et le compte fournisseur. Ne jamais committer la clé ni la saisir dans l’app. Redéployer la configuration, puis dans l’app ouvrir **Choisir mon IA → Autorisation et confidentialité de mon profil**, choisir « Cloud possible, avec consentement », enregistrer et sélectionner Cloud ou Automatique. L’activation serveur et l’autorisation du profil ne déclenchent aucun envoi à elles seules. Une clé configurée ne prouve ni un crédit disponible ni un accès au modèle ; une erreur déclenche le repli explicite.

Le cloud reçoit uniquement une intention reconnue et les agrégats minimisés autorisés. La question libre, l’historique, les noms et les documents ne lui sont pas transmis. Son périmètre reste donc limité ; il ne s’agit pas d’un chat généraliste avec accès à tous les documents. Voir [le contrat IA](CONFIDENTIALITE-IA.md). Faire la première recette sur un profil synthétique et contrôler consentement, disponibilité, refus et retour au moteur.

Le push sur `main` déclenche automatiquement le déploiement de cette ressource. La migration `0025` ajoute la progression du parcours ; elle conserve les données financières et reprend le budget de l’ancien formulaire terminé.
