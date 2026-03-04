-- =============================================================================
-- Migration: 000001_initial_schema.down.sql
-- Description: Rolls back the initial Neuco schema in strict reverse
--              dependency order (children before parents).
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Triggers (must be dropped before the function they reference)
-- ---------------------------------------------------------------------------
DROP TRIGGER IF EXISTS specs_updated_at          ON specs;
DROP TRIGGER IF EXISTS organizations_updated_at  ON organizations;

-- ---------------------------------------------------------------------------
-- Trigger function
-- ---------------------------------------------------------------------------
DROP FUNCTION IF EXISTS update_updated_at();

-- ---------------------------------------------------------------------------
-- Leaf / child tables first
-- ---------------------------------------------------------------------------
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS copilot_notes;
DROP TABLE IF EXISTS integrations;
DROP TABLE IF EXISTS pipeline_tasks;

-- Remove the deferred FK before dropping pipeline_runs so that we can drop
-- generations and pipeline_runs independently without ordering concerns.
ALTER TABLE IF EXISTS generations
    DROP CONSTRAINT IF EXISTS generations_pipeline_run_fk;

DROP TABLE IF EXISTS pipeline_runs;
DROP TABLE IF EXISTS generations;
DROP TABLE IF EXISTS candidate_signals;
DROP TABLE IF EXISTS specs;
DROP TABLE IF EXISTS feature_candidates;
DROP TABLE IF EXISTS signals;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;

-- ---------------------------------------------------------------------------
-- Extensions
-- ---------------------------------------------------------------------------
DROP EXTENSION IF EXISTS vector;
