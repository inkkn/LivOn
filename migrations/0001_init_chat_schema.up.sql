-- Anonymous Users (Internal identity tied to phone)
CREATE TABLE users (
    id TEXT PRIMARY KEY, 
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Conversations
CREATE TABLE conversations (
    id          UUID PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Session-Based Participants (The Privacy Bridge)
CREATE TABLE conversation_participants (
    id              UUID PRIMARY KEY, -- The public 'sender_id'
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at         TIMESTAMPTZ, -- Only set if they click "Leave"
    
    CONSTRAINT unique_active_participant UNIQUE (user_id, conversation_id, left_at)
);

-- Optimized for the 5-minute resume lookup
CREATE INDEX idx_resume_session 
ON conversation_participants (user_id, conversation_id, last_seen_at DESC)
WHERE left_at IS NULL;

-- Sequences for Guaranteed Ordering
CREATE TABLE conversation_sequences (
    conversation_id UUID PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
    last_seq        BIGINT NOT NULL DEFAULT 0
);
ALTER TABLE conversation_sequences
ADD CONSTRAINT seq_conversation_not_zero
CHECK (conversation_id <> '00000000-0000-0000-0000-000000000000');

-- Persistent Messages
CREATE TABLE messages (
    id              UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL REFERENCES conversation_participants(id), 
    seq             BIGINT NOT NULL, 
    payload         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uniq_conversation_seq UNIQUE (conversation_id, seq)
);

-- Visibility Index (Join-Onward + Recent 5-min Window)
CREATE INDEX idx_messages_visibility 
ON messages (conversation_id, created_at DESC);