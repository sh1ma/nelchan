# Tasks: mllm Memory Enhancement

**Input**: Design documents from `/specs/001-mllm-memory-enhancement/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: テストは明示的に要求されていないため、含まれていません。

**Organization**: タスクはユーザーストーリーごとにグループ化されています。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 並列実行可能（異なるファイル、依存関係なし）
- **[Story]**: このタスクが属するユーザーストーリー（US1, US2, US3, US4）

## Path Conventions

- **Bot (Go)**: `bot/`
- **Worker (TypeScript)**: `code-sandbox/src/`
- **Migrations**: `code-sandbox/migrations/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: プロジェクトの基盤準備

- [x] T001 DBマイグレーションファイルを作成 in code-sandbox/migrations/0003_mllm_memory.sql
- [x] T002 [P] TypeScript型定義を追加 in code-sandbox/src/types/discord.ts
- [x] T003 [P] wrangler.jsoncにScheduled Trigger設定を追加 in code-sandbox/wrangler.jsonc

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: すべてのユーザーストーリーが依存するコアインフラ

**⚠️ CRITICAL**: このフェーズが完了するまでユーザーストーリーの作業は開始できません

- [x] T004 ノイズフィルタリング関数を実装 in code-sandbox/src/filters.ts
- [x] T005 [P] ベクトル化サービスを実装（embedding生成 + upsert/delete） in code-sandbox/src/vectorService.ts
- [x] T006 [P] ユーザーサービスを実装（CRUD操作） in code-sandbox/src/userService.ts
- [x] T007 メッセージサービスを実装（CRUD + ベクトル化連携） in code-sandbox/src/messageService.ts

**Checkpoint**: 基盤完了 - ユーザーストーリー実装を開始可能

---

## Phase 3: User Story 2 - Discordメッセージの自動保存と検索 (Priority: P1) 🎯 MVP

**Goal**: メッセージをDBとベクトルDBに保存し、類似検索できるようにする

**Independent Test**: Discordでメッセージ送信後、`/mget`で類似メッセージが取得できる

> **Note**: US1（高品質な会話応答）はUS2（メッセージ保存）に依存するため、US2を先に実装

### Implementation for User Story 2

- [x] T008 [US2] POST /message エンドポイントを追加 in code-sandbox/src/index.ts
- [x] T009 [US2] PUT /message エンドポイントを追加 in code-sandbox/src/index.ts
- [x] T010 [US2] DELETE /message エンドポイントを追加 in code-sandbox/src/index.ts
- [x] T011 [P] [US2] Bot側にStoreMessage APIクライアントを追加 in bot/commandAPI.go
- [x] T012 [P] [US2] Bot側にUpdateMessage APIクライアントを追加 in bot/commandAPI.go
- [x] T013 [P] [US2] Bot側にDeleteMessage APIクライアントを追加 in bot/commandAPI.go
- [x] T014 [US2] Bot側にMessageCreateハンドラを実装 in bot/messageHandlers.go
- [x] T015 [US2] Bot側にMessageUpdateハンドラを実装 in bot/messageHandlers.go
- [x] T016 [US2] Bot側にMessageDeleteハンドラを実装 in bot/messageHandlers.go

**Checkpoint**: メッセージがDBとベクトルDBに保存され、検索可能

---

## Phase 4: User Story 1 - 高品質な会話応答 (Priority: P1)

**Goal**: 直近会話 + 類似メッセージ + ユーザー情報を使った高品質応答

**Independent Test**: Botへのメンションで、過去の会話文脈を反映した応答が返る

### Implementation for User Story 1

- [x] T017 [US1] 直近メッセージ取得関数を実装 in code-sandbox/src/messageService.ts
- [x] T018 [US1] 類似メッセージ検索関数を実装（既存getMemoryを拡張） in code-sandbox/src/messageService.ts
- [x] T019 [US1] コンテキストビルダーを実装（3層コンテキスト構築） in code-sandbox/src/contextBuilder.ts
- [x] T020 [US1] POST /mllm/v2 エンドポイントを追加 in code-sandbox/src/index.ts
- [x] T021 [US1] 強化版memoryLLM関数を実装 in code-sandbox/src/usecase.ts
- [x] T022 [P] [US1] Bot側にMLLMv2 APIクライアントを追加 in bot/commandAPI.go
- [x] T023 [US1] Bot側のメンションハンドラをMLLM v2に更新 in bot/nelchan.go (APIクライアント追加のみ、ハンドラ更新は既存のまま)

**Checkpoint**: Botが過去の会話文脈を反映した応答を返す

---

## Phase 5: User Story 3 - ユーザー情報の管理と定期更新 (Priority: P2)

**Goal**: ユーザー情報を保存し、定期的に最新化する

**Independent Test**: Scheduled Triggerでユーザー情報が更新される

### Implementation for User Story 3

