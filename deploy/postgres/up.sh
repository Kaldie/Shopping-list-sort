#!/usr/bin/env bash
# Start/update the Postgres stack on the database host, taking the admin
# password from the k8s Secret postgres/postgres-admin. It goes to the host
# over ssh stdin and lives only in the containers' environment, not in a file.
#   ./up.sh            # docker compose up -d
#   ./up.sh logs -f    # any other compose command
set -euo pipefail
DB_SSH="${DB_SSH:-ruud@192.168.1.170}"
args="${*:-up -d}"
kubectl -n postgres get secret postgres-admin -o jsonpath='{.data.password}' | base64 -d \
  | ssh -o BatchMode=yes "$DB_SSH" "IFS= read -r POSTGRES_PASSWORD; export POSTGRES_PASSWORD; cd ~/postgres && docker compose $args"
