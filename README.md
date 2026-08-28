# Super Search

Super Search is a two-stage cloud-drive search application. Search results come
from melost.cn, selected metadata is staged in URLDB, and a new share link is
created only when a visitor requests it from the resource detail page.

## Repository layout

- `backend/`: URLDB Go API and its original Nuxt administration frontend.
- `frontend/`: Next.js 16 public search and resource detail application.
- `compose.yml`: PostgreSQL, URLDB, and Next.js production stack.

## Search flow

1. `POST /api/melost/search` searches melost.cn.
2. Clicking a result calls `POST /api/melost/resources` and stores metadata only.
3. The resource detail page reads the staged resource by its public key.
4. `GET /api/resources/:id/link` transfers the resource and returns only the new
   share link.

Visitors do not need to sign in. Original melost share links are not returned by
the public detail or transfer APIs.

## Run with Docker

```bash
cp .env.example .env
# Set a strong POSTGRES_PASSWORD in .env.
docker compose -f compose.yml up -d --build
```

The public Next.js application listens on `127.0.0.1:13000` by default. Put
Nginx or another reverse proxy in front of it for public access.

## Frontend development

```bash
cd frontend
cp .env.example .env.local
pnpm install
pnpm dev
```

`URLDB_API_BASE` defaults to `http://localhost:3030/api` for local development.

## Backend development

See [`backend/README.md`](backend/README.md) for the original URLDB setup and
administration frontend documentation.
