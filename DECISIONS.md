# Neuco — Decision Log

## Architectural Decisions

| # | Decision | Choice | Rationale | Date |
|---|----------|--------|-----------|------|
| 1 | Team model | Org → Members → Projects with RBAC | Team-first from day one. Solo users get auto-created personal org. | 2026-03-02 |
| 2 | Admin panel scope | Both SaaS operator + team admin | Operator admin for internal ops; team admin for org owners managing members/billing/usage. | 2026-03-02 |
| 3 | Solo user experience | Async AI co-pilot | System proactively surfaces insights, challenges specs, flags risks. Not AI "personas" — a thoughtful async collaborator. | 2026-03-02 |
| 4 | MVP scope | PRD MVP + team/admin layered in | CSV upload, synthesis, spec gen, React codegen, GitHub PR, pipeline visibility. Team model and admin from the start. | 2026-03-02 |
| 5 | AI co-pilot timing | Full in v1 | Ship proactive insights, spec review, risk flagging as part of the first release. | 2026-03-02 |
| 6 | Repo structure | Monorepo | Single repo: Go backend + SvelteKit frontend + migrations + shared type generation. Simpler at early stage. | 2026-03-02 |
| 7 | River version | Open-source River only | No River Pro dependency. Implement DAG-like behavior with manual job chaining. Design abstractions for easy Pro upgrade later. | 2026-03-02 |
| 8 | HTTP router | Chi (from architecture doc) | Minimal, composable. SSE streaming needs direct ResponseWriter control that GoFr doesn't support well. | 2026-03-02 |
| 9 | LLM framework | Eino (CloudWeGo) | Go-idiomatic, built for production agentic workloads. Not a Python port like LangChainGo. | 2026-03-02 |
| 10 | Frontend framework | SvelteKit + TanStack Query | Established in architecture doc. Auto-generated types/hooks from Go structs. | 2026-03-02 |
| 11 | GitHub repo access | GitHub App (not OAuth) | GitHub App for repo operations (indexing, PR creation). GitHub OAuth for user auth/login only. Better permissions model, webhook support. | 2026-03-02 |
| 12 | Operator admin UI | Same app, gated routes | `/operator/*` routes in the same SvelteKit app, gated by internal token or admin role. Shares design system. | 2026-03-02 |
| 13 | Email service | Resend | Modern DX, React Email templates, simple pricing. Good startup fit. | 2026-03-02 |
| 15 | UI component library | shadcn-svelte | Keep UI code DRY, fast-track styling. Uses Tailwind + Bits UI under the hood. | 2026-03-02 |
| 14 | Pipeline tracking (no River Pro) | Custom tables | `pipeline_runs` + `pipeline_tasks` tables. Workers update task status. Easy to swap for River Pro later. | 2026-03-02 |

## Open Questions

| # | Question | Status | Notes |
|---|----------|--------|-------|
| 1 | Stripe integration for billing — which tier is MVP? | Open | Pricing tiers defined in PRD but billing explicitly excluded from MVP. When to add? |
| 2 | GitHub App: create as org-wide or per-repo installation? | Open | Org-wide is simpler UX but broader permissions. Per-repo is more granular. |
| 3 | Resend: React Email templates or plain HTML? | Open | React Email is nice DX but adds a build step. |
| 4 | S3 bucket structure for repo index cache + CSV uploads? | Open | Need to define key structure. |
| 5 | pgvector: HNSW or IVFFlat index for MVP scale? | Resolved | HNSW — no training step, better query performance. Architecture doc already specifies this. |
