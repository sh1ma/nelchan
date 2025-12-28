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
