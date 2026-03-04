CREATE TABLE feature_flags (
    key         TEXT PRIMARY KEY,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    metadata    JSONB DEFAULT '{}',
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_by  UUID REFERENCES users(id)
);

INSERT INTO feature_flags (key, enabled, description) VALUES
    ('eino_integration', false, 'Use Eino framework instead of raw HTTP for LLM calls'),
    ('rlm_agent', false, 'Enable RLM transcript agent for long-form signal processing'),
    ('github_app', false, 'Enable GitHub App integration for repo indexing and PR creation'),
    ('email_notifications', false, 'Send email notifications via Resend'),
    ('natural_language_query', false, 'Enable natural language signal query'),
    ('weekly_digest', true, 'Run weekly synthesis digest for all projects');
