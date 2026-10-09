# Déployer le service Opale de Vaycode

État du 9 octobre 2026. Les applications livrées utilisent exclusivement **https://opale.vaycode.com** ; aucun utilisateur ne choisit son backend. Le serveur est distant, géré dans Coolify ; le PC Windows à domicile peut seulement fournir l’inférence IA privée. Un push sur `main` déclenche automatiquement la production.

## Source et configuration Coolify

- Dépôt : [AzmogEx/Opale](https://github.com/AzmogEx/Opale), branche `main`.
- Build : Docker Compose ; Base Directory `/`, Docker Compose Location `/docker-compose.yml`.
- Services : PostgreSQL `db`, API Go `api`, SPA/nginx `web`.
- Domaine : `https://opale.vaycode.com` sur **web**, port interne **80**, HTTPS public 443. Nginx relaie `/v1` vers `api:8080`.
- Ne pas charger `docker-compose.override.yml` : il sert seulement aux ports de boucle locale du poste de développement. La pile Coolify ne publie aucun port hôte, ce qui évite le conflit sur 8080.

Le [guide Coolify](COOLIFY.md) détaille les champs de l’interface, les variables et le diagnostic de disponibilité.

## Secrets et données existantes

Générer une clé du coffre avec `openssl rand -hex 32` et un mot de passe PostgreSQL unique. Les stocker dans Coolify, jamais dans Git ou l’app. La clé du coffre doit aussi être sauvegardée séparément : une nouvelle clé ne déchiffre pas les anciens documents.

Sur une ressource existante, conserver volume, nom de projet, mot de passe du rôle PostgreSQL et clé du coffre. Changer la variable de mot de passe ne change pas un rôle déjà créé dans le volume. L’API applique ses migrations au démarrage ; 0024 ajoute contrats/revenus et 0025 le parcours, sans remise à zéro des profils.

## Disponibilité et contrôle après livraison

PostgreSQL : `pg_isready`. API : exécutable `/healthcheck`, qui interroge `/readyz` et la connexion PostgreSQL. Web : requête `/readyz` à travers nginx. Le démarrage attend les dépendances saines.

```bash
curl --fail https://opale.vaycode.com/healthz
curl --fail https://opale.vaycode.com/readyz
```

Vérifier les trois services sains dans Coolify et l’absence d’erreur de migration. Ces sondes ne prouvent pas le fonctionnement d’Ollama, d’une banque ou d’APNs ; leurs recettes sont séparées. Les métriques `/metrics` restent sur le réseau API privé et sont bloquées par nginx public.

## IA et intégrations facultatives

[PC Windows/Tailscale/Ollama et cloud Claude](COOLIFY.md#choisir-lia--pc-windows-ou-cloud) sont deux options indépendantes. Le serveur Vaycode joint le fournisseur, l’iPhone ne choisit pas un serveur alternatif. Le cloud exige aussi politique du profil et consentement par demande. Les clés restent dans les secrets API.

Banque, cours, coffre et APNs : variables et limites dans [EXPLOITATION.md](EXPLOITATION.md). Ne pas présenter une configuration renseignée comme une intégration testée avec le fournisseur.

## Sauvegarde et restauration

Sur la machine qui possède l’accès au conteneur PostgreSQL, `scripts/backup.sh` crée un dump ; `scripts/restore.sh fichier.dump nouvelle_base` restaure exclusivement dans une **nouvelle base** et révoque les sessions du clone. Garder la clé du coffre séparément et vérifier un téléchargement/déchiffrement après restauration. Commandes complètes, modes Docker/natif et preuves dans [EXPLOITATION.md](EXPLOITATION.md#sauvegarder-et-restaurer).

## Livraison iPhone et recette locale

La destination iOS est fixée dans `AppBackend`. Installer une version signée valide sur l’appareil, puis ouvrir un profil Vaycode ; aucun champ serveur à renseigner. Les anciennes adresses sont ignorées et les jetons étrangers ne sont pas transmis.

Pour les tests qui écrivent, utiliser une base et API jetables. La dérogation iOS de boucle locale n’est compilée qu’en Debug sur simulateur ; Playwright intercepte la destination Vaycode dans le seul navigateur de recette. Détails : [backend imposé](fonctionnalites/serveur-vaycode.md), [recette](RECETTE.md) et [livraison iOS](DELIVERY-IOS.md).
