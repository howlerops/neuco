-- =============================================================================
-- Migration: 000002_github_app_installation.up.sql
-- Description: Store the GitHub App installation ID per organization.
--              When a user installs the Neuco GitHub App on their GitHub org,
--              GitHub calls back with an installation_id that is stored here.
--              The installation_id is used by the codegen workers to obtain
--              short-lived installation access tokens via the GitHub App JWT
--              exchange.
-- =============================================================================

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS github_installation_id BIGINT;

COMMENT ON COLUMN organizations.github_installation_id IS
    'GitHub App installation ID set when the org owner installs the Neuco App. '
    'NULL means the App has not been installed for this org.';
