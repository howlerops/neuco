-- Expand signals type and source check constraints to match domain model.
-- New signal types for Slack, Linear, Jira, GitHub integrations and analytics.
-- New signal sources for GitHub, Zendesk, Amplitude, Mixpanel, and manual entry.
ALTER TABLE signals DROP CONSTRAINT IF EXISTS signals_type_check;
ALTER TABLE signals ADD CONSTRAINT signals_type_check
    CHECK (type IN (
        'call_transcript', 'support_ticket', 'feature_request',
        'bug_report', 'review', 'note', 'event',
        'user_interview', 'survey_response', 'nps_comment',
        'churn_reason', 'product_review', 'slack_message',
        'github_issue', 'linear_issue', 'jira_issue',
        'usage_anomaly'
    ));

ALTER TABLE signals DROP CONSTRAINT IF EXISTS signals_source_check;
ALTER TABLE signals ADD CONSTRAINT signals_source_check
    CHECK (source IN (
        'gong', 'intercom', 'linear', 'jira', 'hubspot',
        'notion', 'salesforce', 'csv', 'slack', 'webhook',
        'github', 'zendesk', 'amplitude', 'mixpanel', 'manual'
    ));
