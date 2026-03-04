-- =============================================================================
-- Migration: 000001_initial_schema.up.sql
-- Description: Initial schema for Neuco - product intelligence platform
-- PostgreSQL 16 | pgvector extension
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. Extensions
-- ---------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS vector;


-- ---------------------------------------------------------------------------
-- 2. users
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id   TEXT        NOT NULL UNIQUE,
    github_login TEXT        NOT NULL,
    email       TEXT,
    avatar_url  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- ---------------------------------------------------------------------------
-- 3. organizations
-- ---------------------------------------------------------------------------
CREATE TABLE organizations (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL UNIQUE,
    plan       TEXT        NOT NULL DEFAULT 'starter'
                           CHECK (plan IN ('starter', 'builder', 'team', 'enterprise')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- ---------------------------------------------------------------------------
-- 4. org_members
-- ---------------------------------------------------------------------------
CREATE TABLE org_members (
    org_id     UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    role       TEXT        NOT NULL DEFAULT 'member'
                           CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    invited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_at  TIMESTAMPTZ,
    PRIMARY KEY (org_id, user_id)
);


-- ---------------------------------------------------------------------------
-- 5. projects
-- ---------------------------------------------------------------------------
CREATE TABLE projects (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    github_repo TEXT,
    framework   TEXT        NOT NULL DEFAULT 'react'
                            CHECK (framework IN ('react', 'nextjs', 'vue', 'svelte', 'angular', 'solid')),
    styling     TEXT        NOT NULL DEFAULT 'tailwind'
                            CHECK (styling IN ('tailwind', 'css_modules', 'styled_components', 'vanilla')),
    created_by  UUID        NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- ---------------------------------------------------------------------------
-- 6. signals
-- ---------------------------------------------------------------------------
CREATE TABLE signals (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source      TEXT        NOT NULL
                            CHECK (source IN ('gong', 'intercom', 'linear', 'jira', 'hubspot',
                                              'notion', 'salesforce', 'csv', 'slack', 'webhook')),
    source_ref  TEXT,
    type        TEXT        NOT NULL
                            CHECK (type IN ('call_transcript', 'support_ticket', 'feature_request',
                                            'bug_report', 'review', 'note', 'event')),
    content     TEXT        NOT NULL,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    embedding   vector(1536)
);

-- Composite index: primary access pattern for a project's signals feed
CREATE INDEX signals_project_idx
    ON signals (project_id, ingested_at DESC);

-- HNSW index for approximate nearest-neighbour vector search (cosine distance).
-- ef_construction=200 trades index-build time for higher recall; m=16 is
-- the default connectivity parameter and suits 1536-dim embeddings well.
CREATE INDEX signals_embedding_idx
    ON signals USING hnsw (embedding vector_cosine_ops)
    WITH (ef_construction = 200, m = 16);

-- Filtering indexes for source and type facets within a project
CREATE INDEX signals_source_idx ON signals (project_id, source);
CREATE INDEX signals_type_idx   ON signals (project_id, type);


-- ---------------------------------------------------------------------------
-- 7. feature_candidates
-- ---------------------------------------------------------------------------
CREATE TABLE feature_candidates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title           TEXT        NOT NULL,
    problem_summary TEXT,
    signal_count    INT         NOT NULL DEFAULT 0,
    score           FLOAT       NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'new'
                                CHECK (status IN ('new', 'specced', 'in_progress', 'shipped', 'rejected')),
    suggested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    centroid        vector(1536)
);


-- ---------------------------------------------------------------------------
-- 8. candidate_signals  (junction: feature_candidates <-> signals)
-- ---------------------------------------------------------------------------
CREATE TABLE candidate_signals (
    candidate_id UUID  NOT NULL REFERENCES feature_candidates(id) ON DELETE CASCADE,
    signal_id    UUID  NOT NULL REFERENCES signals(id)            ON DELETE CASCADE,
    relevance    FLOAT,
    PRIMARY KEY (candidate_id, signal_id)
);


-- ---------------------------------------------------------------------------
-- 9. specs
-- ---------------------------------------------------------------------------
CREATE TABLE specs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id         UUID        NOT NULL REFERENCES feature_candidates(id) ON DELETE CASCADE,
    project_id           UUID        NOT NULL REFERENCES projects(id)            ON DELETE CASCADE,
    problem_statement    TEXT,
    proposed_solution    TEXT,
    user_stories         JSONB       NOT NULL DEFAULT '[]',
    acceptance_criteria  JSONB       NOT NULL DEFAULT '[]',
    out_of_scope         JSONB       NOT NULL DEFAULT '[]',
    ui_changes           TEXT,
    data_model_changes   TEXT,
    open_questions       JSONB       NOT NULL DEFAULT '[]',
    version              INT         NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- ---------------------------------------------------------------------------
-- 10. generations
-- Note: pipeline_run_id FK is deferred; constraint added after pipeline_runs
-- is created in step 11.
-- ---------------------------------------------------------------------------
CREATE TABLE generations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    spec_id         UUID        NOT NULL REFERENCES specs(id),
    project_id      UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    pipeline_run_id UUID,                          -- FK added below after pipeline_runs
    status          TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    branch_name     TEXT,
    pr_url          TEXT,
    pr_number       INT,
    files           JSONB       NOT NULL DEFAULT '[]',
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX generations_pipeline_idx ON generations (pipeline_run_id);


-- ---------------------------------------------------------------------------
-- 11. pipeline_runs
-- ---------------------------------------------------------------------------
CREATE TABLE pipeline_runs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type        TEXT        NOT NULL
                            CHECK (type IN ('ingest', 'synthesis', 'codegen', 'digest', 'spec_gen', 'copilot')),
    status      TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    metadata    JSONB       NOT NULL DEFAULT '{}',
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX pipeline_runs_project_idx ON pipeline_runs (project_id, created_at DESC);


-- ---------------------------------------------------------------------------
-- 12. pipeline_tasks
-- ---------------------------------------------------------------------------
CREATE TABLE pipeline_tasks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id UUID        NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    river_job_id    BIGINT,
    name            TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    attempt         INT         NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    error           TEXT,
    sort_order      INT         NOT NULL DEFAULT 0
);

CREATE INDEX pipeline_tasks_run_idx ON pipeline_tasks (pipeline_run_id, sort_order);


-- ---------------------------------------------------------------------------
-- Now that pipeline_runs exists, wire up the deferred FK on generations
-- ---------------------------------------------------------------------------
ALTER TABLE generations
    ADD CONSTRAINT generations_pipeline_run_fk
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id);


