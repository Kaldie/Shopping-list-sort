#!/usr/bin/env bash
# TLS for the shared PostgreSQL, and TLS ONLY once every resident has moved over.
#
#   ./tls.sh enable    make the CA (once) and the server certificate, and switch ssl on. A reload: nobody is disconnected,
#                      and plain connections are still accepted, so every resident can move to TLS at its own pace.
#   ./tls.sh status    ssl on or off, the certificate's expiry, every live connection and whether it is encrypted, the pg_hba rules.
#   ./tls.sh enforce   TLS only: pg_hba.conf gets `hostssl` rules and a `hostnossl ... reject`. Refuses while any client is still
#                      connected without TLS (it lists them); --force overrides. Local socket access inside the container (docker exec) stays.
#   ./tls.sh relax     the way back (an emergency): plain connections are accepted again.
#
# Residents connect with sslmode=require (encrypted) or, better, sslmode=verify-ca with the CA from the ConfigMap/Secret below.
# Where things live ("secrets live in k8s, not on the server"): the CA key and certificate in the Secret postgres/postgres-ca; the server's
# key and certificate in the data directory (server.key, server.crt); the public CA certificate can be given to a resident with
#   kubectl -n postgres get secret postgres-ca -o jsonpath='{.data.ca\.crt}' | base64 -d
# The server certificate lasts 825 days; `enable` issues a new one when fewer than 60 days are left. The CA lasts 10 years.
#
# Run from a machine with kubectl access to the cluster and ssh access to the database host, like create-app-db.sh.
set -euo pipefail
export MSYS_NO_PATHCONV=1   # git bash on Windows must not rewrite the "/CN=..." of openssl -subj into a path
cmd="${1:-status}"
DB_SSH="${DB_SSH:-ruud@192.168.1.170}"
DB_ADDR="${DB_ADDR:-192.168.1.170}"
C=postgres-postgres-1
DATA=/var/lib/postgresql/data

ssh_() { ssh -o BatchMode=yes "$DB_SSH" "$@"; }
psql_() { ssh_ "docker exec -i $C psql -q -At -v ON_ERROR_STOP=1 -U postgres" ; }   # SQL on stdin

ca_secret_exists() { kubectl -n postgres get secret postgres-ca >/dev/null 2>&1; }

make_ca() {
  if ca_secret_exists; then return 0; fi   # (not `a && return 0`: a failing test would end the function with status 1, and set -e would stop the script silently)
  local d; d="$(mktemp -d)"; trap 'rm -rf "$d"' RETURN
  # every openssl and kubectl step runs INSIDE the temp dir with relative names: on Windows (git bash) these are native programs that cannot read /tmp paths
  (
    cd "$d"
    openssl ecparam -name prime256v1 -genkey -noout -out ca.key
    openssl req -new -x509 -key ca.key -sha256 -days 3650 -subj "/CN=shared-postgres-ca" -out ca.crt
    kubectl -n postgres create secret generic postgres-ca --from-file=ca.crt=ca.crt --from-file=ca.key=ca.key >/dev/null
  )
  echo "created the CA (Secret postgres/postgres-ca)"
}

server_cert_days_left() {   # days until the server certificate expires, 0 if there is none or ssl is off
  local end
  end="$(echo | openssl s_client -starttls postgres -connect "$DB_ADDR:5432" 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2 || true)"
  [ -z "$end" ] && { echo 0; return; }
  echo $(( ( $(date -d "$end" +%s) - $(date +%s) ) / 86400 ))
}

enable() {
  make_ca
  local left; left="$(server_cert_days_left)"
  if [ "$left" -gt 60 ]; then
    echo "server certificate is fine ($left days left); ssl is on"
    return 0
  fi
  local d; d="$(mktemp -d)"; trap 'rm -rf "$d"' RETURN
  (
    cd "$d"
    kubectl -n postgres get secret postgres-ca -o jsonpath='{.data.ca\.crt}' | base64 -d > ca.crt
    kubectl -n postgres get secret postgres-ca -o jsonpath='{.data.ca\.key}' | base64 -d > ca.key
    openssl ecparam -name prime256v1 -genkey -noout -out server.key
    openssl req -new -key server.key -subj "/CN=postgres" -out server.csr
    printf 'subjectAltName=IP:%s,IP:127.0.0.1,DNS:postgres,DNS:localhost\nextendedKeyUsage=serverAuth\n' "$DB_ADDR" > ext.cnf
    openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 825 -sha256 -extfile ext.cnf
  )
  # key and certificate go into the data directory over ssh stdin; the key must belong to postgres and be 0600
  ssh_ "docker exec -i $C sh -c 'cat > $DATA/server.key && chown postgres:postgres $DATA/server.key && chmod 600 $DATA/server.key'" < "$d/server.key"
  ssh_ "docker exec -i $C sh -c 'cat > $DATA/server.crt && chown postgres:postgres $DATA/server.crt && chmod 644 $DATA/server.crt'" < "$d/server.crt"
  psql_ <<'SQL'
ALTER SYSTEM SET ssl = 'on';
ALTER SYSTEM SET ssl_cert_file = 'server.crt';
ALTER SYSTEM SET ssl_key_file = 'server.key';
ALTER SYSTEM SET ssl_min_protocol_version = 'TLSv1.2';
SELECT pg_reload_conf();
SQL
  sleep 2
  [ "$(echo 'SHOW ssl;' | psql_)" = "on" ] || { echo "✗ ssl did not switch on: docker logs $C" >&2; exit 1; }
  echo "✓ ssl is on, certificate valid for $(server_cert_days_left) days; plain connections are still accepted"
}

