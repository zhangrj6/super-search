# Super Search

Super Search is a two-stage cloud-drive search application. Search results come
from melost.cn, selected metadata is staged in URLDB, and a new share link is
created only when a visitor requests it from the resource detail page.

## Repository layout

- `backend/`: URLDB Go API and its original Nuxt administration frontend.
- `frontend/`: Next.js 16 public search and resource detail application.
- `compose.yml`: PostgreSQL, URLDB, and Next.js production stack.

## Search flow

1. `POST /api/search` searches melost.cn for resource mode. With
   `search_type: "video"`, URLDB obtains a short-lived token from quanpan.xyz,
   searches the selected Quark or Xunlei source, and paginates the normalized
   results before returning JSON.
2. Clicking a result calls `POST /api/resources/stage` and stores metadata only.
3. The resource detail page reads the staged resource by its public key.
4. `GET /api/resources/:id/link` transfers the resource and returns only the new
   share link.

Visitors do not need to sign in. Original links from melost and quanpan are not
returned by the public detail or transfer APIs.

For quanpan video results, the stage request carries `source: "quanpan"` and
continues through the existing staging and transfer/share flow. The quanpan
token is fetched and cached server-side; browser cookies and fixed `X-QP-K`
values are not required.

## Run with Docker

```bash
cp .env.example .env
# Set a strong POSTGRES_PASSWORD in .env.
docker compose -f compose.yml up -d --build
```

The public Next.js application listens on `127.0.0.1:13000` by default. The
URLDB administration frontend listens on `127.0.0.1:13001`, and the Go API
listens on `127.0.0.1:18080`. Put Nginx or another reverse proxy in front of
these loopback-only ports for public access. The provided Nginx configuration
keeps the search application at `/`, exposes the API at `/api/`, and serves the
administration frontend at `/admin`.

On a new database, sign in to `/login` with the URLDB default administrator
account (`admin` / `password`) and change the password immediately.

## Frontend development

```bash
cd frontend
cp .env.example .env.local
pnpm install
pnpm dev
```

`URLDB_API_BASE` defaults to `https://52juyou.com/api` for local development.

## Backend development

See [`backend/README.md`](backend/README.md) for the original URLDB setup and
administration frontend documentation.
