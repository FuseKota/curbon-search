# carbon-relay 開発者ドキュメント

## アーキテクチャ概要

```
[Carbon Pulse / QCI]
       ↓ スクレイピング
[Headline Collection]
       ↓
[Search Query Generation] ← 見出し + 戦略的キーワード
       ↓
[OpenAI Web Search] ← OpenAI Responses API
       ↓ URL抽出
[URL → Title Generation] ← 疑似タイトル生成
       ↓
[Candidate Pool] ← すべての検索結果
       ↓
[IDF Construction] ← コーパス全体から逆文書頻度計算
       ↓
[Similarity Matching] ← TF-IDF + Signals + Quality
       ↓
[Top-K Selection] → relatedFree
```

---

## 各モジュールの詳細

### 1. headlines.go - ヘッドライン収集

#### Carbon Pulse スクレイピング
```go
// 対象ページ
- https://carbon-pulse.com/daily-timeline/
- https://carbon-pulse.com/category/newsletters/

// 収集ロジック
1. すべての<a>タグを走査
2. href="/数字/" パターンにマッチするもののみ採用（例：/470597/）
3. リンクテキストが空 or "Read more" 等 → 除外
4. 最小文字数チェック（len < 10 → 除外）
```

**重要な発見（2025-12-29）：**
- "Read more" のような無意味なリンクテキストが大量に取得されていた
- → 除外フィルタを追加（headlines.go:64-68）

#### QCI スクレイピング
```go
// 対象ページ
- https://www.qcintel.com/carbon/

// 収集ロジック
1. すべての<a>タグを走査
2. href に "/carbon/article/" を含むもののみ採用
3. Carbon Pulse と同じフィルタを適用
```

---

### 2. search_openai.go - OpenAI検索統合

#### 🚨 重要：OpenAI Responses API の挙動

**期待していた動作：**
```json
{
  "output": [
    {
      "type": "web_search_call",
      "results": [
        {"title": "...", "url": "...", "snippet": "..."}
      ]
    }
  ]
}
```

**実際の動作：**
```json
{
  "output": [
    {
      "type": "web_search_call",
      "results": [],  // ← 常に空！
      "action": {}    // sources も空
    },
    {
      "type": "message",
      "content": [
        {
          "text": "I searched and found: https://example.com ..."
        }
      ]
    }
  ]
}
```

**結論：**
- OpenAI Responses API は検索結果を構造化データとして返さない
- message.content にテキスト形式で統合される
- → **正規表現でURL抽出**するしかない

#### URL抽出ロジック

```go
// search_openai.go:177-217
reURL := regexp.MustCompile(`https?://[^\s\)]+`)

for _, it := range r.Output {
    if it.Type != "message" { continue }
    for _, cp := range it.Content {
        if cp.Text != "" {
            urls := reURL.FindAllString(cp.Text, -1)
            for _, u := range urls {
                u = strings.TrimRight(u, ".,;:!?")  // 末尾の句読点除去
                // ... URL追加
            }
        }
    }
}
```

#### 疑似タイトル生成（generateTitleFromURL）

**問題：**
- 抽出したURLにはタイトル情報がない
- マッチングにはタイトルが必須
- → URLから疑似タイトルを生成

**アルゴリズム（search_openai.go:53-101）：**
```go
// 入力：https://www.lse.ac.uk/granthaminstitute/wp-content/uploads/2025/06/Global-Trends-in-Climate-Change-Litigation-2025-Snapshot.pdf
// 出力：Lse Granthaminstitute Wp Content Uploads Global Trends In Climate Change Litigation 2025 Snapshot.pdf

1. ドメイン抽出：lse.ac.uk → lse
2. パス分解：/granthaminstitute/wp-content/uploads/2025/06/Global-Trends...
3. 意味のある部分を抽出：
   - 数字のみのパート（06等）→ 除外
   - 短すぎるパート（wp等、len < 3）→ 除外
   - 残り：granthaminstitute, content, uploads, Global-Trends-in-Climate...
