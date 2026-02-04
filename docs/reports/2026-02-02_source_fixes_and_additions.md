# ソース修正・追加レポート（2026-02-02）

## 概要

VCM認証団体ソースの修正、および新規ソース（UK ETS、Puro.earth、UN News）の実装を完了した。

**作業期間**: 2026-02-01〜02-02
**実装者**: Claude Code

---

## 1. VCM認証団体ソースの修正

### 1.1 ACR（American Carbon Registry）

**問題**: 記事本文が取得できていなかった
**原因**: WordPressブロックベースのレイアウトで`.entry-content`セレクタが機能しない

**修正内容**:
- nav/header/footerを先に削除してからコンテンツ抽出
- 複数セレクタを優先順位付きで試行
- タイトルから「PUBLISHED」サフィックスを除去

```go
selectors := []string{
    ".entry-content",
    ".wp-block-group",
    ".post-content",
    "article",
    "main",
    ".wp-site-blocks",
    "body",
}
```

**ステータス**: ✅ 修正完了・動作確認済み

### 1.2 CAR（Climate Action Reserve）

**問題**: 外部リンク（Zoomなど）が収集されていた
**原因**: ニュースページに外部イベントリンクが含まれていた

**修正内容**:
```go
// CAR-specific: Only allow internal climateactionreserve.org links
if !strings.Contains(articleURL, "climateactionreserve.org") {
    return
}
```

**ステータス**: ✅ 修正完了・動作確認済み

### 1.3 Gold Standard

**問題**: Notionに日付が表示されなかった
**原因**: `+0000`形式のタイムゾーンオフセットをGoのRFC3339パーサーが解析できなかった

**修正内容** (`notion.go`):
```go
// Try ISO 8601 with timezone offset without colon (e.g., +0000)
t, err = time.Parse("2006-01-02T15:04:05-0700", dateStr)
```

**ステータス**: ✅ 修正完了・Notion動作確認済み

### 1.4 Verra

**問題**: リンクベースの記事選択が不安定
**修正内容**: 柔軟なリンク選択ロジックと記事ページからの本文取得を実装

**ステータス**: ✅ 修正完了・動作確認済み

---

## 2. 新規ソース実装

### 2.1 UK ETS（英国排出量取引制度）

**識別子**: `uk-ets`
**ファイル**: `sources_regional_ets.go`
**方式**: HTMLスクレイピング（gov.uk検索ページ）

**背景**:
- 当初のAtomフィード（`gov.uk/government/publications.atom?topics[]=uk-emissions-trading-scheme`）が空だった
- gov.uk検索ページを代替として使用

**実装詳細**:
```go
func collectHeadlinesUKETSHTML(limit int, cfg headlineSourceConfig) ([]Headline, error) {
    searchURL := "https://www.gov.uk/search/all?keywords=%22UK+Emissions+Trading+Scheme%22&order=updated-newest"
    // gem-c-document-list セレクタを使用
}
```

**テスト結果**:
```json
{
  "source": "UK ETS",
  "title": "The Merchant Shipping (Monitoring, Reporting and Verification of Carbon Dioxide Emissions) (Revocation) Regulations 2026",
  "url": "https://www.gov.uk/government/publications/...",
  "publishedAt": "2026-01-22T00:00:00Z",
  "excerpt": "This instrument seeks to revoke the existing monitoring..."
}
```

**ステータス**: ✅ 実装完了・Notion動作確認済み

### 2.2 Puro.earth

**識別子**: `puro-earth`
**ファイル**: `sources_html.go`
**方式**: HTMLスクレイピング（ブログページ）

**背景**:
- 当初のニュースルーム（`/news-room/`）に構造化されたニュースがなかった
- ブログページ（`/our-blog/`）を使用

**実装詳細**:
- ブログページからリンクを収集
- 各記事ページをフェッチして日付と抜粋を取得
- 日付形式: `DD/MM/YYYY`（ヨーロッパ形式）

```go
datePatterns := []struct {
    regex  string
    format string
}{
    {`\d{2}/\d{2}/\d{4}`, "02/01/2006"},
    {`(Jan|Feb|Mar|...)...\d{4}`, "Jan 2, 2006"},
}
```

**テスト結果**:
```json
{
  "source": "Puro.earth",
  "title": "Scaling Biochar With Integrity And Innovation",
  "url": "https://puro.earth/our-blog/362-...",
  "publishedAt": "2026-01-29T00:00:00Z",
  "excerpt": "By Elias Azzi, Head of Eligibility"
}
```

**ステータス**: ✅ 実装完了・Notion動作確認済み

### 2.3 UN News（国連ニュース気候変動）

