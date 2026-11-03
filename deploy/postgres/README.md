# Shared Postgres (outside k3s)

Runs on the server via Docker Compose, data in `~/postgres/data`, nightly dumps in
`~/postgres/backups`. Listens on the LAN address only; never forward 5432.

Secrets live in k8s, not on the server:

- Admin password: Secret `postgres/postgres-admin` (key `password`). `./up.sh`
  streams it to the host over ssh into the container env. No password file.
  Lost the cluster? Reset it via the local socket:
  `docker exec -it postgres-postgres-1 psql -U postgres -c "\password postgres"`
- Per-app credentials: `./create-app-db.sh <app> <namespace>` creates a role +
  database and a Secret `<app>-db` with `DATABASE_URL` in that namespace.
  Use it in a pod with `envFrom: [{secretRef: {name: <app>-db}}]`.

Server-side `.env` only holds non-secret paths (`DATA_DIR`, `BACKUP_DIR`).

## TLS

The server speaks TLS and, once enforced, **only** TLS: `pg_hba.conf` has `hostssl` rules and rejects anything unencrypted. Every
resident (the shop, the sovereign data platform) connects with `sslmode=require`.

    ./tls.sh enable     # CA once, server certificate, ssl on (a reload; plain still accepted)
    ./tls.sh status     # ssl, certificate expiry, every live connection and whether it is encrypted, the pg_hba rules
    ./tls.sh enforce    # TLS only; refuses while a client still connects without TLS (lists it), --force overrides
    ./tls.sh relax      # the way back, in an emergency

Order for a new resident: create its database with `./create-app-db.sh` (the URL already has `?sslmode=require`), or add
`sslmode=require` to an existing URL; check `./tls.sh status` shows its connections encrypted; then `enforce`.

The CA (key and certificate) is the Secret `postgres/postgres-ca`; the server key and certificate live in the data directory. The server
certificate lasts 825 days and `./tls.sh enable` renews it when fewer than 60 remain. `sslmode=require` encrypts but does not check the
server's identity; for that give the app the CA certificate (`kubectl -n postgres get secret postgres-ca -o jsonpath='{.data.ca\.crt}' | base64 -d`)
and use `sslmode=verify-ca&sslrootcert=<file>`.
