CREATE TABLE IF NOT EXISTS routing_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT,
    user_id BIGINT,
    api_key_id BIGINT,
    group_id BIGINT,
    account_id BIGINT,
    model TEXT,
    inbound_endpoint TEXT,
    upstream_endpoint TEXT,
    request_type TEXT,
    stream BOOLEAN NOT NULL DEFAULT FALSE,
    routing_policy_version TEXT NOT NULL DEFAULT 'routing-audit-v1',
    precheck_rules_version TEXT NOT NULL DEFAULT 'precheck-v1',
    candidate_pools JSONB NOT NULL DEFAULT '[]'::jsonb,
    selected_pool TEXT,
    selected_account_id BIGINT,
    decision_reason TEXT,
    fallback_reason TEXT,
    privacy_decision TEXT,
    privacy_redirect BOOLEAN NOT NULL DEFAULT FALSE,
    quota_redirect BOOLEAN NOT NULL DEFAULT FALSE,
    error_redirect BOOLEAN NOT NULL DEFAULT FALSE,
    schedule_layer TEXT,
    sticky_previous_hit BOOLEAN NOT NULL DEFAULT FALSE,
    sticky_session_hit BOOLEAN NOT NULL DEFAULT FALSE,
    candidate_count INTEGER NOT NULL DEFAULT 0,
    top_k INTEGER NOT NULL DEFAULT 0,
    scheduler_latency_ms BIGINT NOT NULL DEFAULT 0,
    load_skew DOUBLE PRECISION NOT NULL DEFAULT 0,
    plus_pool_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    relay_quota_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    final_route TEXT,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    error_type TEXT,
    upstream_status INTEGER,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    image_output_tokens INTEGER NOT NULL DEFAULT 0,
    total_cost NUMERIC(18, 8) NOT NULL DEFAULT 0,
    actual_cost NUMERIC(18, 8) NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routing_audit_logs_created_at
    ON routing_audit_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_routing_audit_logs_user_created
    ON routing_audit_logs (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_routing_audit_logs_api_key_created
    ON routing_audit_logs (api_key_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_routing_audit_logs_selected_pool_created
    ON routing_audit_logs (selected_pool, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_routing_audit_logs_decision_reason_created
    ON routing_audit_logs (decision_reason, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_routing_audit_logs_policy_created
    ON routing_audit_logs (routing_policy_version, created_at DESC);