**識別子**: `un-news`
**ファイル**: `sources_rss.go`
**方式**: RSSフィード

**背景**:
- UNFCCCサイトはIncapsula保護でスクレイピング不可
- 代替として国連公式ニュースの気候変動RSSフィードを採用

**実装詳細**:
```go
func collectHeadlinesUNNews(limit int, cfg headlineSourceConfig) ([]Headline, error) {
    feedURL := "https://news.un.org/feed/subscribe/en/news/topic/climate-change/feed/rss.xml"
    // 標準的なgofeedパターン
}
```

**テスト結果**:
```json
{
  "source": "UN News",
  "title": "From family farm to climate tech: How one Kenyan woman is helping farmers outsmart drought",
  "url": "https://news.un.org/feed/view/en/story/2026/01/1166823",
  "publishedAt": "2026-01-25T12:00:00Z",
  "excerpt": ""Giving up is not an option - so many people depend on you,"..."
}
```

**ステータス**: ✅ 実装完了・Notion動作確認済み

---

## 3. 現在のソースステータス一覧

### 3.1 動作中のソース（34ソース）

| カテゴリ | ソース | 識別子 | 方式 |
|---------|--------|--------|------|
| 有料 | Carbon Pulse | `carbonpulse` | HTML |
| 有料 | QCI | `qci` | HTML |
| WordPress | CarbonCredits.jp | `carboncredits.jp` | REST API |
| WordPress | Carbon Herald | `carbonherald` | REST API |
| WordPress | Climate Home News | `climatehomenews` | REST API |
| WordPress | CarbonCredits.com | `carboncredits.com` | REST API |
| WordPress | Sandbag | `sandbag` | REST API |
| WordPress | Ecosystem Marketplace | `ecosystem-marketplace` | REST API |
| WordPress | Carbon Brief | `carbon-brief` | REST API |
| HTML | ICAP | `icap` | HTML |
| HTML | IETA | `ieta` | HTML |
| HTML | Energy Monitor | `energy-monitor` | HTML |
| HTML | World Bank | `world-bank` | HTML |
| HTML | NewClimate | `newclimate` | HTML |
| HTML | Carbon Knowledge Hub | `carbon-knowledge-hub` | HTML |
| VCM認証 | Verra | `verra` | HTML |
| VCM認証 | Gold Standard | `gold-standard` | HTML |
| VCM認証 | ACR | `acr` | HTML |
| VCM認証 | CAR | `car` | HTML |
| 国際機関 | IISD ENB | `iisd` | HTML |
| 国際機関 | Climate Focus | `climate-focus` | HTML |
| 国際機関 | **UN News** | `un-news` | RSS |
| 地域ETS | EU ETS | `eu-ets` | HTML |
| 地域ETS | **UK ETS** | `uk-ets` | HTML |
| 地域ETS | California CARB | `carb` | HTML |
| 地域ETS | RGGI | `rggi` | HTML |
| 地域ETS | Australia CER | `australia-cer` | HTML |
| CDR | **Puro.earth** | `puro-earth` | HTML |
| CDR | Isometric | `isometric` | HTML |
| 日本語 | JRI | `jri` | RSS |
| 日本語 | 環境省 | `env-ministry` | HTML |
| 日本語 | METI | `meti` | RSS |
| 日本語 | PwC Japan | `pwc-japan` | HTML |
| 日本語 | Mizuho R&T | `mizuho-rt` | HTML |
| RSS | Politico EU | `politico-eu` | RSS |
| RSS | Euractiv | `euractiv` | RSS |
| 学術 | arXiv | `arxiv` | XML API |
| 学術 | Nature Communications | `nature-comms` | RSS |

### 3.2 一時無効化中のソース（3ソース）

| ソース | 識別子 | 問題 | 代替 |
|--------|--------|------|------|
| UNFCCC | `unfccc` | Incapsula保護 | UN News |
| OIES | `oies` | JavaScript描画必須 | なし |
| Carbon Market Watch | `carbon-market-watch` | 403 Forbidden | なし |

---

## 4. 変更ファイル一覧

| ファイル | 変更内容 |
|----------|----------|
| `cmd/pipeline/sources_html.go` | ACR、CAR、Verra、Puro.earth修正 |
| `cmd/pipeline/sources_regional_ets.go` | UK ETS HTML実装追加 |
| `cmd/pipeline/sources_rss.go` | UN News追加 |
| `cmd/pipeline/headlines.go` | sourceCollectors更新、コメント更新 |
| `cmd/pipeline/notion.go` | 日付パース形式追加（+0000対応） |

---

## 5. テスト方法

