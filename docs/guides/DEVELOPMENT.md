# carbon-relay 開発者ドキュメント

## アーキテクチャ概要

```
[36の無料ソース]
       ↓ スクレイピング（WordPress API / HTML / RSS）
[Headline Collection]
       ↓
[キーワードフィルタリング]（日本ソースのみ）
       ↓
[JSON出力 / Notion / メール送信]
```

---

## 各モジュールの詳細

### 1. internal/pipeline/headlines.go - ヘッドライン収集

#### ソースコレクター登録（36ソース）
```go
var sourceCollectors = map[string]HeadlineCollector{
    // WordPress REST APIソース（7ソース）
    "carboncredits.jp": collectHeadlinesCarbonCreditsJP,
    "carbonherald":     collectHeadlinesCarbonHerald,
    "climatehomenews":  collectHeadlinesClimateHomeNews,
    "carboncredits.com": collectHeadlinesCarbonCreditscom,
    "sandbag":          collectHeadlinesSandbag,
    "ecosystem-marketplace": collectHeadlinesEcosystemMarketplace,
    "carbon-brief":     collectHeadlinesCarbonBrief,

    // HTMLスクレイピングソース（6ソース）
    "icap":             collectHeadlinesICAP,
    "ieta":             collectHeadlinesIETA,
    "energy-monitor":   collectHeadlinesEnergyMonitor,
    "world-bank":       collectHeadlinesWorldBank,
    "newclimate":       collectHeadlinesNewClimate,
    "carbon-knowledge-hub": collectHeadlinesCarbonKnowledgeHub,

    // 日本語ソース（6ソース）
    "jri":          collectHeadlinesJRI,
    "env-ministry": collectHeadlinesEnvMinistry,
    "meti":         collectHeadlinesMETI,
    "pwc-japan":    collectHeadlinesPwCJapan,
    "mizuho-rt":    collectHeadlinesMizuhoRT,
    "jpx":          collectHeadlinesJPX,

    // RSSフィード（2ソース）
    "politico-eu": collectHeadlinesPoliticoEU,
    "euractiv":    collectHeadlinesEuractiv,

    // 学術・研究機関（2ソース）
    "arxiv": collectHeadlinesArXiv,
    "oies":  collectHeadlinesOIES,

    // VCM認証団体（4ソース）
    "verra":         collectHeadlinesVerra,
    "gold-standard": collectHeadlinesGoldStandard,
    "acr":           collectHeadlinesACR,
    "car":           collectHeadlinesCAR,

    // 国際機関（2ソース）
    "iisd":          collectHeadlinesIISD,
    "climate-focus": collectHeadlinesClimateFocus,

    // 地域ETS（5ソース）
    "eu-ets":        collectHeadlinesEUETS,
    "uk-ets":        collectHeadlinesUKETS,
    "carb":          collectHeadlinesCARB,
    "rggi":          collectHeadlinesRGGI,
    "australia-cer": collectHeadlinesAustraliaCER,

    // CDR関連（2ソース）
    "puro-earth": collectHeadlinesPuroEarth,
    "isometric":  collectHeadlinesIsometric,
}
```

#### WordPress REST API パターン
```go
// Carbon Herald、Sandbag 等
url := "https://carbonherald.com/wp-json/wp/v2/posts?per_page=100"
resp, _ := client.Get(url)
var posts []WordPressPost
json.NewDecoder(resp.Body).Decode(&posts)
```

#### HTML スクレイピングパターン
```go
// JRI、環境省、METI 等
doc, _ := goquery.NewDocumentFromReader(resp.Body)
doc.Find("article, .post, .news-item").Each(func(i int, s *goquery.Selection) {
    title := s.Find("h2, h3, .title").Text()
    link, _ := s.Find("a").Attr("href")
    // ...
})
```

#### RSS フィードパターン
```go
// Carbon Brief 等
fp := gofeed.NewParser()
feed, _ := fp.ParseURL(rssURL)
for _, item := range feed.Items {
    // item.Title, item.Link, item.Published
}
```

---

### 2. キーワードフィルタリング

日本語ソース（JRI、環境省、METI、Mizuho R&T）では、カーボン関連キーワードでフィルタリングを行います。

