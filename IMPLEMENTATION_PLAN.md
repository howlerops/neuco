# Neuco — Implementation Plan

**Created:** 2026-03-02
**Target:** 14-week MVP (team-first, admin-ready, AI co-pilot included)

---

## Guiding Principles

1. **Team-first data model from day zero** — every table is org-scoped, every endpoint is tenant-isolated.
2. **Admin is not an afterthought** — operator and team admin routes/UI are built alongside customer features.
3. **Open-source River with upgrade path** — no River Pro dependency. Custom pipeline tracking tables that can be replaced by River Pro's workflow tables later.
4. **AI co-pilot woven in, not bolted on** — co-pilot behaviors ship alongside each feature area, not as a separate phase.
5. **Type safety end-to-end** — Go structs → auto-generated TypeScript types + Zod schemas + TanStack Query hooks.

---

## Schema Changes from Architecture Doc

The architecture doc's schema assumes single-user ownership. Here's what changes for team-first:

```sql
-- NEW: Organization layer
CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,     -- URL-safe identifier
    plan        TEXT NOT NULL DEFAULT 'starter',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- NEW: Org membership with roles
CREATE TABLE org_members (
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'member',  -- owner | admin | member | viewer
    invited_at  TIMESTAMPTZ DEFAULT NOW(),
    joined_at   TIMESTAMPTZ,
    PRIMARY KEY (org_id, user_id)
);

-- CHANGED: Projects belong to orgs, not users
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,  -- was owner_id
    name        TEXT NOT NULL,
    github_repo TEXT,
    framework   TEXT NOT NULL DEFAULT 'react',
    styling     TEXT NOT NULL DEFAULT 'tailwind',
    created_by  UUID NOT NULL REFERENCES users(id),  -- who created it, not who owns it
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- NEW: Pipeline tracking (replaces River Pro's workflow tables)
CREATE TABLE pipeline_runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,  -- ingest | synthesis | codegen | digest
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending | running | completed | failed
    metadata    JSONB DEFAULT '{}',
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error       TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE pipeline_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    river_job_id    BIGINT,  -- reference to river_job.id
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    attempt         INT DEFAULT 0,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    error           TEXT,
    sort_order      INT NOT NULL DEFAULT 0
);

-- NEW: AI co-pilot insights
CREATE TABLE copilot_notes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,       -- spec | candidate | generation | signal_batch
    target_id   UUID NOT NULL,
    note_type   TEXT NOT NULL,       -- review | risk | suggestion | insight
    content     TEXT NOT NULL,
    metadata    JSONB DEFAULT '{}',
    dismissed   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- NEW: Audit log (team admin + operator visibility)
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID REFERENCES organizations(id),
    user_id     UUID REFERENCES users(id),
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id UUID,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX audit_log_org_idx ON audit_log(org_id, created_at DESC);
```

---

## Phase Breakdown

### Phase 0: Project Scaffolding (Week 1)

**Goal:** Monorepo structure, local dev environment, database running, build tooling working.

| Task | Details |
|------|---------|
| **0.1** Init Go module | `go mod init github.com/neuco-ai/neuco`, set up `cmd/server/`, `cmd/worker/`, `internal/` structure per architecture doc |
| **0.2** SvelteKit project | `neuco-web/` with SvelteKit, TanStack Query, Tailwind CSS, TypeScript |
| **0.3** Docker Compose | postgres:16 with pgvector extension pre-loaded |
| **0.4** Makefile | `run-api`, `run-worker`, `migrate-up`, `migrate-down`, `gen`, `test`, `build` |
| **0.5** Air hot-reload configs | `.air.api.toml`, `.air.worker.toml` |
| **0.6** Initial migrations | All tables from schema above (org, members, projects, signals, candidates, specs, generations, integrations, pipeline_runs, pipeline_tasks, copilot_notes, audit_log) + River base migrations |
| **0.7** Config/env setup | Viper config loading, `.env.example`, env validation on startup |
| **0.8** Type generation script | `scripts/gen_types.go` — Go domain structs → TypeScript types + Zod schemas |
| **0.9** CI skeleton | GitHub Actions: `go test`, `make gen` (fail on diff), Docker build |

**Deliverable:** `make run-api` and `cd neuco-web && npm run dev` both start. Migrations run. Types generate.

---

### Phase 1: Auth & Team Model (Week 2-3)

**Goal:** Users can sign in, create/join orgs, manage projects. RBAC enforced on every endpoint.

