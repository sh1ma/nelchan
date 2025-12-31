# Research: mllm Memory Enhancement

**Date**: 2025-12-29
**Branch**: `001-mllm-memory-enhancement`

## 1. Discord Message Handling

### Decision
discordgo (bwmarrin/discordgo) ライブラリを使用してメッセージイベントを処理する。

### Rationale
- 既存のBot実装がdiscordgoを使用している
- MessageCreate, MessageUpdate, MessageDelete イベントが利用可能
- Message構造体に必要なフィールドがすべて含まれている

### Key Message Fields (from discordgo)
```go
type Message struct {
    ID               string            // メッセージID
    ChannelID        string            // チャンネルID
    GuildID          string            // サーバーID
    Content          string            // テキスト内容
    Timestamp        time.Time         // 投稿時刻
    EditedTimestamp  *time.Time        // 編集時刻
    Author           *User             // 投稿者
    Attachments      []*MessageAttachment // 添付ファイル
    MentionRoles     []string          // メンションされたロール
    Mentions         []*User           // メンションされたユーザー
    MessageReference *MessageReference // 返信先
}
```

### Event Handlers Required
- `MessageCreate`: 新規メッセージ → DB保存 + ベクトル化
- `MessageUpdate`: メッセージ編集 → DB更新 + 再ベクトル化
- `MessageDelete`: メッセージ削除 → DB削除 + ベクトル削除

### Alternatives Considered
- Discord API直接呼び出し: discordgoが既に使用されているため不要

---

## 2. Cloudflare Vectorize Best Practices

### Decision
既存の `nelchan-memory-index` を拡張使用し、メッセージIDをベクトルIDとして使用する。

### Rationale
- 既存インデックスの再利用でインフラ変更を最小化
- Discord Message IDはユニークで永続的
- upsert操作で編集時の更新が容易

### Implementation Pattern
```typescript
// 保存
await env.VECTORIZE.upsert([{
  id: messageId,  // Discord Message ID
  values: embedding,
  metadata: {
    content: messageContent,
    channelId: channelId,
    userId: userId,
    timestamp: timestamp
  }
}]);

// 検索
const results = await env.VECTORIZE.query(embedding, {
  topK: 5,
  returnMetadata: "all",
  filter: { channelId: targetChannelId }  // オプション: チャンネル絞り込み
});

// 削除
await env.VECTORIZE.deleteByIds([messageId]);
```

### Alternatives Considered
- 新規インデックス作成: 追加のインフラ管理が必要なため却下
- 従来のランダムUUID方式: メッセージとの紐付けが困難なため却下

---

## 3. D1 Database Schema Design

### Decision
2つの新テーブル `discord_messages` と `discord_users` を追加する。

### Rationale
- 既存の `commands`, `dictionaries`, `codes` テーブルと分離
- Discord固有のデータモデルを明確に表現
- 外部キー制約なしでシンプルに保つ（Discordデータは外部ソース）

### Schema
```sql
-- discord_messages テーブル
CREATE TABLE discord_messages (
    id TEXT PRIMARY KEY,              -- Discord Message ID
    channel_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    timestamp TEXT NOT NULL,          -- ISO 8601形式
    edited_timestamp TEXT,
    reference_message_id TEXT,        -- 返信先メッセージID
    mention_user_ids TEXT,            -- JSON配列
    mention_role_ids TEXT,            -- JSON配列
    is_vectorized INTEGER DEFAULT 0,  -- ベクトル化済みフラグ
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX idx_messages_channel ON discord_messages(channel_id);
CREATE INDEX idx_messages_user ON discord_messages(user_id);
CREATE INDEX idx_messages_timestamp ON discord_messages(timestamp);

-- discord_users テーブル
CREATE TABLE discord_users (
    id TEXT PRIMARY KEY,              -- Discord User ID
    username TEXT NOT NULL,
    display_name TEXT,
    avatar_url TEXT,
    updated_at TEXT DEFAULT (datetime('now'))
);
```

### Alternatives Considered
- 単一テーブル設計: 正規化のため分離を選択
- PostgreSQL/外部DB: Cloudflare D1との統合を維持するため却下

---

## 4. Scheduled Trigger for User Updates

