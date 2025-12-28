# Quickstart: mllm Memory Enhancement

**Date**: 2025-12-29
**Branch**: `001-mllm-memory-enhancement`

## Prerequisites

- Go 1.21+
- Node.js 20+ / pnpm
- Cloudflare Wrangler CLI
- 有効なNELCHAN_API_KEY

## 1. データベースマイグレーション

```bash
# code-sandboxディレクトリで
pnpm run cf-typegen  # 型生成

# マイグレーションを適用（ローカル）
wrangler d1 execute nelchan_db --local --file=migrations/0003_mllm_memory.sql

# マイグレーションを適用（本番）
wrangler d1 execute nelchan_db --file=migrations/0003_mllm_memory.sql
```

## 2. Workerのデプロイ

```bash
# code-sandboxディレクトリで
pnpm run typecheck  # 型チェック
pnpm run deploy     # デプロイ
```

## 3. Botの更新

```bash
# botディレクトリで
go build ./...      # ビルド確認
docker-compose up   # ローカル実行
```

## 4. 初回データ投入

```bash
# 特定チャンネルの過去メッセージを取得
curl -X POST https://my-sandbox.sh1ma.workers.dev/admin/fetch_channel \
  -H "Authorization: Bearer $NELCHAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "YOUR_CHANNEL_ID", "limit": 100}'
```

## 5. 動作確認

### メッセージ保存の確認

Discordでメッセージを投稿し、D1データベースを確認:

```bash
wrangler d1 execute nelchan_db --local --command="SELECT * FROM discord_messages LIMIT 5"
```

### mllm v2の確認

```bash
curl -X POST https://my-sandbox.sh1ma.workers.dev/mllm/v2 \
  -H "Authorization: Bearer $NELCHAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "テストメッセージ",
    "channel_id": "YOUR_CHANNEL_ID",
    "user_id": "YOUR_USER_ID"
  }'
```

## API Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/message` | メッセージ保存 |
| PUT | `/message` | メッセージ更新 |
| DELETE | `/message` | メッセージ削除 |
| POST | `/mllm/v2` | 強化版mllm |
| POST | `/admin/fetch_channel` | 一括取得（管理者用） |

## Configuration

### wrangler.jsonc (Scheduled Trigger)

```jsonc
{
  "triggers": {
    "crons": ["0 0 * * *"]  // 毎日00:00 UTCにユーザー情報更新
  }
}
```

### Default Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| recent_count | 10 | 直近会話の取得件数 |
| similar_count | 5 | 類似メッセージの取得件数 |
| min_content_length | 5 | ベクトル化対象の最小文字数 |

## Troubleshooting

### メッセージがベクトル化されない

1. 文字数が5文字以上か確認
2. `!`で始まっていないか確認
3. URL/絵文字のみでないか確認

### mllm v2の応答が遅い

1. `recent_count`/`similar_count`を減らす
2. Vectorizeのインデックスサイズを確認

### Scheduled Triggerが動かない

1. wrangler.jsoncの`triggers`設定を確認
2. `wrangler tail`でログを確認
