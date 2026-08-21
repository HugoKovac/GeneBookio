# Deploying to Dokploy

This repo's root [docker-compose.yaml](docker-compose.yaml) is the Dokploy deployment
stack — separate from `backend/docker-compose.yaml`, which stays the local/dev
orchestration described in the main [CLAUDE.md](CLAUDE.md).

Deploy it as **one Dokploy project with two Compose environments** ("dev" and
"prod"), both pointing at this same `docker-compose.yaml`, each with its own
Environment Variables (see [.env.example](.env.example),
[.env.dev.example](.env.dev.example), [.env.prod.example](.env.prod.example)).

## One-time setup

1. **`dokploy-network`** — Dokploy creates this external network on install;
   nothing to do unless it's been renamed.

2. **JWT keys** — `backend/keys/` is gitignored (it holds real private keys),
   so a fresh git checkout Dokploy builds from won't have it. Before the first
   deploy of each environment, generate a keypair and put it at
   `backend/keys/` in the checkout Dokploy builds from (e.g. `make genNewKeys`
   from `backend/`, run once per environment so dev and prod don't share a
   signing key, then get the files onto the Dokploy host/build context).

3. **Admin Basic Auth** — `frontend-admin` (backend/cmd/admin) has no
   authentication of its own by design (see CLAUDE.md), so it's only exposed
   here behind a Traefik Basic Auth middleware. Generate a hash:

   ```bash
   htpasswd -nB admin
   ```

   Paste the result into `ADMIN_BASIC_AUTH` as `user:hash`. Docker Compose
   treats `$` as a variable-expansion character, so if you paste the hash
   directly as an env var value (rather than through Dokploy's UI, which
   handles this for you), double every literal `$` to `$$`.

4. **Domains / TLS** — set per environment in `.env.<env>.example`:
   - `APP_DOMAIN` — the hostname both the app (`/`) and admin (`/admin`) are
     served from (path-based, not subdomain-based, since a Tailscale
     MagicDNS name like `hugo-gemini-j34.taile50a8b.ts.net` has no wildcard
     subdomains).
   - `CERT_RESOLVER` — Dokploy's Let's Encrypt resolver (usually
     `letsencrypt`). **Leave blank** for a domain Let's Encrypt can't validate
     over the public internet — a `.ts.net` Tailscale name, a
     `traefik.me`/127.0.0.1 name, a bare IP. Traefik then serves its own
     self-signed certificate (expect a browser warning; fine for a private
     tailnet). Switch to `letsencrypt` once `APP_DOMAIN` is a real,
     publicly-reachable domain — nothing else in the compose file needs to
     change.
   - `TRAEFIK_ENTRYPOINT` — since dev and prod currently share the exact same
     `APP_DOMAIN` (no separate DNS name for dev yet), they're split by
     **port** instead: prod keeps Dokploy's default `websecure` (:443), dev
     uses a distinct entrypoint (e.g. `websecure-dev`, suggested :8443). A
     custom entrypoint/port is Traefik **static** config, which isn't
     something an app's docker-compose labels can add — add it once in
     Dokploy's own Traefik config (Server → Advanced → Traefik, or the
     `traefik.yml`/`traefik.dynamic` files Dokploy manages) as an extra
     `entryPoints` block bound to that port, then restart Traefik. Once dev
     gets its own domain, you can drop this port split and just give it its
     own `TRAEFIK_ENTRYPOINT=websecure` + distinct `APP_DOMAIN`.

## Notes

- Internal services (`db`, `rabbitmq`, `seaweed`, `api`, `admin`, and the four
  pipeline workers) publish no host ports and carry no Traefik labels — only
  `frontend` and `frontend-admin` join `dokploy-network` and get routed.
- `frontend-admin` is built with `VITE_BASE_PATH=/admin/` and
  `VITE_API_URL=/admin/api` (see its `Dockerfile`/`vite.config.ts`) so it
  works correctly served under the `/admin` path prefix on the same domain as
  `frontend`. Traefik router priority (`admin` = 10, `app` = 1) makes sure
  `/admin` requests reach `frontend-admin` rather than falling through to
  `frontend`'s catch-all router.
- `AI_TEST_MODE` defaults to `true` if unset — each environment's real values
  should set it explicitly (`false` only where spending real OpenAI credit is
  intended).
- SeaweedFS seeds its four buckets (`books`, `scripts`, `audio`, `prompts`)
  from `S3_BUCKET` on first boot — nothing to do there. To bring over data
  from an existing host, see `backend/scripts/migrate_all.sh`.
