#!/usr/bin/env bash
# Back up the app's own data: accounts, saved queries, history and lesson progress.
# Learner database copies are not included; the app recreates them on demand.
#
#   sudo ./backup.sh    writes /var/backups/sql-playground/app-YYYY-MM-DD.sql.gz, keeps 14
#
# Restore on a fresh install (from deploy/, after `docker compose up -d`):
#   gunzip -c app-YYYY-MM-DD.sql.gz | docker compose exec -T mysql \
#     sh -c 'MYSQL_PWD=$MYSQL_ROOT_PASSWORD mysql -uroot'
#   docker compose restart app
set -euo pipefail
cd "$(dirname "$0")/.."   # the deploy/ folder with docker-compose.yml

dir=${BACKUP_DIR:-/var/backups/sql-playground}
mkdir -p "$dir"
file="$dir/app-$(date +%F).sql.gz"
docker compose exec -T mysql sh -c \
  'MYSQL_PWD=$MYSQL_ROOT_PASSWORD mysqldump -uroot --single-transaction --databases playground_app' \
  | gzip > "$file.tmp"
mv "$file.tmp" "$file"
ls -1t "$dir"/app-*.sql.gz | tail -n +15 | xargs -r rm --
echo "backup written: $file ($(du -h "$file" | cut -f1))"