| Task | Details |
|------|---------|
| **1.1** GitHub OAuth flow | `/api/v1/auth/github/callback` — exchange code for token, upsert user, create JWT |
| **1.2** JWT middleware | Access token (24h) + refresh token (30d). Extracts user + current org from token claims. |
| **1.3** Auto-create personal org | On first login, create `{username}'s Workspace` org with user as owner |
| **1.4** Org CRUD endpoints | `GET/POST /orgs`, `GET/PATCH /orgs/:id` |
| **1.5** Member management | `POST /orgs/:id/members/invite`, `DELETE /orgs/:id/members/:userId`, `PATCH /orgs/:id/members/:userId` (role change) |
| **1.6** Tenant isolation middleware | `tenant.go` — resolves org from JWT, injects into context. All downstream queries scoped by org_id. |
| **1.7** RBAC middleware | Role checks: owner can do everything, admin can manage members + projects, member can CRUD project content, viewer is read-only |
| **1.8** Project CRUD | `POST/GET/PATCH /orgs/:id/projects`, `GET/PATCH /orgs/:id/projects/:pid` |
| **1.9** Audit logging | Middleware that writes to audit_log for write operations. Captures who did what. |
| **1.10** Frontend: Auth flow | GitHub OAuth redirect, token storage, auth guard on routes |
| **1.11** Frontend: Org switcher | Dropdown in nav for users in multiple orgs |
| **1.12** Frontend: Project list + creation | Project cards, create dialog, framework/styling selector |
| **1.13** Frontend: Team settings page | Member list, invite form, role management (for admin+ roles) |

**Deliverable:** Full auth flow, org creation, project creation, team member invite/manage. All API requests org-scoped.

**Admin tasks in this phase:**
- **Team admin:** Org settings page (name, slug), member management UI
- **Operator admin:** `/operator/orgs` — list all orgs, user counts, created dates

---

### Phase 2: Signal Ingestion (Week 3-5)

**Goal:** Users can upload CSV/text signals. Webhook endpoint receives Make.com payloads. Signals are stored, listed, and searchable.

| Task | Details |
|------|---------|
| **2.1** Signal store | CRUD operations on signals table, paginated list with filters (source, type, date range) |
| **2.2** CSV upload endpoint | `POST /projects/:id/signals/upload` — parse CSV, validate, insert raw signals, enqueue ingest jobs |
| **2.3** Plain text upload | Same endpoint, detect format, split into logical signals |
| **2.4** River setup (open-source) | River client init in both API (insert-only) and worker (processing). Queue config: ingest(5), synthesis(2), codegen(3), default(10) |
| **2.5** Ingest worker (basic) | Parse raw payload, normalize fields, store as structured signal. No RLM agent yet — simple extraction. |
| **2.6** Pipeline tracking | On job insert, create `pipeline_run` + `pipeline_task` rows. Worker updates task status on start/complete/fail. |
| **2.7** Webhook endpoint | `POST /webhooks/make/:projectId/:secret` — validates secret, inserts signal + ingest job atomically (River transactional insert) |
| **2.8** Embedder setup | Eino client init, OpenAI text-embedding-3-small integration. Batch embedding worker (100 signals at a time). |
| **2.9** Frontend: Signals page | Upload UI (drag-drop CSV/text), signal list with filters, signal detail view |
| **2.10** Frontend: Upload progress | Show ingest pipeline progress using pipeline tracking data |

**Deliverable:** Signals flowing in via CSV upload and webhook. Stored with embeddings. Visible in UI.

**Admin tasks in this phase:**
- **Operator admin:** `/operator/signals` — signal volume across all tenants, ingestion error rates

---

### Phase 3: Synthesis & Candidates (Week 5-7)

**Goal:** Signals are clustered into themes, scored, and surfaced as feature candidates. On-demand synthesis works.

