# Déployer Opale dans Coolify

Le dépôt fournit une pile API Go + PostgreSQL + web nginx. Coolify construit les deux images et termine HTTPS. Configuration fondée sur la [documentation Compose](https://coolify.io/docs/applications/builds/docker-compose) et les [domaines Coolify](https://coolify.io/docs/core/networking/domains).

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
