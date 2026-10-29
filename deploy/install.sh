#!/usr/bin/env bash
# Installs or updates SQL Playground on Ubuntu, without Docker:
#   MySQL from Ubuntu's packages, the app as the systemd service "sql-playground".
#
#   sudo ./deploy/install.sh              first install, and again after every `git pull`
#   sudo ./deploy/install.sh --port 8081  first install on another local port (default 8080)
#
# Safe to run again at any time: it keeps your settings, passwords and all data.
# Go and Node.js are downloaded into /usr/local/lib/sql-playground/toolchain and are
# only used to build; they don't touch any Go or Node you have installed yourself.
set -euo pipefail

GO_VERSION=${GO_VERSION:-go1.27.3}
GO_DL=${GO_DL:-https://go.dev/dl}
NODE_MAJOR=${NODE_MAJOR:-24}
NODE_DL=${NODE_DL:-https://nodejs.org/dist}

PREFIX=/usr/local/lib/sql-playground
TOOLS=$PREFIX/toolchain
BUILD=$PREFIX/build
CACHE=$PREFIX/cache
CONF=/etc/sql-playground/config.env
SERVICE=sql-playground
SRC=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PORT=8080

say() { printf '\n\033[1;33m==>\033[0m \033[1m%s\033[0m\n' "$*"; }
die() { printf '\n\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case $1 in
    --port) [ $# -ge 2 ] || die "--port needs a port number"; PORT=$2; shift 2 ;;
    -h | --help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "unknown option: $1 (see --help)" ;;
  esac
done
[[ $PORT =~ ^[0-9]+$ ]] || die "--port needs a port number"
[ "$(id -u)" -eq 0 ] || die "run it with sudo: sudo $0"
[ -f "$SRC/backend/go.mod" ] && [ -f "$SRC/frontend/package.json" ] ||
  die "run this script from a checkout of the repository"
. /etc/os-release
[ "${ID:-}" = ubuntu ] || die "this script supports Ubuntu (found ${PRETTY_NAME:-an unknown system})"
case $(uname -m) in
  x86_64) GOARCH=amd64 NODEARCH=x64 ;;
  aarch64) GOARCH=arm64 NODEARCH=arm64 ;;
  *) die "unsupported CPU architecture: $(uname -m)" ;;
esac

# ---------------------------------------------------------------- packages
say "Installing MySQL and build prerequisites"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq mysql-server curl ca-certificates xz-utils rsync python3 openssl \
  >/dev/null
systemctl enable --now --quiet mysql

mysql_version=$(mysqld --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)
[ "$(printf '%s\n' 8.0.31 "$mysql_version" | sort -V | head -n 1)" = 8.0.31 ] ||
  die "MySQL $mysql_version is too old; SQL Playground needs 8.0.31 or newer"
echo "MySQL $mysql_version"

# ---------------------------------------------------------------- MySQL settings
say "Configuring MySQL"
cnf_dir=/etc/mysql/mysql.conf.d
[ -d "$cnf_dir" ] || cnf_dir=/etc/mysql/conf.d
cnf=$cnf_dir/zz-sql-playground.cnf
if cmp -s "$SRC/deploy/mysql/playground.cnf" "$cnf"; then
  echo "$cnf is up to date"
else
  install -m 644 "$SRC/deploy/mysql/playground.cnf" "$cnf"
  if ! mysqld --validate-config --user=mysql; then
    rm -f "$cnf"
    die "MySQL rejected deploy/mysql/playground.cnf, so it was not installed"
  fi
  echo "installed $cnf, restarting MySQL"
  systemctl restart mysql
fi

# ---------------------------------------------------------------- settings + MySQL account
say "Creating the app's settings and MySQL account"
install -d -m 700 "$(dirname "$CONF")"
if [ ! -f "$CONF" ]; then
  sed -e "s|^SQLPG_LISTEN=.*|SQLPG_LISTEN=127.0.0.1:$PORT|" \
    -e "s|^SQLPG_MYSQL_ADMIN_PASSWORD=.*|SQLPG_MYSQL_ADMIN_PASSWORD=$(openssl rand -hex 24)|" \
    "$SRC/deploy/config.env.example" | install -m 600 /dev/stdin "$CONF"
  echo "wrote $CONF"
else
  echo "keeping $CONF"
