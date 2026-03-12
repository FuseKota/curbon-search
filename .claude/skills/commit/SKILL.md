---
name: commit
description: コミットを作成（prefix必須、細かく論理的に区切る）
allowed-tools: Bash
disable-model-invocation: true
---

# Git Commit

変更内容を確認し、適切な粒度に区切り、日本語でわかりやすく簡潔にコミットを作成する。

## 手順

1. `git status` と `git diff` で変更内容を確認
2. 論理単位でステージング
3. 適切なprefixでコミットメッセージを作成

## Prefix（必須）

| Prefix      | 用途                                     |
| ----------- | ---------------------------------------- |
| `feat:`     | 新機能追加                               |
| `fix:`      | バグ修正                                 |
| `refactor:` | リファクタリング（機能変更なし）         |
| `docs:`     | ドキュメントのみの変更                   |
| `style:`    | フォーマット変更（コード動作に影響なし） |
| `test:`     | テストの追加・修正                       |
| `chore:`    | ビルド、設定、依存関係など               |

スコープを付ける場合: `feat(auth): add login form`

## コミットの粒度

### 原則

- 1コミット = 1つの論理的変更
- 「このコミットで何をしたか」が一言で説明できる粒度
- 複数の目的が混在するコミットは避ける

### 分割の例

```
❌ 悪い例: 1コミットに複数の変更
"Add user form, fix validation bug, and update styles"

✅ 良い例: 論理単位で分割
1. "feat(user): add registration form component"
2. "fix(validation): fix email regex pattern"
3. "style(form): update input styles"
```

### 分割の判断基準

| 分割すべき                       | まとめてよい                     |
| -------------------------------- | -------------------------------- |
| 異なる機能への変更               | 同一機能の関連ファイル           |
| バグ修正と新機能追加             | リファクタリングの一連の変更     |
| 本体コードとテストコード（別々） | 型定義とその使用箇所（密結合時） |

## コミット後

コミット完了後、以下を実行してリモートにプッシュしてください：

```bash
# upstream が設定済みの場合
git push

# 新しいブランチで初回プッシュの場合
git push -u origin {現在のブランチ名}
```

> **注意**: `git push --force` は絶対に使用しないこと。
