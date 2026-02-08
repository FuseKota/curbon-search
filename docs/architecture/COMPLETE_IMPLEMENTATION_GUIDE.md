# Carbon Relay - Complete Implementation Guide 2026

**最終更新**: 2026年1月4日
**バージョン**: 1.0
**ステータス**: Production Ready

---

## 📑 目次

1. [プロジェクト概要](#1-プロジェクト概要)
2. [システムアーキテクチャ](#2-システムアーキテクチャ)
3. [全ソースの実装詳細](#3-全ソースの実装詳細)
4. [データ処理パイプライン](#4-データ処理パイプライン)
5. [スコアリング・マッチングアルゴリズム](#5-スコアリングマッチングアルゴリズム)
6. [Notion統合](#6-notion統合)
7. [設定とコンフィグレーション](#7-設定とコンフィグレーション)
8. [使用方法と実行例](#8-使用方法と実行例)
9. [最近の修正と改善](#9-最近の修正と改善)
10. [トラブルシューティング](#10-トラブルシューティング)

---

## 1. プロジェクト概要

### 1.1 プロジェクトの目的

**Carbon Relay**は、カーボン関連ニュースの収集・分析・配信を自動化するGo製インテリジェンスシステムです。

### 1.2 運用モード

#### 🟢 無料記事収集モード（Free Article Collection Mode）

**目的**: Carbon関連の無料記事を幅広く収集し、要約してメール配信

**フロー**:
```
Carbon関連の無料記事を幅広く確認
    ↓
その日のニュースをまとめて、各記事300文字程度のNotionAiで要約
    ↓
まとめたニュースをメール配信
```

**使用例**:
```bash
# 20の無料ソースから幅広く記事を収集
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -sendEmail
```

**特徴**:
- 20の無料ソースから直接記事を収集
- 高速実行（5-15秒）
- メール配信・Notion統合に対応

---

### 1.3 主要機能

- ✅ 20の情報ソースからのニュース自動収集
- ✅ HTML/RSS/WordPress API によるスクレイピング
- ✅ メール送信機能（Gmail SMTP）
- ✅ Notion Databaseへの自動クリッピング

### 1.4 プロジェクト統計

| 項目 | 値 |
|------|-----|
| 総コード行数 | 4,751行（Go） |
| 実装ソース数 | 20（無料ソースのみ） |
| テスト成功率 | 100%（15/15テスト合格） |
| 実装期間 | 2025年12月29日 - 2026年1月4日 |
| ステータス | 本番環境対応済み |

### 1.5 技術スタック

**プログラミング言語**: Go 1.23
**主要ライブラリ**:
- `github.com/PuerkitoBio/goquery v1.10.2` - HTML解析
- `github.com/mmcdole/gofeed v1.3.0` - RSS/Atomフィード解析
- `github.com/jomei/notionapi v1.13.3` - Notion API クライアント
- `github.com/joho/godotenv v1.5.1` - 環境変数管理

**API統合**:
- OpenAI Responses API（Web検索）
- Notion API（データベース統合）
- Gmail SMTP（メール送信）

---

## 2. システムアーキテクチャ

### 2.1 ディレクトリ構造

```
/Users/kotafuse/Yasui/Prog/Test/carbon-relay/
├── cmd/
│   └── pipeline/
│       ├── main.go              (515行) - パイプライン制御とCLI
│       ├── headlines.go         (2,354行) - 18ソース実装
│       ├── matcher.go           (506行) - IDFスコアリング
│       ├── search_openai.go     (295行) - OpenAI検索統合
│       ├── search_queries.go    (232行) - クエリ生成
│       ├── notion.go            (554行) - Notion統合
│       ├── email.go             (175行) - メール送信
│       ├── types.go             (42行) - データ構造
│       └── utils.go             (78行) - ユーティリティ
├── .env                         - 環境変数設定
├── .env.example                 - 環境変数サンプル
├── go.mod                       - Go依存関係
├── go.sum                       - 依存関係ハッシュ
└── [各種ドキュメント]
```

### 2.2 コアモジュール

#### main.go (パイプライン制御)
**責務**:
- コマンドラインフラグ解析（13フラグ）
- 環境変数読み込み（godotenv）
- パイプライン全体の制御
- エラーハンドリングとログ出力
- Database ID の自動保存

**主要フラグ**:
```go
-sources           // 収集するソース（CSV形式）
-perSource         // ソースあたりの最大記事数
-queriesPerHeadline // 記事あたりの検索クエリ数
-notionClip        // Notionへクリップ
-sendEmail         // メール送信
```

#### headlines.go (ソース実装)
**責務**:
- 18ニュースソースの実装
- 複数のスクレイピングパターン:
  - WordPress REST API（7ソース）
  - HTML Scraping with goquery（8ソース）
  - RSS Feed with gofeed（3ソース）
- キーワードフィルタリング（日本語ソース）
- URL重複排除
- Excerpt抽出

**コード比率**: 全体の49.6%

#### matcher.go (スコアリング)
**責務**:
- IDF（逆文書頻度）ベーススコアリング
- シグナル抽出（Markets, Topics, Geos）
- トークン正規化とストップワード除去
- 複数の類似度メトリクス計算
- ソース品質ブースト
- 新しさスコア計算

#### search_openai.go (Web検索)
**責務**:
- OpenAI Responses API統合
- 3段階フォールバックでURL抽出
- 擬似タイトル生成
- 正規表現ベースURL抽出
- デバッグログサポート

#### notion.go (Notion統合)
**責務**:
- Notion Database統合
- データベース自動作成
- Database IDの自動永続化
- リッチテキスト分割（2000文字/ブロック）
- コンテンツブロック作成
- 公開日パース（複数フォーマット対応）

### 2.3 データフロー図

```
┌─────────────────────────────────────────────────────────────┐
│ Phase 1: Collection                                         │
├─────────────────────────────────────────────────────────────┤
│  ユーザー入力（CLI flags）                                     │
│         ↓                                                    │
│  ソース選択（-sources flag）                                  │
│         ↓                                                    │
│  ソースごとのスクレイピング（limit: -perSource）                │
│         ↓                                                    │
│  URL重複排除（uniqueHeadlinesByURL）                          │
│         ↓                                                    │
│  Headline[] with Excerpts                                   │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ Phase 2: Search (if queriesPerHeadline > 0)                │
├─────────────────────────────────────────────────────────────┤
│  各Headlineに対して:                                          │
│    ↓                                                        │
│  検索クエリ生成（buildSearchQueries - 7戦略）                 │
│    ↓                                                        │
│  OpenAI Web Search（クエリごと）                             │
│    - HTTP POST to api.openai.com/v1/responses              │
│    - Model: gpt-4o-mini                                    │
│    - Tool: web_search                                      │
│    ↓                                                        │
│  URL抽出（3段階フォールバック）                                │
│    - web_search_call.results（理想、通常は空）                │
│    - action.sources（URLのみ）                               │
│    - テキスト正規表現抽出（主要手法）                           │
│    ↓                                                        │
│  擬似タイトル生成                                              │
│    ↓                                                        │
│  URL重複排除（global seen map）                              │
│    ↓                                                        │
│  FreeArticle[] candidates（見出しごと）                       │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ Phase 3: Matching                                           │
├─────────────────────────────────────────────────────────────┤
│  IDFコーパス構築                                               │
│    - 全見出しタイトル + 全候補タイトル                           │
│    ↓                                                        │
│  各Headline-Candidateペアに対して:                            │
│    ↓                                                        │
│  トークン化（reTok regex）                                    │
│    ↓                                                        │
│  シグナル抽出                                                 │
│    - Markets: eua, uka, rggi, cca, etc.                    │
│    - Topics: vcm, cdr, dac, beccs, etc.                    │
│    - Geos: US, UK, EU, China, Japan, etc.                  │
│    ↓                                                        │
│  類似度スコア計算                                              │
│    - IDF加重リコール類似度（56%）                              │
│    - IDF加重Jaccard（28%）                                   │
│    - マーケットマッチ（6%）                                    │
│    - トピックマッチ（4%）                                      │
│    - 地理的マッチ（2%）                                        │
│    - 新しさ（4%）                                            │
│    - 品質ブースト（最大+0.18）                                │
│    ↓                                                        │
│  フィルタリング:                                              │
│    - 最小スコア（-minScore, default 0.32）                   │
│    - 厳格マーケットマッチ（見出しにマーケットシグナルがある場合） │
│    - 特定地域マッチ（見出しに特定地域がある場合）                 │
│    - 共有トークン >= 2（titleSim >= 0.90でない限り）           │
│    ↓                                                        │
│  スコアでソート（降順）                                         │
│    ↓                                                        │
│  上位K件取得（-topK, default 3）                              │
│    ↓                                                        │
│  RelatedFree[] per headline                                 │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ Phase 4: Output                                             │
├─────────────────────────────────────────────────────────────┤
│  JSON出力                                                    │
│    - stdout または ファイル（-out flag）                       │
│    ↓                                                        │
│  Notionクリッピング（-notionClip指定時）                       │
│    - データベース作成/再利用                                   │
│    - 見出し + 関連記事をクリップ                               │
│    - フルコンテンツをブロックに保存                             │
│    - AI Summaryフィールド保存                                 │
│    ↓                                                        │
│  メール送信（-sendEmail指定時）                                │
│    - Notionから取得                                          │
│    - プレーンテキストサマリー生成                               │
│    - Gmail SMTP経由で送信                                    │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 全ソースの実装詳細

### 3.1 無料ソース - 日本市場（7ソース）

#### ソース1: CarbonCredits.jp
**実装**: `collectHeadlinesCarbonCreditsJP()`
**手法**: WordPress REST API
**エンドポイント**: `https://carboncredits.jp/wp-json/wp/v2/posts`
**フィールド**: title, link, date, content

**特徴**:
- 日本語での完全な記事コンテンツ
- 標準WordPressフィールド
- HTMLタグのクリーニング

**ステータス**: ✅ 完全動作

**コード例**:
```go
apiURL := "https://carboncredits.jp/wp-json/wp/v2/posts?per_page=30&_fields=title,link,date,content"

type WPPost struct {
    Title   struct{ Rendered string } `json:"title"`
    Link    string `json:"link"`
    Date    string `json:"date"`
    Content struct{ Rendered string } `json:"content"`
}

var posts []WPPost
json.Unmarshal(body, &posts)

for _, p := range posts {
    title := cleanHTMLTags(p.Title.Rendered)
    excerpt := extractExcerpt(p.Content.Rendered, 500)
}
```

#### ソース2: Japan Research Institute (JRI - 日本総研)
**実装**: `collectHeadlinesJRI()`
**手法**: RSS Feed（gofeed）
**フィードURL**: `https://www.jri.co.jp/xml.jsp?id=12966`

**特徴**:
- カーボン関連キーワードフィルタリング（オプション）
- 記事ページから完全コンテンツ取得
- 複数のコンテンツセレクタを試行

**キーワード例**: カーボン, 炭素, 脱炭素, CO2, GHG, etc.

**ステータス**: ✅ 完全動作

**コード例**:
```go
fp := gofeed.NewParser()
feed, _ := fp.Parse(resp.Body)

for _, item := range feed.Items {
    title := item.Title
    url := item.Link
    publishedAt := item.Published

    // 記事ページから完全コンテンツ取得
    excerpt := fetchFullContent(url)
}
```

#### ソース3: Japan Environment Ministry（環境省）
**実装**: `collectHeadlinesEnvMinistry()`
**手法**: HTML Scraping + キーワードフィルタリング
**収集URL**: `https://www.env.go.jp/press/`

**特徴**:
- 日本語形式からの日付抽出（YYYY年MM月DD日）
- カーボン/気候トピックのキーワードフィルタリング
- プレスリリースページから完全コンテンツ取得

**キーワード**: 18のカーボン関連用語（日本語）
```go
carbonKeywords := []string{
    "カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
    "気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
    "カーボンクレジット", "クレジット市場", "JCM", "二国間クレジット",
    "カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
}
```

**ステータス**: ✅ 完全動作

**日付解析例**:
```go
dateText := "2025年12月26日発表"
var year, month, day int
fmt.Sscanf(dateText, "%d年%d月%d日", &year, &month, &day)
currentDate = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
```

#### ソース4: Japan Exchange Group (JPX)
**実装**: `collectHeadlinesJPX()`
**手法**: RSS Feed（gofeed）
**フィードURL**: `https://www.jpx.co.jp/rss/jpx-news.xml`

**特徴**:
- カーボンクレジットトピックのキーワードフィルタリング
- RSS標準日付解析

**キーワード**: カーボン, クレジット, GX, 取引, etc.

**ステータス**: ✅ 完全動作

#### ソース5: Japan Ministry of Economy (METI - 経済産業省)
**実装**: `collectHeadlinesMETI()`
**手法**: RSS Feed（gofeed）
**フィードURL**: `https://www.chusho.meti.go.jp/rss/index.xml`

**特徴**:
- 拡張タイムアウト（60秒）
- 包括的キーワードリスト（20+用語）
- テスト用にすべての記事を収集（キーワードフィルタ無効）

**ステータス**: ✅ 完全動作（マッチなしの場合空を返す）

#### ソース6: Mizuho Research & Technologies（みずほリサーチ＆テクノロジーズ）
**実装**: `collectHeadlinesMizuhoRT()`
**手法**: HTML Scraping + キーワードフィルタリング
**収集URL**: `https://www.mizuho-rt.co.jp/publication/2025/index.html`

**特徴**:
- 正規表現パターンでの日付抽出
- サステナビリティキーワードフィルタリング（20+用語）
- /business/ と /publication/ パスのリンクフィルタリング

**日付解析例**:
```go
datePattern := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
matches := datePattern.FindStringSubmatch(dateText)
if len(matches) == 4 {
    year, month, day := matches[1], matches[2], matches[3]
    publishedAt = fmt.Sprintf("%s-%02s-%02sT00:00:00Z", year, month, day)
}
```

**ステータス**: ✅ 完全動作

#### ソース7: PwC Japan
**実装**: `collectHeadlinesPwCJapan()`
**手法**: HTML Scraping（複雑なJSON抽出）
**収集URL**: `https://www.pwc.com/jp/ja/knowledge/column/sustainability.html`

**特徴**:
- angular.loadFacetedNavigationスクリプトからJSON抽出
- 3重エスケープされたJSONのアンエスケープ
- 日付解析（YYYY-MM-DD形式）
- 動的コンテンツ処理

**特別処理**:
- JSON抽出用正規表現パターン
- 複数回のアンエスケープイテレーション
- ブロッキング回避のためのブラウザ風ヘッダー

**実装詳細**（2026年1月4日修正）:
```go
// JavaScript関数呼び出しから埋め込みJSONを抽出
jsonPattern := regexp.MustCompile(`"(\{\\x22numberHits\\x22:\d+,\\x22elements\\x22:.*?\\x22filterTags\\x22:.*?\})"`)
matches := jsonPattern.FindAllStringSubmatch(bodyStr, -1)

// 16進エスケープされた引用符をアンエスケープ
jsonStr = strings.ReplaceAll(jsonStr, `\x22`, `"`)
jsonStr = strings.ReplaceAll(jsonStr, `\/`, `/`)
jsonStr = strings.ReplaceAll(jsonStr, `\u002D`, `-`)

// 3重エスケープされた要素配列をアンエスケープ（2回実行）
for i := 0; i < 2; i++ {
    elementsStr = strings.ReplaceAll(elementsStr, `\\`, "\x00")
    elementsStr = strings.ReplaceAll(elementsStr, `\"`, `"`)
    elementsStr = strings.ReplaceAll(elementsStr, "\x00", `\`)
}

// 個別記事オブジェクトを解析
titlePattern := regexp.MustCompile(`"title":"([^"]+)"`)
hrefPattern := regexp.MustCompile(`"href":"([^"]+)"`)
datePattern := regexp.MustCompile(`"publishDate":"([^"]*)"`)
```

**ステータス**: ✅ 実装済み（2026年1月4日修正で動作確認）

---

### 3.2 無料ソース - ヨーロッパ＆国際（6ソース）

#### ソース8: Sandbag
**実装**: `collectHeadlinesSandbag()`
**手法**: WordPress REST API
**エンドポイント**: `https://sandbag.be/wp-json/wp/v2/posts`
**焦点**: EU ETS分析

**コンテンツ**: HTMLクリーニング付き完全記事
**ステータス**: ✅ 完全動作

#### ソース9: Ecosystem Marketplace
**実装**: `collectHeadlinesEcosystemMarketplace()`
**手法**: WordPress REST API
**エンドポイント**: `https://www.ecosystemmarketplace.com/wp-json/wp/v2/posts`
**焦点**: 自然ベース解決策（NbS）市場

**コンテンツ**: 完全記事
**ステータス**: ✅ 完全動作

#### ソース10: Carbon Brief
**実装**: `collectHeadlinesCarbonBrief()`
**手法**: WordPress REST API
**エンドポイント**: `https://www.carbonbrief.org/wp-json/wp/v2/posts`
**焦点**: 気候科学と政策

**コンテンツ**: 完全記事
**ステータス**: ✅ 完全動作

#### ソース11: Climate Home News
**実装**: `collectHeadlinesClimateHomeNews()`
**手法**: WordPress REST API
**エンドポイント**: `https://www.climatechangenews.com/wp-json/wp/v2/posts`
**焦点**: 国際交渉と政策

**コンテンツ**: 完全記事
**ステータス**: ✅ 完全動作

#### ソース12: ICAP (International Carbon Action Partnership)
**実装**: `collectHeadlinesICAP()`
**手法**: HTML Scraping + 完全コンテンツ取得
**収集URL**: `https://icapcarbonaction.com/en/news`

**特徴**:
- 記事グリッド解析
- time要素からの日付抽出
- 記事ページから完全コンテンツ取得
- コンテンツセレクタ: `div.field-body`

**ステータス**: ✅ 完全動作

#### ソース13: IETA (International Emissions Trading Association)
**実装**: `collectHeadlinesIETA()`
**手法**: HTML Scraping + 完全コンテンツ取得
**収集URL**: `https://www.ieta.org/`

**特徴**:
- card-bodyコンテナ解析
- 日付解析（"Dec 18, 2025" 形式）
- 兄弟要素 a.link-cover からリンク抽出
- 完全コンテンツ取得

**ステータス**: ✅ 完全動作

---

### 3.3 無料ソース - グローバルメディア（3ソース）

#### ソース14: Carbon Herald
**実装**: `collectHeadlinesCarbonHerald()`
**手法**: WordPress REST API
**エンドポイント**: `https://carbonherald.com/wp-json/wp/v2/posts`
**焦点**: CDR技術とスタートアップ

**コンテンツ**: 完全記事
**ステータス**: ✅ 完全動作

#### ソース15: CarbonCredits.com
**実装**: `collectHeadlinesCarbonCreditscom()`
**手法**: WordPress REST API
**エンドポイント**: `https://carboncredits.com/wp-json/wp/v2/posts`
**焦点**: 初心者向けコンテンツ

**コンテンツ**: 完全記事
**ステータス**: ✅ 完全動作

#### ソース16: Energy Monitor
**実装**: `collectHeadlinesEnergyMonitor()`
**手法**: HTML Scraping + 完全コンテンツ取得
**収集URL**: `https://www.energymonitor.ai/news/`

**特徴**:
- article要素解析
- 記事ページから完全コンテンツ取得
- time要素からの日付抽出
- コンテンツセレクタ: `article .entry-content, .article-content`

**ステータス**: ✅ 完全動作

---

### 3.4 追加実装ソース

#### Carbon Knowledge Hub
**実装**: `collectHeadlinesCarbonKnowledgeHub()`
**手法**: HTML Scraping（CSS-in-JS対応）
**収集URL**: `https://www.carbonknowledgehub.com`

**特徴**:
- 広範なセレクタ: `a.css-oxwq25, a[class*='css-']`
- コンテンツパスのURLフィルタリング
- ナビゲーションテキストのスキップ（Read more、Learn more等）
- URLパスからタイプ抽出（/factsheet/、/story/等）

**コンテンツURLパターン**（2026年1月4日修正）:
```go
isContentURL := (strings.Contains(href, "/factsheet") ||
                strings.Contains(href, "/story") ||
                strings.Contains(href, "/stories") ||
                strings.Contains(href, "/audio") ||
                strings.Contains(href, "/media") ||
                strings.Contains(href, "/news")) &&
                strings.Count(href, "/") > 1 // カテゴリページではない
```

**特別処理**:
- 複数形パス対応（/factsheets、/stories）
- カテゴリページの除外（スラッシュ数チェック）
- タイプ自動判定（Factsheet、Story、Audio等）

**ステータス**: ✅ 実装済み（2026年1月4日修正で動作確認）

### 3.5 追加実装ソース（2026年2月6日）

#### RMI (Rocky Mountain Institute)
**実装**: `collectHeadlinesRMI()`
**手法**: WordPress REST API
**エンドポイント**: `https://rmi.org/wp-json/wp/v2/posts`
**ファイル**: `sources_wordpress.go`

**特徴**:
- エネルギー転換に特化したシンクタンク
- 標準WordPress REST API（共通関数使用）
- 全文コンテンツ取得

**ステータス**: ✅ 完全動作

#### IOP Science (Environmental Research Letters)
**実装**: `collectHeadlinesIOPScience()`
**手法**: RSS Feed（gofeed） + キーワードフィルタ
**フィードURL**: `https://iopscience.iop.org/journal/rss/1748-9326`
**フォーマット**: RDF/RSS 1.0
**ファイル**: `sources_academic.go`

**特徴**:
- 環境科学全般をカバーする学術誌
- `carbonKeywordsAcademic`によるキーワードフィルタリング
- gofeedがRDF/RSS 1.0を自動処理

**ステータス**: ✅ 完全動作

#### Nature Ecology & Evolution
**実装**: `collectHeadlinesNatureEcoEvo()`
**手法**: RSS Feed（gofeed） + キーワードフィルタ
**フィードURL**: `https://www.nature.com/natecolevol.rss`
**ファイル**: `sources_academic.go`

**特徴**:
- 生態学・進化学の学術誌
- Nature.comのbot保護により空スライスを返す場合あり（Nature Commsと同様）
- エラー時はgracefulに空スライスを返却

**ステータス**: ⚠️ bot保護により不安定（空スライス返却で対応）

#### ScienceDirect (Resources, Conservation & Recycling Advances)
**実装**: `collectHeadlinesScienceDirect()`
**手法**: RSS Feed（gofeed） + キーワードフィルタ
**フィードURL**: `https://rss.sciencedirect.com/publication/science/2950631X`
**ファイル**: `sources_academic.go`

**特徴**:
- Elsevier社の学術誌プラットフォーム
- 持続可能性・資源管理に特化したジャーナル
- `carbonKeywordsAcademic`によるキーワードフィルタリング

**ステータス**: ✅ 完全動作

---

## 4. データ処理パイプライン

### 4.1 フェーズ1: 収集

**入力**: コマンドラインフラグ
**出力**: `Headline[]` with Excerpts

**処理フロー**:
```
1. ユーザー入力（CLI flags）
   ↓
2. ソース選択（-sources flag）
   - デフォルト: 全18ソース
   - カスタム: カンマ区切りリスト
   ↓
3. ソースごとのスクレイピング
   - 各ソースから最大N件（-perSource, default 30）
   - ソース固有の実装を呼び出し
   ↓
4. URL重複排除
   - uniqueHeadlinesByURL()
   - URLをキーとしたマップで重複削除
   ↓
5. Headline[]配列の構築
   - Source: ソース名
   - Title: 記事タイトル
   - URL: 記事URL
   - PublishedAt: RFC3339形式の日付
   - Excerpt: 記事抜粋（ソースによる）
   - IsHeadline: true（見出しであることを示す）
```

**データ構造**:
```go
type Headline struct {
    Source        string        `json:"source"`
    Title         string        `json:"title"`
    URL           string        `json:"url"`
    PublishedAt   string        `json:"publishedAt"`
    Excerpt       string        `json:"excerpt,omitempty"`
    IsHeadline    bool          `json:"isHeadline"`
    RelatedFree   []RelatedFree `json:"relatedFree,omitempty"`
    SearchQueries []string      `json:"searchQueries,omitempty"`
}
```

### 4.2 フェーズ2: 検索（オプション）

**条件**: `-queriesPerHeadline > 0`
**入力**: `Headline[]`
**出力**: `FreeArticle[]` candidates per headline

**処理フロー**:
```
1. 各Headlineに対して:
   ↓
2. 検索クエリ生成（buildSearchQueries）
   - 7つの戦略を適用:
     a) 引用符付き完全一致
     b) excerptの最初の150文字
     c) 固有名詞抽出（組織、プロジェクト）
     d) 数値抽出（$50 million、30%、2025）
     e) 地理的site:演算子（site:go.kr等）
     f) PDF優先（filetype:pdf）
     g) 一次情報源優先（site:unfccc.int等）
   ↓
3. クエリごとにOpenAI Web Search実行
   - HTTP POST to api.openai.com/v1/responses
   - Model: gpt-4o-mini（デフォルト）
   - Tool: web_search
   - Max results per query: 10（-resultsPerQuery）
   ↓
4. URL抽出（3段階フォールバック）
   a) web_search_call.results（理想だが通常空）
      - 構造化データ（title、snippet）
   b) action.sources（URLのみ）
      - URLフィールドのみ
   c) テキスト正規表現抽出（主要手法）
      - 本文テキストから正規表現でURL抽出
   ↓
5. 擬似タイトル生成
   - URLからドメイン抽出
   - パス部分をクリーンアップ
   - 人間可読な形式に変換
   ↓
6. グローバルURL重複排除
   - globalSeenマップでURL追跡
   - 全見出し間で重複排除
   ↓
7. 候補のマージ
   - 見出しあたり最大N件（-searchPerHeadline, default 25）
   - URLをキーとしたマップでマージ
   ↓
8. FreeArticle[]配列の構築
   - Source: "OpenAI Search"（または抽出元）
   - Title: 実際のタイトルまたは擬似タイトル
   - URL: 記事URL
   - PublishedAt: 抽出可能な場合
   - Excerpt: 抽出可能な場合
```

**データ構造**:
```go
type FreeArticle struct {
    Source      string `json:"source"`
    Title       string `json:"title"`
    URL         string `json:"url"`
    PublishedAt string `json:"publishedAt,omitempty"`
    Excerpt     string `json:"excerpt,omitempty"`
}
```

**検索クエリ生成例**:
```go
// 戦略1: 引用符付き完全一致
query := fmt.Sprintf("\"%s\"", headline.Title)

// 戦略3: 固有名詞抽出
properNouns := extractProperNouns(headline.Title)
// 例: "European Commission", "Greenhouse Gas Protocol"

// 戦略4: 数値抽出
numbers := extractNumbers(headline.Title)
// 例: "$50 million", "30%", "2025"

// 戦略5: 地理的site:演算子
if containsGeo(headline.Title, "South Korea") {
    query += " site:go.kr"
}
```

### 4.3 フェーズ3: マッチング

**入力**: `Headline[]` + `FreeArticle[][]` candidates
**出力**: `RelatedFree[]` per headline

**処理フロー**:
```
1. IDFコーパス構築
   - 全見出しタイトル + 全候補タイトル
   - N個のドキュメント
   ↓
2. IDF計算
   - 各トークンに対して:
     IDF(term) = log(1 + N / (1 + df))
   - df = ドキュメント頻度
   ↓
3. 各Headline-Candidateペアに対して:
   ↓
4. トークン化
   - 正規表現: [A-Za-z0-9]+(?:-[A-Za-z0-9]+)*
   - ハイフン付き単語を保持
   - 小文字に変換
   ↓
5. トークン正規化
   - normToken マップ適用
   - 例: "euas" → "eua", "credits" → "credit"
   ↓
6. ストップワード除去
   - 25の一般的英単語
   - 例: the, a, an, to, of, in, on
   ↓
7. シグナル抽出
   a) Markets:
      - eua, uka, rggi, cca, accu, nzu, irec, ccer, corsia
   b) Topics:
      - vcm, cdr, dac, beccs, biochar, methane, forest
      - offset, credit, redd, nature-based, carbon removal
   c) Geos:
      - US, UK, EU, South Korea, New Zealand, Taiwan
      - China, Japan, India, Australia, etc.
   ↓
8. 類似度スコア計算
   a) IDF加重リコール類似度（56%重み）
      overlap = Σ(IDF(t) for t in intersection) / Σ(IDF(t) for t in headline)

   b) IDF加重Jaccard（28%重み）
      jaccard = Σ(IDF(t) for t in intersection) / Σ(IDF(t) for t in union)

   c) マーケットマッチ（6%重み）
      marketMatch = |h.Markets ∩ c.Markets| / |h.Markets|

   d) トピックマッチ（4%重み）
      topicMatch = |h.Topics ∩ c.Topics| / |h.Topics|

   e) 地理的マッチ（2%重み）
      geoMatch = |h.Geos ∩ c.Geos| / |h.Geos|

   f) 新しさスコア（4%重み）
      recency = exp(-age_days / 14.0)  // 14日半減期

   g) ソース品質ブースト（加算）
      - .gov/.gov.uk/.go.jp: +0.18
      - .pdf files: +0.18
      - europa.eu: +0.16
      - IR paths: +0.12
      - NGOs: +0.12
      - Press wires: +0.08
   ↓
9. 必須フィルタリング
   a) 見出しにマーケットシグナルがあり、strictMarket=trueの場合:
      → 候補も同じマーケットを持つ必要がある

   b) 見出しに特定地域がある場合:
      → 候補も同じ地域を持つ必要がある

   c) 曖昧な地理的マッチの除外:
      if marketMatch=0 AND topicMatch=0 AND geoMatch>0:
         → overlap >= 0.50 OR titleSim >= 0.84 が必要

   d) 最小共通単語数:
      → sharedTokens >= 2（titleSim >= 0.90でない限り）
   ↓
10. スコアフィルタ
    - score >= minScore（デフォルト0.32）
    ↓
11. スコアでソート（降順）
    ↓
12. 上位K件取得（-topK, default 3）
    ↓
13. RelatedFree[]配列の構築
    - Source、Title、URL、PublishedAt、Excerpt
    - Score: 計算されたスコア
    - Reason: スコアの内訳
```

**最終スコア計算式**:
```
score = 0.56 * overlap +
        0.28 * titleSim +
        0.06 * marketMatch +
        0.04 * topicMatch +
        0.02 * geoMatch +
        0.04 * recency +
        qBoost
```

**データ構造**:
```go
type RelatedFree struct {
    Source      string  `json:"source"`
    Title       string  `json:"title"`
    URL         string  `json:"url"`
    PublishedAt string  `json:"publishedAt,omitempty"`
    Excerpt     string  `json:"excerpt,omitempty"`
    Score       float64 `json:"score"`
    Reason      string  `json:"reason"`
}
```

### 4.4 フェーズ4: 出力

**入力**: `Headline[]` with `RelatedFree[]`
**出力**: JSON、Notion、Email

**処理フロー**:
```
1. JSON出力
   - stdout または ファイル（-out flag）
   - 整形されたJSON（2スペースインデント）
   ↓
2. Notionクリッピング（-notionClip指定時）
   a) データベース作成/再利用
      - 新規の場合: -notionPageID必須
      - 既存の場合: .env の NOTION_DATABASE_ID 使用

   b) 各Headlineをクリップ
      - ページ作成
      - プロパティ設定:
        * Title
        * URL
        * Source（色分けSelectオプション）
        * Type: "Headline"
        * Published Date
        * AI Summary（後でAIサマリー用の最初の2000文字）
      - 完全コンテンツをブロックに追加（2000文字/ブロック）

   c) 各RelatedFreeをクリップ
      - ページ作成
      - プロパティ設定（+ Score）
      - Type: "Related Free"
      - 完全コンテンツをブロックに追加

   d) Database IDの永続化
      - 新規作成時に.envに自動保存
      - appendToEnvFile()関数使用
   ↓
3. メール送信（-sendEmail指定時）
   a) Notionから最近の見出し取得
      - 過去N日間（-emailDaysBack, default 1）
      - Published Dateでフィルタ

   b) プレーンテキストサマリー生成
      - 見出しリスト
      - 各見出しの関連記事
      - URL付き

   c) Gmail SMTP経由で送信
      - EMAIL_FROM、EMAIL_PASSWORD、EMAIL_TO使用
      - RFC 5322準拠のメッセージ
      - 指数バックオフ付きリトライロジック
```

---

## 5. スコアリング・マッチングアルゴリズム

### 5.1 IDF（逆文書頻度）加重システム

**目的**: 一般的な単語にペナルティを与え、特徴的な用語をブーストする

**IDF計算式**:
```
IDF(term) = log(1 + N / (1 + df))

ここで:
- N = コーパス内の総ドキュメント数
- df = ドキュメント頻度（termを含むドキュメント数）
```

**コーパス**: 全見出しタイトル + 全候補タイトル

**実装例**:
```go
func buildIDF(docs [][]string) map[string]float64 {
    N := len(docs)
    df := make(map[string]int)

    // ドキュメント頻度をカウント
    for _, doc := range docs {
        seen := make(map[string]bool)
        for _, term := range doc {
            if !seen[term] {
                df[term]++
                seen[term] = true
            }
        }
    }

    // IDF計算
    idf := make(map[string]float64)
    for term, freq := range df {
        idf[term] = math.Log(1.0 + float64(N) / (1.0 + float64(freq)))
    }

    return idf
}
```

### 5.2 トークン正規化

**トークン化正規表現**:
```go
reTok = regexp.MustCompile(`[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*`)
```

**正規化マッピング**（40+エントリ）:
```go
normToken := map[string]string{
    // 市場
    "euas": "eua", "eua": "eua",
    "ukas": "uka", "uka": "uka",
    "rggi": "rggi",

    // トピック
    "credits": "credit", "credit": "credit",
    "offsets": "offset", "offset": "offset",
    "removal": "removal", "removals": "removal",

    // 一般用語
    "emissions": "emission", "emission": "emission",
    "countries": "country", "country": "country",
    // ... 更に多数
}
```

**ストップワード**（25語）:
```go
stopwords := map[string]bool{
    "the": true, "a": true, "an": true, "to": true,
    "of": true, "in": true, "on": true, "at": true,
    "for": true, "with": true, "by": true, "as": true,
    "is": true, "are": true, "was": true, "were": true,
    "be": true, "been": true, "being": true, "have": true,
    "has": true, "had": true, "do": true, "does": true,
    "did": true,
}
```

### 5.3 シグナル抽出

#### Markets（市場シグナル）
```go
markets := map[string]bool{
    "eua":        strings.Contains(lower, "eua") || reEUA.MatchString(raw),
    "uka":        strings.Contains(lower, "uka") || reUKA.MatchString(raw),
    "rggi":       strings.Contains(lower, "rggi"),
    "cca":        strings.Contains(lower, "cca") || tokSet["cca"],
    "accu":       strings.Contains(lower, "accu") || tokSet["accu"],
    "nzu":        strings.Contains(lower, "nzu") || tokSet["nzu"],
    "irec":       strings.Contains(lower, "irec") || strings.Contains(lower, "i-rec"),
    "ccer":       strings.Contains(lower, "ccer") || tokSet["ccer"],
    "corsia":     strings.Contains(lower, "corsia"),
    "article 6":  strings.Contains(lower, "article 6"),
    "jcm":        strings.Contains(lower, "jcm") || strings.Contains(lower, "二国間クレジット"),
}
```

#### Topics（トピックシグナル）
```go
topics := map[string]bool{
    "vcm":           strings.Contains(lower, "vcm") || strings.Contains(lower, "voluntary"),
    "cdr":           strings.Contains(lower, "cdr") || strings.Contains(lower, "carbon removal"),
    "dac":           strings.Contains(lower, "dac") || strings.Contains(lower, "direct air capture"),
    "beccs":         strings.Contains(lower, "beccs"),
    "biochar":       strings.Contains(lower, "biochar"),
    "methane":       strings.Contains(lower, "methane") || strings.Contains(lower, "ch4"),
    "forest":        strings.Contains(lower, "forest") || strings.Contains(lower, "deforestation"),
    "offset":        tokSet["offset"],
    "credit":        tokSet["credit"],
    "redd":          strings.Contains(lower, "redd"),
    "nature-based":  strings.Contains(lower, "nature-based") || strings.Contains(lower, "nbs"),
}
```

#### Geos（地理シグナル）
```go
geos := map[string]bool{
    "united_states": reUS.MatchString(raw),
    "united_kingdom": reUK.MatchString(raw),
    "eu": reEU.MatchString(raw),
    "europe": strings.Contains(lower, "europe"),
    "south_korea": strings.Contains(lower, "south korea"),
    "new_zealand": strings.Contains(lower, "new zealand"),
    // ... 更に多数
}
```

### 5.4 類似度メトリクス

#### 1. IDF加重リコール類似度（56%重み）
```go
func idfWeightedRecallOverlap(htok, ctok []string, idf map[string]float64) (float64, int) {
    hset := toSet(htok)
    cset := toSet(ctok)

    intersection := intersect(hset, cset)
    sharedTokens := len(intersection)

    sumIntersection := 0.0
    for term := range intersection {
        sumIntersection += idf[term]
    }

    sumHeadline := 0.0
    for term := range hset {
        sumHeadline += idf[term]
    }

    if sumHeadline == 0 {
        return 0, 0
    }

    return sumIntersection / sumHeadline, sharedTokens
}
```

**特徴**:
- 見出しの重要な単語が候補にどれだけカバーされているか
- 見出しの語彙を基準にリコールを測定

#### 2. IDF加重Jaccard類似度（28%重み）
```go
func idfWeightedJaccard(htok, ctok []string, idf map[string]float64) float64 {
    hset := toSet(htok)
    cset := toSet(ctok)

    intersection := intersect(hset, cset)
    union := union(hset, cset)

    sumIntersection := 0.0
    for term := range intersection {
        sumIntersection += idf[term]
    }

    sumUnion := 0.0
    for term := range union {
        sumUnion += idf[term]
    }

    if sumUnion == 0 {
        return 0
    }

    return sumIntersection / sumUnion
}
```

**特徴**:
- 両方のドキュメントの全体的な類似度
- 対称的な尺度

#### 3. シグナルマッチスコア（Markets 6%、Topics 4%、Geos 2%）
```go
func intersectScore(a, b map[string]bool) float64 {
    if len(a) == 0 {
        return 0
    }

    overlap := 0
    for key := range a {
        if b[key] {
            overlap++
        }
    }

    return float64(overlap) / float64(len(a))
}
```

**特徴**:
- 見出しのシグナルを基準にリコール測定
- 候補が見出しのシグナルをどれだけカバーするか

#### 4. 新しさスコア（4%重み）
```go
func recencyScoreRFC3339(publishedAt string, now time.Time, daysBack int) float64 {
    t, err := time.Parse(time.RFC3339, publishedAt)
    if err != nil {
        return 0
    }

    age := now.Sub(t)
    ageDays := age.Hours() / 24

    // 期間外の記事を除外
    if daysBack > 0 && ageDays > float64(daysBack) {
        return 0
    }

    // 指数減衰（14日半減期）
    return math.Exp(-ageDays / 14.0)
}
```

**特徴**:
- 14日半減期の指数減衰
- 期間ウィンドウ外の記事は0スコア

#### 5. ソース品質ブースト（加算）
```go
func sourceQualityBoost(url string) float64 {
    lower := strings.ToLower(url)

    // 政府サイト
    if strings.Contains(lower, ".gov") ||
       strings.Contains(lower, ".gov.uk") ||
       strings.Contains(lower, ".go.jp") {
        return 0.18
    }

    // PDFドキュメント
    if strings.HasSuffix(lower, ".pdf") {
        return 0.18
    }

    // EU公式サイト
    if strings.Contains(lower, "europa.eu") {
        return 0.16
    }

    // 投資家向け情報
    if strings.Contains(lower, "/investor") ||
       strings.Contains(lower, "/ir/") {
        return 0.12
    }

    // NGO
    if strings.Contains(lower, "carbonmarketwatch.org") ||
       strings.Contains(lower, "clientearth.org") {
        return 0.12
    }

    // プレスリリース配信
    if strings.Contains(lower, "prnewswire") ||
       strings.Contains(lower, "businesswire") {
        return 0.08
    }

    return 0.0
}
```

### 5.5 フィルタリングルール

#### 必須フィルタ

**1. 厳格マーケットマッチ**:
```go
if strictMarket && len(hs.Markets) > 0 && marketMatch == 0 {
    return scored{}, false
}
```
- 見出しにマーケットシグナルがある場合
- 候補も同じマーケットを持つ必要がある

**2. 特定地域マッチ**:
```go
if hasSpecificGeo(hs) && geoMatch == 0 {
    return scored{}, false
}

func hasSpecificGeo(s Signals) bool {
    for geo := range s.Geos {
        if geo != "united_states" && geo != "united_kingdom" &&
           geo != "eu" && geo != "europe" {
            return true
        }
    }
    return false
}
```
- 見出しに特定地域（US/UK/EU以外）がある場合
- 候補も同じ地域を持つ必要がある

**3. 曖昧な地理的マッチの除外**:
```go
if marketMatch == 0 && topicMatch == 0 && geoMatch > 0 &&
   overlap < 0.50 && titleSim < 0.84 {
    return scored{}, false
}
```
- 地域だけが一致して内容が異なる記事を除外

**4. 最小共通単語数**:
```go
if sharedTokens < 2 && titleSim < 0.90 {
    return scored{}, false
}
```
- 共通単語が2未満で類似度も低い記事を除外

**5. 最小スコア閾値**:
```go
if score < minScore {
    return scored{}, false
}
```
- デフォルト: 0.32未満の記事を除外

---

## 6. Notion統合

### 6.1 Notionデータベーススキーマ

**テーブル名**: "Carbon News Clippings"（自動作成）

**プロパティ定義**:

| プロパティ名 | タイプ | 用途 | 備考 |
|------------|-------|------|------|
| Title | Title | 記事見出し | 必須、ページタイトル |
| URL | URL | 記事リンク | クリック可能リンク |
| Source | Select | ソース名 | 22の色分けオプション |
| Type | Select | 記事タイプ | "Headline" または "Related Free" |
| Score | Number | マッチングスコア | Related Freeのみ、0-1の範囲 |
| Published Date | Date | 公開日 | RFC3339からパース |
| AI Summary | Rich Text | AIサマリー用 | 最初の2000文字を保存 |

**Sourceオプション（色分け）**:
```go
sourceOptions := []notionapi.Option{
    {Name: "CarbonCredits.jp", Color: notionapi.ColorYellow},
    {Name: "Carbon Herald", Color: notionapi.ColorGreen},
    {Name: "Climate Home News", Color: notionapi.ColorBlue},
    {Name: "CarbonCredits.com", Color: notionapi.ColorPurple},
    {Name: "Sandbag", Color: notionapi.ColorPink},
    {Name: "Ecosystem Marketplace", Color: notionapi.ColorBrown},
    {Name: "Carbon Brief", Color: notionapi.ColorGray},
    {Name: "ICAP", Color: notionapi.ColorDefault},
    {Name: "IETA", Color: notionapi.ColorRed},
    {Name: "Energy Monitor", Color: notionapi.ColorOrange},
    {Name: "JRI", Color: notionapi.ColorYellow},
    {Name: "Environment Ministry", Color: notionapi.ColorGreen},
    {Name: "JPX", Color: notionapi.ColorBlue},
    {Name: "METI", Color: notionapi.ColorPurple},
    {Name: "World Bank", Color: notionapi.ColorPink},
    {Name: "Carbon Market Watch", Color: notionapi.ColorBrown},
    {Name: "NewClimate", Color: notionapi.ColorGray},
    {Name: "Carbon Knowledge Hub", Color: notionapi.ColorDefault},
    {Name: "PwC Japan", Color: notionapi.ColorRed},
    {Name: "Mizuho R&T", Color: notionapi.ColorOrange},
}
```

**Typeオプション**:
```go
typeOptions := []notionapi.Option{
    {Name: "Headline", Color: notionapi.ColorBlue},
    {Name: "Related Free", Color: notionapi.ColorGreen},
}
```

### 6.2 データベース作成フロー

**シーケンス図**:
```
ユーザー
  ↓ (初回実行: -notionClip -notionPageID=xxx)
main.go
  ↓ (notionDatabaseID == "")
NewNotionClipper()
  ↓
CreateDatabase(ctx, pageID)
  ↓ (POST /v1/databases)
Notion API
  ↓ (返却: database ID)
appendToEnvFile(".env", "NOTION_DATABASE_ID", dbID)
  ↓ (NOTION_DATABASE_ID=xxx を .env に追加)
.env ファイル
  ↓
以降の実行で自動的に使用
```

**実装コード**:
```go
// main.goでの処理
if *notionDatabaseID == "" {
    if *notionPageID == "" {
        fatalf("ERROR: -notionPageID is required when creating a new Notion database")
    }

    fmt.Fprintln(os.Stderr, "Creating new Notion database...")
    dbID, err := clipper.CreateDatabase(ctx, *notionPageID)
    if err != nil {
        fatalf("ERROR creating Notion database: %v", err)
    }

    // Database IDを.envファイルに保存
    if err := appendToEnvFile(".env", "NOTION_DATABASE_ID", dbID); err != nil {
        fmt.Fprintf(os.Stderr, "WARN: Failed to save database ID to .env: %v\n", err)
        fmt.Fprintf(os.Stderr, "Please manually add to .env:\nNOTION_DATABASE_ID=%s\n", dbID)
    } else {
        fmt.Fprintf(os.Stderr, "✅ Database ID saved to .env file\n")
    }
} else {
    fmt.Fprintf(os.Stderr, "Using existing Notion database: %s\n", *notionDatabaseID)
}
```

**appendToEnvFile()関数**:
```go
func appendToEnvFile(path, key, value string) error {
    // .envファイルを読み込み
    content, err := os.ReadFile(path)
    if err != nil && !os.IsNotExist(err) {
        return err
    }

    lines := strings.Split(string(content), "\n")
    found := false

    // 既存のキーを更新
    for i, line := range lines {
        if strings.HasPrefix(line, key+"=") {
            lines[i] = fmt.Sprintf("%s=%s", key, value)
            found = true
            break
        }
    }

    // 新しいキーを追加
    if !found {
        lines = append(lines, fmt.Sprintf("%s=%s", key, value))
    }

    // ファイルに書き戻し
    return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}
```

### 6.3 記事クリッピングフロー

**処理フロー**:
```
1. ClipHeadlineWithRelated(headline)
   ↓
2. ClipHeadline(headline)
   a) ページプロパティ設定
      - Title: headline.Title
      - URL: headline.URL
      - Source: headline.Source (Select)
      - Type: "Headline" (Select)
      - Published Date: parseDate(headline.PublishedAt)
      - AI Summary: excerpt[:2000] (Rich Text)

   b) ページコンテンツ作成
      - Excerpt/Full Contentを段落ブロックに分割
      - 各ブロック最大2000文字
      - 空白行で段落分割

   c) Notion APIへPOST
      - POST /v1/pages
      - 親: database_id
      - プロパティ + コンテンツブロック
   ↓
3. For each RelatedFree:
   ClipRelatedFree(related)
   a) ページプロパティ設定
      - Title: related.Title
      - URL: related.URL
      - Source: related.Source (Select)
      - Type: "Related Free" (Select)
      - Score: related.Score (Number)
      - Published Date: parseDate(related.PublishedAt)
      - AI Summary: excerpt[:2000]

   b) ページコンテンツ作成（同上）

   c) Notion APIへPOST
```

**リッチテキスト分割実装**:
```go
func splitRichText(text string, maxLen int) []notionapi.RichText {
    if len(text) <= maxLen {
        return []notionapi.RichText{
            {Text: &notionapi.Text{Content: text}},
        }
    }

    var chunks []notionapi.RichText
    for len(text) > 0 {
        end := min(maxLen, len(text))
        chunks = append(chunks, notionapi.RichText{
            Text: &notionapi.Text{Content: text[:end]},
        })
        text = text[end:]
    }

    return chunks
}
```

**コンテンツブロック作成**:
```go
func createContentBlocks(content string) []notionapi.Block {
    if content == "" {
        return nil
    }

    // 段落で分割
    paragraphs := strings.Split(content, "\n\n")
    blocks := make([]notionapi.Block, 0, len(paragraphs))

    for _, para := range paragraphs {
        para = strings.TrimSpace(para)
        if para == "" {
            continue
        }

        // 2000文字制限でリッチテキスト分割
        richText := splitRichText(para, 2000)

        block := notionapi.ParagraphBlock{
            BasicBlock: notionapi.BasicBlock{
                Object: notionapi.ObjectTypeBlock,
                Type:   notionapi.BlockTypeParagraph,
            },
            Paragraph: notionapi.Paragraph{
                RichText: richText,
            },
        }

        blocks = append(blocks, block)
    }

    return blocks
}
```

### 6.4 日付パース

**複数フォーマット対応**:
```go
func parsePublishedDate(dateStr string) *time.Time {
    if dateStr == "" {
        return nil
    }

    formats := []string{
        time.RFC3339,           // "2025-12-26T10:30:00Z"
        "2006-01-02T15:04:05",  // "2025-12-26T10:30:00"
        "2006-01-02",           // "2025-12-26"
        "Jan 2, 2006",          // "Dec 26, 2025"
        "2 Jan 2006",           // "26 Dec 2025"
    }

    for _, format := range formats {
        if t, err := time.Parse(format, dateStr); err == nil {
            return &t
        }
    }

    return nil
}
```

### 6.5 メール送信用の最近見出し取得

**実装**:
```go
func (nc *NotionClipper) FetchRecentHeadlines(ctx context.Context, daysBack int) ([]NotionHeadline, error) {
    cutoffDate := time.Now().AddDate(0, 0, -daysBack)

    // Notionデータベースをクエリ
    query := &notionapi.DatabaseQueryRequest{
        Filter: notionapi.PropertyFilter{
            Property: "Published Date",
            Date: &notionapi.DateFilterCondition{
                OnOrAfter: (*notionapi.Date)(&cutoffDate),
            },
        },
        Sorts: []notionapi.SortObject{
            {
                Property:  "Published Date",
                Direction: notionapi.SortOrderDESC,
            },
        },
    }

    resp, err := nc.client.Database.Query(ctx, notionapi.DatabaseID(nc.databaseID), query)
    if err != nil {
        return nil, err
    }

    headlines := make([]NotionHeadline, 0, len(resp.Results))
    for _, page := range resp.Results {
        // プロパティからデータ抽出
        headline := extractHeadlineFromPage(page)
        headlines = append(headlines, headline)
    }

    return headlines, nil
}
```

---

## 7. 設定とコンフィグレーション

### 7.1 環境変数（.env）

**ファイルパス**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/.env`

**必須変数**:
```bash
# OpenAI API Key（Web検索に必須）
OPENAI_API_KEY=sk-your-openai-api-key-here

# Notion統合トークン（Notionクリッピングに必須）
NOTION_TOKEN=secret_your-notion-integration-token-here
```

**オプション変数**:
```bash
# Notion Page ID（新規データベース作成時に必須）
# URLから取得: https://www.notion.so/Page-Title-<THIS_PART>
NOTION_PAGE_ID=your-notion-page-id-here

# Notion Database ID（初回作成後に自動保存される）
NOTION_DATABASE_ID=your-notion-database-id-here

# メール設定（メール送信機能を使う場合）
EMAIL_FROM=your-email@gmail.com
EMAIL_PASSWORD=your-gmail-app-password
EMAIL_TO=recipient@example.com
```

**デバッグフラグ**:
```bash
# OpenAI検索結果サマリーを表示
DEBUG_OPENAI=1

# OpenAI APIの完全なレスポンスを表示
DEBUG_OPENAI_FULL=1

# スクレイピング詳細を表示
DEBUG_SCRAPING=1

# スクレイピング中のHTMLコンテンツを表示
DEBUG_HTML=1
```

### 7.2 コマンドラインフラグ

**全13フラグ**:

#### 入力制御（3フラグ）
```bash
-headlines <path>
  # JSONファイルから見出しを読み込み（スクレイピングをスキップ）
  # デフォルト: ""（スクレイピングを実行）

-sources <csv>
  # 収集するソースのカンマ区切りリスト
  # デフォルト: 全18ソース
  # 例: -sources=carbonpulse,sandbag,carbon-brief

-perSource <int>
  # ソースあたりの最大見出し数
  # デフォルト: 30
```

#### 検索制御（6フラグ）
```bash
-queriesPerHeadline <int>
  # 見出しあたりのクエリ数
  # デフォルト: 3
  # 0に設定すると検索を無効化

-resultsPerQuery <int>
  # クエリあたりの結果数
  # デフォルト: 10

-searchPerHeadline <int>
  # 見出しあたりの最大候補数
  # デフォルト: 25

-searchProvider <string>
  # 検索プロバイダ
  # デフォルト: "openai"
  # 現在openaiのみサポート

-openaiModel <string>
  # OpenAIモデル
  # デフォルト: "gpt-4o-mini"
  # 他のオプション: "gpt-4o", "gpt-4-turbo"

-openaiTool <string>
  # ツールタイプ
  # デフォルト: "web_search"
  # オプション: "web_search_preview"
```

#### マッチング制御（4フラグ）
```bash
-topK <int>
  # 見出しあたりの最大関連記事数
  # デフォルト: 3

-minScore <float>
  # 最小スコア閾値
  # デフォルト: 0.32
  # 範囲: 0.0 - 1.0

-daysBack <int>
  # 新しさウィンドウ（日数）
  # デフォルト: 60
  # 0で無効化

-strictMarket <bool>
  # 見出しにマーケットシグナルがある場合、マーケットマッチを要求
  # デフォルト: true
```

#### 出力制御（2フラグ）
```bash
-out <path>
  # 出力JSONをファイルに書き込み
  # デフォルト: ""（stdoutに出力）

-saveFree <path>
  # プールされた無料候補をファイルに保存
  # デフォルト: ""（保存しない）
```

#### Notion統合（3フラグ）
```bash
-notionClip <bool>
  # Notionにクリップ
  # デフォルト: false

-notionPageID <string>
  # 新規DB作成用の親ページID
  # 初回実行時に必須
  # 以降は.envのNOTION_DATABASE_IDを使用

-notionDatabaseID <string>
  # 既存データベースID
  # デフォルト: ""（.envから読み込み）
```

#### メール統合（2フラグ）
```bash
-sendEmail <bool>
  # メールサマリーを送信
  # デフォルト: false

-emailDaysBack <int>
  # Notionから取得する日数
  # デフォルト: 1
```

### 7.3 デフォルトソースリスト

**`-sources`フラグ未指定時の全18ソース**:
```go
defaultSources := []string{
    "carbonpulse",
    "qci",
    "carboncredits.jp",
    "carbonherald",
    "climatehomenews",
    "carboncredits.com",
    "sandbag",
    "ecosystem-marketplace",
    "carbon-brief",
    "icap",
    "ieta",
    "energy-monitor",
    "jri",
    "env-ministry",
    "jpx",
    "meti",
    "world-bank",
    "carbon-market-watch",
    "newclimate",
    "carbon-knowledge-hub",
    "pwc-japan",
    "mizuho-rt",
}
```

**ソース名の対応**:
| CLI名 | 実装関数 | ソース名 |
|-------|----------|---------|
| carboncredits.jp | collectHeadlinesCarbonCreditsJP | CarbonCredits.jp |
| jri | collectHeadlinesJRI | Japan Research Institute |
| env-ministry | collectHeadlinesEnvMinistry | Environment Ministry |
| pwc-japan | collectHeadlinesPwCJapan | PwC Japan |
| mizuho-rt | collectHeadlinesMizuhoRT | Mizuho R&T |
| sandbag | collectHeadlinesSandbag | Sandbag |
| carbon-brief | collectHeadlinesCarbonBrief | Carbon Brief |
| icap | collectHeadlinesICAP | ICAP |
| ieta | collectHeadlinesIETA | IETA |
| energy-monitor | collectHeadlinesEnergyMonitor | Energy Monitor |
| carbon-knowledge-hub | collectHeadlinesCarbonKnowledgeHub | Carbon Knowledge Hub |
| ... | ... | ... |

---

## 8. 使用方法と実行例

### 8.1 🟢 無料記事収集モード

**使用シーン**: Carbon関連の無料記事を幅広く収集し、要約してメール配信したい場合

#### 基本コマンド

```bash
# 無料ソースから記事を収集
./pipeline \
  -sources=sandbag,carbon-brief,climate-home,carbon-herald,carboncredits-com,carbon-knowledge-hub,ecosystem,icap,ieta,energy-monitor,carbon-market-watch,new-climate,carboncredits-jp,jri,env-ministry,jpx \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=free_articles.json
```

#### メール配信付き

```bash
# 無料記事を収集してメール送信
./pipeline \
  -sources=all-free \
  -perSource=15 \
  -queriesPerHeadline=0 \
  -sendEmail
```

**特徴**:
- ✅ 20の無料ソースから直接記事を収集
- ✅ 実行速度が速い（5-15秒程度）
- ✅ メール配信・Notion統合に対応

**ユースケース**:
- 日次の無料記事レビュー
- 業界トレンドの幅広い把握

---

### 8.2 詳細なワークフロー例

#### ワークフロー1: 全ソースから収集

**目的**: 全無料ソースから見出しとexcerptを収集

**コマンド**:
```bash
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=headlines.json
```

**出力**:
- `headlines.json`: Headline[]配列（excerptあり）

**ユースケース**:
- 日次のニュースレビュー
- 業界トレンドの把握

---

#### ワークフロー2: Notionクリッピング

**目的**: 記事をNotionにクリッピング

**コマンド**:
```bash
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -notionClip
```

**出力**:
- Notionデータベースに記事が追加される

---

#### ワークフロー3: メール送信

**コマンド**:
```bash
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -sendEmail
```

**出力**:
- `EMAIL_TO`にメールサマリー送信

---

#### ワークフロー4: デバッグモード

**スクレイピングのデバッグ**:
```bash
DEBUG_SCRAPING=1 ./pipeline \
  -sources=carbonherald \
  -perSource=2 \
  -queriesPerHeadline=0 \
  -out=debug.json
```

---

### 8.3 高度な使用例

#### 例1: 日本市場のみに焦点

```bash
./pipeline \
  -sources=jri,env-ministry,jpx,meti,pwc-japan,mizuho-rt,carboncredits.jp \
  -perSource=20 \
  -queriesPerHeadline=0 \
  -notionClip
```

#### 例2: EU市場のみに焦点

```bash
./pipeline \
  -sources=sandbag,icap,ieta,politico-eu \
  -perSource=15 \
  -queriesPerHeadline=0 \
  -notionClip
```

#### 例3: 高精度マッチング（厳格設定）

```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=10 \
  -queriesPerHeadline=5 \
  -resultsPerQuery=15 \
  -searchPerHeadline=40 \
  -topK=5 \
  -minScore=0.40 \
  -strictMarket=true \
  -out=high_precision.json
```

#### 例4: 低精度マッチング（緩い設定）

```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=10 \
  -queriesPerHeadline=2 \
  -topK=2 \
  -minScore=0.25 \
  -strictMarket=false \
  -out=high_recall.json
```

#### 例5: 新しい記事のみ（7日間）

```bash
./carbon-relay \
  -sources=carbonpulse,qci \
  -perSource=20 \
  -queriesPerHeadline=3 \
  -daysBack=7 \
  -notionClip
```

---

### 8.3 バッチ処理用スクリプト

**cron用スクリプト例**（`daily_clip.sh`）:
```bash
#!/bin/bash
set -e

cd /Users/kotafuse/Yasui/Prog/Test/carbon-relay

# 環境変数読み込み
source .env

# ログディレクトリ作成
mkdir -p logs

# タイムスタンプ
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 見出し収集 + Notionクリップ
./carbon-relay \
  -sources=carbonpulse,qci,sandbag,carbon-brief,icap,ieta \
  -perSource=30 \
  -queriesPerHeadline=3 \
  -topK=3 \
  -minScore=0.32 \
  -daysBack=60 \
  -notionClip \
  > logs/clip_${TIMESTAMP}.log 2>&1

# メール送信（前日の記事）
./carbon-relay \
  -sendEmail \
  -emailDaysBack=1 \
  >> logs/email_${TIMESTAMP}.log 2>&1

echo "Daily clip completed: ${TIMESTAMP}"
```

**crontab設定例**:
```cron
# 毎日朝9時に実行
0 9 * * * /Users/kotafuse/Yasui/Prog/Test/carbon-relay/daily_clip.sh
```

---

## 9. 最近の修正と改善

### 9.1 PwC Japan修正（2026年1月4日）

**問題**:
- PwC JapanサイトがAngular.jsで動的にコンテンツを読み込み
- 通常のHTML scrapingでは記事データを取得できない
- `angular.loadFacetedNavigation()`関数呼び出しにJSONデータが埋め込まれている

**解決策**:
1. **JavaScript関数からJSON抽出**:
   ```go
   // パターン: "{\x22numberHits\x22:...\x22filterTags\x22:...}"
   jsonPattern := regexp.MustCompile(`"(\{\\x22numberHits\\x22:\d+,\\x22elements\\x22:.*?\\x22filterTags\\x22:.*?\})"`)
   matches := jsonPattern.FindAllStringSubmatch(bodyStr, -1)
   ```

2. **16進エスケープのアンエスケープ**:
   ```go
   jsonStr = strings.ReplaceAll(jsonStr, `\x22`, `"`)
   jsonStr = strings.ReplaceAll(jsonStr, `\/`, `/`)
   jsonStr = strings.ReplaceAll(jsonStr, `\u002D`, `-`)
   ```

3. **3重エスケープの処理**:
   ```go
   // 2回のアンエスケープイテレーション
   for i := 0; i < 2; i++ {
       elementsStr = strings.ReplaceAll(elementsStr, `\\`, "\x00")
       elementsStr = strings.ReplaceAll(elementsStr, `\"`, `"`)
       elementsStr = strings.ReplaceAll(elementsStr, "\x00", `\`)
   }
   ```

4. **記事データの抽出**:
   ```go
   titlePattern := regexp.MustCompile(`"title":"([^"]+)"`)
   hrefPattern := regexp.MustCompile(`"href":"([^"]+)"`)
   datePattern := regexp.MustCompile(`"publishDate":"([^"]*)"`)
   ```

5. **Accept-Encoding削除**:
   - gzip圧縮レスポンスの問題を回避
   - 非圧縮レスポンスを受信

**結果**:
- ✅ 3件の記事を正常に収集
- ✅ NotionDBへの保存成功
- ✅ タイトル、URL、日付を正確に抽出

**収集例**:
```json
{
  "source": "PwC Japan",
  "title": "企業のサステナビリティ経営の成熟度／業界別分析からの考察 第3回：食品業界",
  "url": "https://www.pwc.com/jp/ja/knowledge/column/sustainability/sustainability-value-assessment03.html",
  "publishedAt": "2025-12-18T00:00:00Z",
  "isHeadline": true
}
```

---

### 9.2 Carbon Knowledge Hub修正（2026年1月4日）

**問題**:
- CSS-in-JSを使用しているため、CSSクラス名が動的
- URLフィルタが実際のサイト構造と一致していない
- 複数形パス（`/factsheets`、`/stories`）への対応が不足

**解決策**:
1. **広範なセレクタ**:
   ```go
   doc.Find("a.css-oxwq25, a[class*='css-']").Each(...)
   ```

2. **柔軟なURLフィルタリング**:
   ```go
   isContentURL := (strings.Contains(href, "/factsheet") ||
                   strings.Contains(href, "/story") ||
                   strings.Contains(href, "/stories") ||
                   strings.Contains(href, "/audio") ||
                   strings.Contains(href, "/media") ||
                   strings.Contains(href, "/news")) &&
                   strings.Count(href, "/") > 1 // カテゴリページではない
   ```

3. **タイプ自動判定**:
   ```go
   contentType := ""
   switch {
   case strings.Contains(href, "/factsheet/"):
       contentType = "Factsheet"
   case strings.Contains(href, "/story/"):
       contentType = "Story"
   case strings.Contains(href, "/audio/"):
       contentType = "Audio"
   case strings.Contains(href, "/news/"):
       contentType = "News"
   }
   ```

4. **重複排除**:
   ```go
   seen := make(map[string]bool)
   if seen[articleURL] {
       return
   }
   seen[articleURL] = true
   ```

**結果**:
- ✅ 5件の記事を正常に収集
- ✅ NotionDBへの保存成功
- ✅ Factsheet、Story、Audioなど多様なコンテンツタイプに対応

**収集例**:
```json
{
  "source": "Carbon Knowledge Hub",
  "title": "Offset use in South Africa's carbon tax",
  "url": "https://www.carbonknowledgehub.com/factsheets/south-africa-carbon-tax-offset-use",
  "publishedAt": "2026-01-04T17:28:25+09:00",
  "excerpt": "Type: Factsheet",
  "isHeadline": true
}
```

---

### 9.3 Mizuho R&T実装（2026年1月4日）

**新規実装**:
- HTML Scraping + キーワードフィルタリング
- 2025年の出版物ページから収集
- 日本語の日付形式に対応
- サステナビリティ関連キーワードで絞り込み

**実装コード概要**:
```go
func collectHeadlinesMizuhoRT(limit int, cfg headlineSourceConfig) ([]Headline, error) {
    newsURL := "https://www.mizuho-rt.co.jp/publication/2025/index.html"

    // キーワードリスト
    sustainabilityKeywords := []string{
        "サステナビリティ", "カーボン", "脱炭素", "GX", "ESG",
        "気候変動", "クリーンエネルギー", "環境", "再生可能エネルギー",
        // ... 20+キーワード
    }

    // リンクをフィルタリング
    doc.Find("a").Each(func(_ int, link *goquery.Selection) {
        href, _ := link.Attr("href")

        // /business/ または /publication/ パスのみ
        if !strings.Contains(href, "/business/") &&
           !strings.Contains(href, "/publication/") {
            return
        }

        title := link.Text()

        // キーワードチェック
        containsKeyword := false
        for _, kw := range sustainabilityKeywords {
            if strings.Contains(title, kw) {
                containsKeyword = true
                break
            }
        }

        // 日付抽出
        datePattern := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
        // ...
    })
}
```

**結果**:
- ✅ 実装完了
- ✅ 日本語キーワードフィルタリング動作
- ✅ 日付解析正常

---

## 10. トラブルシューティング

### 10.1 よくある問題と解決策

#### 問題1: OpenAI API Key Error
**エラーメッセージ**:
```
ERROR: set OPENAI_API_KEY (OpenAI API key) in your environment
```

**原因**:
- `.env`ファイルに`OPENAI_API_KEY`が設定されていない
- 環境変数が読み込まれていない

**解決策**:
```bash
# .envファイルを確認
cat .env | grep OPENAI_API_KEY

# または環境変数を直接設定
export OPENAI_API_KEY=sk-your-key-here

# または検索を無効化
./carbon-relay -queriesPerHeadline=0 ...
```

---

#### 問題2: Notion Token Error
**エラーメッセージ**:
```
ERROR: NOTION_TOKEN environment variable is required for Notion integration
```

**原因**:
- `.env`ファイルに`NOTION_TOKEN`が設定されていない

**解決策**:
```bash
# Notion統合トークンを取得
# https://www.notion.so/my-integrations

# .envに追加
echo "NOTION_TOKEN=secret_your-token-here" >> .env

# または-notionClipを外す
./carbon-relay ... # （-notionClipなし）
```

---

#### 問題3: Database ID Not Found
**エラーメッセージ**:
```
ERROR: -notionPageID is required when creating a new Notion database
```

**原因**:
- 初回実行時に`-notionPageID`が指定されていない
- `.env`に`NOTION_DATABASE_ID`がない

**解決策**:
```bash
# 初回実行時は必ず-notionPageIDを指定
./carbon-relay \
  -notionClip \
  -notionPageID=1234567890abcdef1234567890abcdef \
  ...

# Page IDの取得方法:
# NotionページのURLから取得
# https://www.notion.so/Page-Title-1234567890abcdef1234567890abcdef
#                               ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

---

#### 問題4: No Headlines Collected
**エラーメッセージ**:
```
ERROR collecting [Source] headlines: no [Source] headlines found
```

**原因**:
- サイトのHTML構造が変更された
- キーワードフィルタが厳しすぎる
- ネットワークエラー

**解決策**:
```bash
# デバッグモードで実行
DEBUG_SCRAPING=1 ./carbon-relay -sources=problem-source ...

# キーワードフィルタを確認（該当する場合）
# headlines.goのキーワードリストをチェック

# 別のソースで試す
./carbon-relay -sources=carbon-brief ...
```

---

#### 問題5: OpenAI Search Returns No Results
**症状**:
- 検索は実行されるが、候補が0件

**原因**:
- OpenAI API の`web_search_call.results`が常に空
- 3段階フォールバックでもURL抽出失敗

**解決策**:
```bash
# デバッグモードで確認
DEBUG_OPENAI_FULL=1 ./carbon-relay -sources=carbonpulse -perSource=1 -queriesPerHeadline=1

# 検索クエリを確認
# search_queries.goの戦略を調整

# 一時的に検索を無効化
./carbon-relay -queriesPerHeadline=0 ...
```

---

#### 問題6: Notion Clipping Fails
**エラーメッセージ**:
```
WARN: failed to clip headline 'xxx': API error
```

**原因**:
- Notion APIレート制限
- 不正なデータベースID
- トークンの権限不足

**解決策**:
```bash
# トークンの権限を確認
# Notion統合ページで以下を確認:
# - Content Capabilities: Insert content, Update content, Read content
# - Database: Create new databases（初回のみ）

# データベースIDを確認
cat .env | grep NOTION_DATABASE_ID

# レート制限の場合は待機後に再試行
# Notion API: 3 requests per second
```

---

#### 問題7: Email Sending Fails
**エラーメッセージ**:
```
ERROR sending email: authentication failed
```

**原因**:
- Gmailアプリパスワードが正しくない
- 2段階認証が有効化されていない

**解決策**:
```bash
# Gmailアプリパスワードを生成
# 1. Googleアカウント設定 > セキュリティ
# 2. 2段階認証を有効化
# 3. アプリパスワードを生成

# .envに設定
echo "EMAIL_PASSWORD=your-16-char-app-password" >> .env
```

---

### 10.2 デバッグテクニック

#### テクニック1: 段階的テスト

**ステップ1: 見出し収集のみ**:
```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=5 \
  -queriesPerHeadline=0 \
  -out=test_headlines.json
```

**ステップ2: 検索追加**:
```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=2 \
  -queriesPerHeadline=1 \
  -out=test_search.json
```

**ステップ3: マッチング確認**:
```bash
cat test_search.json | jq '.[] | select(.relatedFree | length > 0)'
```

**ステップ4: Notionクリップ**:
```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=1 \
  -queriesPerHeadline=1 \
  -notionClip
```

---

#### テクニック2: ソース別テスト

**各ソースを個別にテスト**:
```bash
for source in carbonpulse qci jri pwc-japan carbon-knowledge-hub; do
  echo "Testing: $source"
  ./carbon-relay \
    -sources=$source \
    -perSource=3 \
    -queriesPerHeadline=0 \
    -out=test_${source}.json 2>&1 | tee test_${source}.log
done
```

---

#### テクニック3: JSON出力の検証

**記事数カウント**:
```bash
cat headlines.json | jq 'length'
```

**ソース別カウント**:
```bash
cat headlines.json | jq 'group_by(.source) | map({source: .[0].source, count: length})'
```

**関連記事あり/なし**:
```bash
cat matched.json | jq 'map(select(.relatedFree | length > 0)) | length'
```

**平均スコア**:
```bash
cat matched.json | jq '[.[].relatedFree[]?.score] | add / length'
```

---

#### テクニック4: ログ分析

**OpenAI検索ログ**:
```bash
DEBUG_OPENAI=1 ./carbon-relay ... 2>&1 | grep "OpenAI"
```

**スクレイピングエラーログ**:
```bash
./carbon-relay ... 2>&1 | grep "ERROR"
```

**タイミング分析**:
```bash
time ./carbon-relay -sources=carbonpulse -perSource=10 -queriesPerHeadline=3
```

---

### 10.3 パフォーマンス最適化

#### 最適化1: 並列処理（将来の改善）

現在の実装は順次処理:
```
見出し1 → 検索1 → 検索2 → 検索3
見出し2 → 検索1 → 検索2 → 検索3
...
```

将来の並列化:
```
見出し1-10 → 並列検索 → マッチング
```

---

#### 最適化2: キャッシング（将来の改善）

**OpenAI検索結果のキャッシュ**:
- 同じクエリの再利用
- Redis/ファイルベースのキャッシュ

**HTML取得のキャッシュ**:
- 同じURLの再取得を避ける
- 有効期限付きキャッシュ

---

#### 最適化3: バッチサイズ調整

**少数の見出しで高精度**:
```bash
./carbon-relay \
  -perSource=5 \
  -queriesPerHeadline=5 \
  -resultsPerQuery=15 \
  -searchPerHeadline=50
```

**多数の見出しで高速**:
```bash
./carbon-relay \
  -perSource=50 \
  -queriesPerHeadline=2 \
  -resultsPerQuery=8 \
  -searchPerHeadline=20
```

---

## まとめ

このドキュメントは、Carbon Relayプロジェクトの完全な実装ガイドです。

### 2つの運用モード

本システムは**2つの異なる運用モード**をサポートします：

#### 🟢 無料記事収集モード
- **用途**: 幅広いCarbon関連無料記事の収集と要約配信
- **コマンド例**: `./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -sendEmail`
- **特徴**: コスト効率が高く、高速実行
- **詳細**: セクション1.2、セクション8.1

---

### 主要セクション参照ガイド

新しいClaude Codeセッションで参照する際は、以下のセクションを参照してください：

- **運用モードの理解**: セクション1.2
- **使用方法とコマンド例**: セクション8.1、8.2
- **アーキテクチャ理解**: セクション2
- **ソース追加**: セクション3
- **Notion統合**: セクション6
- **トラブルシューティング**: セクション10

---

### プロジェクト情報

**プロジェクトパス**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/`

**主要ファイル**:
- `cmd/pipeline/main.go` - エントリーポイント
- `internal/pipeline/headlines.go` - 共通ロジック
- `internal/pipeline/sources_*.go` - ソース実装
- `.env` - 環境変数設定

**ステータス**: 本番環境対応済み ✅
