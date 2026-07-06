#!/usr/bin/env bash
# Opale — sauvegarde PostgreSQL avec rotation (audit : données patrimoniales
# + coffre chiffré SANS backup = risque réel).
#
# Usage :  ./scripts/backup.sh [dossier-destination]
# Cron :   0 3 * * *  /chemin/vers/Opale/scripts/backup.sh /backups/opale
#
# ⚠️ La clé OPALE_VAULT_KEY ne vit PAS en base : sauvegarde-la séparément
#    (gestionnaire de mots de passe). Sans elle, les documents du coffre
#    contenus dans le dump resteront chiffrés à jamais.

set -euo pipefail

CONTAINER="${OPALE_DB_CONTAINER:-opale-db}"
DB_USER="${OPALE_DB_USER:-opale}"
DB_NAME="${OPALE_DB_NAME:-opale}"
DEST="${1:-./backups}"
KEEP_DAYS="${OPALE_BACKUP_KEEP_DAYS:-14}"

mkdir -p "$DEST"
STAMP="$(date +%Y-%m-%d_%H%M%S)"
FILE="$DEST/opale-$STAMP.sql.gz"

# Dump compressé, format custom → restauration ciblée possible (pg_restore).
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --format=custom \
  | gzip > "$FILE"

# Vérification minimale : le fichier n'est pas vide.
if [ ! -s "$FILE" ]; then
  echo "ERREUR : dump vide — sauvegarde échouée" >&2
  rm -f "$FILE"
  exit 1
fi

# Rotation : supprime les sauvegardes plus vieilles que KEEP_DAYS jours.
find "$DEST" -name "opale-*.sql.gz" -mtime "+$KEEP_DAYS" -delete

echo "OK : $FILE ($(du -h "$FILE" | cut -f1)) — rotation à $KEEP_DAYS jours"
