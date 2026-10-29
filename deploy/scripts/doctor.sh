#!/usr/bin/env bash
# Checks each hop between a visitor and SQL Playground (app → nginx → Cloudflare)
# and says where loading breaks. Changes nothing.
#
#   sudo /opt/sql-playground/deploy/scripts/doctor.sh sql.example.com
set -uo pipefail
[ "$(id -u)" -eq 0 ] || exec sudo "$0" "$@"
domain=${1:-}
[ -n "$domain" ] || { echo "usage: sudo $0 your.domain"; exit 2; }

CONF=/etc/sql-playground/config.env
WEB=/usr/local/lib/sql-playground/web
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
problems=0

ok() { printf '  \033[32mok\033[0m    %s\n' "$*"; }
bad() { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; problems=$((problems + 1)); }
warn() { printf '  \033[33mwarn\033[0m  %s\n' "$*"; }
head1() { printf '\n\033[1m%s\033[0m\n' "$*"; }

# fetch NAME URL [curl options]: prints status, type, bytes and seconds, and
# checks that a script or stylesheet arrives complete (same size as on disk).
fetch() {
  local name=$1 url=$2 out
  shift 2
  rm -f "$tmp/body"
  out=$(curl -sSk --compressed --max-time 20 -o "$tmp/body" \
    -w '%{http_code} %{content_type} %{time_total}' "$@" "$url" 2>"$tmp/err")
  local code=${out%% *} rest=${out#* } size
  local type=${rest% *} secs=${rest##* }
  size=$(cat "$tmp/body" 2>/dev/null | wc -c)
  local line="$name  $code  ${type:-?}  $size bytes  ${secs}s"
  if [ "$code" != 200 ]; then
    bad "$line  $(head -c 200 "$tmp/err")"
    return 1
  fi
  local path=${url#*://*/}
  if [ -f "$WEB/$path" ] && [ "$size" -ne "$(wc -c <"$WEB/$path")" ]; then
    bad "$line  (expected $(wc -c <"$WEB/$path") bytes: the file arrives cut off)"
    return 1
  fi
  case $path in
    *.js) [[ $type == *javascript* ]] || { bad "$line  (not served as JavaScript)"; return 1; } ;;
  esac
  ok "$line"
}

# check_hop LABEL BASE [curl options]: the page and every file it loads
check_hop() {
  local label=$1 base=$2
  shift 2
  head1 "$label"
  fetch "/" "$base/" "$@" || return 1
  cp "$tmp/body" "$tmp/index.html"
  if grep -q 'rocket-loader\|rocketscript' "$tmp/index.html"; then
    warn "Cloudflare Rocket Loader is rewriting the page. Turn it off: Speed → Optimization → Rocket Loader."
  fi
  local failed=0
  for asset in $(grep -oE '/assets/[^"]+\.(js|css)' "$tmp/index.html" | sort -u); do
    fetch "$asset" "$base$asset" "$@" || failed=1
  done
  fetch "/api/config" "$base/api/config" "$@" || failed=1
  return $failed
}

head1 "Services"
for s in mysql sql-playground nginx; do
  if systemctl is-active --quiet "$s"; then ok "$s is running"; else bad "$s is not running (systemctl status $s)"; fi
done
listen=$(sed -n 's/^SQLPG_LISTEN=//p' "$CONF" 2>/dev/null | tail -n 1)
port=${listen##*:}
port=${port:-8080}
[ -f "$WEB/index.html" ] && ok "website files in $WEB" || bad "no website in $WEB: run deploy/install.sh again"

check_hop "1. App, directly (http://127.0.0.1:$port)" "http://127.0.0.1:$port"
app=$?
check_hop "2. nginx on this server (https://$domain via 127.0.0.1)" "https://$domain" \
  --resolve "$domain:443:127.0.0.1"
local_nginx=$?
check_hop "3. Public, through Cloudflare (https://$domain)" "https://$domain"
public=$?
if [ $public -eq 0 ] || [ -s "$tmp/index.html" ]; then
  hdrs=$(curl -sk -I --max-time 20 "https://$domain/" 2>/dev/null | tr -d '\r')
  grep -qi '^server: cloudflare' <<<"$hdrs" && ok "answered by Cloudflare" ||
    warn "the public address isn't answered by Cloudflare (DNS record not proxied?)"
fi

head1 "Server"
df -h / | awk 'NR==2 { print "  disk  " $4 " free of " $2 " (" $5 " used)" }'
free -h | awk '/^Mem/ { print "  RAM   " $7 " available of " $2 }'
avail=$(df -Pk / | awk 'NR==2 { print $4 }')
[ "$avail" -gt 1048576 ] || bad "less than 1 GB of disk left"
nginx -t >/dev/null 2>&1 && ok "nginx configuration is valid" || bad "nginx -t reports errors (run: sudo nginx -t)"
if grep -qE 'proxy.*(Permission denied|No space left)|(Permission denied|No space left).*proxy' /var/log/nginx/error.log 2>/dev/null; then
  bad "nginx couldn't write its temporary files, which cuts large downloads short. Free up disk space, or add"
  echo "        proxy_max_temp_file_size 0;   to the location block of your nginx site, then reload nginx."
fi
if [ -s /var/log/nginx/error.log ]; then
  echo "  last nginx errors:"
  tail -n 5 /var/log/nginx/error.log | cut -c1-220 | sed 's/^/    /'
fi
echo "  last app log lines:"
journalctl -u sql-playground -n 5 --no-pager -o cat 2>/dev/null | cut -c1-220 | sed 's/^/    /'

head1 "Verdict"
if [ $app -ne 0 ]; then
  echo "  The app itself doesn't serve the site correctly. Check: journalctl -u sql-playground -n 50"
elif [ $local_nginx -ne 0 ]; then
  echo "  The app works, but nginx on this server doesn't pass it through. Check the site file in"
  echo "  /etc/nginx/sites-enabled/ (server_name, certificate, upstream port $port) and the nginx errors above."
elif [ $public -ne 0 ]; then
  echo "  This server works; the problem is between Cloudflare and the server, or in Cloudflare's"
  echo "  settings. Check SSL/TLS mode = Full (strict), Rocket Loader off, and Caching → Purge Everything."
elif [ $problems -gt 0 ]; then
  echo "  Every hop delivers the site, but see the warnings above."
else
  echo "  Every hop delivers the site completely. If a browser still shows a blank page, open it in a"
  echo "  private window (to rule out extensions and old cache) and look at the Console tab in DevTools."
fi
