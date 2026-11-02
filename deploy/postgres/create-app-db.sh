#!/usr/bin/env bash
# Create a role + database for one app and store its connection URL as a k8s
# Secret (key DATABASE_URL). The password is generated here and only travels
# over stdin: never printed, never in a process argument list.
#
#   ./create-app-db.sh <app-name> <k8s-namespace> [secret-name]
#   ./create-app-db.sh shoppinglist shopping-list-sort
#
# Run from a machine that has kubectl access to the cluster and ssh access to
# the database host.
set -euo pipefail
name="${1:?usage: $0 <app-name> <k8s-namespace> [secret-name]}"
ns="${2:?usage: $0 <app-name> <k8s-namespace> [secret-name]}"
secret="${3:-$name-db}"
DB_SSH="${DB_SSH:-ruud@192.168.1.170}"
DB_ADDR="${DB_ADDR:-192.168.1.170}"
[[ "$name" =~ ^[a-z][a-z0-9_]*$ ]] || { echo "app name must match [a-z][a-z0-9_]*" >&2; exit 1; }

kubectl get namespace "$ns" >/dev/null
pw="$(openssl rand -hex 24)"

ssh -o BatchMode=yes "$DB_SSH" 'docker exec -i postgres-postgres-1 psql -v ON_ERROR_STOP=1 -U postgres' <<SQL
CREATE ROLE "$name" LOGIN PASSWORD '$pw';
CREATE DATABASE "$name" OWNER "$name";
REVOKE ALL ON DATABASE "$name" FROM PUBLIC;
SQL

kubectl apply -f - <<YAML
apiVersion: v1
kind: Secret
metadata:
  name: $secret
  namespace: $ns
type: Opaque
stringData:
  DATABASE_URL: postgres://$name:$pw@$DB_ADDR:5432/$name
YAML
echo "database '$name' created; secret $ns/$secret has DATABASE_URL"
