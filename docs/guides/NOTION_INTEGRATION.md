# Notion統合ガイド

## 🎯 概要

carbon-relayで収集した記事をNotion Databaseに自動的にクリッピングできます。

### クリッピングされる記事

- ✅ **カーボン関連ニュース**: 16の無料ソースから収集したヘッドラインと要約

---

## 📋 事前準備

### 1. Notion Integration を作成

1. [https://www.notion.so/my-integrations](https://www.notion.so/my-integrations) にアクセス
2. 「+ New integration」をクリック
3. 名前を入力（例：`carbon-relay`）
4. Capabilitiesで以下を有効化：
   - ✅ Read content
   - ✅ Update content
   - ✅ Insert content
5. 「Submit」をクリック
6. **Internal Integration Token** をコピー（`secret_...` で始まる文字列）

### 2. 親ページを作成（新規DB作成の場合）

1. Notionで新しいページを作成
2. ページのURLから **Page ID** を取得
   ```
   https://www.notion.so/My-Page-abc123def456...
                                  ^^^^^^^^^^^
                                  これがPage ID
   ```
3. ページの右上「...」→「Connections」→ 作成したIntegrationを接続

---

## 🚀 使い方

### パターン1: 新規データベース作成 + クリッピング

```bash
# 環境変数設定
export NOTION_TOKEN="secret_..."

# 実行（新規DB作成）
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -out=results.json \
  -notionClip \
  -notionPageID="abc123def456..."
```

**実行後：**
- Notionに「Carbon News Clippings」データベースが自動作成されます
- 全ての記事がクリッピングされます

### パターン2: 既存データベースにクリッピング

```bash
# 2回目以降は既存のDatabase IDを指定
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -out=results.json \
  -notionClip \
  -notionDatabaseID="xyz789abc123..."
```

**Database IDの取得方法：**
```
https://www.notion.so/xyz789abc123...?v=...
                    ^^^^^^^^^^^
                    これがDatabase ID
```

---

## 📊 Notion Database の構造

自動作成されるデータベースには以下のフィールドが含まれます：

| フィールド名 | タイプ | 説明 | 例 |
|------------|--------|------|-----|
| **Title** | Title | 記事タイトル | "EU carbon price hits record high amid supply concerns" |
| **URL** | URL | 記事URL | https://carbonherald.com/article/... |
| **Source** | Select | 記事ソース | "Carbon Herald", "JRI", "Carbon Brief" 等 |
| **Type** | Select | 記事タイプ | "Headline" |
| **Excerpt** | Rich Text | 記事要約 | "EU carbon prices reached..." |

---

## 🎨 Notion での活用例

### フィルタ設定

```
Source = "Carbon Herald" → Carbon Heraldのみ
Source = "JRI" → JRIのみ
```

### ソート設定

```
Created time（降順） → 新しい記事から表示
```

### ビュー作成例

1. **全記事一覧**（Table View）
   - Sort: `Created time`（降順）

2. **日本ソースのみ**（Table View）
   - Filter: `Source contains "JRI" OR Source contains "METI" OR Source contains "環境省"`

3. **ソース別**（Board View）
   - Group by: `Source`

---

## ⚙️ コマンドラインオプション

| オプション | 必須/任意 | 説明 |
|-----------|----------|------|
| `-notionClip` | 任意 | Notionクリッピングを有効化（デフォルト: false） |
| `-notionPageID` | 新規DB作成時のみ必須 | 親ページのID |
| `-notionDatabaseID` | 任意 | 既存データベースのID（指定しない場合は新規作成） |

### 環境変数

| 環境変数 | 必須 | 説明 |
|---------|------|------|
| `NOTION_TOKEN` | ✅ | Notion Integration Token |

---

## 📝 実行例

### 例1: ヘッドライン収集 → Notionにクリッピング

```bash
# 環境変数設定
export NOTION_TOKEN="secret_..."

# 全無料ソースから収集してNotionにクリッピング
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -out=notion_clips.json \
  -notionClip \
  -notionPageID="abc123def456..."
```

### 例2: 特定ソースのみNotionにクリッピング

```bash
# 日本ソースのみ
./pipeline \
  -sources=jri,env-ministry,meti \
  -perSource=10 \
  -notionClip \
  -notionDatabaseID="xyz789abc123..."
```

---

## 🆘 トラブルシューティング

### エラー: "NOTION_TOKEN is required"

```bash
# 環境変数を設定
export NOTION_TOKEN="secret_..."
```

### エラー: "notionPageID is required when creating a new Notion database"

```bash
# 新規DB作成時は親ページIDが必要
./pipeline ... -notionClip -notionPageID="abc123..."
```

### エラー: "Could not find database"

→ IntegrationがデータベースまたはページにConnectされていません

**解決方法：**
1. Notionでデータベース/ページを開く
2. 右上「...」→「Connections」
3. 作成したIntegrationを選択

### クリッピングが遅い

→ Notion APIには rate limit があります（1秒あたり3リクエスト）

**対策：**
- 一度に大量の記事をクリッピングしない
- `-perSource`を減らす（例：10件ずつ）

---

## 💡 ベストプラクティス

### 1. 毎日の定期実行

```bash
#!/bin/bash
# daily_notion_clip.sh

export NOTION_TOKEN="secret_..."
DB_ID="xyz789abc123..."  # 既存のDB ID

./pipeline \
  -sources=all-free \
  -perSource=10 \
  -out="$(date +%Y%m%d)_clips.json" \
  -notionClip \
  -notionDatabaseID="$DB_ID"
```

### 2. 日本ソースのみクリッピング

```bash
# 日本のカーボン関連ニュースのみ
./pipeline \
  -sources=jri,env-ministry,meti,mizuho-rt,jpx,carboncredits.jp \
  -perSource=10 \
  -notionClip \
  -notionDatabaseID="$DB_ID"
```

### 3. 国際ソースのみクリッピング

```bash
# 国際的なカーボン関連ニュースのみ
./pipeline \
  -sources=carbonherald,carbon-brief,sandbag,icap,ieta \
  -perSource=10 \
  -notionClip \
  -notionPageID="abc123..."
```

---

## 🔗 参考リンク

- [Notion API Documentation](https://developers.notion.com/)
- [Notion Integration Guide](https://www.notion.so/help/add-and-manage-integrations-with-the-api)
- [jomei/notionapi (Go Package)](https://github.com/jomei/notionapi)

---

**Happy Clipping! 📎**