4. ハイフン・アンダースコアをスペースに変換
5. 各単語を先頭大文字化
```

**制約：**
- PDF名がランダム文字列の場合は意味がない
- ドメイン名が略称の場合（例：lse）も情報が少ない
- → **Brave Search API等で本物のタイトルを取得すべき**

---

### 3. search_queries.go - 検索クエリ生成

#### 戦略

```go
// 基本戦略
queries := []string{
    `"見出し完全一致"`,                          // ① 引用符で完全一致
    "見出し + カーボン市場キーワード",            // ② VCM, ETS等
    "見出し + 地域別site:演算子",                // ③ site:go.kr等
    "見出し + filetype:pdf",                    // ④ PDF優先
    "見出し + official announcement",          // ⑤ 公式発表
    "見出し + site:unfccc.int OR ...",        // ⑥ NGO優先
}
```

#### 地域別site:演算子マッピング

| 検出キーワード | site:演算子 |
|--------------|-----------|
| "south korea", "korea" | `site:go.kr` |
| "eu", "europe" | `site:europa.eu` |
| "japan" | `site:go.jp` |
| "uk", "united kingdom" | `site:gov.uk` |
| "china" | `site:gov.cn` |
| "australia" | `site:gov.au` |

#### カーボン市場キーワード拡張

| 略語 | 拡張 |
|-----|------|
| VCM | voluntary carbon market |
| ETS | emissions trading system |
| CORSIA | CORSIA ICAO |
| CCER | CCER China |

---

### 4. matcher.go - マッチングアルゴリズム

#### シグナル抽出（extractSignals）

```go
type Signals struct {
    Markets map[string]bool  // EUA, UKA, RGGI, CCA, ACCU, NZU, etc.
    Topics  map[string]bool  // VCM, CDR, DAC, biochar, methane, etc.
    Geos    map[string]bool  // united_states, eu, south_korea, etc.
}
```

**Market シグナル例：**
- "EU ETS" → `markets["eua"] = true`
- "UK ETS" → `markets["uka"] = true`
- "RGGI" → `markets["rggi"] = true`

**Topic シグナル例：**
- "voluntary carbon market" → `topics["vcm"] = true`
- "biochar" → `topics["biochar"] = true`

**Geo シグナル例：**
- 正規表現：`\bUS\b` → `geos["united_states"] = true`
- 文字列検出：`"south korea"` → `geos["south_korea"] = true`

#### IDF（逆文書頻度）計算

```go
// すべての見出し + 候補のタイトルをコーパスとして使用
docs := [][]string{
    tokenize("Climate litigation marks turning point"),
    tokenize("LSE Grantham Institute PDF"),
    // ...
}

idf := buildIDF(docs)
// idf["climate"] = log(1 + N / (1 + df["climate"]))
```

#### スコアリング（scoreHeadlineCandidate）

```go
score = 0.56 * overlap       // IDF加重Recall
      + 0.28 * titleSim      // IDF加重Jaccard
      + 0.06 * marketMatch   // Market信号一致度
      + 0.04 * topicMatch    // Topic信号一致度
      + 0.02 * geoMatch      // Geo信号一致度
      + 0.04 * recency       // 新しさ（exp(-age/14))
      + qualityBoost         // ドメイン品質（最大0.18）
```

**ドメイン品質スコア（sourceQualityBoost）：**

| ドメイン種別 | スコア |
|------------|-------|
| `.gov`, `.gov.uk`, `europa.eu` | +0.18 |
| `.pdf` ファイル | +0.18 |
| NGO（carbonmarketwatch.org等） | +0.12 |
| IR（/investor/, /ir/） | +0.12 |
| プレスリリース配信 | +0.08 |

#### 除外ルール

```go
// 1. Market厳格マッチング（strictMarket=true）
if strictMarket && len(hs.Markets) > 0 && marketMatch == 0 {
    return false  // 見出しにmarket信号があるのに候補にない → 除外
}

// 2. 特定地域マッチング
if hasSpecificGeo(hs) && geoMatch == 0 {
    return false  // 見出しに特定地域（韓国等）があるのに候補にない → 除外
}