```go
var carbonKeywords = []string{
    "カーボン", "炭素", "CO2", "排出", "脱炭素",
    "グリーン", "温室効果", "GHG", "クレジット",
    "ネットゼロ", "気候変動", "climate",
}

func matchesCarbonKeywords(title, excerpt string) bool {
    combined := strings.ToLower(title + " " + excerpt)
    for _, kw := range carbonKeywords {
        if strings.Contains(combined, strings.ToLower(kw)) {
            return true
        }
    }
    return false
}
```

---

### 3. internal/pipeline/notion.go - Notion統合

#### データベース自動作成
```go
func createNotionDatabase(client *notionapi.Client, pageID string) (string, error) {
    db := &notionapi.DatabaseCreateRequest{
        Parent: notionapi.Parent{PageID: pageID},
        Title:  []notionapi.RichText{{Text: &notionapi.Text{Content: "Carbon News Clippings"}}},
        Properties: map[string]notionapi.PropertyConfig{
            "Title":   notionapi.TitlePropertyConfig{},
            "URL":     notionapi.URLPropertyConfig{},
            "Source":  notionapi.SelectPropertyConfig{},
            "Excerpt": notionapi.RichTextPropertyConfig{},
        },
    }
    // ...
}
```

#### リッチテキスト分割（2000文字制限対応）
```go
func splitRichText(text string, limit int) []notionapi.RichText {
    var result []notionapi.RichText
    for len(text) > 0 {
        chunk := text
        if len(chunk) > limit {
            chunk = text[:limit]
        }
        result = append(result, notionapi.RichText{
            Text: &notionapi.Text{Content: chunk},
        })
        text = text[len(chunk):]
    }
    return result
}
```

---

### 4. cmd/pipeline/main.go - メインエントリーポイント

#### コマンドラインフラグ
```go
sources        = flag.String("sources", "all-free", "Source names, comma-separated or 'all-free'")
perSource      = flag.Int("perSource", 30, "Max headlines per source")
queriesPerHL   = flag.Int("queriesPerHeadline", 0, "Search queries per headline (0 to skip)")
hoursBack      = flag.Int("hoursBack", 0, "Only include headlines from last N hours (0 = no limit)")
outFile        = flag.String("out", "", "Output file path")
notionClip     = flag.Bool("notionClip", false, "Enable Notion clipping")
sendEmail      = flag.Bool("sendEmail", false, "Send email with results")
```

---

## トークン化（tokenize）

