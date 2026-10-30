# Shopping List Sort

Sorts your shopping list into walking order for a supermarket whose layout
you don't know up front. The layout is learned from self-scan receipts,
your tick-off order, aisle-sign photos and a manual section list, and
everyone's trips vote on a shared layout per supermarket.

See [PLAN.md](PLAN.md) for the design and algorithm.

## Develop

```
cd backend && go run ./cmd/server          # http://localhost:8080 (dev login, no Google)
cd frontend && npm install && npm run dev  # http://localhost:5173
```

## Run locally with Docker Compose (Caddy + app)

```
cp .env.example .env          # optional: GOOGLE_CLIENT_ID, SITE_ADDRESS
docker compose up -d --build
```

Open https://localhost. Caddy serves HTTPS with its own local CA; your browser
will warn once unless you trust it (`docker compose exec caddy caddy trust`
installs it in the container only — on your machine, import
`/data/caddy/pki/authorities/local/root.crt` from the `caddy-data` volume).

Stack: `caddy` (ports 80/443, HTTPS, security headers, 25 MB uploads) →
`app` (Go API + built Vue frontend + tesseract/pdftotext, runs as non-root,
data in the `app-data` volume).

## Home server (DuckDNS + port forwarding)

1. Claim `yourname.duckdns.org` at https://www.duckdns.org and copy the token.
2. Check you have a public IPv4: the WAN IP in your router must equal
   https://whatismyip.com. If not (CGNAT), port forwarding can't work.
3. On the router, forward **TCP 80, TCP 443 and UDP 443** to the home server.
4. In `.env`:
   ```
   SITE_ADDRESS=yourname.duckdns.org
   DUCKDNS_SUBDOMAIN=yourname
   DUCKDNS_TOKEN=<token>
   GOOGLE_CLIENT_ID=<client id>   # required when exposed: otherwise anyone can use the dev login
   ```
5. `docker compose --profile duckdns up -d --build`
6. Check: `docker compose logs duckdns` should say `OK`, and `docker compose logs caddy`
   should show a certificate obtained for your domain. Then open https://yourname.duckdns.org.

Add `https://yourname.duckdns.org` to the Google client ID's *Authorized JavaScript origins*.

## Kubernetes

Manifests live in `deploy/k8s` (Kustomize), all in namespace `shopping-list-sort`:

```
docker build -t ghcr.io/kaldie/shopping-list-sort:latest . && docker push ghcr.io/kaldie/shopping-list-sort:latest
# edit deploy/k8s/kustomization.yaml (image, GOOGLE_CLIENT_ID) and ingress.yaml (host, class, issuer)
kubectl apply -k deploy/k8s
```

- In the cluster **Traefik** (ingress) plays Caddy's role: TLS via cert-manager or Traefik's own ACME resolver (see `ingress.yaml`).
- The image is built by `.github/workflows/image.yml` and pushed to `ghcr.io/kaldie/shopping-list-sort` on every push to `main`. After the first push, set the package to **Public** (GitHub → your profile → Packages → shopping-list-sort → Package settings) so the cluster can pull it without credentials.
- One replica with `Recreate` strategy: the data is a single JSON file on a
  ReadWriteOnce volume.
- Pods run under the `restricted` Pod Security Standard: non-root, read-only
  root filesystem, no capabilities.

## Google Sign-In setup

1. Google Cloud Console → APIs & Services → Credentials → *Create OAuth client ID*,
   type **Web application**.
2. *Authorized JavaScript origins*: your HTTPS address, e.g. `https://shop.example.nl`
   (add `https://localhost` and `http://localhost:5173` for development).
3. Put the client ID in `GOOGLE_CLIENT_ID`. No client secret is needed.

Without `GOOGLE_CLIENT_ID` the server runs a name-only dev login: never expose
that to the internet.
