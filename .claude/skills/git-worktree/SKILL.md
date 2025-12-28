---
name: git-worktree
description: Git worktreeを使った並列開発を支援する。複数のClaude Codeインスタンスで同時に異なるブランチの作業を行う際に使用する。"worktree"、"並列開発"、"別ブランチで作業"などのキーワードで発動。
---

# Git Worktree スキル

## 概要

git worktreeは、1つのリポジトリで複数の作業ディレクトリを持つ機能。
これにより、複数のClaude Codeインスタンスで異なるブランチを並列開発できる。

## ディレクトリ構成

```
~/workspace/
├── nelchan/              # メインリポジトリ (main)
└── .worktrees/
    └── nelchan/          # nelchanのworktree格納場所
        ├── 001-feature-a/  # 機能Aのworktree
        └── 002-feature-b/  # 機能Bのworktree
```

## 基本操作

### Worktreeの作成

spec-kit連携の場合（推奨）：

```bash
# 1. mainリポジトリで仕様を作成（ブランチも作成される）
/speckit.specify <機能の説明>

# 2. 作成されたブランチでworktreeを作成
git worktree add ~/.worktrees/nelchan/<branch-name> <branch-name>
```

手動でブランチとworktreeを同時に作成：

```bash
git worktree add -b <new-branch> ~/.worktrees/nelchan/<new-branch> main
```

既存ブランチからworktreeを作成：

```bash
git worktree add ~/.worktrees/nelchan/<branch-name> <branch-name>
```

### Worktree一覧

```bash
git worktree list
```

### Worktreeの削除

```bash
# 作業完了・マージ後
git worktree remove ~/.worktrees/nelchan/<branch-name>

# ブランチも削除する場合
git worktree remove ~/.worktrees/nelchan/<branch-name>
git branch -d <branch-name>
```

### 強制削除（未コミットの変更がある場合）

```bash
git worktree remove --force ~/.worktrees/nelchan/<branch-name>
```

## spec-kit + git-worktree ワークフロー

### Phase 1: 仕様策定（mainリポジトリ）

```bash
# mainリポジトリで作業
# 1. 仕様を作成
/speckit.specify ユーザー認証機能を追加

# → 001-user-auth ブランチが作成され、specs/001-user-auth/spec.md が生成される

# 2. 計画を作成
/speckit.plan

# 3. タスクを生成
/speckit.tasks
```

### Phase 2: 並列開発環境の準備

```bash
# Worktreeを作成して別ディレクトリで開発
git worktree add ~/.worktrees/nelchan/001-user-auth 001-user-auth
```

### Phase 3: 実装（Worktreeで）

```bash
# 別のClaude Codeインスタンスを起動
claude --cwd ~/.worktrees/nelchan/001-user-auth

# 実装開始
/speckit.implement
```

### Phase 4: PR作成

```bash
# Worktreeで
git push -u origin 001-user-auth

# PR作成
gh pr create --title "feat: ユーザー認証機能を追加" --body "..."
```

### Phase 5: クリーンアップ

```bash
# マージ後、mainリポジトリに戻って
git worktree remove ~/.worktrees/nelchan/001-user-auth
git branch -d 001-user-auth
```

## 並列開発のパターン

### 複数機能を同時開発

```
Terminal 1 (main):
  └── 仕様策定・レビュー作業

Terminal 2 (001-feature-a worktree):
  └── 機能A実装

Terminal 3 (002-feature-b worktree):
  └── 機能B実装
```

### 注意点

1. **specsディレクトリはmainで管理**
   - 仕様書はmainブランチで作成・管理
   - worktreeでは実装のみ行う

2. **コミットはworktreeごとに独立**
   - 各worktreeで独立してコミット可能
   - pushはworktreeから直接実行

3. **ロックされたブランチ**
   - あるworktreeでチェックアウト中のブランチは他でチェックアウト不可
   - `git worktree list`で確認

## トラブルシューティング

### Worktreeが残っている場合

```bash
# 孤立したworktreeを削除
git worktree prune
```

### ブランチがロックされている場合

```bash
# どのworktreeがブランチを使っているか確認
git worktree list

# 該当worktreeを削除してからブランチを操作
git worktree remove <path>
```
