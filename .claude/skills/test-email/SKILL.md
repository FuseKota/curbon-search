---
name: test-email
description: メール送信機能をテストする（診断モードと実送信モードを切り替え可能）
allowed-tools: Bash, AskUserQuestion
---

# Carbon Relay: メール送信テストスキル

引数 `$ARGUMENTS` を解析し、環境変数・ビルドを確認後、ユーザーに実行モードを確認してから実行する。

---

## Step 1: 引数解析

`$ARGUMENTS` から数値を抽出して `DAYS` に設定する（なければ `1`）。

| 引数パターン | `-emailDaysBack` |
|-------------|-----------------|
| （なし） | `1` |
| 数値のみ（例: `3`） | その数値 |

## Step 2: 環境変数チェック

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
source .env 2>/dev/null || true
echo "=== 環境変数チェック ==="
[ -n "$NOTION_TOKEN" ]       && echo "✓ NOTION_TOKEN" || echo "✗ NOTION_TOKEN（未設定）"
[ -n "$NOTION_DATABASE_ID" ] && echo "✓ NOTION_DATABASE_ID" || echo "✗ NOTION_DATABASE_ID（未設定）"
[ -n "$EMAIL_FROM" ]         && echo "✓ EMAIL_FROM: $EMAIL_FROM" || echo "✗ EMAIL_FROM（未設定）"
[ -n "$EMAIL_PASSWORD" ]     && echo "✓ EMAIL_PASSWORD" || echo "✗ EMAIL_PASSWORD（未設定）"
[ -n "$EMAIL_TO" ]           && echo "✓ EMAIL_TO: $EMAIL_TO" || echo "✗ EMAIL_TO（未設定）"
```

未設定の必須変数（`NOTION_TOKEN`, `NOTION_DATABASE_ID`, `EMAIL_FROM`, `EMAIL_PASSWORD`, `EMAIL_TO`）があれば、中断してエラーを表示する。

## Step 3: ビルド確認

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
go build -o pipeline ./cmd/pipeline 2>&1 && echo "BUILD_OK" || echo "BUILD_FAILED"
```

`BUILD_FAILED` が出力された場合は即座に中断し、エラー内容を表示する。

## Step 4: モード確認（AskUserQuestion）

ユーザーに以下を確認する：

> 実行モードを選んでください:
> - `list` → 診断のみ（送信対象の記事を確認、メール送信なし）
> - `send` → 実際にメール送信（送信先: EMAIL_TO の値）

## Step 5a: 診断モード（list を選択した場合）

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
./pipeline -listShortHeadlines -emailDaysBack={DAYS} 2>&1
```

結果をそのまま表示する。

## Step 5b: 送信モード（send を選択した場合）

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
./pipeline -sendShortEmail -emailDaysBack={DAYS} 2>&1
```

結果をそのまま表示する。

---

## 表示例

### 診断モード（list）

```
=== 環境変数チェック ===
✓ NOTION_TOKEN
✓ NOTION_DATABASE_ID
✓ EMAIL_FROM: example@gmail.com
✓ EMAIL_PASSWORD
✓ EMAIL_TO: recipient@example.com

[list] 過去1日の送信対象記事:
1. Carbon Markets Weekly: ... (carbonherald)
2. The Case for Clean Energy ... (rmi)
...
合計: 12件
```

### 送信モード（send）

```
=== 環境変数チェック ===
✓ NOTION_TOKEN
✓ NOTION_DATABASE_ID
✓ EMAIL_FROM: example@gmail.com
✓ EMAIL_PASSWORD
✓ EMAIL_TO: recipient@example.com

メール送信完了: recipient@example.com 宛に12件の記事を送信しました
```

---

## 注意事項

- メール送信には Notion DB からの記事取得が必要なため、`NOTION_TOKEN` と `NOTION_DATABASE_ID` も必須
- `-emailDaysBack` のデフォルトは `1`（過去24時間）
- 送信モードは実際にメールが送信されるため、確認後に実行すること