// 3. 語彙的実質性
if sharedTokens < 2 && titleSim < 0.90 {
    return false  // 共有トークンが2未満 かつ 類似度が0.9未満 → 除外
}

// 4. 広すぎる地域のみのマッチ回避
if marketMatch == 0 && topicMatch == 0 && geoMatch > 0 && overlap < 0.50 {
    return false  // market/topic無し、geoのみ、overlapが低い → 除外
}
```

---

## トークン化（tokenize）

### 正規表現パターン
```go
reTok = regexp.MustCompile(`[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*`)
// マッチ例：
// - "carbon-pulse" → 1トークン
// - "climate-change-litigation" → 1トークン
// - "EUA" → 1トークン
```

### 正規化マッピング
```go
normToken = map[string]string{
    "euas": "eua", "eua": "eua",
    "credits": "credit", "credit": "credit",
    "offsets": "offset", "offset": "offset",
    // ...
}
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

**注意：** ストップワードは最小限に留める（過度に除去するとマッチング精度が下がる）

---

## パフォーマンス最適化

### 現在の処理フロー

```
見出し収集：      ~5秒（perSource=10の場合）
検索実行：        ~2秒/query（OpenAI API）
                 → 見出し10件 × クエリ3件 = ~60秒
IDF構築：         ~0.1秒
マッチング：      ~0.5秒
合計：           ~65秒（10見出しの場合）
```

### ボトルネック

1. **OpenAI API レスポンスタイム**
   - 平均2秒/query
   - 並列化できない（API制限）

2. **構造化データが取れない**
   - message.contentのパースが必要
   - タイトル生成のオーバーヘッド

### 最適化案

#### すぐできる改善
```go
// 1. クエリ数を動的調整
if hasMarketSignal(headline) {
    queries = queries[:2]  // market特化クエリのみ
}

// 2. 並列化（goroutine）
for i, h := range headlines {
    go func(idx int, headline Headline) {
        // 検索実行
    }(i, h)
}
```

#### 長期的改善
- **Brave Search API**: レスポンスタイム ~500ms、構造化データあり
- **ローカルキャッシュ**: 同じクエリは再検索しない
- **バッチ処理**: 複数見出しをまとめて処理

---

## テスト戦略

### 単体テスト（現在未実装）

```go
// matcher_test.go
func TestExtractSignals(t *testing.T) {
    sig := extractSignals("EU ETS carbon price hits record high")
    assert.True(t, sig.Markets["eua"])
    assert.True(t, sig.Geos["eu"])
}

func TestGenerateTitleFromURL(t *testing.T) {
    title := generateTitleFromURL("https://energy.gov/sites/default/files/clean-hydrogen.pdf")
    assert.Contains(t, title, "Energy")
    assert.Contains(t, title, "Clean Hydrogen")
}
```

### 統合テスト

```bash
# 小規模テスト
./carbon-relay -sources=carbonpulse -perSource=1 -queriesPerHeadline=1

# 期待される動作：
# - 1件のヘッドラインが収集される
# - 1件の検索クエリが実行される
# - relatedFree が0〜3件返される
```

### 品質チェック

```bash
# 候補プールを確認
./carbon-relay -saveFree=candidates.json

# 確認ポイント：
# 1. URLが正しく抽出されているか
# 2. Titleが意味のあるものか（URLそのままでないか）
# 3. Sourceが "OpenAI(text_extract)" になっているか
```

---

## デバッグガイド

### DEBUG_OPENAI=1

```bash
DEBUG_OPENAI=1 ./carbon-relay ...

# 出力例：
[DEBUG] OpenAI response for query '"Climate litigation"':
[DEBUG] Output items: 2
[DEBUG]   [0] Type=web_search_call, Results=0
[DEBUG]       Action.Sources=0
[DEBUG]   [1] Type=message, Results=0
[DEBUG] Processing Action.Sources: 0 items
[DEBUG] Total candidates collected: 0
[DEBUG] Attempting URL extraction from message.content.text
[DEBUG] Found message item with 1 content parts
[DEBUG] Content text: I searched and found https://example.com ...
[DEBUG] Extracted 3 URLs from text
[DEBUG]   -> Added URL: https://example.com/article1
```

