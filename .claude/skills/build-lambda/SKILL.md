---
name: build-lambda
description: AWS Lambda用のzipファイルをビルドする（collect-headlines / send-email）
allowed-tools: Bash
disable-model-invocation: true
---

引数 `$ARGUMENTS` に応じてビルド対象を決定し、Lambda zip ファイルを生成する。

## 引数の解釈

| 引数 | 対象 |
|------|------|
| （なし） | 両方ビルド |
| `collect` | collect-headlines のみ |
| `email` | send-email のみ |

## 実行手順

**引数なし（両方ビルド）の場合：**

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
./scripts/build_lambda.sh
```

**`collect` の場合：**

```bash
PROJECT_ROOT=/Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
mkdir -p "$PROJECT_ROOT/dist"
cd "$PROJECT_ROOT"
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o dist/bootstrap ./cmd/lambda/collect/
cd dist && zip -j collect-headlines.zip bootstrap && rm bootstrap
echo "Built: dist/collect-headlines.zip"
ls -lh "$PROJECT_ROOT/dist/collect-headlines.zip"
```

**`email` の場合：**

```bash
PROJECT_ROOT=/Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
mkdir -p "$PROJECT_ROOT/dist"
cd "$PROJECT_ROOT"
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o dist/bootstrap ./cmd/lambda/email/
cd dist && zip -j send-email.zip bootstrap && rm bootstrap
echo "Built: dist/send-email.zip"
ls -lh "$PROJECT_ROOT/dist/send-email.zip"
```

## ビルド後

ビルド完了後、次のステップを案内する：

```
Next steps:
  1. AWS Console > Lambda > 関数の作成
  2. ランタイム: Amazon Linux 2023 (provided.al2023)
  3. アーキテクチャ: arm64
  4. ZIPファイルをアップロード
  5. 環境変数を設定
  6. EventBridgeでスケジュール設定
```