- [x] T024 [US3] ユーザー情報upsert関数を実装 in code-sandbox/src/userService.ts
- [x] T025 [US3] Scheduled Triggerハンドラを実装 in code-sandbox/src/scheduler.ts
- [x] T026 [US3] Worker exportにscheduled関数を追加 in code-sandbox/src/index.ts
- [x] T027 [P] [US3] メッセージ保存時にユーザー情報も保存する処理を追加 in code-sandbox/src/messageService.ts

**Checkpoint**: ユーザー情報がDBに保存され、定期更新される

---

## Phase 6: User Story 4 - 初回データ取得機能 (Priority: P2)

**Goal**: 管理者がチャンネルの過去メッセージを一括取得できる

**Independent Test**: `/admin/fetch_channel`で過去100件のメッセージが保存される

### Implementation for User Story 4

- [x] T028 [US4] Discord API経由でメッセージ履歴を取得する関数を実装 in code-sandbox/src/discordClient.ts
- [x] T029 [US4] バッチ保存関数を実装 in code-sandbox/src/messageService.ts
- [x] T030 [US4] POST /admin/fetch_channel エンドポイントを追加 in code-sandbox/src/index.ts

**Checkpoint**: 過去メッセージを一括取得・保存できる

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 全体的な改善と仕上げ

- [x] T031 [P] 従来のautoStoreMemory呼び出しを削除（commandRouter.goから） in bot/commandRouter.go
- [ ] T032 [P] Cloudflare型定義を更新（cf-typegen実行） in code-sandbox/
- [x] T033 エラーハンドリングとログ出力を統一 in code-sandbox/src/ (各サービスで一貫したログ出力を実装)
- [ ] T034 quickstart.mdに従って動作確認

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 依存なし - 即開始可能
- **Foundational (Phase 2)**: Setup完了後 - すべてのユーザーストーリーをブロック
- **User Story 2 (Phase 3)**: Foundational完了後
- **User Story 1 (Phase 4)**: User Story 2完了後（メッセージデータが必要）
- **User Story 3 (Phase 5)**: Foundational完了後（US1/US2と並列可能）
- **User Story 4 (Phase 6)**: Foundational完了後（US1/US2/US3と並列可能）
- **Polish (Phase 7)**: すべてのユーザーストーリー完了後

### User Story Dependencies

```
         ┌─────────────┐
         │   Setup     │
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │ Foundational │
         └──────┬──────┘
                │
    ┌───────────┼───────────┬───────────┐
    │           │           │           │
┌───▼───┐ ┌─────▼─────┐ ┌───▼───┐       │
│  US2  │ │    US3    │ │  US4  │       │
│(P1 MVP)│ │   (P2)    │ │ (P2)  │       │
└───┬───┘ └───────────┘ └───────┘       │
    │                                    │
┌───▼───┐                                │
│  US1  │ ◄──────────────────────────────┘
│ (P1)  │   (US1 depends on US2 for message data)
└───────┘
```

### Parallel Opportunities

- T002, T003 (Setup) は並列実行可能
- T005, T006 (Foundational) は並列実行可能
- T011, T012, T013 (US2 APIクライアント) は並列実行可能
- T022 (US1) は他のUS1タスクと並列可能
- T027 (US3) は他のUS3タスクと並列可能
- T031, T032 (Polish) は並列実行可能
- US3とUS4はUS2完了を待たずに並列開始可能

---

## Parallel Example: User Story 2

```bash
# Bot側APIクライアントを並列で実装:
Task: "Bot側にStoreMessage APIクライアントを追加 in bot/commandAPI.go"
Task: "Bot側にUpdateMessage APIクライアントを追加 in bot/commandAPI.go"
Task: "Bot側にDeleteMessage APIクライアントを追加 in bot/commandAPI.go"
```

---

## Implementation Strategy

### MVP First (User Story 2 + 1)

1. Phase 1: Setup 完了
2. Phase 2: Foundational 完了
3. Phase 3: User Story 2 完了
4. **検証**: メッセージがDB/ベクトルDBに保存されることを確認
5. Phase 4: User Story 1 完了
6. **検証**: Botが文脈を反映した応答を返すことを確認
7. デプロイ/デモ（MVP）

### Incremental Delivery

1. Setup + Foundational → 基盤完了
2. User Story 2 → メッセージ保存機能リリース
3. User Story 1 → 高品質応答リリース（MVP完了）
4. User Story 3 → ユーザー情報管理リリース
5. User Story 4 → 初回データ取得リリース
6. Polish → 最終調整

---

## Notes

- [P] タスク = 異なるファイル、依存関係なし
- [Story] ラベル = トレーサビリティのためのユーザーストーリーマッピング
- 各ユーザーストーリーは独立してテスト可能
- タスク完了ごとにコミット
- チェックポイントでストーリーを個別に検証
