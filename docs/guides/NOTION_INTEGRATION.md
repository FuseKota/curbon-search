# Notion統合ガイド

## 🎯 概要

carbon-relayで収集した記事（有料ヘッドライン + 関連無料記事）をNotion Databaseに自動的にクリッピングできます。

### クリッピングされる記事

- ✅ **有料記事のヘッドライン**: Carbon Pulse / QCI の見出しと要約
- ✅ **関連無料記事**: OpenAI検索で見つかった一次情報

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
export OPENAI_API_KEY="sk-..."
export NOTION_TOKEN="secret_..."

# 実行（新規DB作成）
./carbon-relay \
  -headlines=collected_headlines.json \
  -queriesPerHeadline=5 \
  -topK=3 \
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
./carbon-relay \
  -headlines=collected_headlines.json \
  -queriesPerHeadline=5 \
  -topK=3 \
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
| **Title** | Title | 記事タイトル | "Climate litigation marks 'turning point' in 2025" |
| **URL** | URL | 記事URL | https://carbon-pulse.com/470719/ |
| **Source** | Select | 記事ソース | "Carbon Pulse", "QCI", "OpenAI(text_extract)" |
| **Type** | Select | 記事タイプ | "Headline" または "Related Free" |
| **Excerpt** | Rich Text | 記事要約 | "Global climate litigation grew..." |
| **Score** | Number | マッチングスコア | 0.79（Related Freeのみ） |

---

## 🎨 Notion での活用例

### フィルタ設定

```
Type = "Headline" → 有料記事のみ表示
Type = "Related Free" → 無料記事のみ表示
Source = "Carbon Pulse" → Carbon Pulseのみ
Score > 0.5 → 高スコアの記事のみ
```

### ソート設定

```
Score（降順） → スコアの高い記事から表示
Created time（降順） → 新しい記事から表示
```

### ビュー作成例

1. **ヘッドライン一覧**（Table View）
   - Filter: `Type = "Headline"`
   - Sort: `Created time`（降順）

2. **高品質な無料記事**（Gallery View）
   - Filter: `Type = "Related Free" AND Score > 0.5`
   - Sort: `Score`（降順）

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
| `OPENAI_API_KEY` | ✅ | OpenAI API Key（検索時） |

---

## 📝 実行例

### 例1: ヘッドライン収集 → 検索 → Notionにクリッピング（一気通貫）

```bash
# 環境変数設定
export OPENAI_API_KEY="sk-..."
export NOTION_TOKEN="secret_..."

# 一気通貫実行
./carbon-relay \
  -sources=carbonpulse \
  -perSource=10 \
  -queriesPerHeadline=5 \
  -resultsPerQuery=10 \
  -topK=3 \
  -out=notion_clips.json \
  -notionClip \
  -notionPageID="abc123def456..."
```

### 例2: 既存のヘッドラインファイルをNotionにクリッピング

```bash
# 既に検索済みのresults.jsonをNotionにクリッピング
./carbon-relay \
  -headlines=search_results.json \
  -queriesPerHeadline=0 \
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
./carbon-relay ... -notionClip -notionPageID="abc123..."
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

export OPENAI_API_KEY="sk-..."
export NOTION_TOKEN="secret_..."
DB_ID="xyz789abc123..."  # 既存のDB ID

./carbon-relay \
  -sources=carbonpulse,qci \
  -perSource=20 \
  -queriesPerHeadline=5 \
  -topK=3 \
  -out="$(date +%Y%m%d)_clips.json" \
  -notionClip \
  -notionDatabaseID="$DB_ID"
```

### 2. 高品質記事のみクリッピング

```bash
# 事前にminScoreを高めに設定して高品質記事のみ収集
./carbon-relay \
  -headlines=collected_headlines.json \
  -queriesPerHeadline=5 \
  -minScore=0.5 \
  -topK=2 \
  -out=high_quality.json \
  -notionClip \
  -notionDatabaseID="$DB_ID"
```

### 3. ヘッドラインのみクリッピング（検索なし）

```bash
# 検索をスキップしてヘッドラインのみNotionに保存
./carbon-relay \
  -sources=carbonpulse \
  -perSource=30 \
  -queriesPerHeadline=0 \
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