### 正規表現パターン
```go
reTok = regexp.MustCompile(`[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*`)
// マッチ例：
// - "carbon-pulse" → 1トークン
// - "climate-change" → 1トークン
// - "EUA" → 1トークン
```

### ストップワード
```go
stop = map[string]bool{
    "the": true, "a": true, "an": true,
    "to": true, "of": true, "in": true,
    "new": true, "year": true,
    // ...
}
```

---

## テスト戦略

### 単体テスト

```bash
# 特定ソースのテスト
./pipeline -sources=carbonherald -perSource=3 -queriesPerHeadline=0

# 日本ソースのテスト
./pipeline -sources=jri,env-ministry -perSource=3 -queriesPerHeadline=0

# 全ソースのクイックテスト
./pipeline -sources=all-free -perSource=1 -queriesPerHeadline=0
```

### デバッグモード

```bash
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=2

# 出力例：
[DEBUG] Fetching https://carbonherald.com/wp-json/wp/v2/posts
[DEBUG] Found 10 posts
[DEBUG] Processing: "EU carbon price hits record high"
```

---

## デバッグガイド

### DEBUG_SCRAPING=1

```bash
DEBUG_SCRAPING=1 ./pipeline -sources=jri -perSource=5

# 出力例：
[DEBUG] Fetching JRI page: https://www.jri.co.jp/...
[DEBUG] Found 20 articles
[DEBUG] After keyword filter: 8 articles
[DEBUG]   - カーボンニュートラル達成に向けた...
```

### よくあるデバッグシナリオ

#### ヘッドラインが収集されない

```bash
# 1. デバッグ出力で状況確認
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=1

# 2. サイトに直接アクセスできるか確認
curl -I https://carbonherald.com/wp-json/wp/v2/posts

# 3. HTMLパース結果を確認（HTMLスクレイピングの場合）
# → internal/pipeline/headlines.go のセレクタを確認
```

#### 日本ソースの記事が少ない

```bash
# キーワードフィルタの影響を確認
# → carbonKeywords に必要なキーワードがあるか確認
# → matchesCarbonKeywords の判定ロジックを確認
```

---

## よくある質問（FAQ）

### Q1: 新しいソースを追加したい

```go
// 1. internal/pipeline/headlines.go に収集関数を追加
func collectHeadlinesNewSource(ctx context.Context, cfg *HeadlineSourceConfig) ([]Headline, error) {
    // WordPress API / HTML / RSS のいずれかのパターンで実装
    // 既存の関数を参考に
}

// 2. sourceCollectors に登録
var sourceCollectors = map[string]HeadlineCollector{
    // ...
    "new-source": collectHeadlinesNewSource,
}
```

### Q2: キーワードフィルタを調整したい

```go
// internal/pipeline/headlines.go の carbonKeywords を編集
var carbonKeywords = []string{
    "カーボン", "炭素", "CO2", "排出", "脱炭素",
    "グリーン", "温室効果", "GHG", "クレジット",
    "ネットゼロ", "気候変動", "climate",
    "新しいキーワード",  // ← 追加
}
```

### Q3: 特定ドメインをNotionソースリストに追加したい

```go
// internal/pipeline/notion.go の sourceOptions を編集
sourceOptions := []struct {
    Name  string
    Color notionapi.Color
}{
    {Name: "Carbon Herald", Color: notionapi.ColorBlue},
    {Name: "JRI", Color: notionapi.ColorGreen},
    {Name: "New Source", Color: notionapi.ColorPurple},  // ← 追加
    // ...
}
```

---

## コントリビューションガイド

### コーディング規約

1. **エラーハンドリング**
   ```go
   // ✅ Good
   if err != nil {
       return nil, fmt.Errorf("failed to parse URL: %w", err)
   }

   // ❌ Bad
   if err != nil {
       panic(err)  // 本番環境でpanicは禁止
   }
   ```

2. **コメント**
   ```go
   // ✅ Good：なぜそうするのかを説明
   // 日本語ソースはカーボン関連キーワードでフィルタリング
   if matchesCarbonKeywords(title, excerpt) {
       // ...
   }

   // ❌ Bad：コードを繰り返すだけ
   // キーワードをチェック
   if matchesCarbonKeywords(title, excerpt) {
       // ...
   }
   ```

3. **命名**
   - 変数：`camelCase`
   - 関数：`camelCase`
   - 定数：`PascalCase`（Goの慣習）
   - エクスポート：`PascalCase`

---

## リリースチェックリスト

- [ ] すべてのデバッグ出力を削除（または環境変数で制御）
- [ ] go.mod / go.sum が正しい
- [ ] README.md が最新
- [ ] DEVELOPMENT.md が最新
- [ ] エラーメッセージがユーザーフレンドリー
- [ ] APIキーがハードコードされていない
- [ ] 全ソースのテスト実行（`./pipeline -sources=all-free -perSource=1`）

---

## 参考リンク

- [WordPress REST API Documentation](https://developer.wordpress.org/rest-api/)
- [goquery Documentation](https://github.com/PuerkitoBio/goquery)
- [gofeed Documentation](https://github.com/mmcdole/gofeed)
- [Notion API Documentation](https://developers.notion.com/)

---

## ソース一覧

### 日本ソース
| ソース | 実装方式 | URL |
|-------|---------|-----|
| JRI | HTML | https://www.jri.co.jp/ |
| 環境省 | HTML | https://www.env.go.jp/ |
| METI | HTML | https://www.meti.go.jp/ |
| PwC Japan | JSON | https://www.pwc.com/jp/ |
| Mizuho R&T | HTML | https://www.mizuho-rt.co.jp/ |
| JPX | HTML | https://www.jpx.co.jp/ |
| カーボンクレジット.jp | HTML | https://carboncredits.jp/ |

### 国際ソース
| ソース | 実装方式 | URL |
|-------|---------|-----|
| Carbon Herald | WordPress API | https://carbonherald.com/ |
| Carbon Brief | RSS | https://www.carbonbrief.org/ |
| Sandbag | WordPress API | https://sandbag.be/ |
| ICAP | HTML | https://icapcarbonaction.com/ |
| IETA | HTML | https://www.ieta.org/ |
| Politico EU | HTML | https://www.politico.eu/ |
| IISD | HTML | https://sdg.iisd.org/ |
| UNFCCC | HTML | https://unfccc.int/ |
| GEF | HTML | https://www.thegef.org/ |