### 個別ソーステスト
```bash
# UK ETS
./pipeline -sources=uk-ets -perSource=5 -queriesPerHeadline=0 -out=/tmp/test.json

# Puro.earth
./pipeline -sources=puro-earth -perSource=5 -queriesPerHeadline=0 -out=/tmp/test.json

# UN News
./pipeline -sources=un-news -perSource=5 -queriesPerHeadline=0 -out=/tmp/test.json
```

### Notionクリップテスト
```bash
./pipeline -sources={source} -perSource=5 -queriesPerHeadline=0 -notionClip
```

### デバッグモード
```bash
DEBUG_SCRAPING=1 ./pipeline -sources={source} -perSource=1 -queriesPerHeadline=0
```

---

## 6. 今後の課題

### 6.1 未解決のソース

| ソース | 問題 | 検討中の対策 |
|--------|------|------------|
| Euractiv | Cloudflare WAF | ヘッドレスブラウザ、またはニュースアグリゲータ経由 |
| OIES | SPA/JavaScript | Puppeteer/Playwright導入 |
| Carbon Market Watch | IP制限? | プロキシ、または別のNGOソース |

### 6.2 改善候補

- [ ] Puro.earthの日付取得を最適化（現在は各記事をフェッチ）
- [ ] UK ETSの検索結果ページング対応
- [ ] エラーハンドリングの強化

---

## 7. まとめ

今回の作業で以下を達成:

1. **VCM認証団体ソース4つの修正完了**
   - 記事本文、日付、URLの取得が正常化
   - Notion統合でも正しく表示

2. **新規ソース3つの実装完了**
   - UK ETS: gov.uk検索ページから取得
   - Puro.earth: ブログページから取得
   - UN News: RSSフィードから取得（UNFCCC代替）

3. **有効ソース数: 32 → 34（+2）**
   - UN Newsを追加（UNFCCCの代替として）
   - Euractivが動作確認済み（RSSフィード + キーワードフィルタリング）

**2026-02-04 追記**: Euractivの動作を確認。メインRSSフィードからキーワードフィルタリングで気候関連記事を取得。Notion統合も正常動作。

---

## 8. 追加修正（ソース品質改善）

### 8.1 Isometric - タイトル抽出修正

**問題**: タイトルに日付・著者情報が混入
- 例: `"ScienceJan 21, 2026A new module for...Stacy Kauk, P.Eng.Chief Science Officer"`

**原因**: `<a>`要素全体のテキストを取得していた

**修正内容**:
- セレクタを`a[href*='/writing-articles/']`に限定
- 最初の`<p>`要素（クラスなし）からタイトルを抽出
- 著者セクション（img要素を含む親）を除外

**結果**: ✅ タイトルのみ正しく抽出

### 8.2 arXiv - カテゴリ制限追加

**問題**: 物理学論文が混入（"positron emission"など）

**原因**: "emission"が物理学文脈でもマッチ

**修正内容**:
- API検索にカテゴリ制限を追加: `econ.GN`, `q-fin.*`, `stat.AP`
- キーワードフィルタを気候専用フレーズに変更

```go
categories := "cat:econ.GN+OR+cat:q-fin.GN+OR+cat:q-fin.PM+OR+cat:stat.AP"
keywords := "carbon+OR+climate+OR+emission+OR+environmental+policy"
searchQuery := fmt.Sprintf("(%s)+AND+(%s)", categories, keywords)
```

**結果**: ✅ 気候経済関連論文のみ返却

### 8.3 IISD ENB - 日付抽出追加

**問題**: 日付が取得されなかった

**原因**: 日付がテキスト内に埋め込まれていた

**修正内容**:
- 正規表現パターンでテキストから日付を抽出
- 形式: `"2 February 2026"`

```go
datePatterns := []struct {
    regex  string
    format string
}{
    {`(\d{1,2}\s+(?:January|February|...|December)\s+\d{4})`, "2 January 2006"},
}
```

**結果**: ✅ イベント日付を正しく取得

### 8.4 Climate Focus - JSON-LD日付取得

**問題**: リストページに日付が表示されていなかった

**原因**: 日付は個別記事ページのJSON-LDにのみ存在

**修正内容**:
- 各記事ページをフェッチ
- JSON-LDスキーマから`datePublished`を抽出
- `og:description`からexcerptも取得

```go
articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
    re := regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
    if match := re.FindStringSubmatch(text); len(match) > 1 {
        dateStr = match[1]
    }
})
```

**結果**: ✅ 正確な公開日を取得

---

**作成日**: 2026-02-02
**作成者**: Claude Code (Opus 4.5)
