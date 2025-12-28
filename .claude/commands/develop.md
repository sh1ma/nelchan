---
description: 仕様策定からPR作成までの統合開発ワークフローをガイドします。新機能開発の開始時に使用してください。
---

# 統合開発ワークフロー

このコマンドは、spec-kit、git-worktree、ghを組み合わせた開発フローをガイドします。

## ユーザー入力

```text
$ARGUMENTS
```

## ワークフロー

### 1. 現在の状態を確認

まず、現在のリポジトリ状態を確認します：

```bash
# 現在のブランチとworktree状態
git branch --show-current
git worktree list

# 進行中の機能があるか確認
ls -la specs/ 2>/dev/null || echo "specsディレクトリなし"
```

### 2. フェーズ判定

ユーザーの入力と状態から、現在のフェーズを判定：

**Case A: 新機能開始**（`$ARGUMENTS`に機能の説明がある場合）
- フェーズ1（仕様策定）から開始
- `/speckit.specify $ARGUMENTS` を実行

**Case B: 仕様策定済み・計画未作成**
- specs/<branch>/spec.md が存在
- specs/<branch>/plan.md が存在しない
- `/speckit.plan` を提案

**Case C: 計画済み・タスク未生成**
- specs/<branch>/plan.md が存在
- specs/<branch>/tasks.md が存在しない
- `/speckit.tasks` を提案

**Case D: タスク生成済み・実装未開始**
- specs/<branch>/tasks.md が存在
- 実装準備として worktree 作成を提案

**Case E: 実装中**
- worktree内で作業中
- `/speckit.implement` を提案

**Case F: 実装完了・PR未作成**
- 変更がコミット済み
- PR作成を提案

### 3. ワークフロー実行ガイド

各フェーズで実行すべきコマンドを提示：

---

## フェーズ1: 仕様策定（mainリポジトリ）

```
/speckit.specify <機能の説明>
```

→ ブランチ作成、spec.md 生成

---

## フェーズ2: 計画作成（mainリポジトリ）

```
/speckit.plan
```

→ plan.md 生成（技術選定、アーキテクチャ）

---

## フェーズ3: タスク生成（mainリポジトリ）

```
/speckit.tasks
```

→ tasks.md 生成（実装タスクリスト）

---

## フェーズ4: 実装準備

### 並列開発する場合（推奨）

```bash
# Worktreeを作成
git worktree add ~/.worktrees/nelchan/<branch-name> <branch-name>

# 新しいClaude Codeセッションで作業
claude --cwd ~/.worktrees/nelchan/<branch-name>
```

### 単一作業の場合

```bash
# ブランチにそのまま滞在して作業
git checkout <branch-name>
```

---

## フェーズ5: 実装

```
/speckit.implement
```

→ tasks.md に従って実装

---

## フェーズ6: コミット・プッシュ

```
/git  # コミット支援
git push -u origin <branch-name>
```

---

## フェーズ7: PR作成

```
/gh  # PR作成支援
```

または直接：

```bash
gh pr create --title "feat: <機能名>" --body "## Summary
- 変更内容

## Test plan
- [ ] テスト項目"
```

---

## フェーズ8: クリーンアップ（マージ後）

```bash
# mainに戻る
git checkout main
git pull origin main

# Worktreeを削除
git worktree remove ~/.worktrees/nelchan/<branch-name>

# ブランチを削除（マージ済みの場合）
git branch -d <branch-name>
```

---

## クイックリファレンス

| フェーズ | コマンド | 出力 |
|---------|---------|------|
| 仕様策定 | `/speckit.specify <desc>` | spec.md |
| 明確化 | `/speckit.clarify` | spec.md更新 |
| 計画 | `/speckit.plan` | plan.md |
| タスク | `/speckit.tasks` | tasks.md |
| 実装 | `/speckit.implement` | コード |
| コミット | `/git` | コミット |
| PR | `/gh` | PR作成 |

## 次のステップ

現在の状態を確認して、次に実行すべきコマンドを提案します。
