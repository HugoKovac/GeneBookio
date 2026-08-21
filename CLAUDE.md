# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository layout

Independent projects in one repo:

- `backend/` — Go 1.26 service pipeline that turns an uploaded EPUB into narrated audio (module `hkorpo/book`)
- `frontend/` — React 19 + TypeScript + Vite + Mantine SPA (auth + dashboard UI), talks to `cmd/api`
- `frontend-admin/` — React 19 + TypeScript + Vite + Mantine SPA for admins only: upload a book, watch its pipeline progress, browse the catalogue. Talks to `cmd/admin`, which has no auth (local-admin-only by design)

## Backend

### Commands (run from `backend/`)

```
make api                # run the HTTP API (cmd/api) — user auth + book search endpoints, :3000
make admin                # run the admin/backoffice service (cmd/admin) — EPUB upload + catalogue endpoints, :3001
make epub_split           # run the epub_parser worker (consumes "split", produces "prepare")
make prepare_chapters     # run the prepare_chapters worker (consumes "prepare", produces "generate_script")
make generate_script       # run the generate_script worker (consumes "generate_script", produces "generate_tts")
make generate_tts          # run the generate_tts worker (consumes "generate_tts")

make docker               # docker compose up -d (all services incl. db/rabbitmq/seaweed)
make docker-hybrid        # only infra: db, rabbitmq, seaweed (run Go services locally against them)

make ent                  # regenerate ent ORM code after editing pkg/ent/schema/*.go
make genNewKeys            # generate a new RSA keypair into keys/ (KEYS_DIR/KEY_NAME override the path/prefix)
```

Each `cmd/*` binary reads its own `.env` file (see `cmd/<name>/.env.example`) via `pkg/env` + `envconfig`; `backend/.env` holds the shared docker-compose credentials (MySQL/SeaweedFS/RabbitMQ). Copy the relevant `.env.example` to `.env` before running a service.

