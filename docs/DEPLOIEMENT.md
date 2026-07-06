# Déploiement homelab — guide pas à pas

Objectif : Opale en production chez toi (Coolify), avec TLS, sauvegardes,
et la cascade IA branchée sur la RTX 5080.

## 1. Prérequis

- Homelab avec Docker + [Coolify](https://coolify.io) (déjà en place)
- Un domaine (ou sous-domaine) pointant vers le homelab, ex. `opale.mondomaine.fr`
- Le dépôt Git accessible depuis Coolify (GitHub `AzmogEx/Opale`)

## 2. Secrets à générer AVANT le premier démarrage

```bash
openssl rand -hex 32     # → OPALE_VAULT_KEY (coffre-fort)
openssl rand -hex 24     # → OPALE_DB_PASSWORD
```

> ⚠️ **OPALE_VAULT_KEY** : à stocker dans ton gestionnaire de mots de passe.
> Clé perdue = documents du coffre définitivement illisibles (y compris dans
> les sauvegardes).

## 3. Coolify

1. **Nouvelle ressource → Docker Compose**, pointe sur le dépôt (le
   `docker-compose.yml` à la racine décrit `db` + `api`).
2. Renseigne les variables d'environnement depuis `.env.example` :
   - obligatoires : `OPALE_DB_PASSWORD`, `OPALE_VAULT_KEY`, `OPALE_ENV=prod`,
     `OPALE_DB_SSLMODE=disable` (réseau interne Docker) ;
   - cascade IA : `OPALE_OLLAMA_URL=http://<ip-du-gpu>:11434` (voir §5) ;
   - cloud (optionnel) : `OPALE_ANTHROPIC_API_KEY` ;
   - banque (optionnel) : `OPALE_GC_SECRET_ID/KEY` (portail GoCardless →
     « User secrets »).
3. **Domaine + TLS** : attache `opale.mondomaine.fr` au service `api` ;
   Coolify provisionne Let's Encrypt (ENF-003 : le TLS se termine ici).
4. Vérifie : `curl https://opale.mondomaine.fr/readyz` → `{"status":"ready"}`
   et dans les logs : `coffre-fort activé`, `migrations appliquées`.

## 4. Sauvegardes (critique)

```bash
# Test manuel :
OPALE_DB_CONTAINER=<nom-du-conteneur-db> ./scripts/backup.sh /backups/opale
# Cron quotidien (3h du matin, rotation 14 jours) :
0 3 * * * OPALE_DB_CONTAINER=<nom> /opt/opale/scripts/backup.sh /backups/opale
```

Idéalement : réplique le dossier `/backups/opale` hors du homelab
(rclone vers un stockage chiffré, disque externe…). Test de restauration :
`gunzip -c dump.sql.gz | docker exec -i <db> pg_restore -U opale -d opale_restore`.

## 5. Cascade IA — N2 (RTX 5080)

Sur la machine GPU :

```bash
# Ollama expose son API sur le réseau local :
OLLAMA_HOST=0.0.0.0 ollama serve
ollama pull llama3.1:8b        # ou un modèle plus costaud (qwen2.5:14b…)
```

Puis `OPALE_OLLAMA_URL=http://<ip-gpu>:11434` côté API. Vérification :
l'app iOS → Assistant → menu ⋯ → « Homelab en ligne », et les réponses
portent le badge « Homelab — privé ».

## 6. iOS en production

Dans l'app : porte de profils → champ serveur → `https://opale.mondomaine.fr`.
(ATS exige HTTPS hors réseau local — c'est déjà le cas via Coolify.)

## 7. Check-list de mise en service

- [ ] `/readyz` répond en HTTPS
- [ ] `OPALE_VAULT_KEY` sauvegardée hors homelab
- [ ] Cron de backup actif + un dump testé en restauration
- [ ] Brute-force vérifié : 5 PIN faux → HTTP 429 (verrou 15 min)
- [ ] Ollama joignable depuis le conteneur API (`homelab_available: true`)
- [ ] Export ZIP testé depuis l'app (menu Assistant)
- [ ] Widget ajouté à l'écran d'accueil de l'iPhone (validation visuelle)
