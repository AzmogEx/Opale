#!/usr/bin/env bash
# Dump PostgreSQL cohérent incluant les documents chiffrés. Clé du coffre à conserver séparément.
set -euo pipefail
umask 077
source "$(dirname "$0")/postgres-common.sh"
DEST="${1:-$ROOT_DIR/backups}"
KEEP_DAYS="${OPALE_BACKUP_KEEP_DAYS:-14}"
[[ "$KEEP_DAYS" =~ ^[0-9]+$ ]] || { echo 'Durée de rétention invalide.' >&2; exit 1; }
mkdir -p "$DEST"
DEST="$(cd "$DEST" && pwd)"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$DEST/opale-$STAMP-$$.dump"
TEMP="$(mktemp "$DEST/.opale-partial.XXXXXX")"
trap 'rm -f "$TEMP"' EXIT
pg_command pg_dump -U "$DB_USER" -d "$DB_NAME" --format=custom --no-owner --no-privileges > "$TEMP"
[[ -s "$TEMP" ]] || { echo 'Sauvegarde vide.' >&2; exit 1; }
pg_command pg_restore --list < "$TEMP" > /dev/null
mv "$TEMP" "$FILE"
sha256_file "$FILE" > "$FILE.sha256"
printf 'created_utc=%s\ndatabase=%s\nformat=pg_dump_custom\nvault_key=external_required\n' "$STAMP" "$DB_NAME" > "$FILE.info"
# La rétention ne vise que nos dumps nommés ; aucune suppression en cas d'échec du nouveau dump.
while IFS= read -r OLD; do rm -f "$OLD" "$OLD.sha256" "$OLD.info"; done < <(find "$DEST" -maxdepth 1 -type f -name 'opale-*.dump' -mtime "+$KEEP_DAYS")
printf 'Sauvegarde vérifiée : %s\n' "$FILE"
