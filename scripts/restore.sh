#!/usr/bin/env bash
# Restaure exclusivement dans une NOUVELLE base. Aucune base existante ne peut être écrasée.
set -euo pipefail
umask 077
source "$(dirname "$0")/postgres-common.sh"
FILE="${1:?Usage: restore.sh sauvegarde.dump nouvelle_base}"
TARGET="${2:?Indique une nouvelle base isolée}"
[[ "$TARGET" =~ ^[a-z][a-z0-9_]{0,62}$ ]] || { echo 'Nom de base invalide (minuscules, chiffres, _).' >&2; exit 1; }
[[ "$TARGET" != "$DB_NAME" ]] || { echo 'La cible doit être différente de la base source.' >&2; exit 1; }
[[ -r "$FILE" && -r "$FILE.sha256" ]] || { echo 'Dump ou empreinte manquant.' >&2; exit 1; }
[[ "$(sha256_file "$FILE")" == "$(cat "$FILE.sha256")" ]] || { echo 'Empreinte incorrecte : restauration refusée.' >&2; exit 1; }
pg_command pg_restore --list < "$FILE" > /dev/null
EXISTS="$(pg_command psql -U "$DB_USER" -d postgres -Atc "SELECT 1 FROM pg_database WHERE datname='$TARGET'")"
[[ "$EXISTS" != 1 ]] || { echo 'La base cible existe déjà : aucune modification.' >&2; exit 1; }
pg_command createdb -U "$DB_USER" -- "$TARGET"
cleanup_failure() { pg_command dropdb -U "$DB_USER" -- "$TARGET" >/dev/null 2>&1 || true; }
trap cleanup_failure ERR
pg_command pg_restore -U "$DB_USER" -d "$TARGET" --exit-on-error --single-transaction --no-owner --no-privileges < "$FILE"
# Un ancien jeton ne doit pas redevenir actif lors d'une restauration.
pg_command psql -U "$DB_USER" -d "$TARGET" -v ON_ERROR_STOP=1 -c 'TRUNCATE TABLE sessions;' > /dev/null
pg_command psql -U "$DB_USER" -d "$TARGET" -v ON_ERROR_STOP=1 -c 'ANALYZE;' > /dev/null
trap - ERR
printf 'Base restaurée : %s. Sessions révoquées. Vérifier avec une API isolée et la même clé de coffre avant toute bascule.\n' "$TARGET"
