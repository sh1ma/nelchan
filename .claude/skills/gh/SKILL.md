---
name: gh
description: GitHub CLI（gh）を使った操作を支援します。Issue管理、PR操作、リリース作成時に使用します。「/gh」で明示的に呼び出すこともできます。
---

# GitHub CLI スキル

## 認証確認

```bash
gh auth status
```

## Issue 操作

### Issue 一覧

```bash
gh issue list                      # オープンなIssue一覧
gh issue list --state all          # 全Issue
gh issue list --label "bug"        # ラベルでフィルタ
gh issue list --assignee @me       # 自分にアサインされたIssue
```

### Issue 作成

```bash
gh issue create --title "タイトル" --body "本文"
gh issue create --title "タイトル" --label "bug" --assignee @me
```

### Issue 詳細・操作

```bash
gh issue view <number>             # 詳細表示
gh issue close <number>            # クローズ
gh issue reopen <number>           # 再オープン
gh issue edit <number> --add-label "bug"
```

## Pull Request 操作

### PR 一覧

```bash
gh pr list                         # オープンなPR一覧
gh pr list --state merged          # マージ済みPR
gh pr list --author @me            # 自分が作成したPR
```

### PR 作成

```bash
gh pr create --title "タイトル" --body "本文"
gh pr create --fill                # コミットメッセージから自動入力
gh pr create --draft               # ドラフトPRとして作成
gh pr create --base main           # ベースブランチを指定
```

### PR 詳細・操作

```bash
gh pr view <number>                # 詳細表示
gh pr view --web                   # ブラウザで開く
gh pr checkout <number>            # PRのブランチをチェックアウト
gh pr merge <number>               # マージ
gh pr merge <number> --squash      # スカッシュマージ
gh pr close <number>               # クローズ
```

### PR レビュー

```bash
gh pr review <number> --approve    # 承認
gh pr review <number> --comment -b "コメント"
gh pr review <number> --request-changes -b "理由"
```

## リリース操作

### リリース一覧

```bash
gh release list
```

### リリース作成

```bash
gh release create <tag> --title "タイトル" --notes "リリースノート"
gh release create <tag> --generate-notes   # 自動でリリースノート生成
gh release create <tag> --draft            # ドラフトとして作成
```

### リリース詳細

```bash
gh release view <tag>
gh release download <tag>          # アセットをダウンロード
```

## リポジトリ操作

```bash
gh repo view                       # 現在のリポジトリ情報
gh repo view --web                 # ブラウザで開く
gh repo clone <owner/repo>         # クローン
gh repo fork                       # フォーク作成
```

## ワークフロー（GitHub Actions）

```bash
gh run list                        # ワークフロー実行一覧
gh run view <run-id>               # 実行詳細
gh run watch <run-id>              # 実行をリアルタイム監視
gh workflow list                   # ワークフロー一覧
gh workflow run <workflow>         # ワークフローを手動実行
```

## 便利なオプション

- `--json <fields>`: JSON形式で出力
- `--jq <expression>`: jqでフィルタリング
- `--web`: ブラウザで開く
- `-R <owner/repo>`: リポジトリを指定して実行