| Task | Details |
|------|---------|
| **3.1** Eino client full setup | Sonnet for spec/codegen, Haiku for theme naming/sub-queries, embedder |
| **3.2** RLM ingest agent | Replace basic ingest worker with full ReAct agent (peek, search, sub_query, emit_signal tools). For long-form transcripts only; CSV signals skip this. |
| **3.3** Clustering service | pgvector k-means clustering in pure Go. Groups signals by embedding similarity. |
| **3.4** Theme naming | Haiku call per cluster: generate title + problem summary from representative signals |
| **3.5** Candidate scoring | `frequency * recency * segment_weight * churn_risk` scoring formula |
| **3.6** Synthesis workflow | Chain: fetch_signals → embed_missing → cluster → name → score → write_candidates. Manual job chaining (no River Pro DAGs — each job enqueues the next on completion). |
| **3.7** On-demand synthesis | `POST /projects/:id/candidates/refresh` — triggers synthesis workflow immediately |
| **3.8** Weekly digest (cron) | River periodic job: Monday 8am UTC, runs synthesis for all projects with new signals |
| **3.9** Natural language query | `POST /projects/:id/signals/query` — filtered semantic search over signals, returns ranked themes |
| **3.10** Frontend: Candidates page | Ranked list of feature candidates, score breakdown, supporting signal excerpts |
| **3.11** Frontend: Candidate detail | Drill-down to individual signals, source attribution, sentiment |
| **3.12** AI Co-pilot: Synthesis insights | After synthesis completes, generate co-pilot notes: "3 of your top 5 themes relate to onboarding — consider a dedicated onboarding epic" |

**Deliverable:** Working synthesis pipeline. Feature candidates ranked and visible. Co-pilot providing synthesis-level insights.

**Admin tasks in this phase:**
- **Operator admin:** Synthesis pipeline health dashboard, LLM token usage tracking
- **Team admin:** Usage stats on project dashboard (signals processed, candidates generated)

---

### Phase 4: Spec Generation (Week 7-9)

**Goal:** Users can generate structured specs from candidates, edit them inline, and get AI co-pilot review.

| Task | Details |
|------|---------|
| **4.1** Spec generation pipeline | Candidate + supporting signals → Sonnet call → structured spec (problem, solution, user stories, acceptance criteria, out of scope, UI changes, data model changes, open questions) |
| **4.2** Spec store | Versioned specs — each edit bumps version, old versions kept |
| **4.3** Spec generation endpoint | `POST /projects/:id/candidates/:cId/spec/generate` — enqueues spec generation job |
| **4.4** Spec CRUD | `GET/PATCH /projects/:id/candidates/:cId/spec` — read and inline edit |
| **4.5** Frontend: Spec editor | Rich inline editor for all spec fields. Version history sidebar. |
| **4.6** Frontend: Spec generation progress | SSE-based live progress while spec generates |
| **4.7** AI Co-pilot: Spec review | After spec generation, automatically generate review notes: risks, ambiguities, missing acceptance criteria, scope concerns. Stored as `copilot_notes`. |
| **4.8** AI Co-pilot: Spec challenge | When user edits a spec, co-pilot can flag: "This change contradicts signal evidence from 3 customers — are you sure?" |
| **4.9** Frontend: Co-pilot notes panel | Sidebar or inline cards showing co-pilot review notes on specs. Dismissable. |

**Deliverable:** Full spec generation, editing, versioning. Co-pilot reviews every spec and flags risks.

**Admin tasks in this phase:**
- **Team admin:** Spec generation history, who generated/edited which specs

---

### Phase 5: Code Generation & GitHub (Week 9-11)

**Goal:** Specs produce React/Next.js components + Storybook stories, committed as draft PRs.

| Task | Details |
|------|---------|
| **5.1** GitHub connection | GitHub App installation flow (per org). Store installation ID, repo access tokens. |
| **5.2** Codebase indexer | Clone/fetch repo → index component library, design tokens, existing stories, TS types, styling approach. Cache in S3 as `{repo}:{sha}.json`. |
| **5.3** Context builder | Select top-N similar components + stories as few-shot examples for code generation |
| **5.4** Code generation pipeline | Spec + codebase context → Sonnet call → component file, story file, types file, test scaffold |
| **5.5** Output parser/validator | Validate file paths, strip markdown fences, ensure valid TypeScript/JSX |
| **5.6** PR creation | Create branch `neuco/feature-slug-YYYY-MM-DD`, commit files, open draft PR with structured description linking back to signals |
| **5.7** Codegen workflow (job chain) | fetch_spec → index_repo → build_context → generate_code → create_pr → notify. Manual chaining with pipeline_run tracking. |
| **5.8** SSE progress stream | `GET /projects/:id/generations/:gId/stream` — live task-by-task progress |
| **5.9** Frontend: Generate button | On spec page, "Generate Code" button → progress view → PR link |
| **5.10** Frontend: Generation history | List of all generations for a project, status, PR links |
| **5.11** AI Co-pilot: PR review notes | After PR is created, co-pilot generates review notes: "Component follows project patterns", "Missing error state handling", "Consider accessibility for this input" |

