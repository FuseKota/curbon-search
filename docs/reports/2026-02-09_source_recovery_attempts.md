# ソース復旧試行レポート（2026-02-09）

## 概要

以前403エラーやボット保護で無効化されていたソースについて、改めてアクセス可能か調査・修正を行った。

**作業期間**: 2026-02-09
**実装者**: Claude Code

---

## 1. Carbon Market Watch - 復旧成功

### 問題（2026-01以降）

HTMLスクレイピング（`/publications/`ページ）が403 Forbiddenで完全ブロック。
WordPress REST APIも同様に403。

### 調査結果

| エンドポイント | ステータス |
|---------------|-----------|
| `/publications/` (HTML) | 403 |
| `/wp-json/wp/v2/posts` (WP API) | 403 |
| `/feed/` (RSS) | **200** |

RSSフィード（`/feed/`）のみアクセス可能。`content:encoded`に記事全文が含まれている。

### 修正内容

- **方式変更**: HTMLスクレイピング → RSSフィード方式
- `sources_html.go` から旧実装（約90行）を削除
- `sources_rss.go` にRSS版 `collectHeadlinesCarbonMarketWatch()` を追加
- `headlines.go` のsourceMapをRSSセクションに移動・有効化
- `config.go` のデフォルトソースリストに追加（40→41ソース）

### テスト結果

```
Total articles: 3

  Source: Carbon Market Watch
  Title: The EU's approach to carbon removals won't remove the climate crisis...
  URL: https://carbonmarketwatch.org/2026/01/29/...
  Date: 2026-01-29T13:24:48Z
  Excerpt: 5833 chars

  Source: Carbon Market Watch
  Title: Make impact assessments great again (MIAGA)
  URL: https://carbonmarketwatch.org/2026/01/19/...
  Date: 2026-01-19T14:59:35Z
  Excerpt: 14464 chars

  Source: Carbon Market Watch
  Title: Proposed CBAM reforms serving industrial lobbies need climate refocus
  URL: https://carbonmarketwatch.org/2025/12/18/...
  Date: 2025-12-18T12:00:50Z
  Excerpt: 5738 chars
```

- Notion送信: 3件すべて成功

---

## 2. UNFCCC - 復旧不可

### 問題（2026-01以降）

Imperva/Incapsula のボット保護により全エンドポイントがブロック。
HTTPステータスは200を返すが、レスポンスボディはIncapsulaのJavaScriptチャレンジページ。

### 調査した全アプローチ

| アプローチ | 結果 |
|-----------|------|
| 直接HTMLアクセス (`/news`) | Incapsula チャレンジページ |
| RSSフィード (`/feed/`, `/rss.xml`, `/news/rss.xml`) | Incapsula チャレンジページ |
| Drupal JSON:API (`/jsonapi/node/news`) | Incapsula チャレンジページ |
| Drupal `?_format=json` | Incapsula チャレンジページ |
| www4サブドメイン | Incapsula チャレンジページ |
| Chrome User-Agent + フルブラウザヘッダー | Incapsula チャレンジページ |
| Safari User-Agent + Sec-Fetch ヘッダー | Incapsula チャレンジページ |
| Accept: application/vnd.api+json | Incapsula チャレンジページ |
| Referer: google.com 付きリクエスト | Incapsula チャレンジページ |
| Google News RSS (`site:unfccc.int`) | 記事はあるがURLが暗号化（protobuf）されており解決不可 |
| Wayback Machine CDX API | スナップショット自体がIncapsulaページを保存 |
| Googleキャッシュ | Google検索ページが返される |
| `/sitemap.xml` | Incapsula チャレンジページ |

### 技術的詳細

- **保護方式**: Imperva Incapsula（JavaScriptチャレンジ型）
- **対象範囲**: `unfccc.int` の全パス、全サブドメイン（`www4.unfccc.int` 含む）
- **HTTPステータス**: 200を返すが、ボディは約850バイトのHTML（JavaScriptチャレンジ）
- **Incapsula識別**: `<META NAME="ROBOTS" CONTENT="NOINDEX, NOFOLLOW">` + `_Incapsula_Resource` スクリプト
- **要件**: JavaScriptチャレンジを実行できるヘッドレスブラウザ（Puppeteer等）が必要

### 結論

`curl` やGoの `http.Client` ではIncapsulaのJavaScriptチャレンジを突破できない。
ヘッドレスブラウザの導入なしには復旧不可能。

### 対応

- `headlines.go` のsourceMapでコメントアウトを維持
- UN News Climate (`un-news`) も独立したソースとしてコメントアウト状態を維持

---

## 3. Nature Communications - 復旧成功

### 問題（2026-02以降）

Nature.comがFastlyのbot保護を導入。GoのTLSフィンガープリントを検出し、JavaScriptチャレンジページ（`Client Challenge`）を返す。HTTPステータスは200だがContent-Typeは`text/html`。

### 調査結果

| アクセス方法 | メインRSS (`/ncomms.rss`) | サブジェクトRSS (`/subjects/climate-change/ncomms.rss`) |
|-------------|-------------------------|------------------------------------------------------|
| curl | OK (RDF 1.0, 8件, 汎用) | OK (RSS 2.0, 30件, 気候変動フィルタ済み) |
| Go http.Client | JSチャレンジ | JSチャレンジ |

- **原因**: Fastlyのbot保護がTLSフィンガープリント（JA3/JA4）でGoを検出
- curlは異なるTLSフィンガープリントのためブロックされない

### 修正内容

- `fetchViaCurl()` ヘルパー関数を `headlines.go` に追加
  - `exec.Command("curl")` でHTTPリクエストを実行
  - GoのTLSフィンガープリント問題を回避
- サブジェクトフィード（`/subjects/climate-change/ncomms.rss`）を使用
  - 気候変動関連に事前フィルタ済み（30件）
  - キーワードフィルタ不要
- 各記事ページからAbstractを取得（`#Abs1-content p` セレクタ）

### テスト結果

```
Total articles: 3

  Source: Nature Communications
  Title: Record-breaking emergence of upstream-downstream zonal-consistent variation...
  Date: 2026-02-04T00:00:00Z
  Excerpt: 1403 chars

  Source: Nature Communications
  Title: Anthropogenically-driven escalating impact of soil-based compound dry-hot...
  Date: 2026-02-03T00:00:00Z
  Excerpt: 1399 chars

  Source: Nature Communications
  Title: Anthropogenic climate change drives rising global heat stress...
  Date: 2026-02-03T00:00:00Z
  Excerpt: 1045 chars
```

- Notion送信: 3件すべて成功

---

## 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `cmd/pipeline/sources_rss.go` | Carbon Market Watch RSS実装を追加 |
| `cmd/pipeline/sources_html.go` | Carbon Market Watch旧実装を削除 |
| `cmd/pipeline/headlines.go` | sourceMap更新、コメント整理、`fetchViaCurl()` 追加 |
| `cmd/pipeline/sources_academic.go` | Nature Communications: curl方式で復活、Abstract取得追加 |
| `cmd/pipeline/config.go` | デフォルトソースに `carbon-market-watch` 追加 |
