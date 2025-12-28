# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## コミュニケーション

このリポジトリでは日本語でやりとりを行うこと。

## 作業ルール

- `cd dir && command` のようにcdとコマンドを&&で繋げて実行してはいけない。代わりに絶対パスを使用するか、作業ディレクトリを指定する
- ビルドや型チェック（`go build`, `go test`, `pnpm run typecheck` など）はユーザーに任せる

## Project Overview

nelchan is a Discord bot with two main components:
- **bot/** - Go Discord bot that handles messages and routes commands
- **code-sandbox/** - Cloudflare Worker that provides REST API for command execution, LLM, and memory management

## Development Commands

### Bot (Go)
```bash
cd bot
go test ./...                    # Run tests
docker build -t nelchan:latest . --secret id=token,src=token.txt  # Build
docker-compose up                # Run locally
```

### Code Sandbox (TypeScript/Cloudflare Worker)
```bash
cd code-sandbox
pnpm install                     # Install dependencies
pnpm run dev                     # Local development (port 8787)
pnpm run deploy                  # Deploy to Cloudflare
pnpm run typecheck               # TypeScript checking
pnpm run cf-typegen              # Generate Cloudflare Worker types
```

## Architecture

### Message Flow
```
Discord Message → commandRouter.Handle()
    ├─ "!command" → handleCodeCommand() → Python sandbox execution
    └─ "command"  → handleTextCommand() → Text response + 30% memory storage
```

### Key Components

**Bot (Go)**
- `nelchan.go` - Main bot logic, command handlers (`handleCodeCommand`, `handleTextCommand`)
- `commandRouter.go` - Discord message routing, memory storage with 30% probability
- `commandAPI.go` - HTTP client for sandbox API calls
- `commandParser.go` - Command parsing (has tests in `commandParser_test.go`)

**Code Sandbox (TypeScript)**
- `src/index.ts` - Hono HTTP routes (all API endpoints)
- `src/usecase.ts` - Business logic: DB operations, LLM calls, vector storage
- `src/agent.ts` - NelchanAgent with MCP (Model Context Protocol) integration

### API Endpoints
- `POST /register_command` - Register code/text command
- `POST /run_command` - Execute registered command
- `POST /memory` - Store memory manually
- `POST /mget` - Semantic search memories (vector DB)
- `POST /mllm` - LLM with memory context (RAG)
- `POST /llmWithAgent/:path` - MCP agent endpoint

### Database (D1 SQLite)
- `commands` - Command metadata (id, name, author_id)
- `dictionaries` - Text command content
- `codes` - Python code command content

### Cloudflare Bindings
- `nelchan_db` - D1 database
- `AI` - Workers AI (deepseek-r1-distill, llama-4-scout models)
- `VECTORIZE` - Vector database for memory (nelchan-memory-index)
- `Sandbox` - Container for Python code execution

## Code Commands

Code commands (`!command args`) execute Python in sandbox with injected variables:
- `username`, `user_id`, `user_avatar`, `args`
- Helper functions: `llm()`, `mget()`, `mllm()`, `automemory()`, `llmWithAgent()`

## Environment

- `DISCORD_BOT_TOKEN` - Discord bot token
- `NELCHAN_API_KEY` - API key for authenticating requests to code-sandbox
- `ENV` - `development` (localhost:8787) or `production` (workers.dev URL)

## 開発ワークフロー

### 概要

spec-kit + git-worktree + gh を組み合わせた開発フロー:

```
仕様策定 → 計画 → タスク生成 → Worktree作成 → 実装 → PR作成
```

### クイックスタート

```bash
# 1. 新機能の仕様を作成（mainで）
/speckit.specify <機能の説明>

# 2. 計画を作成
/speckit.plan

# 3. タスクを生成
/speckit.tasks

# 4. Worktreeを作成して並列開発準備
git worktree add ~/.worktrees/nelchan/<branch-name> <branch-name>

# 5. 別セッションで実装
claude --cwd ~/.worktrees/nelchan/<branch-name>
/speckit.implement

# 6. PR作成
/gh
```

### ワークフロー詳細

#### Phase 1: 仕様策定（mainリポジトリ）

| コマンド | 説明 | 出力 |
|---------|------|------|
| `/speckit.specify <desc>` | 仕様書を作成 | `specs/<branch>/spec.md` |
| `/speckit.clarify` | 仕様の曖昧な点を明確化 | spec.md 更新 |
| `/speckit.plan` | 技術計画を作成 | `specs/<branch>/plan.md` |
| `/speckit.tasks` | 実装タスクを生成 | `specs/<branch>/tasks.md` |

#### Phase 2: 並列開発準備

```bash
# 作成されたブランチでworktreeを作成
git worktree add ~/.worktrees/nelchan/<branch> <branch>

# Worktree一覧確認
git worktree list
```

#### Phase 3: 実装（Worktreeで）

```bash
# 新しいClaude Codeセッションを起動
claude --cwd ~/.worktrees/nelchan/<branch>

# 実装開始
/speckit.implement
```

#### Phase 4: PR作成

```bash
# コミット
/git

# プッシュ & PR作成
git push -u origin <branch>
/gh
```

#### Phase 5: クリーンアップ（マージ後）

```bash
# mainに戻る
git checkout main
git pull

# Worktree削除
git worktree remove ~/.worktrees/nelchan/<branch>

# ブランチ削除
git branch -d <branch>
```

### Worktreeディレクトリ構成

```
~/workspace/
├── nelchan/                    # メインリポジトリ (main)
│   └── specs/                  # 仕様書はここで管理
│       ├── 001-feature-a/
│       └── 002-feature-b/
└── .worktrees/
    └── nelchan/                # Worktree格納場所
        ├── 001-feature-a/      # 機能A実装用
        └── 002-feature-b/      # 機能B実装用
```

### 関連スキル

- `/develop` - 現在の状態を確認して次のステップを提案
- `/git-worktree` - Worktree操作の支援
- `/git` - Git操作の支援
- `/gh` - GitHub CLI操作の支援