fi
setting() { sed -n "s/^$1=//p" "$CONF" | tail -n 1; }
db_user=$(setting SQLPG_MYSQL_ADMIN_USER)
db_pass=$(setting SQLPG_MYSQL_ADMIN_PASSWORD)
listen=$(setting SQLPG_LISTEN)
PORT=${listen##*:}
[[ $db_user =~ ^[A-Za-z0-9_]+$ ]] || die "SQLPG_MYSQL_ADMIN_USER in $CONF must be letters, digits or _"
[[ $db_pass =~ ^[A-Za-z0-9_.+-]+$ ]] ||
  die "SQLPG_MYSQL_ADMIN_PASSWORD in $CONF is empty or has characters other than letters, digits and _.+-"

mysql --protocol=socket -e 'SELECT 1' >/dev/null 2>&1 ||
  die "can't log in to MySQL as root through its socket (sudo mysql). If root has a password, put it in /root/.my.cnf under [client]."
# The app connects over TCP to 127.0.0.1, which matches the host '127.0.0.1'.
mysql --protocol=socket <<SQL
CREATE USER IF NOT EXISTS '$db_user'@'127.0.0.1' IDENTIFIED BY '$db_pass';
ALTER USER '$db_user'@'127.0.0.1' IDENTIFIED BY '$db_pass';
GRANT ALL PRIVILEGES ON *.* TO '$db_user'@'127.0.0.1' WITH GRANT OPTION;
SQL
echo "MySQL account $db_user@127.0.0.1 is ready"

# ---------------------------------------------------------------- build tools
say "Getting Go and Node.js (used only for building)"
install -d "$TOOLS"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

install_go() {
  local tgz=$GO_VERSION.linux-$GOARCH.tar.gz sum
  if [ "$(head -n 1 "$TOOLS/go/VERSION" 2>/dev/null)" = "$GO_VERSION" ]; then
    echo "Go ${GO_VERSION#go} is already there"
    return
  fi
  sum=$(curl -fsSL "$GO_DL/?mode=json&include=all" | python3 -c '
import json, sys
for release in json.load(sys.stdin):
    for f in release["files"]:
        if f["filename"] == sys.argv[1]:
            print(f["sha256"])
            sys.exit()
' "$tgz") || die "couldn't read the Go download list from $GO_DL"
  [ -n "$sum" ] || die "$tgz isn't listed on $GO_DL"
  curl -fsSL "$GO_DL/$tgz" -o "$tmp/$tgz" || die "couldn't download $GO_DL/$tgz"
  echo "$sum  $tmp/$tgz" | sha256sum -c --quiet - || die "the Go download is corrupt (checksum mismatch)"
  rm -rf "$TOOLS/go"
  tar -C "$TOOLS" -xzf "$tmp/$tgz"
  echo "installed Go ${GO_VERSION#go}"
}

install_node() {
  local base=$NODE_DL/latest-v$NODE_MAJOR.x sums line file version
  if ! sums=$(curl -fsSL "$base/SHASUMS256.txt"); then
    [ -x "$TOOLS/node/bin/node" ] || die "couldn't reach $base"
    echo "couldn't check for a newer Node.js; using the one already installed"
    return
  fi
  line=$(grep -E "  node-v[0-9.]+-linux-$NODEARCH\.tar\.xz\$" <<<"$sums" | head -n 1) ||
    die "no Node.js $NODE_MAJOR build for linux-$NODEARCH"
  file=${line##* }
  version=${file%-linux-*}
  if [ "$(cat "$TOOLS/node/.version" 2>/dev/null)" = "$version" ]; then
    echo "Node.js ${version#node-v} is already there"
    return
  fi
  curl -fsSL "$base/$file" -o "$tmp/$file" || die "couldn't download $base/$file"
  (cd "$tmp" && sha256sum -c --quiet <<<"$line") || die "the Node.js download is corrupt (checksum mismatch)"
  rm -rf "$TOOLS/node"
  mkdir "$TOOLS/node"
  tar -C "$TOOLS/node" --strip-components=1 -xJf "$tmp/$file"
  echo "$version" >"$TOOLS/node/.version"
  echo "installed Node.js ${version#node-v}"
}

install_go
install_node

# ---------------------------------------------------------------- build
say "Building the app (a few minutes the first time)"
id -u sqlpg >/dev/null 2>&1 ||
  useradd --system --user-group --home-dir "$PREFIX" --no-create-home --shell /usr/sbin/nologin sqlpg
install -d -o sqlpg -g sqlpg "$BUILD" "$CACHE"
rsync -a --delete --chown=sqlpg:sqlpg \
  --exclude=/.git --exclude=/docs --exclude=node_modules --exclude=/frontend/dist \
  "$SRC/" "$BUILD/"

# Builds run as the unprivileged sqlpg user, with the toolchain above.
as_builder() {
  runuser -u sqlpg -- env HOME="$CACHE" PATH="$TOOLS/go/bin:$TOOLS/node/bin:/usr/bin:/bin" \
    GOPATH="$CACHE/go" GOCACHE="$CACHE/go-build" GOTOOLCHAIN=local CGO_ENABLED=0 \
    npm_config_cache="$CACHE/npm" npm_config_update_notifier=false "$@"
}
(cd "$BUILD/frontend" &&
  as_builder npm ci --ignore-scripts --no-audit --no-fund --loglevel=error &&
  as_builder npm run build --silent -- --logLevel warn) || die "the website build failed (output above)"
(cd "$BUILD/backend" &&
  as_builder go build -trimpath -ldflags='-s -w' -o "$BUILD/sqlpg" ./cmd/sqlpg) ||
  die "the server build failed (output above)"
rm -rf "$BUILD/frontend/node_modules"   # npm ci starts from scratch anyway; its cache stays

# ---------------------------------------------------------------- install
say "Installing the service"
install -d "$PREFIX/bin"
install -m 755 "$BUILD/sqlpg" "$PREFIX/bin/sqlpg.new"
mv -f "$PREFIX/bin/sqlpg.new" "$PREFIX/bin/sqlpg"
rm -rf "$PREFIX/web.new" "$PREFIX/web.old"
cp -r "$BUILD/frontend/dist" "$PREFIX/web.new"
chown -R root:root "$PREFIX/web.new"
chmod -R u=rwX,go=rX "$PREFIX/web.new"
[ ! -d "$PREFIX/web" ] || mv "$PREFIX/web" "$PREFIX/web.old"
mv "$PREFIX/web.new" "$PREFIX/web"
rm -rf "$PREFIX/web.old"

install -m 755 "$SRC/deploy/scripts/backup.sh" "$PREFIX/backup.sh"
install -m 755 "$SRC/deploy/scripts/sqlpg-admin" /usr/local/bin/sqlpg-admin
install -m 644 "$SRC/deploy/systemd/sql-playground.service" /etc/systemd/system/$SERVICE.service
if [ -d /etc/cron.d ]; then
  printf '%s\n' '# Daily backup of SQL Playground accounts and progress (from deploy/install.sh)' \
    "17 3 * * * root $PREFIX/backup.sh >/dev/null" >/etc/cron.d/sql-playground
fi

if ss -Hltnp "( sport = :$PORT )" | grep -v '"sqlpg"' | grep -q .; then
  die "port $PORT is already used by another program. Set another port in SQLPG_LISTEN in $CONF (and in your nginx site), then run this script again."
fi
systemctl daemon-reload
systemctl enable --quiet $SERVICE
systemctl restart $SERVICE
restarts=$(systemctl show -p NRestarts --value $SERVICE)

say "Starting (the first start generates the dataset, about 30 seconds)"
for _ in $(seq 1 240); do
  if curl -fs -o /dev/null "http://127.0.0.1:$PORT/api/health"; then
    say "SQL Playground is running on http://127.0.0.1:$PORT"
    cat <<EOF
  logs:      journalctl -u $SERVICE -f
  accounts:  sudo sqlpg-admin list-users
  settings:  $CONF  (then: sudo systemctl restart $SERVICE)
  backups:   daily at 03:17 to /var/backups/sql-playground/
  update:    cd $SRC && git pull && sudo ./deploy/install.sh
EOF
    exit 0
  fi
  if [ "$(systemctl show -p NRestarts --value $SERVICE)" -ge $((restarts + 2)) ]; then
    echo
    journalctl -u $SERVICE -n 20 --no-pager -o cat || true
    die "the app keeps stopping (log above). Fix the cause, then run this script again."
  fi
  printf '.'
  sleep 1
done
echo
journalctl -u $SERVICE -n 30 --no-pager -o cat || true
die "the app didn't answer on port $PORT within 4 minutes (log above)"
