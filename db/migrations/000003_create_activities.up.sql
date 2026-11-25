CREATE TABLE IF NOT EXISTS activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_item_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, provider, provider_item_id, occurred_at)
);

CREATE INDEX IF NOT EXISTS idx_activities_user_occurred ON activities (user_id, occurred_at DESC);
