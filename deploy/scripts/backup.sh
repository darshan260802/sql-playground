#!/usr/bin/env bash
# Back up the app's own data: accounts, saved queries, history and lesson progress.
# Learner database copies are not included; the app recreates them on demand.
# deploy/install.sh installs this as /usr/local/lib/sql-playground/backup.sh and runs
# it every night at 03:17 (/etc/cron.d/sql-playground).
#
#   sudo ./backup.sh    writes /var/backups/sql-playground/app-YYYY-MM-DD.sql.gz, keeps 14
#
# Restore (after deploy/install.sh; on a new server, copy the file there first):
#   sudo gunzip -c /var/backups/sql-playground/app-YYYY-MM-DD.sql.gz | sudo mysql
#   sudo systemctl restart sql-playground
set -euo pipefail
[ "$(id -u)" -eq 0 ] || exec sudo "$0" "$@"
umask 077

dir=${BACKUP_DIR:-/var/backups/sql-playground}
keep=${KEEP:-14}
mkdir -p "$dir"
file="$dir/app-$(date +%F).sql.gz"
# root logs in to MySQL through its socket, as on any Ubuntu install
mysqldump --single-transaction --databases playground_app | gzip > "$file.tmp"
mv "$file.tmp" "$file"
ls -1t "$dir"/app-*.sql.gz | tail -n +$((keep + 1)) | xargs -r rm --
echo "backup written: $file ($(du -h "$file" | cut -f1))"