`prepare_chapters`, `generate_script` and `generate_tts` (the three AI-driven stages — the latter now calls OpenAI's TTS API instead of a local model) read `AI_TEST_MODE` (`book.ConfigAi`, envconfig key `AI_TEST_MODE`). When true they wire a substitution client instead of the real OpenAI-backed one — `book.NewSubstitutionAiClient()` (echoes the request back wrapped in `==START/END OF REQUEST==` markers) for the first two, `tts.NewSubstitutionTTSClient()` (returns a tiny silent WAV) for the third — so the pipeline can be exercised without spending real money. `make prepare_chapters`/`make generate_script`/`make generate_tts` default `TEST_MODE=true`; override with e.g. `make generate_tts TEST_MODE=false` (or set `AI_TEST_MODE` directly in `.env`) to hit the real API. `docker-compose.yaml` defaults the same env var to `true` for all three services' containers.

Every AI call's token/character usage is recorded on `Book.TokenUsage` (a `primitive.TokenUsage` JSON column, keyed by model name) via `Repository.AddTokenUsage` — `internal/pricing` turns that into a display cost (`CostUSD`, best-effort `CostEUR` converted via the free Frankfurter exchange-rate API) that `catalog.Service.GetBooks` attaches to each `catalog.BookWithCost`, shown in `frontend-admin`'s catalogue table.

Each AI-driven stage also has a hardcoded per-book EUR budget (`budgetEUR` in each stage's `service.go`: €1 for `prepare_chapters`, €1 for `generate_script`, €2 for `generate_tts`), enforced two ways via `internal/pricing.Calculator`: **pre-call**, `CapOutputTokens` sizes a `max_output_tokens` cap (or, for the fully character-priced TTS stage, `CheckBudget` runs on the exact input size before any request) from the model's price and the estimated/known input size, so a single request can't spend past the whole stage budget on its own; **post-call**, `CheckBudget` runs again on the real reported usage (for `prepare_chapters`, the *aggregate* across all of a book's concurrently-processed chunks) to catch what the pre-call estimate couldn't. Either check failing returns a `*pricing.BudgetExceededError` (`pricing.IsBudgetExceeded`), which each `cmd/*/main.go` queue consumer routes to `book.RecordPermanentFailure` instead of `book.RecordFailure` — this sets `Book.RetryDisabled`, which `catalog.Service.RetryFailedStage` checks and refuses, and which `frontend-admin`'s `CataloguePage` renders as a "Budget exceeded" badge instead of a Retry button.

Standard Go tooling applies for tests/build — there's no test/lint wrapper in the Makefile:
```
go test ./...
go build ./...
go vet ./...
```
`internal/pricing`'s tests make live HTTP calls to the Frankfurter exchange-rate API — expect them to be slow/flaky offline.

### Migrating to another host

`scripts/migrate_mysql.sh`, `scripts/migrate_minio.sh` and `scripts/migrate_all.sh` move the MySQL database and all four object-storage buckets (see Storage layout below) from one host to another. `migrate_minio.sh` is named after the tool it uses (`mc`, the MinIO Client CLI, `brew install minio/stable/mc`) rather than the target — it drives any S3-compatible endpoint, source or destination, including the SeaweedFS S3 gateway this project now runs (its `SRC_MINIO_*`/`DST_MINIO_*` env vars just mean "S3-compatible endpoint"). Configure via `scripts/migrate.env` (copy from `scripts/migrate.env.example`, gitignored) then `source scripts/migrate.env` before running. `migrate_mysql.sh` writes a timestamped dump to `backend/backups/` (also gitignored) before restoring it, so it doubles as a backup step; `migrate_minio.sh` mirrors buckets non-destructively by default (`--exact` also deletes destination objects absent from the source). `migrate_all.sh` runs both and prompts for confirmation before touching the destination host (`--yes` to skip).

This is exactly how the MinIO → SeaweedFS switch below was carried out: `docker-compose.yaml`'s `minio` service was left running and untouched, `seaweed` was brought up alongside it, `migrate_minio.sh` mirrored all four buckets across (verified after with `mc du` — object counts/sizes matched), and only once that was confirmed did the app's `.env` files get repointed at `seaweed`. `minio` and `minio-data/` are still present in the repo/compose file as a fallback and are safe to remove once you're confident in the SeaweedFS data.

### Architecture: queue-driven pipeline

The backend is a set of independent binaries under `cmd/`, each a stage in a book-processing pipeline connected by RabbitMQ. A book moves through stages by publishing its `bookID` (a UUID string) as the message body onto the next queue:

```
admin (HTTP :3001, internal/upload)
   → uploads EPUB to the object store, posts to "split"
epub_parser (worker, internal/parsing)
   → consumes "split": extracts EPUB into text chunks, posts to "prepare"
prepare_chapters (worker, internal/preparation)
   → consumes "prepare": AI-preprocesses each chunk in parallel (errgroup, limit 10), posts to "generate_script"
generate_script (worker, internal/script)
   → consumes "generate_script": AI-merges prepared chunks into one script, posts to "generate_tts"
generate_tts (worker, internal/tts)
   → consumes "generate_tts": calls OpenAI's TTS API to synthesize audio, no further queue message
```

Queue stage names live in `internal/primitive/queue_channel.go` (`Split`, `Prepare`, `GenerateScript`, `GenerateTTS`). Each worker's `main.go` wires an `InitProducer` for the *next* stage and an `InitConsumer` for its *own* stage — read a worker's `main.go` to see both ends of its hop.

`cmd/api` is the separate user-facing HTTP API (auth via `internal/user`, book search/listing via `internal/library` + `internal/catalog`) — it does not participate in the queue pipeline directly.

`cmd/admin` has no auth middleware — it's meant to be run locally by an admin only. It mounts `upload.NewHandler` (POST `/upload`, GET `/search`) and `catalog.NewHandler` under `/books` (GET `/books/`, `/books/audio/:query`) directly on the root app; `frontend-admin`'s Vite dev server proxies `/api` → `http://localhost:3001` the same way `frontend` proxies to `cmd/api`. It used to serve an embedded HTML upload page (`internal/upload/html/upload.html`); that's been replaced by `frontend-admin`.

A `Book`'s progress is tracked both as boolean flags (`Uploaded`/`Parsed`/`Prepared`/`ScriptGenerated`/`TTSGenerated` — persisted via ent) and as the `book.Stage` enum in `internal/book/model.go`; `Repository.UpdateBookStage` is called at the end of each pipeline stage.

**Failure tracking & retry.** Each worker's `queue.InitConsumer` handler wraps its stage's error path with `book.RecordFailure(ctx, repo, bookID, primitive.<Stage>, err)` (see `internal/book/failure.go`), which sets `Book.Failed`/`FailedStage`/`ErrorMessage` via `Repository.MarkBookFailed` before returning — otherwise a stage error is just logged and the message silently dropped (`InitConsumer` auto-acks, there's no dead-lettering). `FailedStage` stores the *queue channel* that failed (`"split"`, `"prepare"`, `"generate_script"`, `"generate_tts"`), not a `Stage` value, because that's what a retry needs to know which queue to re-publish to. `catalog.Service.RetryFailedStage` (admin-only — see below) looks up the book, republishes its ID onto `queueRepos[b.FailedStage]`, and calls `Repository.ClearBookFailure`.

`cmd/admin` is the only binary that constructs `queueRepos` (one `QueueRepo` producer per stage, built in its `main.go`) and passes `enableRetry: true` to `catalog.NewHandler`, which is what mounts `POST /books/:id/retry`. `cmd/api` passes `nil`/`false` for both — retrying a pipeline stage isn't a regular-user action.

### Storage layout (S3-compatible buckets, see `internal/primitive/buckets.go`)

Object storage runs on [SeaweedFS](https://github.com/seaweedfs/seaweedfs) (`seaweed` service in `docker-compose.yaml`, image `chrislusf/seaweedfs`, default command `weed mini -dir=/data` — an all-in-one master+volume+filer+S3-gateway process that seeds these four buckets from `S3_BUCKET` on boot). It replaced MinIO; the app code (`internal/platform/bucket`, via `minio-go`) is unchanged since both speak the same S3 API — only the `MINIO_ENPOINT`/`MINIO_ACCESS_KEY`/`MINIO_SECRET_KEY` env vars (kept under their old MinIO-era names) were repointed. The `minio` service/`minio-data/` volume are kept around as an as-yet-unremoved fallback — see Migrating to another host above.

- `books` — `uploads/<bookID>` (raw EPUB), `chunks/<bookID>/<chunkName>` (split text)
- `scripts` — `<bookID>/preparation/<n>.txt` (per-chunk AI output), `<bookID>/script.txt` (merged script)
- `audio` — `<bookID>` (final synthesized audio)
- `prompts` — static prompt templates (`internal/primitive/prompts.go`), also mirrored in `backend/backup-prompts/*.md` for reference/editing

### Package layout: one package per bounded concern

`internal/book` is the Book *domain* package: the `Book`/`BookDTO`/`Stage` model (`model.go`), the `Repository`/`BucketRepo`/`QueueRepo` ports and their MySQL/SeaweedFS/RabbitMQ implementations (`repository.go`, `bucket_repository.go`, `queue_repository.go`), plus the `OpenAiClient` shared by the two AI-driven stages (`openai_client.go`). It has no `Service` and no HTTP handlers of its own — every pipeline stage and read path is a separate *use-case* package built on top of it:

| Package | Owns | Depends on `book` for |
|---|---|---|
| `internal/library` | Book search/lookup via Google Books — `LibraryAPI`'s only implementation is `GoogleBooksClient` (used by both `cmd/api` and `cmd/admin`, each with their own `GOOGLE_API_BOOKS` key) | `Book` (return type only) |
| `internal/catalog` | saved-book persistence + audio streaming | `Repository`, `BucketRepo` |
| `internal/upload` | EPUB intake (`UploadNewBook`/`GetUploadBook`) | `Repository`, `BucketRepo`, `QueueRepo`, `Stage` |
| `internal/parsing` | EPUB → chapter chunks (`EpubParserImpl`) | same, plus its own `EpubParser` interface |
| `internal/preparation` | per-chunk AI preprocessing | same, plus its own `AiAPI` interface |
| `internal/script` | AI merge of chunks into one script | same, plus its own `AiAPI` interface |
| `internal/tts` | audio synthesis (`OpenAiTTSClient`) | `Repository`, `BucketRepo`, plus its own `TTSAPI` interface |

Each of these exposes a plain `Service` with a positional constructor (`library.NewService(bookAPI)`, `catalog.NewService(repo, bucketRepo)`, ...) — no functional options — so a missing dependency is a compile error, not a nil-pointer panic. Every dependency interface is declared in the same file as the code that needs it (or, for `Repository`/`BucketRepo`/`QueueRepo`, colocated with the implementation in `internal/book`); check a package's `service.go` before assuming a method exists there (e.g. `tts.Service` has no `AiAPI`, only `TTSAPI`).

`internal/user` is the one exception that stays a single self-contained domain package (model, repository, service, middleware, *and* handler) — its handler only ever talks to `user.Service`, so there's no cross-package composition to pull apart.

Every `internal/<domain>` package that has HTTP routes owns its own `handler.go` with an exported `Handler` type and `NewHandler(router fiber.Router, ...services)` constructor that registers its routes directly on the passed-in router group — mirroring `internal/user`. `cmd/*/main.go` stays wiring-only: it builds services, creates the route group (attaching shared middleware like `user.MiddlewareAuth` there), and calls each domain's `NewHandler`. When an endpoint's *own* domain logic is genuinely single-package (e.g. `internal/upload`'s `Upload` handler), its handler composes other services directly rather than being pulled out into a separate wrapper — composing services from a domain's handler is fine, only composing them from a *fake* `cmd`-only domain was the anti-pattern. Endpoints that expose another domain's public contract (e.g. `cmd/api`'s `/books/search/:query` and `/books/:query`, which are `library`'s JSON API) live in that owning domain's handler (`library.NewHandler`) even though they're mounted under a shared route group (`/books`) alongside a sibling domain's handler (`catalog.NewHandler` for `/books/` and `/books/audio/:query`).

`internal/platform/*` are thin infra clients (MySQL via ent, the SeaweedFS/S3 bucket store via `minio-go`, RabbitMQ, Fiber httpserver) each with their own `Config*` struct populated via `envconfig`; `cmd/*/main.go` composes the `Config*` structs it needs into one local `Config`.

### Data layer

Uses [ent](https://entgo.io) for MySQL. Schemas live in `pkg/ent/schema/*.go`; generated code lives in the rest of `pkg/ent/` and must be regenerated with `make ent` after editing a schema. `internal/book/repository.go` and `internal/user/repository.go` wrap the generated ent client behind the domain `Repository` interfaces (declared in the same file as the `*Impl`).

### Auth

`internal/user`: RSA-signed JWT (access + refresh, separate keypairs in `keys/`) via `golang-jwt/jwt`, Argon2 password hashing via `matthewhartstonge/argon2`. Keys are generated with `make genNewKeys` and read from paths in the API's `.env` (`PRIVATE_KEY_PATH` etc).

## Frontend

### Commands (run from `frontend/`)

```
npm run dev        # vite dev server, proxies /api → http://localhost:3000 (see vite.config.ts)
npm run build       # tsc -b && vite build
npm run lint         # oxlint
npm run preview      # preview production build
```

No test runner is configured yet.

### Architecture

Vite + React 19 + TypeScript + React Router 7 + Mantine. `VITE_API_URL` (default `/api`, proxied to the backend `cmd/api` service) is the base for all backend calls — see `src/features/auth/api.ts`.

Folder conventions (see `frontend/good_practices.md` for the full rationale):
- `pages/` — one page per route, orchestrates data + components, no complex UI logic itself
- `components/` — reusable, mostly stateless UI building blocks; no routing or direct API calls
- `layouts/` — structural wrappers used via `<Outlet />` in `routes.tsx` (e.g. `MainLayout`)
- `features/<domain>/` — business-domain modules bundling their own components/hooks/api/types (currently just `features/auth`)
- Routes are centralized in `src/routes.tsx`; protected routes are gated via `components/ProtectedRoute.tsx` wrapping children in the router config, auth state comes from `features/auth/AuthContext.tsx` / `useAuth.ts`

When adding a new business area, prefer a new `features/<domain>/` folder over growing `components/` or `pages/` with domain logic.

## Admin frontend (`frontend-admin/`)

Same stack and folder conventions as `frontend/`, but standalone (own `package.json`, no shared code between the two SPAs) and simpler: no auth, since `cmd/admin` has none.

```
npm run dev        # vite dev server on :5174, proxies /api → http://localhost:3001 (cmd/admin)
npm run build       # tsc -b && vite build
npm run lint         # oxlint
```

- `pages/UploadPage.tsx` — search Google Books (via `cmd/admin`'s `/search`), pick a result, choose the book's language (the one per-book option the pipeline reads — see `Book.Language` / `internal/preparation`, `internal/script`, `internal/tts`), attach the `.epub`, POST to `/upload`.
- `pages/CataloguePage.tsx` — polls `GET /books/` every 5s and renders each book's `Uploaded`/`Parsed`/`Prepared`/`ScriptGenerated`/`TTSGenerated` flags as progress dots; a failed book's next-expected dot renders red (tooltip = `ErrorMessage`, mapped from `FailedStage` via `FAILED_STAGE_TO_PROGRESS_KEY`) with a Retry button that `POST`s `/books/:id/retry`. A Cost column shows each book's `CostUSD`/`CostEUR` (from `catalog.BookWithCost`, see backend AI-cost note above), with a tooltip breaking usage down by model from `TokenUsage`.
- `features/books/` — the one feature module (`api.ts`, `types.ts`, `components/BookCover.tsx`).
