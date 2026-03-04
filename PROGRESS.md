# Neuco — Progress Tracker

## Phase 0: Project Scaffolding
- [x] 0.1 Init Go module (`cmd/server/`, `cmd/worker/`, `internal/`)
- [x] 0.2 SvelteKit project (`neuco-web/`)
- [x] 0.3 Docker Compose (postgres:16 + pgvector)
- [x] 0.4 Makefile (run-api, run-worker, migrate-up, gen, test, build)
- [x] 0.5 Air hot-reload configs
- [x] 0.6 Initial migrations (all tables + River base)
- [x] 0.7 Config/env setup (Viper, .env.example)
- [x] 0.8 Type generation script (Go → TypeScript)
- [x] 0.9 CI skeleton (GitHub Actions)

## Phase 1: Auth & Team Model
- [x] 1.1 GitHub OAuth flow
- [x] 1.2 JWT middleware (access + refresh tokens)
- [x] 1.3 Auto-create personal org on first login
- [x] 1.4 Org CRUD endpoints
- [x] 1.5 Member management (invite, remove, role change)
- [x] 1.6 Tenant isolation middleware
- [x] 1.7 RBAC middleware
- [x] 1.8 Project CRUD (org-scoped)
- [x] 1.9 Audit logging middleware
- [x] 1.10 Frontend: Auth flow
- [x] 1.11 Frontend: Org switcher (in app layout)
- [x] 1.12 Frontend: Project list + creation
- [x] 1.13 Frontend: Team settings page

## Phase 2: Signal Ingestion
- [x] 2.1 Signal store (CRUD, paginated list, filters)
- [x] 2.2 CSV upload endpoint
- [x] 2.3 Plain text upload
- [x] 2.4 River setup (open-source, both binaries)
- [x] 2.5 Ingest worker (basic + RLM agent for long content)
- [x] 2.6 Pipeline tracking (pipeline_runs + pipeline_tasks)
- [x] 2.7 Webhook endpoint (Make.com)
- [x] 2.8 Embedder setup (OpenAI text-embedding-3-small, batch support)
- [x] 2.9 Frontend: Signals page (upload + list + filters + pagination)
- [x] 2.10 Frontend: Upload progress

## Phase 3: Synthesis & Candidates
- [x] 3.1 LLM client abstraction (retry, backoff, tool calling support)
- [x] 3.2 RLM ingest agent (ReAct loop: peek/search/sub_query/emit_signal, max 40 steps)
- [x] 3.3 Clustering service (pgvector k-means in pure Go)
- [x] 3.4 Theme naming (Haiku per cluster)
- [x] 3.5 Candidate scoring formula
- [x] 3.6 Synthesis workflow (manual job chain)
- [x] 3.7 On-demand synthesis endpoint
- [x] 3.8 Weekly digest (River periodic job)
- [x] 3.9 Natural language signal query (embedding similarity search via pgvector)
- [x] 3.10 Frontend: Candidates page (ranked cards, status update, co-pilot notes)
- [x] 3.11 Frontend: Candidate detail
- [x] 3.12 AI Co-pilot: Synthesis insights

## Phase 4: Spec Generation
- [x] 4.1 Spec generation pipeline (Sonnet)
- [x] 4.2 Spec store (versioned)
- [x] 4.3 Spec generation endpoint
- [x] 4.4 Spec CRUD
- [x] 4.5 Frontend: Spec editor (full form with user stories, acceptance criteria)
- [x] 4.6 Frontend: Spec generation progress
- [x] 4.7 AI Co-pilot: Spec review notes
- [x] 4.8 AI Co-pilot: Spec change challenges
- [x] 4.9 Frontend: Co-pilot notes panel

## Phase 5: Code Generation & GitHub
- [x] 5.1 GitHub App installation flow (JWT auth, installation token exchange, handler)
- [x] 5.2 Codebase indexer (recursive tree walk, component/story/type/token detection)
- [x] 5.3 Context builder (few-shot selection with token budget)
- [x] 5.4 Code generation pipeline (Sonnet with codebase context)
- [x] 5.5 Output parser/validator
- [x] 5.6 PR creation (Git tree API: blobs → tree → commit → ref → draft PR)
- [x] 5.7 Codegen workflow (manual job chain with index passthrough)
- [x] 5.8 SSE progress stream
- [x] 5.9 Frontend: Generate page (live pipeline progress, success/failure)
- [x] 5.10 Frontend: Generation history + detail (files, co-pilot review)
- [x] 5.11 AI Co-pilot: PR review notes

## Phase 6: Pipeline Visibility & Dashboard
- [x] 6.1 Pipeline list endpoint
- [x] 6.2 Pipeline detail endpoint
- [x] 6.3 Retry endpoint
- [x] 6.4 Project stats endpoint
- [x] 6.5 Frontend: Pipeline activity feed (filterable)
- [x] 6.6 Frontend: Pipeline detail view (task timeline, auto-refresh)
- [x] 6.7 Frontend: Dashboard (stats + activity + needs attention)
- [x] 6.8 Frontend: Retry UI

## Phase 7: Admin Panels
- [x] 7.1 Team admin: Org settings
- [x] 7.2 Team admin: Member management (invite, role change, remove)
- [x] 7.3 Team admin: Usage dashboard (via project stats)
- [x] 7.4 Team admin: Audit log viewer (filterable, paginated)
- [x] 7.5 Operator admin: Auth (internal token middleware)
- [x] 7.6 Operator admin: Tenant list
- [x] 7.7 Operator admin: System health (DB, queue depths)
- [x] 7.8 Operator admin: User management
- [x] 7.9 Operator admin: Feature flags (DB-backed, toggle UI)

## Phase 8: Integration Hub & Polish
- [x] 8.1 Integrations CRUD (backend)
- [x] 8.2 Webhook secret generation
- [x] 8.3 Frontend: Integrations page (add, webhook secret, delete)
- [x] 8.4 Make.com setup guide (6-step in-app guide with examples)
- [x] 8.5 Rate limiting
- [x] 8.6 Error handling hardening
- [x] 8.7 Frontend polish (loading/empty/error states in all pages)
- [x] 8.8 E2E smoke tests (11 test functions covering auth, CRUD, upload, RBAC, tenant isolation)

---
## Quality Status
- Go build: **0 errors**
- Go vet: **0 warnings**
- SvelteKit check: **0 errors, 0 warnings**
- SvelteKit build: **PASS**
- TODOs/FIXMEs: **0**
- Mocks/stubs: **0**
- E2E tests: **11 test functions with subtests**

## All 89 tasks complete.
