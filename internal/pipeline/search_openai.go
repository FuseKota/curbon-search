// =============================================================================
// search_openai.go - OpenAI Web検索統合モジュール
// =============================================================================
//
// このファイルはOpenAI Responses APIを使用して、有料記事の見出しに関連する
// 無料記事をWeb上から検索します。モード2（有料記事マッチング）で使用されます。
//
// =============================================================================
// 【重要な実装の詳細】
// =============================================================================
//
// OpenAI Responses APIには以下の癖があります：
//
// 1. web_search_call.results は通常空で返される
//    → 理想的にはここにタイトル・URL・スニペットが入るはずだが、実際には空
//
// 2. action.sources も通常空
//    → include設定をしても結果が得られないことが多い
//
// 3. 解決策: message.content.text からURLを正規表現で抽出
//    → プロンプトで「URLリストのみを返せ」と指示
//    → 返されたテキストから https://... パターンを抽出
//    → URLから擬似タイトルを生成（generateTitleFromURL）
//
// =============================================================================
// 【URL抽出の3段階フォールバック】
// =============================================================================
//
// 優先度1: web_search_call.results（理想的だが通常は空）
//     ↓
// 優先度2: action.sources（include時、これも通常は空）
//     ↓
// 優先度3: message.content.text から正規表現で抽出 ★主要手法★
//     ↓
// 優先度4: annotations（url_citation）から抽出
//
// =============================================================================
// 【デバッグ方法】
// =============================================================================
//
// 環境変数でデバッグ情報を出力:
//   DEBUG_OPENAI=1      - 検索のサマリーログを出力
//   DEBUG_OPENAI_FULL=1 - 完全なレスポンスJSONを出力
//
// 使用例:
//   DEBUG_OPENAI=1 ./pipeline -sources=carbonpulse -perSource=1 -queriesPerHeadline=1
//
// =============================================================================
// 【初心者向けポイント】
// =============================================================================
//
// - 外部APIを呼び出す際は必ずタイムアウトを設定（ここでは60秒）
// - APIキーは環境変数から取得（OPENAI_API_KEY）
// - レスポンスのJSON構造を事前に定義（構造体で受け取る）
// - エラーハンドリングは各段階で行う
//
// =============================================================================
package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// =============================================================================
// OpenAI Responses API レスポンス構造体
// =============================================================================
//
// OpenAI Responses APIのJSONレスポンスをGoの構造体にマッピングします。
// 各フィールドは json:"xxx" タグでJSONキーと対応付けられています。
//

// openAIResponsesResp は OpenAI Responses API の最上位レスポンス
//
// 【JSON構造】
//
//	{
//	  "output": [
//	    { "type": "web_search_call", ... },
//	    { "type": "message", ... }
//	  ]
//	}
type openAIResponsesResp struct {
	Output []openAIOutputItem `json:"output"` // 出力アイテムの配列
}

// openAIOutputItem は output配列の各要素
//
// 【Typeの種類】
//   - "web_search_call": Web検索の結果（通常は空）
//   - "message": AIの応答メッセージ（URLを抽出する対象）
type openAIOutputItem struct {
	Type    string              `json:"type"`              // アイテムの種類
	Results []openAIWebResult   `json:"results,omitempty"` // 検索結果（通常は空）
	Action  *openAIWebAction    `json:"action,omitempty"`  // アクション情報
	Content []openAIContentPart `json:"content,omitempty"` // メッセージ内容
}

// openAIWebAction は action フィールドの構造
type openAIWebAction struct {
	Sources []openAIWebSource `json:"sources,omitempty"` // ソースURL配列
}

// openAIWebSource は sources配列の各要素
type openAIWebSource struct {
	URL string `json:"url"` // ソースURL
}

// openAIWebResult は results配列の各要素
//
// 【注意】理想的にはここにタイトル・URL・スニペットが入るが、
// 現在のAPIでは通常空で返される
type openAIWebResult struct {
	Title   string `json:"title"`   // 記事タイトル
	URL     string `json:"url"`     // 記事URL
	Snippet string `json:"snippet"` // スニペット（要約）
}

// openAIContentPart は message.content の各パート
//
// 【重要】Textフィールドが主要なURL抽出元
type openAIContentPart struct {
	Type        string             `json:"type"`                  // パートの種類（"text"など）
	Text        string             `json:"text,omitempty"`        // テキスト内容（URL抽出元）
	Annotations []openAIAnnotation `json:"annotations,omitempty"` // アノテーション情報
}

