---
name: add-carbon-source
description: Carbon Relay（Go製カーボンニュース収集システム）に新しいニュースソースを追加するときに必ず使用すること。「新しいソースを追加したい」「このサイトを収集対象に加えたい」「新規ソース実装」など、特定WebサイトURLを提示してカーボンニュース収集を求めたときにトリガー。WordPress API / RSS / HTMLスクレイピングの自動判定から、Go実装・登録・テスト・ドキュメント更新まで一貫してガイドする。
---

# Carbon Relay: 新規ニュースソース追加ガイド

このスキルは、Carbon Relay に新しいカーボン関連ニュースソースを追加する全工程をガイドする。

---

## Phase 0: 情報収集

ユーザーから以下を確認する（会話にURLが既にあれば抽出して確認のみ）：

| 項目 | 説明 | 例 |
|------|------|-----|
| `url` | WebサイトのベースURL | `https://example.com` |
| `display_name` | 表示用ソース名 | `"Carbon Monitor"` |
| `cli_id` | CLI識別子（ケバブケース小文字） | `carbon-monitor` |

---

## Phase 1: パターン判定（優先順位順）

以下の順番でサイトを調査し、実装パターンを決定する。

### 1-A: WordPress REST API チェック
```bash
curl -s "{URL}/wp-json/wp/v2/posts?per_page=1" | jq '.[0].title.rendered'
```
- 200 + JSON レスポンス → **パターンA (WordPress)**

### 1-B: RSSフィード探索
```bash
curl -s -I "{URL}/feed/"
# または
curl -s "{URL}" | grep 'application/rss+xml'
```
- 有効なRSSフィード → **パターンB (RSS)**
- キーワードフィルタが必要な総合メディア・学術系 → **パターンB2 (RSS+フィルタ)**

### 1-C: HTMLスクレイピング
- 上記両方が失敗 → **パターンC (HTML)**
- DevTools で記事一覧のCSSセレクタを調査

### 追加判定

| 条件 | 対応 |
|------|------|
| URLに `.jp` 含む / 日本語サイト | `sources_japan.go` に追加 |
| 大学・研究誌・シンクタンク | `sources_academic.go` に追加 |
| 地域ETS（EU/UK/CARB/RGGI/豪州） | `sources_regional_ets.go` に追加 |
| Fastly / Cloudflare 等 TLS 保護 | `fetchViaCurl()` を使用 |
| bot 保護 (403) | `fetchViaCurl()` を使用 |

---

## Phase 2: パターン別実装テンプレート

### パターンA: WordPress REST API

**追加先**: `internal/pipeline/sources_wordpress.go`

```go
// collectHeadlines{SourceName} は {Display Name} から WordPress REST API でヘッドラインを収集する
func collectHeadlines{SourceName}(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
    return collectWordPressHeadlines("{BASE_URL}", "{Display Name}", limit, cfg)
}
```

### パターンB: RSS（シンプル）

**追加先**: `internal/pipeline/sources_rss.go`

```go
// collectHeadlines{SourceName} は {Display Name} の RSS フィードからヘッドラインを収集する
func collectHeadlines{SourceName}(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
    feed, err := fetchRSSFeed("{FEED_URL}", cfg)
    if err != nil {
        return nil, err
    }

    out := make([]Headline, 0, limit)
    for _, item := range feed.Items {
        if len(out) >= limit {
            break
        }
        title := strings.TrimSpace(item.Title)
        if title == "" {
            continue
        }
        dateStr := ""
        if item.PublishedParsed != nil {
            dateStr = item.PublishedParsed.Format(time.RFC3339)
        }
        out = append(out, Headline{
            Source:      "{Display Name}",
            Title:       title,
            URL:         item.Link,
            PublishedAt: dateStr,
            Excerpt:     extractRSSExcerpt(item),
        })
    }
    return out, nil
}
```

### パターンB2: RSS + キーワードフィルタ

**追加先**: パターンBと同じファイル（総合メディア・学術系向け）