-- ---------------------------------------------------------------------------
-- 13. integrations
-- ---------------------------------------------------------------------------
CREATE TABLE integrations (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider       TEXT        NOT NULL,
    webhook_secret TEXT,
    config         JSONB       NOT NULL DEFAULT '{}',
    last_sync_at   TIMESTAMPTZ,
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- ---------------------------------------------------------------------------
-- 14. copilot_notes
-- ---------------------------------------------------------------------------
CREATE TABLE copilot_notes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    target_type TEXT        NOT NULL
                            CHECK (target_type IN ('spec', 'candidate', 'generation',
                                                   'signal_batch', 'synthesis')),
    target_id   UUID        NOT NULL,
    note_type   TEXT        NOT NULL
                            CHECK (note_type IN ('review', 'risk', 'suggestion', 'insight')),
    content     TEXT        NOT NULL,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    dismissed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX copilot_notes_target_idx ON copilot_notes (project_id, target_type, target_id);


-- ---------------------------------------------------------------------------
-- 15. audit_log
-- ---------------------------------------------------------------------------
CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        REFERENCES organizations(id),
    user_id     UUID        REFERENCES users(id),
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    resource_id UUID,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary query pattern: fetch recent audit events for an org
CREATE INDEX audit_log_org_idx      ON audit_log (org_id,   created_at DESC);
-- Secondary pattern: look up all events touching a specific resource
CREATE INDEX audit_log_resource_idx ON audit_log (resource, resource_id);


-- ---------------------------------------------------------------------------
-- 16. updated_at trigger function + triggers
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to every table that carries an updated_at column
CREATE TRIGGER organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER specs_updated_at
    BEFORE UPDATE ON specs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