status() {
  echo "ssl: $(echo 'SHOW ssl;' | psql_)    certificate days left: $(server_cert_days_left)"
  echo "--- connections (role, client, encrypted):"
  psql_ <<'SQL' | sed 's/|/ | /g'
SELECT coalesce(a.usename,'-'), coalesce(host(a.client_addr),'socket'), coalesce(s.ssl::text,'-'), count(*)
FROM pg_stat_activity a LEFT JOIN pg_stat_ssl s USING (pid) WHERE a.backend_type='client backend' AND a.pid <> pg_backend_pid()
GROUP BY 1,2,3 ORDER BY 1;
SQL
  echo "--- pg_hba rules:"
  psql_ <<'SQL' | sed 's/|/ | /g'
SELECT line_number, type, database, user_name, coalesce(address,''), auth_method, coalesce(error,'') FROM pg_hba_file_rules ORDER BY line_number;
SQL
}

plain_clients() {   # role | client address of every live connection that is not encrypted (a unix socket is not a network client)
  psql_ <<'SQL'
SELECT a.usename || ' from ' || host(a.client_addr) FROM pg_stat_activity a LEFT JOIN pg_stat_ssl s USING (pid)
WHERE a.backend_type='client backend' AND a.client_addr IS NOT NULL AND coalesce(s.ssl,false) = false GROUP BY 1;
SQL
}

write_hba() {   # $1 = enforce | relax
  local rules
  if [ "$1" = enforce ]; then
    rules='local   all         all                          trust
local   replication all                          trust
hostssl all         all         127.0.0.1/32     scram-sha-256
hostssl all         all         ::1/128          scram-sha-256
hostssl replication all         127.0.0.1/32     scram-sha-256
hostssl replication all         ::1/128          scram-sha-256
hostssl all         all         all              scram-sha-256
hostnossl all       all         all              reject'
  else
    rules='local   all         all                          trust
host    all         all         127.0.0.1/32     scram-sha-256
host    all         all         ::1/128          scram-sha-256
local   replication all                          trust
host    replication all         127.0.0.1/32     scram-sha-256
host    replication all         ::1/128          scram-sha-256
host    all         all         all              scram-sha-256'
  fi
  ssh_ "docker exec $C sh -c 'cp $DATA/pg_hba.conf $DATA/pg_hba.conf.bak-\$(date +%Y%m%d%H%M%S)'"
  printf '# managed by deploy/postgres/tls.sh (%s)\n%s\n' "$1" "$rules" | ssh_ "docker exec -i $C sh -c 'cat > $DATA/pg_hba.conf && chown postgres:postgres $DATA/pg_hba.conf && chmod 600 $DATA/pg_hba.conf'"
  echo 'SELECT pg_reload_conf();' | psql_ >/dev/null
  sleep 1
  local bad; bad="$(echo 'SELECT count(*) FROM pg_hba_file_rules WHERE error IS NOT NULL;' | psql_)"
  if [ "$bad" != "0" ]; then
    echo "✗ pg_hba.conf has errors: restoring the previous file" >&2
    ssh_ "docker exec $C sh -c 'ls -t $DATA/pg_hba.conf.bak-* | head -1 | xargs -I{} cp {} $DATA/pg_hba.conf'"
    echo 'SELECT pg_reload_conf();' | psql_ >/dev/null
    exit 1
  fi
}

case "$cmd" in
  enable)  enable ;;
  status)  status ;;
  enforce)
    [ "$(echo 'SHOW ssl;' | psql_)" = "on" ] || { echo "✗ ssl is off: run ./tls.sh enable first" >&2; exit 1; }
    plain="$(plain_clients)"
    if [ -n "$plain" ] && [ "${2:-}" != "--force" ]; then
      echo "✗ these clients still connect without TLS (move them to sslmode=require first, or use --force):" >&2
      echo "$plain" | sed 's/^/    /' >&2
      exit 1
    fi
    write_hba enforce
    echo "✓ TLS only: pg_hba.conf has hostssl rules and rejects everything unencrypted"
    ;;
  relax)   write_hba relax; echo "✓ plain connections are accepted again" ;;
  *) echo "usage: $0 enable|status|enforce [--force]|relax" >&2; exit 2 ;;
esac
