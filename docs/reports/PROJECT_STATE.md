# プロジェクト状態ドキュメント（完全復元用）

**最終更新：2025-12-31**

このドキュメントは、将来的にコンテキストが失われた場合でも、プロジェクトの状態を完全に思い出すためのものです。

---

## 目次

1. [プロジェクト概要](#プロジェクト概要)
2. [現在の実装状況](#現在の実装状況)
3. [重要な技術的決定](#重要な技術的決定)
4. [データフロー](#データフロー)
5. [アーキテクチャ](#アーキテクチャ)
6. [重要なコードセクション](#重要なコードセクション)
7. [トラブルシューティング履歴](#トラブルシューティング履歴)
8. [今後の拡張予定](#今後の拡張予定)

---

## プロジェクト概要

### 目的

カーボンクレジット市場に関する**有料ニュースメディアの見出し**から、その**背景となる無料の一次情報**を自動的に探索・収集するシステム。

### コアコンセプト

```
有料見出し（Carbon Pulse/QCI）
    ↓
OpenAI Web Search
    ↓
無料一次情報（政府資料、NGOレポート、現地メディア等）
    ↓
Notion Database（読みやすい形で保存）
```

### ユーザー価値

- 有料課金せずに記事の背景・根拠・周辺情報を追える
- 一次情報へのアクセスで信頼性の高い情報収集が可能
- Notionで一元管理、AI要約で効率的な情報整理

---

## 現在の実装状況

### ✅ 完全実装済み機能

#### 1. ヘッドライン収集（cmd/pipeline/headlines.go）

**有料ソース（見出しのみ）：**
- `collectHeadlinesCarbonPulse()` - Carbon Pulse timeline/newsletters
- `collectHeadlinesQCI()` - QCI homepage

**無料ソース（全文取得）：**
- `collectHeadlinesCarbonCreditsJP()` - 日本のカーボンクレジット市場ニュース
- `collectHeadlinesCarbonHerald()` - CDR技術・スタートアップ情報
- `collectHeadlinesClimateHomeNews()` - 国際交渉・政策情報
- `collectHeadlinesCarbonCreditscom()` - 初心者向け解説記事

**実装詳細：**
```go
// WordPress REST API パターン
apiURL := fmt.Sprintf(
    "https://carboncredits.jp/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content",
    limit,
)

// Content構造体でレスポンス受け取り
type WPPost struct {
    Title struct {
        Rendered string `json:"rendered"`
    } `json:"title"`
    Link    string `json:"link"`
    Date    string `json:"date"`
    Content struct {
        Rendered string `json:"rendered"`
    } `json:"content"`
}

// HTMLタグ除去 + エンティティデコード
content := cleanHTMLTags(p.Content.Rendered)
content = strings.TrimSpace(content)
```

**重要な技術的決定：**
- `Excerpt`ではなく`Content`を使用（全文取得のため）
- `html.UnescapeString()`で日本語文字・特殊記号をデコード
- Headline構造体の`Excerpt`フィールドに全文を格納（後方互換性）

#### 2. OpenAI検索統合（cmd/pipeline/search_openai.go）

**重要な発見と対策：**

❌ **問題：** OpenAI Responses APIは`web_search_call.results`を返さない
✅ **解決策：** `message.content`から正規表現でURL抽出

```go
// URL抽出パターン
urlPattern := regexp.MustCompile(`https?://[^\s\)\]]+`)
urls := urlPattern.FindAllString(content, -1)

// 疑似タイトル生成
title := generateTitleFromURL(url)
// 例: "carbon-pulse.com/timeline/..." → "Carbon Pulse Timeline"
```

#### 3. Notion統合（cmd/pipeline/notion.go）

**データベース管理：**

```go
// CreateDatabase() - 新規データベース作成
// 戻り値: (データベースID string, エラー error)
func (nc *NotionClipper) CreateDatabase(ctx context.Context, pageID string) (string, error)

// データベースIDを返すことで.envへの自動保存を実現
```

**全文保存機能：**

```go
// ClipHeadline() - 記事をNotionに保存
// 1. Pageプロパティを作成（Title, URL, Source等）
// 2. Pageを作成
// 3. Block.AppendChildren()で全文を追加

if h.Excerpt != "" {
    blocks := createContentBlocks(h.Excerpt)
    _, err = nc.client.Block.AppendChildren(ctx, notionapi.BlockID(page.ID), &notionapi.AppendBlockChildrenRequest{
        Children: blocks,
    })
}
```

**重要な技術的決定：**
- Pageプロパティは2000文字制限
- ページ本文は段落ブロックとして保存（無制限）
- `createContentBlocks()`で2000文字/ブロック単位に自動分割

**AI Summaryフィールド：**

```go
// ExcerptとAI Summaryの両方に同じ内容を入力
properties["Excerpt"] = notionapi.RichTextProperty{...}
properties["AI Summary"] = notionapi.RichTextProperty{...}
// → 後からNotion AIで手動要約可能
```

#### 4. データベースID永続化（cmd/pipeline/main.go）

**実装：**

```go
// appendToEnvFile() - .envファイルへの書き込み
func appendToEnvFile(filename, key, value string) error {
    // 1. 既存の.envファイルを読み込み
    // 2. キーが存在する場合は値を更新（コメントアウトも対応）
    // 3. 存在しない場合は新規追加
    // 4. ファイルに書き戻し
}
```

**動作フロー：**

```
初回実行:
  → 新規DB作成
  → DBIDを.envに保存
  → "✅ Database ID saved to .env file"

2回目以降:
  → .envからDBID読み込み
  → 既存DBに追加
  → "Using existing Notion database: xxx"
```

#### 5. スクリプト統合

**test-notion.sh:**
```bash
# .env読み込み
export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)

# 条件分岐
if [ -n "$NOTION_DATABASE_ID" ]; then
    # 既存DB使用
    ./carbon-relay -notionDatabaseID=$NOTION_DATABASE_ID ...
else
    # 新規DB作成
    ./carbon-relay -notionPageID=$NOTION_PAGE_ID ...
fi
```

**clip-all-sources.sh:**
```bash
# 全4無料ソースから5記事ずつ収集
-sources=carboncredits.jp,carbonherald,climatehomenews,carboncredits.com
```

---

## 重要な技術的決定

### 1. ExcerptフィールドからContentフィールドへの変更

**理由：**
- WordPress REST APIの`excerpt`は要約のみ（短い）
- `content`は全文コンテンツ
- ユーザーは全文を必要としていた（Notion AIで要約するため）

**影響：**
- APIリクエストURL変更: `_fields=...,excerpt` → `_fields=...,content`
- 構造体変更: `Excerpt struct` → `Content struct`
- 処理ロジック変更: `p.Excerpt.Rendered` → `p.Content.Rendered`

### 2. Notionページ本文への全文保存方式

**試行錯誤の歴史：**

❌ **失敗1:** PageCreateRequest.Childrenフィールドにブロックを渡す
```go
pageRequest.Children = createContentBlocks(h.Excerpt)
// → ブロックが追加されない（APIライブラリの制約）
```

✅ **成功:** 2段階アプローチ
```go
// 1. Pageを先に作成
page, err := nc.client.Page.Create(ctx, pageRequest)

// 2. Block.AppendChildren()でブロック追加
nc.client.Block.AppendChildren(ctx, notionapi.BlockID(page.ID), ...)
```

### 3. データベースID永続化の実装

**課題：**
- 毎回新しいデータベースが作成される
- ユーザーは既存DBに追加したい

**解決策：**
1. `CreateDatabase()`の戻り値を`(string, error)`に変更
2. `appendToEnvFile()`ヘルパー関数を実装
3. スクリプト側で`.env`から読み込み→コマンドライン引数として渡す

**重要なポイント：**
- Go側は環境変数を読まない（フラグのみ）
- Bash側で環境変数→フラグ変換が必要

### 4. AI Summaryフィールドの初期値

**決定：**
- 初期値として全文（最初の2000文字）を自動挿入
- ユーザーが後からNotion AIで手動要約

**理由：**
- Notion AI APIは公式にない
- OpenAI API経由の自動要約は追加コスト
- まず手動で試したいというユーザー要望

---

## データフロー

### 全体フロー

```
1. ヘッドライン収集
   ├─ Carbon Pulse (HTML scraping)
   ├─ QCI (HTML scraping)
   ├─ CarbonCredits.jp (WordPress API → 全文)
   ├─ Carbon Herald (WordPress API → 全文)
   ├─ Climate Home News (WordPress API → 全文)
   └─ CarbonCredits.com (WordPress API → 全文)
          ↓
2. 検索クエリ生成
   ├─ タイトル完全一致
   ├─ market/geo/topic キーワード
   └─ site:/filetype: 演算子
          ↓
3. OpenAI Web Search
   ├─ message.contentからURL抽出
   └─ 疑似タイトル生成
          ↓
4. マッチング・スコアリング
   ├─ IDF計算
   ├─ 類似度計算
   ├─ シグナル検出
   └─ ドメイン品質
          ↓
5. Notion保存
   ├─ データベース作成/選択
   ├─ プロパティ設定
   │   ├─ Excerpt (2000文字)
   │   └─ AI Summary (2000文字)
   ├─ ページ本文ブロック追加
   └─ データベースID保存
```

### Notionデータ構造

```
Notion Database
├─ Title (Title)
├─ URL (URL)
├─ Source (Select)
│   ├─ Carbon Pulse (Blue)
│   ├─ QCI (Green)
│   ├─ CarbonCredits.jp (Orange)
│   ├─ Carbon Herald (Pink)
│   ├─ Climate Home News (Purple)
│   ├─ CarbonCredits.com (Yellow)
│   ├─ OpenAI(text_extract) (Gray)
│   └─ Free Article (Default)
├─ Type (Select)
│   ├─ Headline (Red)
│   └─ Related Free (Green)
├─ Score (Number)
├─ Excerpt (Rich Text, 2000文字制限)
├─ AI Summary (Rich Text, 2000文字制限)
├─ Content (Rich Text, 将来の拡張用)
└─ Page Body (Blocks)
    └─ Paragraph Blocks (全文、2000文字/ブロック)
```

---

## アーキテクチャ

### ファイル責務

```
cmd/pipeline/
├── main.go
│   ├─ パイプライン全体のオーケストレーション
│   ├─ コマンドラインフラグ解析
│   ├─ .envファイル管理（appendToEnvFile）
│   └─ Notion統合フロー制御
│
├── headlines.go
│   ├─ 全6ソースのスクレイピング
│   ├─ HTML解析（goquery）
│   ├─ WordPress REST API統合
│   └─ HTML entity decoding
│
├── notion.go
│   ├─ NotionClipper構造体
│   ├─ CreateDatabase() - DB作成・IDを返す
│   ├─ ClipHeadline() - 記事保存
│   ├─ ClipRelatedFree() - 関連記事保存
│   ├─ createContentBlocks() - 2000文字ブロック分割
│   └─ truncateText() - 文字数制限
│
├── search_openai.go
│   ├─ OpenAI Responses API統合
│   ├─ URL抽出（正規表現）
│   └─ 疑似タイトル生成
│
├── search_queries.go
│   ├─ 検索クエリ生成戦略
│   ├─ market/geo/topic抽出
│   └─ site:/filetype:演算子
│
├── matcher.go
│   ├─ IDF計算
│   ├─ 類似度計算
│   ├─ シグナルベースマッチング
│   └─ スコアリング
│
├── types.go
│   ├─ Headline構造体
│   ├─ FreeArticle構造体
│   └─ RelatedFree構造体
│
└── utils.go
    └─ ユーティリティ関数
```

### 依存関係

```
外部ライブラリ:
├─ github.com/PuerkitoBio/goquery - HTMLパース
├─ github.com/jomei/notionapi - Notion API
└─ (標準ライブラリ) - HTTP, JSON, 正規表現等

外部API:
├─ OpenAI Responses API - Web検索
└─ Notion API - データベース管理
```

---

## 重要なコードセクション

### 1. WordPress全文取得

**場所:** `cmd/pipeline/headlines.go:525-596` (CarbonCredits.jp)

```go
// API URLで`content`を指定
apiURL := fmt.Sprintf(
    "https://carboncredits.jp/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content",
    limit,
)

// Content構造体を使用
type WPPost struct {
    // ...
    Content struct {
        Rendered string `json:"rendered"`
    } `json:"content"`
}

// HTML除去 + デコード
content := cleanHTMLTags(p.Content.Rendered)
content = strings.TrimSpace(content)
```

**同じパターン:**
- `collectHeadlinesCarbonHerald()` (lines 608-676)
- `collectHeadlinesClimateHomeNews()` (lines 679-747)
- `collectHeadlinesCarbonCreditscom()` (lines 750-818)

### 2. Notion全文保存（2段階アプローチ）

**場所:** `cmd/pipeline/notion.go:173-202`

```go
// 1段階目: Pageプロパティのみで作成
pageRequest := &notionapi.PageCreateRequest{
    Parent:     notionapi.Parent{...},
    Properties: properties,
}
page, err := nc.client.Page.Create(ctx, pageRequest)

// 2段階目: Block.AppendChildren()で全文追加
if h.Excerpt != "" {
    blocks := createContentBlocks(h.Excerpt)
    _, err = nc.client.Block.AppendChildren(ctx, notionapi.BlockID(page.ID), &notionapi.AppendBlockChildrenRequest{
        Children: blocks,
    })
}
```

### 3. 2000文字ブロック分割

**場所:** `cmd/pipeline/notion.go:324-395`

```go
func createContentBlocks(content string) notionapi.Blocks {
    const maxBlockSize = 2000

    // 1. 段落単位に分割
    paragraphs := []string{}
    for _, line := range strings.Split(content, "\n") {
        // 空行で段落区切り
    }

    // 2. 各段落をブロック化（長すぎる場合は分割）
    for _, para := range paragraphs {
        if len(para) <= maxBlockSize {
            // 1ブロックで収まる
            blocks = append(blocks, notionapi.ParagraphBlock{...})
        } else {
            // 2000文字ごとに分割
            for i := 0; i < len(para); i += maxBlockSize {
                end := i + maxBlockSize
                if end > len(para) {
                    end = len(para)
                }
                blocks = append(blocks, notionapi.ParagraphBlock{
                    Paragraph: notionapi.Paragraph{
                        RichText: []notionapi.RichText{{
                            Text: &notionapi.Text{
                                Content: para[i:end],
                            },
                        }},
                    },
                })
            }
        }
    }
}
```

### 4. データベースID永続化

**場所:** `cmd/pipeline/main.go:240-260`

```go
if *notionDatabaseID == "" {
    // 新規作成
    dbID, err := clipper.CreateDatabase(ctx, *notionPageID)

    // .envに保存
    if err := appendToEnvFile(".env", "NOTION_DATABASE_ID", dbID); err != nil {
        fmt.Fprintf(os.Stderr, "WARN: Failed to save database ID to .env: %v\n", err)
        fmt.Fprintf(os.Stderr, "Please manually add to .env:\nNOTION_DATABASE_ID=%s\n", dbID)
    } else {
        fmt.Fprintf(os.Stderr, "✅ Database ID saved to .env file\n")
    }
} else {
    // 既存DB使用
    fmt.Fprintf(os.Stderr, "Using existing Notion database: %s\n", *notionDatabaseID)
}
```

**場所:** `cmd/pipeline/main.go:285-320`

```go
func appendToEnvFile(filename, key, value string) error {
    // 既存ファイル読み込み
    content := ""
    data, err := os.ReadFile(filename)
    if err == nil {
        content = string(data)
    }

    // キー存在チェック（コメントアウトも対応）
    lines := strings.Split(content, "\n")
    keyExists := false
    for i, line := range lines {
        if strings.HasPrefix(line, key+"=") || strings.HasPrefix(line, "#"+key+"=") {
            lines[i] = key + "=" + value
            keyExists = true
            break
        }
    }

    // 存在しない場合は追加
    if !keyExists {
        lines = append(lines, key+"="+value)
    }

    // 書き戻し
    newContent := strings.Join(lines, "\n")
    return os.WriteFile(filename, []byte(newContent), 0644)
}
```

### 5. AI Summaryフィールド自動入力

**場所:** `cmd/pipeline/notion.go:171-181` (ClipHeadline)

```go
if h.Excerpt != "" {
    // Excerptフィールド
    properties["Excerpt"] = notionapi.RichTextProperty{
        Type: notionapi.PropertyTypeRichText,
        RichText: []notionapi.RichText{{
            Text: &notionapi.Text{
                Content: truncateText(h.Excerpt, 2000),
            },
        }},
    }

    // AI Summaryフィールド（同じ内容）
    properties["AI Summary"] = notionapi.RichTextProperty{
        Type: notionapi.PropertyTypeRichText,
        RichText: []notionapi.RichText{{
            Text: &notionapi.Text{
                Content: truncateText(h.Excerpt, 2000),
            },
        }},
    }
}
```

**同じパターン:** `cmd/pipeline/notion.go:268-278` (ClipRelatedFree)

---

## トラブルシューティング履歴

### 問題1: Notionページ本文が空

**症状:**
- Excerptプロパティには内容がある
- ページ本文（プロパティの下のエリア）は空

**原因:**
- `PageCreateRequest.Children`にブロックを渡しても反映されない
- Notion APIライブラリの制約

**解決策:**
- Page作成後に`Block.AppendChildren()`を使用する2段階アプローチ

**確認方法:**
```bash
DEBUG_SCRAPING=1 ./carbon-relay ...
# [DEBUG] Adding N content blocks to page (total chars: X)
```

### 問題2: 毎回新しいデータベースが作成される

**症状:**
- スクリプトを実行するたびに新規DB作成
- `.env`に`NOTION_DATABASE_ID`があっても無視される

**原因:**
- スクリプトが`.env`から読み込んでも、コマンドライン引数として渡していない
- Go側は環境変数ではなくフラグを見る

**解決策:**
```bash
# スクリプト側で条件分岐
if [ -n "$NOTION_DATABASE_ID" ]; then
    ./carbon-relay -notionDatabaseID=$NOTION_DATABASE_ID ...
else
    ./carbon-relay -notionPageID=$NOTION_PAGE_ID ...
fi
```

### 問題3: 日本語文字が文字化け

**症状:**
- `&nbsp;`、`&rdquo;`等のHTMLエンティティがそのまま表示

**解決策:**
```go
import "html"

content := cleanHTMLTags(p.Content.Rendered)
content = html.UnescapeString(content)  // ← 追加
```

---

## 今後の拡張予定

### 優先度：高

#### 1. 定期実行システム
- **目的:** 毎日自動でニュース収集
- **実装案:** cron or GitHub Actions
- **詳細:**
  ```bash
  # crontab例
  0 9 * * * cd /path/to/carbon-relay && ./clip-all-sources.sh
  ```

#### 2. Notion AI自動要約（API公開待ち）
- **現状:** 手動で要約
- **将来:** AI Summaryフィールドに自動要約挿入

### 優先度：中

#### 3. Brave Search API統合
- **目的:** OpenAI APIコスト削減、構造化データ取得
- **実装ファイル:** `cmd/pipeline/search_brave.go`

#### 4. 重複記事検出
- **目的:** 既にNotionにある記事はスキップ
- **実装案:** URL indexing + 既存ページクエリ

### 優先度：低

#### 5. Webインターフェース
- **技術スタック:** Next.js + Tailwind CSS
- **機能:**
  - ソース選択UI
  - 検索パラメータ調整
  - リアルタイム進捗表示

---

## コマンドリファレンス

### 開発・テスト

```bash
# ビルド
go build -o carbon-relay ./cmd/pipeline

# 単一ソーステスト（検索なし）
./carbon-relay -sources=carboncredits.jp -perSource=1 -queriesPerHeadline=0

# Notion統合テスト
./test-notion.sh

# 全ソースクリッピング
./clip-all-sources.sh

# デバッグモード
DEBUG_SCRAPING=1 ./carbon-relay -sources=carboncredits.jp -perSource=1 -queriesPerHeadline=0
```

### プロダクション想定

```bash
# 環境変数設定
cat > .env << 'EOF'
OPENAI_API_KEY=sk-...
NOTION_TOKEN=ntn_...
NOTION_PAGE_ID=xxx...
# NOTION_DATABASE_ID=xxx... (自動追加される)
EOF

# 初回実行（新規DB作成）
./clip-all-sources.sh

# 2回目以降（既存DB追加）
./clip-all-sources.sh  # 同じコマンド
```

---

## 環境・依存関係

### Go バージョン
```
go 1.21以上推奨
```

### 主要ライブラリ

```go
require (
    github.com/PuerkitoBio/goquery v1.8.1
    github.com/jomei/notionapi v1.13.1
)
```

### 環境変数（.env）

```bash
# 必須（検索機能使用時）
OPENAI_API_KEY=sk-...

# 必須（Notion統合）
NOTION_TOKEN=ntn_...
NOTION_PAGE_ID=xxx...

# 自動生成（手動設定不要）
NOTION_DATABASE_ID=xxx...

# オプション（デバッグ）
DEBUG_OPENAI=1
DEBUG_OPENAI_FULL=1
DEBUG_SCRAPING=1
```

---

## 復元手順

このドキュメントを使ってプロジェクトの状態を完全に思い出す手順：

1. **README.md** を読む - プロジェクト概要・使い方を把握
2. **このドキュメント（PROJECT_STATE.md）** を読む - 詳細な実装状況を把握
3. **重要なコードセクション** を確認 - 各ファイルの該当行を見る
4. **トラブルシューティング履歴** を確認 - 過去の問題と解決策を理解
5. **データフロー** と **アーキテクチャ** を見る - 全体像を掴む

---

## 最後に

このプロジェクトの最も重要な特徴：

1. **WordPress REST APIによる全文取得** - 無料ソースから完全な記事コンテンツを取得
2. **Notion全文保存** - 2000文字/ブロック制限に対応した段落分割
3. **データベースID永続化** - `.env`ファイル管理で重複DB作成を防止
4. **AI Summaryフィールド** - 後から手動要約できる柔軟な設計

**開発の経緯で最も重要だった技術的決定：**
- PageCreateRequestではなくBlock.AppendChildren()を使う
- CreateDatabase()がDBIDを返すようにする
- ExcerptではなくContentを使う
- appendToEnvFile()で#付きコメントも更新対象にする

これらの決定により、ユーザー要望（既存DB再利用、全文保存、AI要約）を実現できました。