**Deliverable:** End-to-end signal → spec → code → PR. SSE progress. Co-pilot PR review.

**Admin tasks in this phase:**
- **Operator admin:** LLM cost tracking per codegen, generation success/failure rates
- **Team admin:** Generation history, PR merge rate stats

---

### Phase 6: Pipeline Visibility & Dashboard (Week 11-12)

**Goal:** Full transparency into every pipeline Neuco runs. Dashboard with aggregate stats.

| Task | Details |
|------|---------|
| **6.1** Pipeline list endpoint | `GET /projects/:id/pipelines` — all pipeline_runs with task summaries |
| **6.2** Pipeline detail endpoint | `GET /projects/:id/pipelines/:runId` — full task breakdown with status, duration, errors |
| **6.3** Retry endpoint | `POST /projects/:id/pipelines/:runId/retry` — re-enqueue failed tasks only |
| **6.4** Project stats endpoint | `GET /projects/:id/stats` — signals ingested, themes found, PRs created, avg gen time, success rate |
| **6.5** Frontend: Pipeline activity feed | Chronological list, filterable by type and status |
| **6.6** Frontend: Pipeline detail view | Step-by-step task visualization (linear DAG), status colors, duration, error messages |
| **6.7** Frontend: Dashboard | Stats cards, recent activity, "Needs Attention" card for failed pipelines |
| **6.8** Frontend: Retry UI | One-click retry on failed pipelines |

**Deliverable:** Full pipeline transparency. Users see exactly what Neuco is doing at all times.

---

### Phase 7: Admin Panels (Week 12-13)

**Goal:** Team admin and operator admin fully functional.

| Task | Details |
|------|---------|
| **7.1** Team admin: Org settings | Name, slug, plan display, danger zone (delete org) |
| **7.2** Team admin: Member management | Full member list, invite via email, role changes, remove members |
| **7.3** Team admin: Usage dashboard | Signals this month, PRs created, pipeline runs, LLM token usage estimate |
| **7.4** Team admin: Audit log viewer | Filterable log of all team actions |
| **7.5** Operator admin: Auth | Separate auth flow — internal token or admin-flagged user |
| **7.6** Operator admin: Tenant list | All orgs with member counts, plan, usage metrics, created date |
| **7.7** Operator admin: System health | Total pipelines running, failure rates, queue depths, LLM cost breakdown |
| **7.8** Operator admin: User management | User list, org associations, ability to impersonate (for support) |
| **7.9** Operator admin: Feature flags | Simple key-value feature flag system for gradual rollout |

**Deliverable:** Both admin panels fully functional. Operator can manage the SaaS. Team admins can manage their orgs.

---

### Phase 8: Integration Hub & Polish (Week 13-14)

**Goal:** Integration management UI, Make.com webhook setup guide, polish and hardening.

| Task | Details |
|------|---------|
| **8.1** Integrations CRUD | `GET/POST/DELETE /projects/:id/integrations` — manage webhook connections |
| **8.2** Webhook secret generation | Auto-generate per-integration secrets, display once |
| **8.3** Frontend: Integrations page | Integration cards with status, last sync, volume, enable/disable toggle |
| **8.4** Make.com setup guide | In-app guide for configuring Make scenarios to push to Neuco webhooks |
| **8.5** Rate limiting | Per-project rate limits on generation endpoint (10/hr), webhook endpoint |
| **8.6** Error handling hardening | Consistent error responses, proper HTTP status codes, user-friendly error messages |
| **8.7** Frontend polish | Loading states, empty states, error states for all views |
| **8.8** E2E smoke tests | Critical path: login → create project → upload CSV → synthesize → generate spec → generate code |

**Deliverable:** Production-ready MVP.

---

## River Open-Source → Pro Upgrade Path

Since we're using open-source River, we need to handle two things River Pro provides for free:

### 1. DAG Workflows → Manual Job Chaining

**Pattern:** Each worker, on successful completion, enqueues the next job in the chain.

```go
func (w *ClusterThemesWorker) Work(ctx context.Context, job *river.Job[ClusterThemesJobArgs]) error {
    // ... do clustering work ...

    // Chain: enqueue next step
    _, err := w.riverClient.Insert(ctx, NameThemesJobArgs{
        ProjectID:     job.Args.ProjectID,
        PipelineRunID: job.Args.PipelineRunID,
    }, &river.InsertOpts{Queue: "synthesis"})
    return err
}
```