// openAIAnnotation は annotations配列の各要素
//
// url_citationタイプの場合、URLとタイトルが含まれる（フォールバック用）
type openAIAnnotation struct {
	Type  string `json:"type"`            // アノテーションの種類
	URL   string `json:"url,omitempty"`   // 引用元URL
	Title string `json:"title,omitempty"` // 引用元タイトル
}

// =============================================================================
// ヘルパー関数
// =============================================================================

// generateTitleFromURL はURLから擬似タイトルを生成する
//
// OpenAI Responses APIは通常タイトルを返さないため、URLから推測します。
// マッチングの精度を上げるために、意味のある単語を抽出します。
//
// 【処理の流れ】
//  1. ドメイン名から www. を除去
//  2. URLパスを / で分割
//  3. 数字のみ・短すぎるパートを除外
//  4. ハイフン/アンダースコアをスペースに変換
//  5. 各単語の先頭を大文字化
//
// 【変換例】
//
//	"https://carbon-pulse.com/timeline/387850/"
//	  → "Carbon Pulse Timeline"
//
//	"https://www.gov.uk/climate-policy-update"
//	  → "Gov Uk Climate Policy Update"
func generateTitleFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // パース失敗時はそのまま返す
	}

	// ドメイン名から www. を除去して最初の部分を取得
	// 例: "www.carbon-pulse.com" → "carbon-pulse"
	host := strings.TrimPrefix(u.Host, "www.")
	hostParts := strings.Split(host, ".")
	domain := hostParts[0]

	// URLパスを分割して意味のある部分を抽出
	// 例: "/timeline/387850/" → ["timeline", "387850"]
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	var meaningfulParts []string

	for _, part := range pathParts {
		// 数字だけのパート（記事IDなど）は除外
		if regexp.MustCompile(`^\d+$`).MatchString(part) {
			continue
		}
		// 短すぎるパートは除外（意味を持たないことが多い）
		if len(part) < 3 {
			continue
		}
		meaningfulParts = append(meaningfulParts, part)
	}

	// タイトル生成
	title := domain
	if len(meaningfulParts) > 0 {
		// ハイフンやアンダースコアをスペースに変換
		for i, part := range meaningfulParts {
			meaningfulParts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "-", " "), "_", " ")
		}
		title = domain + " " + strings.Join(meaningfulParts, " ")
	}

	// 各単語の先頭を大文字化
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// =============================================================================
// メイン関数: OpenAI Web検索
// =============================================================================

