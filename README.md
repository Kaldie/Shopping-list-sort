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

## Run on a home server

```
docker build -t shopping-list-sort .
docker run -d -p 8080:8080 -v sls-data:/data \
  -e GOOGLE_CLIENT_ID=<your-client-id>.apps.googleusercontent.com \
  shopping-list-sort
```

Put it behind HTTPS (e.g. Caddy or a Cloudflare Tunnel): Google Sign-In,
the phone camera and the `Secure` session cookie all need it. For plain
HTTP testing, add `-e SECURE_COOKIES=false`.

### Google Sign-In setup

1. Google Cloud Console → APIs & Services → Credentials → *Create OAuth client ID*,
   type **Web application**.
2. *Authorized JavaScript origins*: your HTTPS address, e.g. `https://shop.example.nl`
   (add `http://localhost:5173` for development).
3. Copy the client ID into `GOOGLE_CLIENT_ID`. No client secret is needed.

Without `GOOGLE_CLIENT_ID` the server runs a name-only dev login — don't
expose that to the internet.
