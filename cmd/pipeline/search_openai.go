// search_openai.go
// OpenAI Responses API を使用したWeb検索統合
//
// このモジュールは、OpenAI Responses API（旧 Web Search API）を使用して
// 有料記事の見出しに関連する無料記事をWeb上から検索します。
//
// 主要な機能:
//   - OpenAI Responses API へのHTTPリクエスト送信
//   - 3段階のフォールバックでURL抽出:
//     1. web_search_call.results（通常は空）
//     2. action.sources（URLのみ）
//     3. message.content から正規表現でURL抽出（主要手法）
//   - URLから擬似タイトル生成（マッチング用）
//   - URL重複排除（global seen map）
//
// 重要な実装の詳細:
//   - OpenAI Responses API は web_search_call.results を返さない（仕様）
//   - action.sources も通常は空
//   - そのため、message.content テキストから正規表現でURLを抽出する
//   - 抽出したURLから generateTitleFromURL() で擬似タイトルを生成
//
// デバッグモード:
//   - DEBUG_OPENAI=1: OpenAI検索のサマリーログ
//   - DEBUG_OPENAI_FULL=1: 完全なレスポンスJSON
package main

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

// OpenAI Responses API のレスポンス構造体

// openAIResponsesResp は OpenAI Responses API の最上位レスポンス
type openAIResponsesResp struct {
	Output []openAIOutputItem `json:"output"`  // アウトプット配列（web_search_call, message等）
}

// openAIOutputItem は output配列の各要素
type openAIOutputItem struct {
	Type    string              `json:"type"`              // "web_search_call" または "message"
	Results []openAIWebResult   `json:"results,omitempty"` // web_search_call の結果（通常は空）
	Action  *openAIWebAction    `json:"action,omitempty"`  // include時のsources（通常は空）
	Content []openAIContentPart `json:"content,omitempty"` // message のコンテンツ（URL抽出元）
}

// openAIWebAction は action フィールドの構造
type openAIWebAction struct {
	Sources []openAIWebSource `json:"sources,omitempty"`  // ソースURL配列（通常は空）
}

// openAIWebSource は sources配列の各要素
type openAIWebSource struct {
	URL string `json:"url"`  // ソースURL
}

// openAIWebResult は results配列の各要素（理想的な形式だが通常は返らない）
type openAIWebResult struct {
	Title   string `json:"title"`    // 記事タイトル
	URL     string `json:"url"`      // 記事URL
	Snippet string `json:"snippet"`  // スニペット
}

// openAIContentPart は message.content の各パート
type openAIContentPart struct {
	Type        string             `json:"type"`                  // "text"
	Text        string             `json:"text,omitempty"`        // テキストコンテンツ（URL抽出元）
	Annotations []openAIAnnotation `json:"annotations,omitempty"` // アノテーション（citations）
}

// openAIAnnotation は annotations配列の各要素
type openAIAnnotation struct {
	Type  string `json:"type"`            // "citation"
	URL   string `json:"url,omitempty"`   // 引用元URL
	Title string `json:"title,omitempty"` // 引用元タイトル
}

// generateTitleFromURL は URLから擬似タイトルを生成（マッチング用）
//
// OpenAI Responses API は通常タイトルを返さないため、URLから推測する。
//
// 例:
//   "https://carbon-pulse.com/timeline/387850/" → "Carbon Pulse Timeline"
//   "https://www.gov.uk/climate-policy" → "Gov Uk Climate Policy"
//
// 処理:
//   1. ドメイン名から意味のある部分を抽出（www. は除去）
//   2. URLパスから意味のある部分を抽出（数字のみのパートは除外）
//   3. ハイフン/アンダースコアをスペースに変換
//   4. 各単語の先頭を大文字化
//
// 引数:
//   rawURL: 対象URL
//
// 戻り値:
//   生成された擬似タイトル
func generateTitleFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// ドメイン名から意味のある部分を抽出
	host := strings.TrimPrefix(u.Host, "www.")
	hostParts := strings.Split(host, ".")
	domain := hostParts[0]

	// パスから意味のある部分を抽出
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	var meaningfulParts []string

	for _, part := range pathParts {
		// 数字だけのパート（IDなど）は除外
		if regexp.MustCompile(`^\d+$`).MatchString(part) {
			continue
		}
		// 短すぎるパートは除外
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

	// 先頭大文字化
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

func openaiWebSearch(query string, limit int, model string, toolType string) ([]FreeArticle, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	// CRITICAL: OpenAI Responses APIはresultsを構造化して返さないため、
	// テキスト形式でURLリストを返させ、後でパースする戦略を取る
	prompt := fmt.Sprintf(`Search for: %s

After searching, list ONLY the URLs you found, one per line. Format:
URL: https://example.com
URL: https://another.com

Do NOT write explanations. ONLY URLs.`, query)

	reqBody := map[string]any{
		"model": model,
		"input": prompt,
		"tools": []map[string]any{
			{"type": toolType}, // "web_search" or "web_search_preview"
		},
		// NOTE: include を指定しない（デフォルトですべて返す）
		// URLリストを返すために少し余裕を持たせる
		"max_output_tokens": 500,
	}

	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai responses error: %s\n%s", resp.Status, string(bodyBytes))
	}

	// DEBUG: レスポンス全体を出力
	if os.Getenv("DEBUG_OPENAI_FULL") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Full OpenAI response:\n%s\n", string(bodyBytes))
	}

	var r openAIResponsesResp
	if err := json.Unmarshal(bodyBytes, &r); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	// DEBUG: レスポンスの内容を確認
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

	// 1) まず web_search_call.results を拾う（ここが本命）
	cands := make([]FreeArticle, 0, limit)
	seen := map[string]bool{}
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

		// 2) include されていれば action.sources も拾える（タイトル無しなのでURLをタイトルにする）
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
					Title:  u,
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

	// 3) web_search_callが結果を返さない場合は、message.content.textからURLを抽出
	if len(cands) == 0 {
		if os.Getenv("DEBUG_OPENAI") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Attempting URL extraction from message.content.text\n")
		}
		reURL := regexp.MustCompile(`https?://[^\s\)]+`)
		for _, it := range r.Output {
			if it.Type != "message" {
				continue
			}
			if os.Getenv("DEBUG_OPENAI") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Found message item with %d content parts\n", len(it.Content))
			}
			for _, cp := range it.Content {
				// まずテキストからURL抽出
				if cp.Text != "" {
					if os.Getenv("DEBUG_OPENAI") != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG] Content text: %s\n", cp.Text[:min(200, len(cp.Text))])
					}
					urls := reURL.FindAllString(cp.Text, -1)
					if os.Getenv("DEBUG_OPENAI") != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG] Extracted %d URLs from text\n", len(urls))
					}
					for _, u := range urls {
						u = strings.TrimRight(u, ".,;:!?")
						if u == "" || seen[u] {
							continue
						}
						seen[u] = true
						// URLから疑似タイトルを生成（マッチング精度向上のため）
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
				// fallback: annotations（url_citation）も拾う
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

	// 安定のため URL ソート
	sort.Slice(cands, func(i, j int) bool { return cands[i].URL < cands[j].URL })

	if limit > 0 && len(cands) > limit {
		cands = cands[:limit]
	}
	return cands, nil
}
