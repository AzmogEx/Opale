#!/usr/bin/env bash
# Chargé par les commandes de sauvegarde/restauration ; ne journalise aucun secret.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_USER="${OPALE_DB_USER:-opale}"
DB_NAME="${OPALE_DB_NAME:-opale}"
pg_command() {
  if [[ "${OPALE_BACKUP_MODE:-docker}" == local ]]; then
    "$@"
  elif [[ -n "${OPALE_DB_CONTAINER:-}" ]]; then
    docker exec -i "$OPALE_DB_CONTAINER" "$@"
  else
    docker compose --project-directory "$ROOT_DIR" exec -T db "$@"
  fi
}
sha256_file() {
  if command -v sha256sum >/dev/null; then sha256sum "$1" | awk '{print $1}';
  else shasum -a 256 "$1" | awk '{print $1}'; fi
}