```go
// {sourceName}Keywords は {Display Name} のキーワードフィルタ
var {sourceName}Keywords = []string{
    "carbon", "emission", "climate", "net zero", "greenhouse",
    "carbon market", "carbon credit", "carbon offset",
    // 必要に応じて追加
}

// collectHeadlines{SourceName} は {Display Name} の RSS フィードから
// カーボン関連記事をキーワードフィルタして収集する
func collectHeadlines{SourceName}(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
    feed, err := fetchRSSFeed("{FEED_URL}", cfg)
    if err != nil {
        return nil, err
    }

    out := make([]Headline, 0, limit)
    for _, item := range feed.Items {
        if len(out) >= limit {
            break
        }
        title := strings.TrimSpace(item.Title)
        if title == "" {
            continue
        }
        excerpt := extractRSSExcerpt(item)
        if !matchesKeywords(title+" "+excerpt, {sourceName}Keywords) {
            continue
        }
        dateStr := ""
        if item.PublishedParsed != nil {
            dateStr = item.PublishedParsed.Format(time.RFC3339)
        }
        out = append(out, Headline{
            Source:      "{Display Name}",
            Title:       title,
            URL:         item.Link,
            PublishedAt: dateStr,
            Excerpt:     excerpt,
        })
    }
    return out, nil
}
```

### パターンC: HTMLスクレイピング

**追加先**: `internal/pipeline/sources_html.go`

```go
// collectHeadlines{SourceName} は {Display Name} の記事一覧ページから
// HTMLスクレイピングでヘッドラインを収集する
func collectHeadlines{SourceName}(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
    doc, err := fetchDoc("{LIST_PAGE_URL}", cfg)
    if err != nil {
        return nil, err
    }

    out := make([]Headline, 0, limit)
    seen := make(map[string]bool)

    doc.Find("{ARTICLE_SELECTOR}").Each(func(_ int, s *goquery.Selection) {
        if len(out) >= limit {
            return
        }

        // タイトルとURL取得
        a := s.Find("{TITLE_LINK_SELECTOR}")
        title := strings.TrimSpace(a.Text())
        href, _ := a.Attr("href")
        if title == "" || href == "" {
            return
        }

        articleURL := resolveURL("{BASE_URL}", href)
        if seen[articleURL] {
            return
        }
        seen[articleURL] = true

        // Excerpt取得（任意）
        excerpt := strings.TrimSpace(s.Find("{EXCERPT_SELECTOR}").Text())

        // 日付取得（任意）
        dateStr := ""
        dateText, _ := s.Find("{DATE_SELECTOR}").Attr("datetime")
        if dateText != "" {
            if t, err := time.Parse("2006-01-02", dateText); err == nil {
                dateStr = t.UTC().Format(time.RFC3339)
            }
        }

        out = append(out, Headline{
            Source:      "{Display Name}",
            Title:       title,
            URL:         articleURL,
            PublishedAt: dateStr,
            Excerpt:     excerpt,
        })
    })

    return out, nil
}
```

---

## Phase 3: ソース登録

### 3-A: `sourceCollectors` マップに追加

ファイル: `internal/pipeline/headlines.go`（行143〜217付近）

対応するセクションを見つけて追加：

```go
// sources_wordpress.go - WordPress REST API ソース
"{cli-id}": collectHeadlines{SourceName},

// sources_rss.go - RSS/Atom フィードソース
"{cli-id}": collectHeadlines{SourceName},

// sources_academic.go - 学術・研究機関ソース
"{cli-id}": collectHeadlines{SourceName},

// sources_regional_ets.go - 地域排出量取引システム
"{cli-id}": collectHeadlines{SourceName},

// sources_html.go - HTMLスクレイピングソース
"{cli-id}": collectHeadlines{SourceName},

// sources_japan.go - 日本語ソース
"{cli-id}": collectHeadlines{SourceName},
```

### 3-B: `DefaultSources` に追加

ファイル: `internal/pipeline/config.go`

```go
// 現在の末尾: "...isometric"
// 変更後:
const DefaultSources = "...isometric,{cli-id}"
```

---

## Phase 4: テストと検証

### 4-A: スクレイピング動作テスト

```bash
# 1. ビルド
go build -o pipeline ./cmd/pipeline

# 2. 最小テスト（1件）
./pipeline -sources={cli-id} -perSource=1

# 3. デバッグモード（問題発生時）
DEBUG_SCRAPING=1 ./pipeline -sources={cli-id} -perSource=1

# 4. 通常テスト（5件）
./pipeline -sources={cli-id} -perSource=5 -out=/tmp/test_{cli-id}.json
cat /tmp/test_{cli-id}.json | jq '.'
```

### 成功判定基準

| チェック項目 | 条件 |
|-------------|------|
| 記事件数 | `count > 0` |
| タイトル | 空でない、意味のある文字列 |
| URL | `https://` で始まる有効なURL |
| PublishedAt | RFC3339形式（例: `2026-01-05T12:00:00Z`） |
| Excerpt | 50文字以上（理想は200文字以上） |

