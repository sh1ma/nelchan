-- Migration: 0003_mllm_memory.sql
-- Date: 2025-12-29
-- Description: Add tables for mllm memory enhancement

-- discord_users table
CREATE TABLE IF NOT EXISTS discord_users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    display_name TEXT,
    avatar_url TEXT,
    updated_at TEXT DEFAULT (datetime('now'))
);

-- discord_messages table
CREATE TABLE IF NOT EXISTS discord_messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    timestamp TEXT NOT NULL,
    edited_timestamp TEXT,
    reference_message_id TEXT,
    mention_user_ids TEXT,
    mention_role_ids TEXT,
    has_attachments INTEGER DEFAULT 0,
    is_vectorized INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

-- Indexes for discord_messages
CREATE INDEX IF NOT EXISTS idx_messages_channel ON discord_messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_messages_user ON discord_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON discord_messages(timestamp);
