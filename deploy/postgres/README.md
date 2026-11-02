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