**Pipeline tracking:** Our custom `pipeline_runs` + `pipeline_tasks` tables track the overall workflow state. Each worker updates its task row on start/complete/fail. The pipeline_run status is derived from its tasks.

### 2. Workflow State → Custom Pipeline Tables

When upgrading to River Pro:
1. Replace `pipeline_runs`/`pipeline_tasks` with River Pro's `river_workflow` table
2. Replace manual job chaining with `riverpro.WorkflowT` DAG definitions
3. Replace pipeline status queries with River Pro's workflow APIs
4. Mount River UI for internal ops

The domain-level pipeline visibility endpoints (`/pipelines`, `/stats`) stay the same — just the data source changes.

---

## API Route Structure (Updated for Team Model)

```
POST   /api/v1/auth/github/callback
GET    /api/v1/auth/me
POST   /api/v1/auth/refresh

# Orgs
GET    /api/v1/orgs
POST   /api/v1/orgs
GET    /api/v1/orgs/:orgId
PATCH  /api/v1/orgs/:orgId

# Members
GET    /api/v1/orgs/:orgId/members
POST   /api/v1/orgs/:orgId/members/invite
PATCH  /api/v1/orgs/:orgId/members/:userId
DELETE /api/v1/orgs/:orgId/members/:userId

# Projects (org-scoped)
GET    /api/v1/orgs/:orgId/projects
POST   /api/v1/orgs/:orgId/projects
GET    /api/v1/orgs/:orgId/projects/:id
PATCH  /api/v1/orgs/:orgId/projects/:id

# Signals
POST   /api/v1/projects/:id/signals/upload
GET    /api/v1/projects/:id/signals
DELETE /api/v1/projects/:id/signals/:signalId
POST   /api/v1/projects/:id/signals/query

# Candidates
GET    /api/v1/projects/:id/candidates
POST   /api/v1/projects/:id/candidates/refresh
PATCH  /api/v1/projects/:id/candidates/:cId

# Specs
GET    /api/v1/projects/:id/candidates/:cId/spec
POST   /api/v1/projects/:id/candidates/:cId/spec/generate
PATCH  /api/v1/projects/:id/candidates/:cId/spec

# Generations
POST   /api/v1/projects/:id/candidates/:cId/generate
GET    /api/v1/projects/:id/generations
GET    /api/v1/projects/:id/generations/:gId
GET    /api/v1/projects/:id/generations/:gId/stream   # SSE

# Pipelines
GET    /api/v1/projects/:id/pipelines
GET    /api/v1/projects/:id/pipelines/:runId
POST   /api/v1/projects/:id/pipelines/:runId/retry
GET    /api/v1/projects/:id/stats

# Co-pilot
GET    /api/v1/projects/:id/copilot/notes?target_type=spec&target_id=xxx
PATCH  /api/v1/projects/:id/copilot/notes/:noteId   # dismiss

# Integrations
GET    /api/v1/projects/:id/integrations
POST   /api/v1/projects/:id/integrations
DELETE /api/v1/projects/:id/integrations/:iId

# Webhooks (no auth — secret in URL)
POST   /api/v1/webhooks/make/:projectId/:secret

# Team Admin
GET    /api/v1/orgs/:orgId/audit-log
GET    /api/v1/orgs/:orgId/usage

# Operator Admin (internal token auth)
GET    /operator/orgs
GET    /operator/orgs/:orgId
GET    /operator/users
GET    /operator/health
GET    /operator/pipelines
GET    /operator/flags
PATCH  /operator/flags/:key
GET    /internal/river/*   # River UI (when upgraded to Pro)
```

---

## Frontend Route Structure

```
/login                          # GitHub OAuth
/                               # Redirect to first org's dashboard

/(app)/
  [orgSlug]/
    dashboard/                  # Stats + recent activity
    projects/                   # Project list
    projects/[id]/
      signals/                  # Signal list + upload
      candidates/               # Feature candidate list
      candidates/[cId]/
        spec/                   # Spec editor + co-pilot notes
        generate/               # Code generation progress
      generations/              # Generation history
      pipelines/                # Pipeline activity feed
      pipelines/[runId]/        # Pipeline detail
      integrations/             # Integration management
    settings/                   # Org settings + members (team admin)
    settings/members/
    settings/audit-log/
    settings/usage/

/(operator)/                    # Operator admin (internal only)
  orgs/
  users/
  health/
  pipelines/
  flags/
```
