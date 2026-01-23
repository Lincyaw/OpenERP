-- Create outbox_events table for reliable event delivery
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    event_id UUID NOT NULL UNIQUE,
    event_type VARCHAR(255) NOT NULL,
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 5,
    last_error TEXT,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for querying by tenant and status
CREATE INDEX idx_outbox_tenant_status ON outbox_events(tenant_id, status);

-- Index for querying pending entries by creation time
CREATE INDEX idx_outbox_status_created ON outbox_events(status, created_at);

-- Partial index for retryable entries
CREATE INDEX idx_outbox_next_retry ON outbox_events(next_retry_at) WHERE status = 'FAILED';

-- Index for cleanup queries
CREATE INDEX idx_outbox_processed_at ON outbox_events(processed_at) WHERE status = 'SENT';

COMMENT ON TABLE outbox_events IS 'Transactional outbox for reliable domain event delivery';
COMMENT ON COLUMN outbox_events.status IS 'PENDING, PROCESSING, SENT, FAILED, DEAD';