### Decision
Cloudflare Scheduled Trigger（Cron Trigger）を使用して1日1回ユーザー情報を更新する。

### Rationale
- Cloudflare Workerネイティブ機能で追加インフラ不要
- wrangler.jsonc に設定を追加するだけで有効化
- Discord APIレート制限を考慮した低頻度更新

### Configuration (wrangler.jsonc)
```jsonc
{
  "triggers": {
    "crons": ["0 0 * * *"]  // 毎日00:00 UTC
  }
}
```

### Implementation
```typescript
export default {
  async scheduled(event: ScheduledEvent, env: Env, ctx: ExecutionContext) {
    // discord_usersテーブルから全ユーザーを取得
    // 各ユーザーのDiscord情報をAPIから取得（キャッシュ活用）
    // 変更があれば更新
  }
}
```

### Alternatives Considered
- 外部Cronサービス: Cloudflareネイティブで十分なため却下
- メッセージ受信時に更新: APIコール増加を避けるため却下

---

## 5. Noise Filtering Strategy

### Decision
複合フィルタリングを適用し、5文字未満、URL/絵文字のみ、Botコマンドを除外する。

### Rationale
- ベクトル検索の精度向上
- ストレージコスト削減
- 意味のあるコンテンツのみを学習対象に

### Filter Rules (Priority Order)
1. 5文字未満 → 除外（DBにも保存しない）
2. `!` プレフィックス（Botコマンド）→ DBには保存、ベクトル化しない
3. URL のみ → DBには保存、ベクトル化しない
4. 絵文字のみ → DBには保存、ベクトル化しない
5. 添付ファイルのみ（テキストなし）→ DBには保存、ベクトル化しない

### Implementation
```typescript
function shouldVectorize(content: string): boolean {
  if (content.length < 5) return false;
  if (content.startsWith('!')) return false;
  if (isUrlOnly(content)) return false;
  if (isEmojiOnly(content)) return false;
  return true;
}

function shouldStore(content: string, hasAttachments: boolean): boolean {
  // 添付ファイルのみでもDBには保存
  if (hasAttachments && content.length === 0) return true;
  if (content.length < 5) return false;
  return true;
}
```

---

## 6. mllm Context Building

### Decision
3層コンテキスト（直近会話 + 類似メッセージ + ユーザー情報）を構築する。

### Rationale
- 直近会話: 即時的な文脈を提供
- 類似メッセージ: 長期記憶からの関連情報
- ユーザー情報: パーソナライズされた応答

### Context Structure
```typescript
interface MLLMContext {
  recentMessages: {  // 直近N件（デフォルト10件）
    userId: string;
    username: string;
    content: string;
    timestamp: string;
  }[];
  similarMessages: {  // ベクトル検索結果M件（デフォルト5件）
    content: string;
    username: string;
    score: number;
  }[];
  userInfo: {  // 発話者情報
    username: string;
    displayName: string;
  } | null;
}
```

### Prompt Template
```
あなたはDiscord上で動くBotのねるちゃんです。
[キャラクター設定...]

## ユーザー情報
発話者: {userInfo.displayName}

## 直近の会話
{recentMessages.map(m => `${m.username}: ${m.content}`).join('\n')}

## 関連する過去の情報
{similarMessages.map(m => `- ${m.content}`).join('\n')}

上記の情報を踏まえて、以下のメッセージに応答してください:
{currentMessage}
```

---

## 7. Initial Fetch Implementation

### Decision
管理者専用APIエンドポイントとして `/admin/fetch_channel` を提供する。

### Rationale
- 既存のBearer Token認証を流用
- Discord APIのレート制限を考慮したバッチ処理
- チャンネル単位での柔軟なデータ投入

### API Design
```
POST /admin/fetch_channel
Authorization: Bearer {ADMIN_API_KEY}
{
  "channel_id": "123456789",
  "limit": 100  // 省略時デフォルト100
}
```

### Discord API Rate Limits
- GET /channels/{channel.id}/messages: 5 req/sec
- 100メッセージ取得には約20秒（50件/リクエスト × 2回）

### Alternatives Considered
- Bot側でのfetch: API認証の一貫性のためWorker側を選択
- 無制限fetch: レート制限とストレージを考慮し上限設定
