# Implementation Plan: mllm Memory Enhancement

**Branch**: `001-mllm-memory-enhancement` | **Date**: 2025-12-29 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-mllm-memory-enhancement/spec.md`

## Summary

mllm（Memory LLM）の応答品質を向上させるため、Discordメッセージとユーザー情報の永続化、メッセージ全文のベクトル化、直近会話・類似メッセージ・ユーザー情報を統合したコンテキスト構築を実装する。

## Technical Context

**Language/Version**: Go 1.21+ (Bot), TypeScript 5.x (Worker)
**Primary Dependencies**: discordgo, Hono, Cloudflare Workers AI
**Storage**: Cloudflare D1 (SQLite), Cloudflare Vectorize
**Testing**: go test, pnpm run typecheck
**Target Platform**: Cloudflare Workers, Docker (Bot)
**Project Type**: Web application (Bot + Worker API)
**Performance Goals**: ベクトル検索 <1秒, メッセージ保存 <100ms
**Constraints**: Discord APIレート制限, Vectorize上限
**Scale/Scope**: 1チャンネル, 数十名のアクティブユーザー

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution未設定のため、プロジェクト固有のルールに従う:
- ✅ 既存のアーキテクチャ（Go Bot + Cloudflare Worker）を維持
- ✅ 既存のVectorizeインデックスを再利用
- ✅ Bearer Token認証を継続使用

## Project Structure

### Documentation (this feature)

```text
specs/001-mllm-memory-enhancement/
├── spec.md              # 機能仕様
├── plan.md              # このファイル
├── research.md          # Phase 0 リサーチ結果
├── data-model.md        # Phase 1 データモデル
├── quickstart.md        # Phase 1 クイックスタート
├── contracts/           # Phase 1 APIコントラクト
│   └── api.yaml         # OpenAPI 3.0仕様
└── checklists/
    └── requirements.md  # 仕様品質チェックリスト
```

### Source Code (repository root)

```text
bot/
├── nelchan.go           # メイン + イベントハンドラ追加
├── commandRouter.go     # メッセージ保存API呼び出し追加
├── commandAPI.go        # 新規APIクライアントメソッド追加
└── messageHandler.go    # 【新規】メッセージ保存ロジック

code-sandbox/
├── src/
│   ├── index.ts         # 新規エンドポイント追加
│   ├── usecase.ts       # 既存関数 + 新規関数追加
│   ├── messageService.ts    # 【新規】メッセージ保存/検索
│   ├── userService.ts       # 【新規】ユーザー管理
│   ├── vectorService.ts     # 【新規】ベクトル化ロジック
│   └── scheduler.ts         # 【新規】Scheduled Trigger
├── migrations/
│   └── 0003_mllm_memory.sql # 【新規】DBマイグレーション
└── wrangler.jsonc       # triggers.crons追加
```

**Structure Decision**: 既存のBot/Worker分離アーキテクチャを維持。Workerに新規サービスファイルを追加してロジックを分離。

## Phase Summary

| Phase | 成果物 | 状態 |
|-------|--------|------|
| Phase 0 | research.md | ✅ 完了 |
| Phase 1 | data-model.md, contracts/, quickstart.md | ✅ 完了 |
| Phase 2 | tasks.md | 未着手（/speckit.tasks で生成） |

## Key Decisions

1. **メッセージ保存範囲**: テキスト重視（Content + ReferencedMessage + Mentions）
2. **返信のベクトル化**: 単体のみ（元メッセージは参照IDのみ）
3. **編集・削除処理**: 編集は更新、削除は両DBから削除
4. **添付ファイルのみ**: DBには保存、ベクトルDBには保存しない
5. **データ保持期間**: 無期限（Discord削除イベントに連動）
6. **Vectorize ID**: Discord Message IDを直接使用（upsert対応）

## Complexity Tracking

> 違反なし。既存アーキテクチャを維持。

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |

## Next Steps

1. `/speckit.tasks` を実行してtasks.mdを生成
2. タスクに従って実装を開始
3. 各マイルストーンでテスト実施