// openaiWebSearch はOpenAI Responses APIを使用してWeb検索を実行する
//
// 【処理の流れ】
//  1. APIキーの確認
//  2. プロンプトの構築（URLリスト形式で返すよう指示）
//  3. HTTPリクエストの送信
//  4. レスポンスのパース
//  5. 3段階のフォールバックでURL抽出
//  6. 結果をソートして返す
//
// 引数:
//
//	query:    検索クエリ
//	limit:    取得する最大記事数
//	model:    使用するOpenAIモデル（例: "gpt-4o-mini"）
//	toolType: ツールタイプ（"web_search" または "web_search_preview"）
//
// 戻り値:
//
//	検索結果の記事リスト、エラー
func openaiWebSearch(query string, limit int, model string, toolType string) ([]FreeArticle, error) {
	// =========================================================================
	// 1. APIキーの確認
	// =========================================================================
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	// =========================================================================
	// 2. プロンプトの構築
	// =========================================================================
	// 重要: OpenAI Responses APIはresultsを構造化して返さないため、
	// テキスト形式でURLリストを返させ、後でパースする戦略を取る
	prompt := fmt.Sprintf(`Search for: %s

After searching, list ONLY the URLs you found, one per line. Format:
URL: https://example.com
URL: https://another.com

Do NOT write explanations. ONLY URLs.`, query)

	// =========================================================================
	// 3. HTTPリクエストの構築
	// =========================================================================
	reqBody := map[string]any{
		"model": model,
		"input": prompt,
		"tools": []map[string]any{
			{"type": toolType}, // "web_search" または "web_search_preview"
		},
		"max_output_tokens": 500, // URLリストには500トークンで十分
	}

	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// =========================================================================
	// 4. HTTPリクエストの送信
	// =========================================================================
	client := &http.Client{Timeout: 60 * time.Second} // タイムアウト60秒
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取り
	bodyBytes, _ := io.ReadAll(resp.Body)

	// HTTPエラーチェック（300番台以上はエラー）
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai responses error: %s\n%s", resp.Status, string(bodyBytes))
	}

	// =========================================================================
	// デバッグ: レスポンス全体を出力
	// =========================================================================
	if os.Getenv("DEBUG_OPENAI_FULL") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Full OpenAI response:\n%s\n", string(bodyBytes))
	}

	// =========================================================================
	// 5. レスポンスのパース
	// =========================================================================
	var r openAIResponsesResp
	if err := json.Unmarshal(bodyBytes, &r); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	// デバッグ: レスポンスの構造を確認
	if os.Getenv("DEBUG_OPENAI") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] OpenAI response for query '%s':\n", query)
		fmt.Fprintf(os.Stderr, "[DEBUG] Output items: %d\n", len(r.Output))
		for i, it := range r.Output {
			fmt.Fprintf(os.Stderr, "[DEBUG]   [%d] Type=%s, Results=%d\n", i, it.Type, len(it.Results))
			if it.Action != nil {
				fmt.Fprintf(os.Stderr, "[DEBUG]       Action.Sources=%d\n", len(it.Action.Sources))
			}
		}
	}

	// =========================================================================
	// 6. URL抽出（3段階フォールバック）
	// =========================================================================

	// 結果を格納するスライスと重複チェック用マップ
	cands := make([]FreeArticle, 0, limit)
	seen := map[string]bool{}

	// -------------------------------------------------------------------------
	// 優先度1: web_search_call.results から抽出（理想的だが通常は空）
	// -------------------------------------------------------------------------
	for _, it := range r.Output {
		if it.Type != "web_search_call" {
			continue
		}
		for _, res := range it.Results {
			u := strings.TrimSpace(res.URL)
			if u == "" || seen[u] {
				continue
			}
			seen[u] = true
			cands = append(cands, FreeArticle{
				Source:  "OpenAI(web_search)",
				Title:   strings.TrimSpace(res.Title),
				URL:     u,
				Excerpt: strings.TrimSpace(res.Snippet),
			})
		}

		// ---------------------------------------------------------------------
		// 優先度2: action.sources から抽出（タイトルなし）
		// ---------------------------------------------------------------------
		if it.Action != nil {
			if os.Getenv("DEBUG_OPENAI") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Processing Action.Sources: %d items\n", len(it.Action.Sources))
			}
			for _, s := range it.Action.Sources {
				u := strings.TrimSpace(s.URL)
				if os.Getenv("DEBUG_OPENAI") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG]   Source URL: %s\n", u)
				}
				if u == "" || seen[u] {
					if os.Getenv("DEBUG_OPENAI") != "" && u != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG]   (skipped: already seen)\n")
					}
					continue
				}
				seen[u] = true
				cands = append(cands, FreeArticle{
					Source: "OpenAI(web_search_sources)",
					Title:  u, // タイトルがないのでURLをそのまま使用
					URL:    u,
				})
				if os.Getenv("DEBUG_OPENAI") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG]   -> Added to candidates\n")
				}
			}
		}
	}

	if os.Getenv("DEBUG_OPENAI") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Total candidates collected: %d\n", len(cands))
	}

	// -------------------------------------------------------------------------
	// 優先度3: message.content.text から正規表現でURL抽出 ★主要手法★
	// -------------------------------------------------------------------------
	if len(cands) == 0 {
		if os.Getenv("DEBUG_OPENAI") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Attempting URL extraction from message.content.text\n")
		}

		// URLを抽出する正規表現パターン
		reURL := regexp.MustCompile(`https?://[^\s\)]+`)

		for _, it := range r.Output {
			if it.Type != "message" {
				continue
			}
			if os.Getenv("DEBUG_OPENAI") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Found message item with %d content parts\n", len(it.Content))
			}

			for _, cp := range it.Content {
				// テキストからURL抽出
				if cp.Text != "" {
					if os.Getenv("DEBUG_OPENAI") != "" {
						// 先頭200文字のみ表示（長すぎるログを避ける）
						fmt.Fprintf(os.Stderr, "[DEBUG] Content text: %s\n", cp.Text[:min(200, len(cp.Text))])
					}

					urls := reURL.FindAllString(cp.Text, -1)
					if os.Getenv("DEBUG_OPENAI") != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG] Extracted %d URLs from text\n", len(urls))
					}

					for _, u := range urls {
						// 末尾の句読点を除去
						u = strings.TrimRight(u, ".,;:!?")
						if u == "" || seen[u] {
							continue
						}
						seen[u] = true

						// URLから擬似タイトルを生成（マッチング精度向上のため）
						title := generateTitleFromURL(u)
						cands = append(cands, FreeArticle{
							Source: "OpenAI(text_extract)",
							Title:  title,
							URL:    u,
						})
						if os.Getenv("DEBUG_OPENAI") != "" {
							fmt.Fprintf(os.Stderr, "[DEBUG]   -> Added URL: %s\n", u)
						}
					}
				}

				// -----------------------------------------------------------------
				// 優先度4: annotations（url_citation）から抽出（フォールバック）
				// -----------------------------------------------------------------
				for _, ann := range cp.Annotations {
					if ann.URL == "" || seen[ann.URL] {
						continue
					}
					seen[ann.URL] = true
					title := ann.Title
					if title == "" {
						title = ann.URL
					}
					cands = append(cands, FreeArticle{
						Source: "OpenAI(citation)",
						Title:  title,
						URL:    ann.URL,
					})
				}
			}
		}
	}

	// =========================================================================
	// 7. 結果の整形
	// =========================================================================

	// URLでソート（結果の安定性のため）
	sort.Slice(cands, func(i, j int) bool { return cands[i].URL < cands[j].URL })

	// 指定された上限で切り詰め
	if limit > 0 && len(cands) > limit {
		cands = cands[:limit]
	}

	return cands, nil
}

