# Data Model: mllm Memory Enhancement

**Date**: 2025-12-29
**Branch**: `001-mllm-memory-enhancement`

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        discord_users                             │
├─────────────────────────────────────────────────────────────────┤
│ PK id: TEXT                    -- Discord User ID               │
│    username: TEXT NOT NULL     -- Discord username              │
│    display_name: TEXT          -- Display name (nullable)       │
│    avatar_url: TEXT            -- Avatar URL (nullable)         │
│    updated_at: TEXT            -- Last update timestamp         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 1:N
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       discord_messages                           │
├─────────────────────────────────────────────────────────────────┤
│ PK id: TEXT                    -- Discord Message ID            │
│    channel_id: TEXT NOT NULL   -- Discord Channel ID            │
│ FK user_id: TEXT NOT NULL      -- Discord User ID               │
│    content: TEXT NOT NULL      -- Message content               │
│    timestamp: TEXT NOT NULL    -- ISO 8601 timestamp            │
│    edited_timestamp: TEXT      -- Edit timestamp (nullable)     │
│    reference_message_id: TEXT  -- Reply target (nullable)       │
│    mention_user_ids: TEXT      -- JSON array of user IDs        │
│    mention_role_ids: TEXT      -- JSON array of role IDs        │
│    has_attachments: INTEGER    -- 0/1 flag                      │
│    is_vectorized: INTEGER      -- 0/1 flag                      │
│    created_at: TEXT            -- DB insert timestamp           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 1:1
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Vectorize Index                              │
│                   (nelchan-memory-index)                         │
├─────────────────────────────────────────────────────────────────┤
│ ID: message_id                 -- Discord Message ID            │
│ Vector: [768 dimensions]       -- bge-base-en-v1.5 embedding    │
│ Metadata:                                                        │
│   - content: TEXT              -- Message content               │
│   - channel_id: TEXT           -- Channel ID                    │
│   - user_id: TEXT              -- User ID                       │
│   - username: TEXT             -- Username at time of message   │
│   - timestamp: TEXT            -- Message timestamp             │
└─────────────────────────────────────────────────────────────────┘
```

## Tables

### discord_users

Discordユーザーの情報を保存する。Scheduled Triggerで定期更新される。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | Discord User ID (snowflake) |
| username | TEXT | NOT NULL | Discord username |
| display_name | TEXT | | グローバル表示名（設定されていない場合NULL） |
| avatar_url | TEXT | | アバター画像URL |
| updated_at | TEXT | DEFAULT now | 最終更新日時（ISO 8601） |

**Indexes:**
- なし（PKのみで十分、小規模テーブル）

**Validation Rules:**
- id: 数字のみ（Discord snowflake形式）
- username: 1-32文字

---

### discord_messages

Discordメッセージを保存する。ベクトル化対象かどうかのフラグを持つ。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | Discord Message ID (snowflake) |
| channel_id | TEXT | NOT NULL | Discord Channel ID |
| user_id | TEXT | NOT NULL | 投稿者のUser ID |
| content | TEXT | NOT NULL | メッセージ本文（空文字列可） |
| timestamp | TEXT | NOT NULL | 投稿日時（ISO 8601） |
| edited_timestamp | TEXT | | 編集日時（未編集はNULL） |
| reference_message_id | TEXT | | 返信先メッセージID |
| mention_user_ids | TEXT | | メンションされたユーザーIDのJSON配列 |
| mention_role_ids | TEXT | | メンションされたロールIDのJSON配列 |
| has_attachments | INTEGER | DEFAULT 0 | 添付ファイルがあるか（0/1） |
| is_vectorized | INTEGER | DEFAULT 0 | ベクトル化済みか（0/1） |
| created_at | TEXT | DEFAULT now | DBへの挿入日時 |

**Indexes:**
- `idx_messages_channel`: channel_id（チャンネル単位の検索用）
- `idx_messages_user`: user_id（ユーザー単位の検索用）
- `idx_messages_timestamp`: timestamp（時系列ソート用）

**Validation Rules:**
- id, channel_id, user_id: 数字のみ（Discord snowflake形式）
- timestamp: ISO 8601形式
- mention_user_ids, mention_role_ids: 有効なJSON配列

---

## State Transitions

### Message Lifecycle

```
[New Message Received]
         │
         ▼
    ┌─────────┐
    │ Pending │  -- メッセージ受信直後
    └────┬────┘
         │
         ▼
  ┌──────────────┐
  │ Filter Check │
  └──────┬───────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐  ┌────────────┐
│ Skip  │  │ Store in   │
│       │  │ Database   │
└───────┘  └─────┬──────┘
                 │
           ┌─────┴─────┐
           │           │
           ▼           ▼
    ┌──────────┐  ┌──────────┐
    │ Vectorize│  │ Skip     │
    │ & Store  │  │ Vectorize│
    └────┬─────┘  └────┬─────┘
         │             │
         ▼             ▼
   is_vectorized=1  is_vectorized=0
```

### Filter Decision Tree

```
1. content.length < 5?
   └─ Yes → Skip entirely (don't store in DB)
   └─ No → Continue

2. content.startsWith('!')?
   └─ Yes → Store in DB, skip vectorize
   └─ No → Continue

3. isUrlOnly(content)?
   └─ Yes → Store in DB, skip vectorize
   └─ No → Continue

4. isEmojiOnly(content)?
   └─ Yes → Store in DB, skip vectorize
   └─ No → Continue

5. content.length === 0 && has_attachments?
   └─ Yes → Store in DB, skip vectorize
   └─ No → Store in DB, vectorize
```

---

## Migration Script

```sql
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
```

---

## TypeScript Types

```typescript
// Discord User entity
interface DiscordUser {
  id: string;
  username: string;
  display_name: string | null;
  avatar_url: string | null;
  updated_at: string;
}

// Discord Message entity
interface DiscordMessage {
  id: string;
  channel_id: string;
  user_id: string;
  content: string;
  timestamp: string;
  edited_timestamp: string | null;
  reference_message_id: string | null;
  mention_user_ids: string | null;  // JSON array
  mention_role_ids: string | null;  // JSON array
  has_attachments: number;  // 0 or 1
  is_vectorized: number;    // 0 or 1
  created_at: string;
}

// Vectorize metadata
interface VectorizedMessageMetadata {
  content: string;
  channel_id: string;
  user_id: string;
  username: string;
  timestamp: string;
}

// Parsed mentions (from JSON)
interface ParsedMentions {
  userIds: string[];
  roleIds: string[];
}
```
