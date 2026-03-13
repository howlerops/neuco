-- Revert to original signal type and source check constraints.
ALTER TABLE signals DROP CONSTRAINT IF EXISTS signals_type_check;
ALTER TABLE signals ADD CONSTRAINT signals_type_check
    CHECK (type IN (
        'call_transcript', 'support_ticket', 'feature_request',
        'bug_report', 'review', 'note', 'event'
    ));

ALTER TABLE signals DROP CONSTRAINT IF EXISTS signals_source_check;
ALTER TABLE signals ADD CONSTRAINT signals_source_check
    CHECK (source IN (
        'gong', 'intercom', 'linear', 'jira', 'hubspot',
        'notion', 'salesforce', 'csv', 'slack', 'webhook'
    ));
