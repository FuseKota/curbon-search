---
name: save-plan
description: 会話とplanの内容をplan.mdに保存し、関連ブランチを作成（移動はしない）
allowed-tools: Bash, Read, Write
disable-model-invocation: true
---

# Save Plan & Create Branch

現在の会話で議論した計画内容を `.claude/plan/plan.md` に保存し、対応するブランチを作成する（ブランチへの移動はしない）。

## 入力

$ARGUMENTS

引数がある場合はブランチ名のヒントとして使用する。

## 手順

### 1. 計画内容の整理

会話の中で議論された以下の情報を整理する:

- 問題の背景・発見経緯
- 根本原因の分析
- 修正方針
- 修正手順（ステップ別）
- 修正ファイル一覧
- 検証方法

### 2. plan.md への保存

整理した内容を `.claude/plan/plan.md` に上書き保存する。

フォーマット:

```markdown
# [計画タイトル]

## 問題の発見経緯

...

## 根本原因

...

## 修正方針

...

## 修正手順

### Step 1: ...

### Step 2: ...

## 修正ファイル一覧

| ファイル | 変更 |
| -------- | ---- |

## 検証

1. ...
```


### 5. 結果報告

- 保存先: `.claude/plan/plan.md`