### DEBUG_OPENAI_FULL=1

```bash
DEBUG_OPENAI_FULL=1 ./carbon-relay ...

# OpenAI APIのレスポンス全体をJSON形式で出力
# 用途：新しいフィールドの発見、エラー詳細の確認
```

### よくあるデバッグシナリオ

#### relatedFreeが常に空

```bash
# 1. minScoreを下げる
./carbon-relay -minScore=0.1

# 2. デバッグ出力で候補数を確認
DEBUG_OPENAI=1 ./carbon-relay -saveFree=candidates.json

# 3. candidates.jsonを確認
# → 候補が0件なら検索の問題
# → 候補はあるがマッチしないならスコアリングの問題
```

#### 無関係な結果ばかり

```bash
# strictMarketをfalseにしてみる
./carbon-relay -strictMarket=false

# 検索クエリを確認
# → search_queries.go の buildSearchQueries をデバッグ出力
```

---

## よくある質問（FAQ）

### Q1: Brave Search APIに移行したい

```go
// 新規ファイル：cmd/pipeline/search_brave.go
package main

import (
    "encoding/json"
    "net/http"
)

type braveSearchResult struct {
    Web struct {
        Results []struct {
            Title       string `json:"title"`
            URL         string `json:"url"`
            Description string `json:"description"`
        } `json:"results"`
    } `json:"web"`
}

func braveWebSearch(query string, limit int) ([]FreeArticle, error) {
    apiKey := os.Getenv("BRAVE_API_KEY")
    url := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
        url.QueryEscape(query), limit)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-Subscription-Token", apiKey)

    // ... レスポンス処理

    for _, res := range result.Web.Results {
        cands = append(cands, FreeArticle{
            Source:  "Brave",
            Title:   res.Title,      // ← 本物のタイトル！
            URL:     res.URL,
            Excerpt: res.Description,
        })
    }

    return cands, nil
}
```

### Q2: 新しいmarketシグナルを追加したい

```go
// matcher.go の marketTerms に追加
var marketTerms = []string{
    "eua", "uka", "rggi", "cca", "accu", "nzu", "irec", "ccer", "corsia",
    "jcm",  // ← 追加例：Japan Credit Mechanism
}

// normToken にも追加
var normToken = map[string]string{
    // ...
    "jcm": "jcm",
    "japan credit mechanism": "jcm",
}
```

### Q3: 特定ドメインを優先したい

```go
// matcher.go の sourceQualityBoost に追加
func sourceQualityBoost(u string) float64 {
    // ...

    // 新規追加例
    priorityDomains := []string{
        "climate-action.info",
        "carbon-neutral.org",
    }
    for _, d := range priorityDomains {
        if strings.HasSuffix(host, d) {
            return 0.15
        }
    }

    return 0
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
   // OpenAI Responses APIはresultsを返さないため、textから抽出
   reURL := regexp.MustCompile(`https?://[^\s\)]+`)

   // ❌ Bad：コードを繰り返すだけ
   // URLを抽出する
   reURL := regexp.MustCompile(`https?://[^\s\)]+`)
   ```

3. **命名**
   - 変数：`camelCase`
   - 関数：`camelCase`
   - 定数：`UPPER_SNAKE_CASE`（Goでは普通はPascalCase）
   - エクスポート：`PascalCase`

---

## リリースチェックリスト

- [ ] すべてのデバッグ出力を削除（または環境変数で制御）
- [ ] go.mod / go.sum が正しい
- [ ] README.md が最新
- [ ] DEVELOPMENT.md が最新
- [ ] エラーメッセージがユーザーフレンドリー
- [ ] APIキーがハードコードされていない
- [ ] パフォーマンステスト（100見出し処理時間）
- [ ] メモリリーク確認

---

## 参考リンク

- [OpenAI Responses API Documentation](https://platform.openai.com/docs/api-reference/responses)
- [Brave Search API](https://brave.com/search/api/)
- [Carbon Pulse](https://carbon-pulse.com/)
- [QCI](https://www.qcintel.com/)