### 4-B: Notion 連携テスト

スクレイピングが成功したら、Notion に実際に記事が登録されるか確認する。

```bash
# 既存 Database ID が .env に設定済みの場合（通常運用）
./pipeline -sources={cli-id} -perSource=3 -notionClip

# 初回セットアップ（NOTION_DATABASE_ID が未設定の場合）
./pipeline -sources={cli-id} -perSource=3 -notionClip -notionPageID=$NOTION_PAGE_ID
```

**Notion 確認ポイント**:
1. Notion のデータベースを開き、新しく追加された記事を確認
2. 以下の項目が正しく入っていることをチェック：

| Notion プロパティ | 期待値 |
|-----------------|--------|
| Title | 記事タイトル（空でない） |
| Source | `{Display Name}` |
| URL | 記事の URL |
| PublishedDate | 日付（空でないことが望ましい） |
| Article Summary 300 | 要約テキスト（メールダイジェスト除外条件に関係） |

> **注意**: PublishedDate が空・Article Summary 300 が空の記事はメールダイジェストから除外される。
> 日付が取れていない場合はスクレイピングロジックの日付パースを見直すこと。

**エラー例と対処**:

| エラー | 原因 | 対処法 |
|--------|------|--------|
| `NOTION_API_KEY is required` | `.env` に `NOTION_API_KEY` が未設定 | `.env` を確認 |
| `NOTION_DATABASE_ID is not set` | データベース未作成 | `-notionPageID` フラグを付けて初回実行 |
| Notion に記事が現れない | `notionClip` が false / ビルド未更新 | ビルドを確認、フラグを確認 |
| 記事が重複登録される | URL の正規化問題 | `resolveURL()` の出力を確認 |

### エラー対処表（スクレイピング）

| エラー | 原因 | 対処法 |
|--------|------|--------|
| `unknown source: {cli-id}` | `sourceCollectors` 登録漏れ | マップへの追加を確認 |
| `status 403` | bot 保護 | `fetchViaCurl()` に変更 or RSS に切り替え |
| `status 429` | レート制限 | `-perSource=1` に下げる、arXiv等は他ソースと同時テストを避ける |
| `returned 0 headlines` | セレクタ不一致 | DevTools で HTML 構造を再確認 |
| Excerpt が空 | セレクタ不一致 | 別セレクタを試す or 記事ページスクレイピング追加 |
| TLS handshake error | Fastly/Cloudflare | `fetchViaCurl()` に変更 |
| Excerpt が短い（< 50文字） | RSS description が貧弱 | 記事ページから `content:encoded` or スクレイピング |

---

## Phase 5: ドキュメント更新

### 5-A: 実装ガイドのソース一覧更新

ファイル: `docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md`

セクション3のソース一覧テーブルに追加：
```
| {Display Name} | {cli-id} | {パターン} | {説明} |
```

### 5-B: CLAUDE.md の変更ログ更新

ファイル: `CLAUDE.md`

「最近の技術的変更」セクションに追記：
```
### 新規ソース追加（{日付}）
- **{Display Name}**: {パターン}（{簡単な説明}）
```

---

## 参照ファイル

| ファイル | 用途 |
|---------|------|
| `internal/pipeline/headlines.go` | `sourceCollectors` マップ（行143-217）、ヘルパー関数群 |
| `internal/pipeline/sources_wordpress.go` | WordPress パターン例 |
| `internal/pipeline/sources_rss.go` | RSS パターン例 |
| `internal/pipeline/sources_html.go` | HTML スクレイピングパターン例 |
| `internal/pipeline/sources_japan.go` | 日本語ソースパターン例 |
| `internal/pipeline/sources_academic.go` | 学術系パターン例 |
| `internal/pipeline/sources_regional_ets.go` | 地域ETS パターン例 |
| `internal/pipeline/config.go` | `DefaultSources` 定数（行102） |

---

## 重要な注意事項

- **リクエスト間隔**: スクレイピング時は適切な間隔を保つ（bot 判定を避ける）
- **テストは小さく始める**: まず `-perSource=1` で動作確認してから増やす
- **arXiv**: IPベースのレート制限（429）が厳しいため、他ソースと同時テストを避ける
- **Excerpt 切り詰め**: `CollectFromSources()` で一括4000文字に切り詰め済み（各ソース関数内で切り詰め不要）
- **UTC統一**: 日付は全て UTC の RFC3339 形式で格納