// =============================================================================
// SearchResult - 検索結果を保持する構造体
// =============================================================================

// SearchResult は SearchForHeadlines の結果を保持する
type SearchResult struct {
	CandsByIdx [][]FreeArticle // 各見出しに対応する候補記事
	GlobalPool []FreeArticle   // 全候補のプール（重複除去済み）
}

// =============================================================================
// SearchForHeadlines - 見出しに対してWeb検索を実行
// =============================================================================

// SearchForHeadlines は各見出しに対してWeb検索を実行し、候補記事を収集する
//
// 【処理の流れ】
//  1. 各見出しから検索クエリを生成
//  2. OpenAI Web検索を実行
//  3. 結果を重複除去しながらマージ
//  4. 見出しごとの候補とグローバルプールを返す
//
// 引数:
//
//	headlines: 検索対象の見出しリスト
//	cfg:       検索設定
//
// 戻り値:
//
//	SearchResult: 検索結果（見出しごとの候補 + グローバルプール）
func SearchForHeadlines(headlines []Headline, cfg *SearchConfig) SearchResult {
	candsByIdx := make([][]FreeArticle, len(headlines))
	globalSeen := map[string]bool{}
	globalPool := make([]FreeArticle, 0, len(headlines)*cfg.SearchPerHeadline)

	if !cfg.IsEnabled() {
		infof("Search disabled (queriesPerHeadline=0), skipping web search phase")
		return SearchResult{CandsByIdx: candsByIdx, GlobalPool: globalPool}
	}

	for i, h := range headlines {
		// クエリを生成
		queries := h.SearchQueries
		if len(queries) == 0 {
			queries = buildSearchQueries(h.Title, h.Excerpt)
		}
		if len(queries) > cfg.QueriesPerHeadline {
			queries = queries[:cfg.QueriesPerHeadline]
		}

		// 各クエリで検索してマージ
		merged := map[string]FreeArticle{}
		for _, q := range queries {
			var res []FreeArticle
			var err error

			switch cfg.Provider {
			case "openai":
				res, err = openaiWebSearch(q, cfg.ResultsPerQuery, cfg.OpenAIModel, cfg.OpenAITool)
			default:
				err = fmt.Errorf("unsupported searchProvider: %s", cfg.Provider)
			}

			if err != nil {
				warnf("search: %v", err)
				continue
			}

			for _, a := range res {
				if a.URL == "" || a.Title == "" {
					continue
				}
				merged[a.URL] = a
				if len(merged) >= cfg.SearchPerHeadline {
					break
				}
			}
			if len(merged) >= cfg.SearchPerHeadline {
				break
			}
		}

		// 重複除去してグローバルプールに追加
		cands := make([]FreeArticle, 0, len(merged))
		for _, a := range merged {
			cands = append(cands, a)
			if !globalSeen[a.URL] {
				globalSeen[a.URL] = true
				globalPool = append(globalPool, a)
			}
		}
		candsByIdx[i] = cands
	}

	return SearchResult{CandsByIdx: candsByIdx, GlobalPool: globalPool}
}
