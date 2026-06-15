# Opale

Application personnelle de **gestion de patrimoine** — *belle, intelligente et
privée*, auto-hébergée. Pas « où est parti mon argent ce mois-ci », mais
**« combien je vaux, et où je vais »**.

> Documents de référence : [`CAHIER-DES-CHARGES.md`](CAHIER-DES-CHARGES.md) ·
> [`CONCEPTION.md`](CONCEPTION.md) · [`docs/DATA-MODEL.md`](docs/DATA-MODEL.md)

## État d'avancement

| Palier | Contenu | Statut |
|---|---|---|
| **P0 — Fondations** | Repo, backend Go + Postgres, modèle de données, auth profils, CI/CD | 🟢 en cours (ce socle) |
| P1 — Patrimoine net | Accueil, saisie actifs/dettes, graphe, design system | ⚪ à venir |
| P2 → P7 | Projection, flux, pilotage, cerveau IA, profondeur, confort | ⚪ à venir |

### Ce qui est posé (P0)

- **Backend Go** (`backend/`) : API REST, logs structurés, arrêt gracieux.
- **PostgreSQL** : modèle de données du cœur patrimoine (profils, actifs, passifs,
  valorisations) + migrations embarquées appliquées au démarrage.
- **Moteur monétaire** (`internal/money`) : montants en **centimes entiers**
  (règle d'or ENF-007), testé unitairement (CA-2).
- **Auth multi-profil** : profils cloisonnés, code (PIN) haché bcrypt, sessions
  par jeton opaque (EF-001/002).
- **Patrimoine net** : calcul déterministe actifs − passifs (CA-1).
- **CI** GitHub Actions + **Docker / docker-compose** pour le homelab.

## Architecture

```
iOS SwiftUI (à venir)   Web (à venir)
            \           /
          API REST (Go) ── Backend Go + PostgreSQL ── Moteur financier déterministe
                                  │
                            AI Router (P5) ── iPhone / Homelab GPU / Cloud
```

Stack tranchée : **iOS SwiftUI**, **backend Go**, **PostgreSQL**, IA hybride en
cascade (paliers ultérieurs). Framework web encore ouvert (React/Svelte).

## Structure du dépôt

```
.
├── backend/                  # API Go
│   ├── cmd/api/              # point d'entrée
│   └── internal/
│       ├── config/           # configuration (env)
│       ├── money/            # type monétaire (centimes) + tests
│       ├── migrations/       # SQL embarqué
│       ├── store/            # accès PostgreSQL (pool, migrations, requêtes)
│       ├── auth/             # hachage PIN + jetons de session
│       └── api/              # serveur HTTP, middlewares, handlers
├── docs/DATA-MODEL.md        # modèle de données détaillé
├── docker-compose.yml        # Postgres + API (dev / homelab)
├── Makefile                  # raccourcis (make help)
└── .github/workflows/ci.yml  # CI
```

## Démarrage rapide

Prérequis : **Go 1.26+**, **Docker**.

```bash
cp .env.example .env          # ajuster si besoin

# Option A — tout via Docker
make up                       # db + api sur http://localhost:8080

# Option B — Postgres en Docker, API en local
make db                       # démarre PostgreSQL
make run                      # lance l'API (applique les migrations au démarrage)
```

Vérifier :

```bash
curl localhost:8080/healthz   # {"status":"ok"}
curl localhost:8080/readyz    # {"status":"ready"} si la base répond
```

Commandes utiles : `make help` · `make test` · `make vet` · `make ci`.

## API (P0)

Toutes les réponses sont en JSON. Les montants sont exprimés en **centimes**
(champs `*_cents`).

### Public

| Méthode | Route | Rôle |
|---|---|---|
| `GET` | `/healthz`, `/readyz` | Sondes liveness / readiness |
| `GET` | `/v1/profiles` | Liste des profils (écran de sélection) |
| `POST` | `/v1/profiles` | Créer un profil `{name, pin, privacy_default?}` |
| `POST` | `/v1/auth/login` | Connexion `{profile_id, pin}` → `{token, expires_at, profile}` |

### Authentifié (`Authorization: Bearer <token>` ou `X-Session-Token: <token>`)

| Méthode | Route | Rôle |
|---|---|---|
| `POST` | `/v1/auth/logout` | Révoque la session |
| `GET` | `/v1/me` | Profil courant |
| `GET` | `/v1/net-worth` | Patrimoine net (actifs − passifs) |
| `GET/POST` | `/v1/assets` | Lister / créer un actif |
| `GET/PATCH/DELETE` | `/v1/assets/{id}` | Détail / mise à jour / suppression |
| `GET/POST` | `/v1/assets/{id}/valuations` | Historique / ajout de valorisation |
| `GET/POST` | `/v1/liabilities` | Lister / créer un passif |
| `GET/PATCH/DELETE` | `/v1/liabilities/{id}` | Détail / mise à jour / suppression |
| `GET/POST` | `/v1/liabilities/{id}/valuations` | Historique / ajout de valorisation |

### Exemple

```bash
# Créer un profil
curl -s localhost:8080/v1/profiles -d '{"name":"Adam","pin":"1234"}'

# Se connecter
TOKEN=$(curl -s localhost:8080/v1/auth/login \
  -d '{"profile_id":"<id>","pin":"1234"}' | jq -r .token)

# Ajouter un actif puis sa valorisation (120 000,00 €)
AID=$(curl -s localhost:8080/v1/assets -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Livret A","kind":"savings"}' | jq -r .id)
curl -s localhost:8080/v1/assets/$AID/valuations -H "Authorization: Bearer $TOKEN" \
  -d '{"value_cents":12000000,"as_of":"2026-06-15"}'

# Patrimoine net
curl -s localhost:8080/v1/net-worth -H "Authorization: Bearer $TOKEN"
```

## Principes non négociables

- **Argent en centimes entiers** — jamais de float (ENF-007).
- **Calculs déterministes** dans le moteur, jamais par l'IA (EIA-040/041).
- **Données privées par profil** ; confidentialité N1/N2/N3 (à venir, P5).
- **Secrets hors code** — variables d'environnement (`.env`, jamais committé).
