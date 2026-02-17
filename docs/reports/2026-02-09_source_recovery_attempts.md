# ソース復旧・改善レポート（2026-02-09）

## 概要

以前403エラーやボット保護で無効化されていたソースの復旧試行と、既存ソースの品質改善・バグ修正を行った。

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

## 4. Euractiv - コンテンツ抽出改善

### 問題

Euractivは以前からRSSフィード方式で動作していたが、以下の課題があった：

- RSSの`content:encoded`が常に空で、`description`は56〜116文字の短い要約のみ
- キーワードフィルタリングがタイトル＋短い説明文のみ対象で、カーボン関連記事の漏れが多い
- セクション別フィード（`/section/emissions-trading-scheme/feed/`等）はCloudflare 403でブロック

### 調査結果

| エンドポイント | ステータス |
|---------------|-----------|
| `/feed/` (メインRSS) | Go http.Client: **200** / curl: Cloudflare 403 |
| セクション別RSS | Go / curl ともに **403** |
| 記事ページ (HTML) | Go http.Client: **200** / curl: Cloudflare 403 |

- Go の http.Client はCloudflareを通過するが、curlはブロックされる（Nature.comと逆パターン）
- 記事ページの`div.c-news-detail__content`に本文テキストが含まれている
- 一部記事はEuractiv Pro（有料）で、本文が「Lorem ipsum」プレースホルダに置き換えられている
- RSSアイテムにカテゴリ情報（例: "Climate Change", "Energy", "Emissions Trading Scheme"）が付与されている

### 修正内容

#### 1. 記事ページスクレイピング追加
- `fetchEuractivArticleExcerpt()` 関数を `sources_rss.go` に追加
- `div.c-news-detail__content` セレクタで本文取得（script, style, ad等除去）
- ペイウォール検出: 「Lorem ipsum」検出時はRSS descriptionにフォールバック

#### 2. カテゴリベースのキーワードマッチング
- RSSアイテムのカテゴリをキーワードフィルタリング対象に追加
- タイトル + 説明文 + カテゴリでマッチング → ヒット率向上（~5/100 → ~18/100）

#### 3. キーワード改善
- `"energy"`, `"environment"`, `"sustainability"` を追加（EU政策サイトとして適切）
- `"eu ets"`, `"ets2"`, `"emissions trading"`, `"uk ets"` を追加
- `"ets"` 単体を削除（`"Markets"`, `"bets"`, `"Metsola"` 等の部分文字列一致を防止）

### テスト結果

```
Total articles: 10

  Free articles (full text):
  1. [2509 chars] Meloni brands violent Olympic protesters 'enemies of Italy'
  2. [6408 chars] Winter Olympics kick off as sustainability and safety concerns linger
  3. [2313 chars] Spain, Portugal brace for fresh storm after deadly flood
  4. [1717 chars] Russia hits Ukraine power grid with massive attack
  5. [1973 chars] Commission rolls out new Russia sanctions package after delays
  6. [1980 chars] US lawmakers urge EU to resist pressure to weaken methane rules
  7. [1974 chars] International Seabed Authority chief urges EU to back deep-sea mining rules

  Paywalled articles (RSS description fallback):
  8. [47 chars] HARVEST: Committees' crossover
  9. [95 chars] Multi-billion credit line to smooth introduction of carbon levy on fuels
  10. [97 chars] Agriculture and environment MEPs to share food safety simplification file
```

- Notion送信: 3件すべて成功（全文テキスト付き）
- **改善前**: Excerpt 56〜116文字（RSS descriptionのみ）
- **改善後**: Excerpt 1,717〜6,408文字（記事全文）

---

## 5. Climate Focus - バグ修正

### 問題

`a[href*='/publications/']` セレクタがページネーションリンクやステージング環境URLも取得してしまい、不正な記事が生成されていた。

### 発生した不正記事

| タイトル | URL | Excerpt |
|---------|-----|---------|
| `?Sf_paged=2` | `publications/?sf_paged=2` | 空 |
| `?Sf_paged=26` | `publications/?sf_paged=26` | 空 |
| `Publications` | `climatefocus.wpengine.com/publications/` | 空 |

### 原因

- ページネーションリンク（`div.nav-links`内の`?sf_paged=N`）が`a[href*='/publications/']`にマッチ
- リンクテキストが`"2"`, `"26"`等で10文字未満だがURLからタイトル生成のフォールバックが発動
- ステージング環境URL（`wpengine.com`）もセレクタにマッチ

### 修正内容

- `sf_paged=` を含むURLをスキップ（ページネーション除外）
- `wpengine.com` を含むURLをスキップ（ステージング環境除外）

### 修正後テスト結果

```
Total: 5件（すべて正常）
1. [941 chars] Carbon Market 2025 Review And Outlook (2026-01-27)
2. [1965 chars] Article 6 Implementation Checklist Tool (2025-12-01)
3. [2029 chars] Carbon Projects Brazil Amazon Guide (2025-11-20)
4. [267 chars] From Forest Pledges To Paris Delivery (2025-11-17)
5. [1501 chars] Food Forward NDCs 3.0 (2025-11-17)
```

---

## 6. METI審議会 - 一時無効化

### 問題

METI審議会（`meti`ソース）は記事に日付情報がなく、`FilterHeadlinesByHours`の「日付なし記事は保持」ルールにより、時間フィルタ使用時に常に全30件が含まれてしまう。本番テスト（48時間フィルタ）で全38件中30件がMETI審議会という偏った結果になった。

### 対応

- `headlines.go` のsourceMapでコメントアウト
- `config.go` のdefaultSourcesから除外（41→40ソース）
- 日付取得の改善後に再有効化を検討

---

## 7. FilterHeadlinesByHours - 未来日付除外の修正

### 問題

`FilterHeadlinesByHours`が未来の公開日付を持つ記事を除外していなかった。ScienceDirectの先行公開記事（日付: `2026-03-01`）が48時間フィルタを通過してしまっていた。

### 原因

フィルタ条件が `pubTime.After(cutoff)` のみで、cutoff時刻より後であれば未来日付でも保持されていた。

### 修正内容

```go
// 修正前
if pubTime.After(cutoff) {

// 修正後
now := time.Now()
if pubTime.After(cutoff) && !pubTime.After(now) {
```

cutoff〜現在時刻の範囲内のみ保持するよう変更。

### 修正後テスト結果（48時間フィルタ）

```
総記事数: 12（ScienceDirect 2件が正しく除外）
ソース数: 5

  Euractiv: 4件  avg 3236文字
  Carbon Herald: 3件  avg 2414文字
  RMI: 2件  avg 21791文字
  IISD ENB: 2件  avg 2385文字
  Politico EU: 1件  avg 2129文字
```

---

## 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `cmd/pipeline/sources_rss.go` | Carbon Market Watch RSS実装追加、Euractiv記事ページスクレイピング・キーワード改善 |
| `cmd/pipeline/sources_html.go` | Carbon Market Watch旧実装削除、Climate Focusページネーション除外 |
| `cmd/pipeline/headlines.go` | sourceMap更新、`fetchViaCurl()`追加、METI無効化、`FilterHeadlinesByHours`未来日付除外 |
| `cmd/pipeline/sources_academic.go` | Nature Communications: curl方式で復活、Abstract取得追加 |
| `cmd/pipeline/config.go` | `carbon-market-watch`追加、`meti`除外（41→40ソース） |
