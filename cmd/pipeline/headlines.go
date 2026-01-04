// headlines.go
// 18のニュースソースから記事見出しと要約を収集するモジュール
//
// 実装ソース:
//   有料ソース（見出しのみ）: Carbon Pulse, QCI
//   無料ソース（全文取得）: 16ソース
//
// スクレイピング手法:
//   - WordPress REST API（7ソース）
//   - HTML Scraping + goquery（8ソース）
//   - RSS Feed + gofeed（3ソース）
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // HTML解析ライブラリ
	"github.com/mmcdole/gofeed"      // RSS/Atomフィード解析
)

// min2 は2つの整数のうち小さい方を返すヘルパー関数
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// グローバル正規表現パターン
var (
	// reCarbonPulseID は Carbon Pulse の記事ID形式をマッチ（例: /12345/）
	reCarbonPulseID = regexp.MustCompile(`^/\d+/$`)

	// reQCIArticle は QCI の記事URLパターンをマッチ
	reQCIArticle    = regexp.MustCompile(`/carbon/article/`)
)

// headlineSourceConfig は見出し収集時の設定を保持
type headlineSourceConfig struct {
	CarbonPulseTimelineURL string        // Carbon Pulse タイムラインページURL
	CarbonPulseNewsletters string        // Carbon Pulse ニュースレターカテゴリURL
	QCIHomeURL             string        // QCI ホームページURL
	UserAgent              string        // HTTPリクエスト時のUser-Agentヘッダー
	Timeout                time.Duration // HTTPリクエストのタイムアウト時間
}

// defaultHeadlineConfig はデフォルトの見出し収集設定を返す
func defaultHeadlineConfig() headlineSourceConfig {
	return headlineSourceConfig{
		CarbonPulseTimelineURL: "https://carbon-pulse.com/daily-timeline/",
		CarbonPulseNewsletters: "https://carbon-pulse.com/category/newsletters/",
		QCIHomeURL:             "https://www.qcintel.com/carbon/",
		UserAgent:              "Mozilla/5.0 (compatible; carbon-relay/1.0; +https://example.invalid)",
		Timeout:                20 * time.Second, // デフォルト20秒タイムアウト
	}
}

// collectHeadlinesCarbonPulse は Carbon Pulse（有料ソース）から見出しと要約を収集
//
// Carbon Pulse は有料サブスクリプションサービスですが、以下のページは無料でアクセス可能：
//   - トップページ: 記事の要約（excerpt）付き
//   - デイリータイムライン: 見出しのみ
//   - ニュースレターカテゴリ: 見出しのみ
//
// 引数:
//   limit: 収集する最大記事数
//   cfg: スクレイピング設定（タイムアウト、User-Agent等）
//
// 戻り値:
//   収集した見出しのスライス、エラー
func collectHeadlinesCarbonPulse(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// 無料でアクセス可能な3つのページを巡回
	// トップページのみ記事の要約（excerpt）が取得可能
	pages := []string{
		"https://carbon-pulse.com/",       // トップページ（excerpt付き）
		cfg.CarbonPulseTimelineURL,        // デイリータイムライン
		cfg.CarbonPulseNewsletters,        // ニュースレターカテゴリ
	}
	out := []Headline{}              // 収集結果を格納するスライス
	seen := map[string]bool{}        // URL重複チェック用マップ

	// デバッグモード時: スクレイピング対象ページを出力
	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Scraping Carbon Pulse from:\n")
		for _, p := range pages {
			fmt.Fprintf(os.Stderr, "  - %s\n", p)
		}
	}

	for pageIdx, pageURL := range pages {
		doc, err := fetchDoc(pageURL, cfg)
		if err != nil {
			// Do not fail hard; just continue.
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Failed to fetch %s: %v\n", pageURL, err)
			}
			continue
		}

		// Debug: Save HTML to inspect structure
		if os.Getenv("DEBUG_HTML") != "" {
			html, _ := doc.Html()
			if len(html) > 0 {
				length := len(html)
				if length > 2000 {
					length = 2000
				}
				fmt.Fprintf(os.Stderr, "[DEBUG] First 2000 chars of HTML:\n%s\n\n", html[:length])
			}
		}

		// Special handling for top page: extract from main content area (.post divs)
		// Always extract excerpts from the top page
		if pageIdx == 0 {
			doc.Find("div.post").Each(func(_ int, postDiv *goquery.Selection) {
				if limit > 0 && len(out) >= limit {
					return
				}

				// Find the title link in this post
				titleLink := postDiv.Find("h2.posttitle a").First()
				if titleLink.Length() == 0 {
					return
				}

				href, ok := titleLink.Attr("href")
				if !ok {
					return
				}

				txt := strings.TrimSpace(titleLink.Text())
				if txt == "" || len(txt) < 10 {
					return
				}

				abs := resolveURL(pageURL, href)
				if abs == "" {
					return
				}
				u, err := url.Parse(abs)
				if err != nil || u.Host == "" {
					return
				}

				if !strings.HasSuffix(u.Host, "carbon-pulse.com") {
					return
				}

				if seen[abs] {
					return
				}
				seen[abs] = true

				// Extract excerpt from this post div
				var excerpt string
				fullText := postDiv.Text()

				if os.Getenv("DEBUG_HTML") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Post full text (first 2000 chars):\n%s\n\n", fullText[:min2(2000, len(fullText))])
				}

				// Remove "Read More" and everything after
				readMoreIdx := strings.Index(fullText, "Read More")
				if readMoreIdx > 0 {
					fullText = fullText[:readMoreIdx]
				}

				// Split into lines and find the excerpt
				lines := strings.Split(fullText, "\n")
				var excerptBuilder strings.Builder
				maxChars := 500

				for i, line := range lines {
					line = strings.TrimSpace(line)

					// Skip empty lines, metadata, tags, and navigation
					if line == "" || len(line) < 30 {
						continue
					}

					// Skip the title itself (exact match)
					if line == txt {
						continue
					}

					// Skip metadata, tags, CSS, and navigation
					if strings.Contains(line, "Published") ||
						strings.Contains(line, "Last updated") ||
						strings.Contains(line, "Carbon Pulse Premium") ||
						strings.Contains(line, "Nature & Biodiversity") ||
						strings.Contains(line, "Net Zero Pulse") ||
						strings.HasPrefix(line, "Top") ||
						strings.HasPrefix(line, "#") ||
						strings.Contains(line, "{") ||
						strings.Contains(line, "}") ||
						strings.Contains(line, "padding:") ||
						strings.Contains(line, "border:") ||
						strings.Contains(line, "/") && strings.Contains(line, "2025") || // Skip date lines
						strings.Count(line, "/") >= 2 || // Skip author/category lines like "Name / Category / Subcategory"
						strings.Contains(line, "Asia Pacific") ||
						strings.Contains(line, "LATAM") ||
						strings.Contains(line, "EMEA") ||
						strings.Contains(line, "VCM Developments") {
						continue
					}

					// This should be the excerpt
					if len(line) > 30 && !strings.HasPrefix(line, "http") {
						excerptBuilder.WriteString(line)

						// Also check next line if it's a continuation
						if i+1 < len(lines) {
							nextLine := strings.TrimSpace(lines[i+1])
							if len(nextLine) > 20 && !strings.Contains(nextLine, "Read More") && !strings.Contains(nextLine, "Published") {
								excerptBuilder.WriteString(" ")
								excerptBuilder.WriteString(nextLine)
							}
						}
						break
					}
				}

				excerpt = strings.TrimSpace(excerptBuilder.String())
				if len(excerpt) > maxChars {
					excerpt = excerpt[:maxChars] + "..."
				}

				if os.Getenv("DEBUG_SCRAPING") != "" && excerpt != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Extracted excerpt from post (%d chars): %s...\n", len(excerpt), excerpt[:min2(100, len(excerpt))])
				}

				out = append(out, Headline{Source: "Carbon Pulse", Title: txt, URL: abs, Excerpt: excerpt, IsHeadline: true})
			})

			// Skip the regular link extraction for top page since we already processed it
			if limit > 0 && len(out) >= limit {
				break
			}
			continue
		}

		// Regular link extraction for other pages
		doc.Find("a").Each(func(_ int, s *goquery.Selection) {
			if limit > 0 && len(out) >= limit {
				return
			}

			href, ok := s.Attr("href")
			if !ok {
				return
			}
			txt := strings.TrimSpace(s.Text())
			if txt == "" {
				return
			}

			// 無意味なリンクテキストを除外
			txtLower := strings.ToLower(txt)
			if txtLower == "read more" || txtLower == "continue reading" || txtLower == "click here" || len(txt) < 10 {
				return
			}

			abs := resolveURL(pageURL, href)
			if abs == "" {
				return
			}
			u, err := url.Parse(abs)
			if err != nil || u.Host == "" {
				return
			}

			if !strings.HasSuffix(u.Host, "carbon-pulse.com") {
				return
			}

			// Keep only numeric article URLs like /470597/ (avoid nav like /register/).
			if !reCarbonPulseID.MatchString(u.Path) {
				return
			}

			if seen[abs] {
				return
			}
			seen[abs] = true

			// Try to extract excerpt from nearby text (parent, siblings, or following paragraphs)
			// Note: Timeline/newsletter pages typically don't have excerpts in the HTML structure
			excerpt := extractExcerptFromContext(s)

			out = append(out, Headline{Source: "Carbon Pulse", Title: txt, URL: abs, Excerpt: excerpt, IsHeadline: true})
		})

		if limit > 0 && len(out) >= limit {
			break
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no Carbon Pulse headlines found (site may block scraping or layout changed)")
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Collected %d Carbon Pulse headlines\n", len(out))
		if len(out) > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Latest: %s\n", out[0].Title)
			fmt.Fprintf(os.Stderr, "[DEBUG] URL: %s\n", out[0].URL)
		}
	}

	return out, nil
}

func collectHeadlinesQCI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Scraping QCI from: %s\n", cfg.QCIHomeURL)
	}

	doc, err := fetchDoc(cfg.QCIHomeURL, cfg)
	if err != nil {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to fetch QCI: %v\n", err)
		}
		return nil, err
	}

	out := []Headline{}
	seen := map[string]bool{}

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		href, ok := s.Attr("href")
		if !ok {
			return
		}
		txt := strings.TrimSpace(s.Text())
		if txt == "" {
			return
		}

		// 無意味なリンクテキストを除外
		txtLower := strings.ToLower(txt)
		if txtLower == "read more" || txtLower == "continue reading" || txtLower == "click here" || len(txt) < 10 {
			return
		}

		abs := resolveURL(cfg.QCIHomeURL, href)
		if abs == "" {
			return
		}
		u, err := url.Parse(abs)
		if err != nil || u.Host == "" {
			return
		}
		if !strings.HasSuffix(u.Host, "qcintel.com") {
			return
		}
		if !reQCIArticle.MatchString(u.Path) {
			return
		}

		if seen[abs] {
			return
		}
		seen[abs] = true

		out = append(out, Headline{Source: "QCI", Title: txt, URL: abs, IsHeadline: true})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no QCI headlines found (site may block scraping or layout changed)")
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Collected %d QCI headlines\n", len(out))
		if len(out) > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Latest: %s\n", out[0].Title)
			fmt.Fprintf(os.Stderr, "[DEBUG] URL: %s\n", out[0].URL)
		}
	}

	return out, nil
}

// fetchDoc は指定URLからHTMLドキュメントを取得してgoqueryでパース
//
// タイムアウト設定と適切なHTTPヘッダー（User-Agent, Accept）を含めて
// HTTPリクエストを送信し、レスポンスをgoquery.Documentとして返す
//
// 引数:
//   u: 取得するURL
//   cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//   パースされたHTMLドキュメント、エラー
func fetchDoc(u string, cfg headlineSourceConfig) (*goquery.Document, error) {
	client := &http.Client{Timeout: cfg.Timeout}  // タイムアウト付きHTTPクライアント
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	// ブロッキング回避のため、ブラウザ風のヘッダーを設定
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// HTTPステータスコードチェック（200番台以外はエラー）
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: status %s", u, resp.Status)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// resolveURL は相対URLを絶対URLに変換
//
// ベースURLと相対URL（href）から完全な絶対URLを生成する。
// 既に絶対URLの場合はそのまま返す。
//
// 引数:
//   baseURL: 基準となるページのURL
//   href: 相対または絶対URL
//
// 戻り値:
//   解決された絶対URL（エラー時は空文字列）
func resolveURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	// 相対URLを絶対URLに解決
	return base.ResolveReference(u).String()
}

// extractExcerptFromContext はリンク周辺のテキストから記事要約を抽出
//
// 個別の記事ページをフェッチせず、タイムライン/一覧ページ内で
// リンク要素の周辺にあるテキストコンテンツを要約として抽出する。
//
// 引数:
//   linkSel: 記事リンクのgoquery Selection
//
// 戻り値:
//   抽出された要約テキスト（最大500文字）
func extractExcerptFromContext(linkSel *goquery.Selection) string {
	var excerpt strings.Builder
	maxChars := 500

	// Debug: show link context - check multiple parent levels
	if os.Getenv("DEBUG_HTML") != "" {
		linkHTML, _ := linkSel.Html()
		parent1HTML, _ := linkSel.Parent().Html()
		parent2HTML, _ := linkSel.Parent().Parent().Html()
		parent3HTML, _ := linkSel.Parent().Parent().Parent().Html()

		fmt.Fprintf(os.Stderr, "\n[DEBUG] ========== Link Context ==========\n")
		fmt.Fprintf(os.Stderr, "[DEBUG] Link HTML: %s\n", linkHTML[:min2(200, len(linkHTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] Parent Level 1: %s\n", parent1HTML[:min2(800, len(parent1HTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] Parent Level 2: %s\n", parent2HTML[:min2(1500, len(parent2HTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] Parent Level 3 (FULL): %s\n", parent3HTML[:min2(5000, len(parent3HTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] =====================================\n\n")
	}

	// Strategy 1: Carbon Pulse top page article structure
	// The excerpt is a text node between <a class="thumbLink"> and <a class="readMore">
	articleContainer := linkSel.Parent().Parent().Parent()

	if os.Getenv("DEBUG_SCRAPING") != "" {
		classes, _ := articleContainer.Attr("class")
		fmt.Fprintf(os.Stderr, "[DEBUG] Article container classes: %s\n", classes)
		fmt.Fprintf(os.Stderr, "[DEBUG] Has 'post' class: %v\n", articleContainer.HasClass("post"))
	}

	// Check if this is a Carbon Pulse article container (has class "post")
	if articleContainer.HasClass("post") {
		// Get all text from the container
		fullText := articleContainer.Text()

		// Remove metadata section (Published ... / Last updated ... / Author ... / Categories ...)
		// Find "Read More" and take text before it
		readMoreIdx := strings.Index(fullText, "Read More")
		if readMoreIdx > 0 {
			fullText = fullText[:readMoreIdx]
		}

		// Split into lines and find the excerpt (typically after tags like "Carbon Pulse Premium")
		lines := strings.Split(fullText, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)

			// Skip empty lines, metadata, tags, and navigation
			if line == "" || len(line) < 30 {
				continue
			}
			if strings.Contains(line, "Published") ||
				strings.Contains(line, "Last updated") ||
				strings.Contains(line, "Carbon Pulse Premium") ||
				strings.Contains(line, "Nature & Biodiversity") ||
				strings.Contains(line, "Net Zero Pulse") ||
				strings.HasPrefix(line, "Top") {
				continue
			}

			// This should be the excerpt
			if len(line) > 30 && !strings.HasPrefix(line, "http") {
				excerpt.WriteString(line)

				// Also check next line if it's a continuation
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if len(nextLine) > 20 && !strings.Contains(nextLine, "Read More") {
						excerpt.WriteString(" ")
						excerpt.WriteString(nextLine)
					}
				}
				break
			}
		}
	}

	// Strategy 2: Fallback - Check for <p> tags (for other page structures)
	if excerpt.Len() == 0 {
		parent := linkSel.Parent()
		parent.Find("p:not(.metaStuff)").Each(func(i int, s *goquery.Selection) {
			if excerpt.Len() >= maxChars {
				return
			}
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				if excerpt.Len() > 0 {
					excerpt.WriteString(" ")
				}
				excerpt.WriteString(text)
			}
		})
	}

	// Strategy 3: Check for <div class="excerpt"> or similar
	if excerpt.Len() == 0 {
		linkSel.Parent().Parent().Find(".excerpt, .summary, .description").Each(func(i int, s *goquery.Selection) {
			if excerpt.Len() >= maxChars {
				return
			}
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				if excerpt.Len() > 0 {
					excerpt.WriteString(" ")
				}
				excerpt.WriteString(text)
			}
		})
	}

	result := strings.TrimSpace(excerpt.String())

	// Truncate if too long
	if len(result) > maxChars {
		result = result[:maxChars] + "..."
	}

	if os.Getenv("DEBUG_SCRAPING") != "" && result != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Extracted excerpt from context (%d chars)\n", len(result))
	}

	return result
}

// collectHeadlinesCarbonCreditsJP collects headlines from carboncredits.jp using WordPress REST API
func collectHeadlinesCarbonCreditsJP(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// WordPress REST API endpoint - get full content for free articles
	apiURL := fmt.Sprintf("https://carboncredits.jp/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch carboncredits.jp API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("carboncredits.jp API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
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

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		// Clean up HTML entities from title
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		// Clean up HTML from full content (free article)
		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "CarbonCredits.jp",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date, // WordPress API returns RFC3339 format
			Excerpt:     content, // Store full content in Excerpt field for free articles
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] CarbonCredits.jp: collected %d headlines\n", len(out))
	}

	return out, nil
}

// cleanHTMLTags removes HTML tags and decodes HTML entities
func cleanHTMLTags(htmlStr string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(htmlStr, "")
	// Decode HTML entities (including Japanese characters)
	text = html.UnescapeString(text)
	return text
}

// collectHeadlinesCarbonHerald collects headlines from carbonherald.com using WordPress REST API
func collectHeadlinesCarbonHerald(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	apiURL := fmt.Sprintf("https://carbonherald.com/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch carbonherald.com API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("carbonherald.com API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

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

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		// Clean up HTML entities from title
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		// Clean up HTML from full content (free article)
		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "Carbon Herald",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date, // WordPress API returns RFC3339 format
			Excerpt:     content, // Store full content in Excerpt field for free articles
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Carbon Herald: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesClimateHomeNews collects headlines from climatechangenews.com using WordPress REST API
func collectHeadlinesClimateHomeNews(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	apiURL := fmt.Sprintf("https://www.climatechangenews.com/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch climatechangenews.com API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("climatechangenews.com API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

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

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		// Clean up HTML entities from title
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		// Clean up HTML from full content (free article)
		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "Climate Home News",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date, // WordPress API returns RFC3339 format
			Excerpt:     content, // Store full content in Excerpt field for free articles
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Climate Home News: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesCarbonCreditscom collects headlines from carboncredits.com using WordPress REST API
func collectHeadlinesCarbonCreditscom(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	apiURL := fmt.Sprintf("https://carboncredits.com/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch carboncredits.com API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("carboncredits.com API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

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

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		// Clean up HTML entities from title
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		// Clean up HTML from full content (free article)
		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "CarbonCredits.com",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date, // WordPress API returns RFC3339 format
			Excerpt:     content, // Store full content in Excerpt field for free articles
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] CarbonCredits.com: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesSandbag fetches articles from Sandbag using WordPress REST API
func collectHeadlinesSandbag(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	apiURL := fmt.Sprintf("https://sandbag.be/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	type WPPost struct {
		Title   struct{ Rendered string `json:"rendered"` } `json:"title"`
		Link    string                                      `json:"link"`
		Date    string                                      `json:"date"`
		Content struct{ Rendered string `json:"rendered"` } `json:"content"`
	}

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)

		// Skip posts without proper title
		if title == "" {
			continue
		}

		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "Sandbag",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date,
			Excerpt:     content,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Sandbag: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesEcosystemMarketplace fetches articles from Ecosystem Marketplace using WordPress REST API
func collectHeadlinesEcosystemMarketplace(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	apiURL := fmt.Sprintf("https://www.ecosystemmarketplace.com/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	type WPPost struct {
		Title   struct{ Rendered string `json:"rendered"` } `json:"title"`
		Link    string                                      `json:"link"`
		Date    string                                      `json:"date"`
		Content struct{ Rendered string `json:"rendered"` } `json:"content"`
	}

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)

		// Skip posts without proper title
		if title == "" {
			continue
		}

		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "Ecosystem Marketplace",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date,
			Excerpt:     content,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Ecosystem Marketplace: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesCarbonBrief fetches articles from Carbon Brief using WordPress REST API
func collectHeadlinesCarbonBrief(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	apiURL := fmt.Sprintf("https://www.carbonbrief.org/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", limit)

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	type WPPost struct {
		Title   struct{ Rendered string `json:"rendered"` } `json:"title"`
		Link    string                                      `json:"link"`
		Date    string                                      `json:"date"`
		Content struct{ Rendered string `json:"rendered"` } `json:"content"`
	}

	var posts []WPPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)

		// Skip posts without proper title
		if title == "" {
			continue
		}

		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      "Carbon Brief",
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date,
			Excerpt:     content,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Carbon Brief: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesICAP fetches articles from ICAP (Drupal site) using HTML scraping
func collectHeadlinesICAP(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://icapcarbonaction.com/en/news"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	doc.Find("article.news-embed-grid").Each(func(_ int, article *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		// Extract title
		titleElem := article.Find("h3.content-title a.link-title span")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			return
		}

		// Extract URL
		linkElem := article.Find("a.link-title")
		href, exists := linkElem.Attr("href")
		if !exists || href == "" {
			return
		}
		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Extract date
		timeElem := article.Find("time")
		datetime, _ := timeElem.Attr("datetime")

		// Fetch full article content
		content := ""
		if articleURL != "" {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						bodyElem := articleDoc.Find("div.field-body")
						content = strings.TrimSpace(bodyElem.Text())
					}
				}
			}
		}

		out = append(out, Headline{
			Source:      "ICAP",
			Title:       title,
			URL:         articleURL,
			PublishedAt: datetime,
			Excerpt:     content,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no ICAP headlines found")
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ICAP: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIETA fetches articles from IETA using HTML scraping
func collectHeadlinesIETA(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	homeURL := "https://www.ieta.org/"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", homeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// Find news items - look for card-body containers
	doc.Find("div.card-body").Each(func(_ int, cardBody *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		// Extract title
		titleElem := cardBody.Find("h3.news-title")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			return
		}

		// Extract date (within the same card-body)
		dateElem := cardBody.Find("div.resource-date")
		dateStr := strings.TrimSpace(dateElem.Text())

		// Extract URL (sibling a.link-cover - need to go up to parent container)
		parent := cardBody.Parent()
		linkElem := parent.Find("a.link-cover")
		href, exists := linkElem.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(homeURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Parse date to RFC3339 format
		publishedAt := ""
		if dateStr != "" {
			// Try to parse "Dec 18, 2025" format
			t, err := time.Parse("Jan 2, 2006", dateStr)
			if err == nil {
				publishedAt = t.Format(time.RFC3339)
			}
		}

		// Fetch full article content
		content := ""
		if articleURL != "" {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						// Try common content selectors
						bodyElem := articleDoc.Find("article, .content, .post-content, .entry-content").First()
						content = strings.TrimSpace(bodyElem.Text())
					}
				}
			}
		}

		out = append(out, Headline{
			Source:      "IETA",
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAt,
			Excerpt:     content,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no IETA headlines found")
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IETA: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesEnergyMonitor fetches articles from Energy Monitor using HTML scraping
func collectHeadlinesEnergyMonitor(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.energymonitor.ai/news/"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// Find article items
	doc.Find("article").Each(func(_ int, article *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		// Extract title and URL from h3 > a
		linkElem := article.Find("h3 a")
		title := strings.TrimSpace(linkElem.Text())
		if title == "" {
			return
		}

		href, exists := linkElem.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Fetch full article content
		content := ""
		publishedAt := ""
		if articleURL != "" {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						// Try to find content
						bodyElem := articleDoc.Find("article .entry-content, article .article-content, .post-content, .content").First()
						content = strings.TrimSpace(bodyElem.Text())

						// Try to find published date
						timeElem := articleDoc.Find("time")
						datetime, exists := timeElem.Attr("datetime")
						if exists {
							publishedAt = datetime
						}
					}
				}
			}
		}

		out = append(out, Headline{
			Source:      "Energy Monitor",
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAt,
			Excerpt:     content,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no Energy Monitor headlines found")
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Energy Monitor: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesJRI は JRI（日本総合研究所）の RSSフィードから見出しを収集
//
// JRI は日本のシンクタンクで、カーボンニュートラルや気候変動に関する
// レポートを公開している。RSSフィードから記事を取得し、carbonKeywordsで
// カーボン関連記事をフィルタリング（現在はフィルタ無効化中）。
//
// 手法: RSS Feed (gofeed)
//
// 引数:
//   limit: 収集する最大記事数
//   cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//   収集した見出しのスライス、エラー
func collectHeadlinesJRI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	rssURL := "https://www.jri.co.jp/xml.jsp?id=12966"  // JRI の RSSフィードURL

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", rssURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// RSSフィードをパース（gofeedライブラリを使用）
	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RSS parse failed: %w", err)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in RSS feed")
	}

	// carbonKeywords: カーボン/気候変動関連記事のフィルタリング用キーワードリスト
	//
	// 日本語ソース（JRI、環境省、METI、Mizuho R&T）で使用。
	// 幅広いトピックをカバーするため、以下のカテゴリを含む：
	//   - カーボン関連: カーボン、炭素、脱炭素、カーボンニュートラル
	//   - 温室効果ガス: CO2、温室効果ガス、GHG
	//   - 市場/取引: 排出量取引、ETS、カーボンプライシング、カーボンクレジット
	//   - 気候変動: 気候変動、クライメート
	//   - 英語キーワード: carbon, climate（英語混在記事用）
	//
	// 注意: 現在はフィルタリング無効化中（全記事を収集）
	carbonKeywords := []string{
		"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
		"気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
		"カーボンクレジット", "クレジット市場", "carbon", "climate",
	}

	out := make([]Headline, 0, limit)
	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		// Check if title contains carbon-related keywords
		title := item.Title
		_ = carbonKeywords // unused for now
		// titleLower := strings.ToLower(title)
		// containsKeyword := false
		// for _, kw := range carbonKeywords {
		// 	if strings.Contains(titleLower, strings.ToLower(kw)) {
		// 		containsKeyword = true
		// 		break
		// 	}
		// }

		// For now, include all articles (filtering can be enabled later)
		// Uncomment to filter only carbon-related articles:
		// if !containsKeyword {
		// 	continue
		// }

		publishedAt := ""
		if item.PublishedParsed != nil {
			publishedAt = item.PublishedParsed.Format(time.RFC3339)
		}

		// Fetch full article content
		excerpt := ""
		if item.Link != "" {
			contentResp, err := client.Get(item.Link)
			if err == nil && contentResp.StatusCode == http.StatusOK {
				defer contentResp.Body.Close()
				contentDoc, err := goquery.NewDocumentFromReader(contentResp.Body)
				if err == nil {
					// Extract content from article page
					// JRI uses various selectors for article content
					contentDoc.Find("div.detail, div.content, div.main-content, article").Each(func(_ int, s *goquery.Selection) {
						if excerpt == "" {
							text := strings.TrimSpace(s.Text())
							if len(text) > 100 { // Only use if substantial content
								excerpt = text
							}
						}
					})
				}
			}
		}

		// If we couldn't get excerpt, use description from RSS
		if excerpt == "" && item.Description != "" {
			excerpt = cleanHTMLTags(item.Description)
		}

		out = append(out, Headline{
			Source:      "Japan Research Institute",
			Title:       title,
			URL:         item.Link,
			PublishedAt: publishedAt,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no JRI headlines found")
	}

	return out, nil
}

// collectHeadlinesEnvMinistry collects headlines from Japan Environment Ministry press releases
func collectHeadlinesEnvMinistry(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	pressURL := "https://www.env.go.jp/press/"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", pressURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Keywords for carbon/climate-related articles
	carbonKeywords := []string{
		"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
		"気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
		"カーボンクレジット", "クレジット市場", "JCM", "二国間クレジット",
		"カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
	}

	out := make([]Headline, 0, limit)
	currentDate := ""

	// Parse press releases
	doc.Find("span.p-press-release-list__heading, li.c-news-link__item").Each(func(i int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Check if this is a date heading
		if s.Is("span.p-press-release-list__heading") {
			dateText := strings.TrimSpace(s.Text())
			// Convert "2025年12月26日発表" to "2025-12-26"
			dateText = strings.Replace(dateText, "発表", "", 1)
			dateText = strings.TrimSpace(dateText)

			// Parse Japanese date format
			var year, month, day int
			if _, parseErr := fmt.Sscanf(dateText, "%d年%d月%d日", &year, &month, &day); parseErr == nil {
				currentDate = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			}
			return
		}

		// Process article items
		if !s.Is("li.c-news-link__item") {
			return
		}

		// Extract title and URL
		link := s.Find("a.c-news-link__link")
		title := strings.TrimSpace(link.Text())
		href, exists := link.Attr("href")
		if !exists || title == "" {
			return
		}

		// Check if title contains carbon-related keywords
		titleLower := strings.ToLower(title)
		containsKeyword := false
		for _, kw := range carbonKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				containsKeyword = true
				break
			}
		}

		if !containsKeyword {
			return
		}

		// Build absolute URL
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			articleURL = "https://www.env.go.jp" + href
		}

		// Fetch full article content
		excerpt := ""
		contentResp, err := client.Get(articleURL)
		if err == nil && contentResp.StatusCode == http.StatusOK {
			defer contentResp.Body.Close()
			contentDoc, err := goquery.NewDocumentFromReader(contentResp.Body)
			if err == nil {
				// Extract main content from article page
				contentDoc.Find("div.l-content, div.c-content, article, main").Each(func(_ int, cs *goquery.Selection) {
					if excerpt == "" {
						text := strings.TrimSpace(cs.Text())
						if len(text) > 100 {
							excerpt = text
						}
					}
				})
			}
		}

		// Format published date
		publishedAt := ""
		if currentDate != "" {
			publishedAt = currentDate + "T00:00:00+09:00"
		}

		out = append(out, Headline{
			Source:      "Japan Environment Ministry",
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAt,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no Environment Ministry headlines found")
	}

	return out, nil
}

// collectHeadlinesJPX collects headlines from Japan Exchange Group (JPX) via RSS
func collectHeadlinesJPX(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// Use JPX RSS feed
	feedURL := "https://www.jpx.co.jp/rss/jpx-news.xml"

	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: cfg.Timeout}

	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JPX RSS: %w", err)
	}

	// Keywords for carbon credit related articles
	carbonKeywords := []string{
		"カーボン", "炭素", "クレジット", "排出", "GX", "グリーン",
		"脱炭素", "CO2", "温室効果ガス", "取引", "市場", "環境",
	}

	out := make([]Headline, 0, limit)

	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		// Check if title or link contains carbon-related keywords
		titleLower := strings.ToLower(item.Title)
		linkLower := strings.ToLower(item.Link)
		containsKeyword := false
		for _, kw := range carbonKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) ||
			   strings.Contains(linkLower, "carbon") ||
			   strings.Contains(linkLower, "クレジット") {
				containsKeyword = true
				break
			}
		}

		if !containsKeyword {
			continue
		}

		// Parse date
		dateStr := time.Now().Format(time.RFC3339)
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// Get content/description
		excerpt := ""
		if item.Description != "" {
			excerpt = html.UnescapeString(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}
		if item.Content != "" && excerpt == "" {
			excerpt = html.UnescapeString(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		}

		out = append(out, Headline{
			Source:      "Japan Exchange Group (JPX)",
			Title:       item.Title,
			URL:         item.Link,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no JPX carbon-related headlines found")
	}

	return out, nil
}

// collectHeadlinesMETI collects headlines from Japan Ministry of Economy, Trade and Industry via RSS
func collectHeadlinesMETI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// Use METI Small and Medium Enterprise Agency RSS feed (verified working)
	feedURL := "https://www.chusho.meti.go.jp/rss/index.xml"

	// Create parser with extended timeout
	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: 60 * time.Second}

	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch METI RSS: %w", err)
	}

	// Keywords for carbon/GX-related articles
	carbonKeywords := []string{
		"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
		"気候変動", "排出量取引", "ETS", "カーボンプライシング",
		"カーボンクレジット", "クレジット", "GX", "グリーントランスフォーメーション",
		"カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
		"水素", "アンモニア", "CCUS", "CCS", "省エネ", "再エネ",
	}

	out := make([]Headline, 0, limit)

	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		// Temporarily collect all articles for testing (keyword filtering disabled)
		// TODO: Re-enable keyword filtering when carbon-related content is available
		_ = carbonKeywords // Avoid unused variable warning

		// Parse date
		dateStr := time.Now().Format(time.RFC3339)
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// Get content/description
		excerpt := ""
		if item.Description != "" {
			excerpt = html.UnescapeString(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}
		if item.Content != "" && excerpt == "" {
			excerpt = html.UnescapeString(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		}

		out = append(out, Headline{
			Source:      "Japan Ministry of Economy (METI)",
			Title:       item.Title,
			URL:         item.Link,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	// Return empty slice if no carbon-related articles found (not an error)
	// METI/SME Agency feed is working but may not always have carbon-specific content
	return out, nil
}

// collectHeadlinesWorldBank collects headlines from World Bank Climate Change publications
func collectHeadlinesWorldBank(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.worldbank.org/en/topic/climatechange"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Keywords for carbon pricing related content
	carbonKeywords := []string{
		"carbon pricing", "carbon tax", "carbon credit", "emissions trading",
		"cap and trade", "carbon market", "climate finance", "carbon border",
		"CBAM", "ETS", "carbon levy", "green bonds", "climate bonds",
	}

	out := make([]Headline, 0, limit)

	// Parse articles (World Bank format)
	// Look for articles in featured and research sections
	doc.Find("div.featured h3 a, div.research h3 a, div[class*='lp__'] h3 a").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		title := strings.TrimSpace(link.Text())
		href, exists := link.Attr("href")
		if !exists || title == "" {
			return
		}

		// Check if title contains carbon-related keywords
		titleLower := strings.ToLower(title)
		containsKeyword := false
		for _, kw := range carbonKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				containsKeyword = true
				break
			}
		}

		if !containsKeyword {
			return
		}

		// Build absolute URL
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			articleURL = "https://www.worldbank.org" + href
		}

		// Extract date if available
		dateStr := time.Now().Format(time.RFC3339)
		// Try to find date in parent elements
		parent := link.Parent().Parent()
		dateElem := parent.Find("time, span.date, div.date")
		if dateElem.Length() > 0 {
			dateText := strings.TrimSpace(dateElem.Text())
			if dateAttr, exists := dateElem.Attr("datetime"); exists {
				dateStr = dateAttr
			} else if dateText != "" {
				// Try to parse common date formats
				if t, err := time.Parse("January 2, 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				} else if t, err := time.Parse("Jan 2, 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				}
			}
		}

		// Extract excerpt from parent element
		excerpt := ""
		excerptElem := parent.Find("p, div.description, div.summary")
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.First().Text())
		}

		out = append(out, Headline{
			Source:      "World Bank",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no World Bank headlines found")
	}

	return out, nil
}

// collectHeadlinesCarbonMarketWatch collects headlines from Carbon Market Watch
func collectHeadlinesCarbonMarketWatch(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://carbonmarketwatch.org/publications/"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	out := make([]Headline, 0, limit)

	// Parse publications/articles
	doc.Find("article, div.post, div.entry, div.publication").Each(func(i int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		link := s.Find("a[href*='/publications/'], a.entry-title, h2 a, h3 a").First()
		if link.Length() == 0 {
			link = s.Find("a").First()
		}

		title := strings.TrimSpace(link.Text())
		if title == "" {
			titleElem := s.Find("h2, h3, h4, .title, .entry-title")
			title = strings.TrimSpace(titleElem.Text())
		}

		href, exists := link.Attr("href")
		if !exists || title == "" || len(title) < 10 {
			return
		}

		// Build absolute URL
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			articleURL = "https://carbonmarketwatch.org" + href
		}

		// Extract date
		dateStr := time.Now().Format(time.RFC3339)
		dateElem := s.Find("time, .date, .published")
		if dateElem.Length() > 0 {
			if dateAttr, exists := dateElem.Attr("datetime"); exists {
				dateStr = dateAttr
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				if t, err := time.Parse("January 2, 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				} else if t, err := time.Parse("2 January 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				}
			}
		}

		// Extract excerpt
		excerpt := ""
		excerptElem := s.Find("p, .excerpt, .summary, .entry-summary")
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.First().Text())
		}

		out = append(out, Headline{
			Source:      "Carbon Market Watch",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no Carbon Market Watch headlines found")
	}

	return out, nil
}

// collectHeadlinesNewClimate collects headlines from NewClimate Institute
func collectHeadlinesNewClimate(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://newclimate.org/news"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	out := make([]Headline, 0, limit)

	// Parse news items
	doc.Find("a[href^='/news/'], a[href^='/resources/publications/']").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists {
			return
		}

		// Get title - may be in the link text or in a child element
		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		// Build absolute URL
		articleURL := "https://newclimate.org" + href

		// Try to extract date from parent or sibling elements
		dateStr := time.Now().Format(time.RFC3339)
		parent := link.Parent().Parent()
		dateElem := parent.Find("time, .date, span[class*='date']")
		if dateElem.Length() > 0 {
			if dateAttr, exists := dateElem.Attr("datetime"); exists {
				dateStr = dateAttr
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				if t, err := time.Parse("2 January 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				} else if t, err := time.Parse("January 2, 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				}
			}
		}

		// Extract excerpt
		excerpt := ""
		excerptElem := parent.Find("p, .description, .summary")
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.First().Text())
		}

		out = append(out, Headline{
			Source:      "NewClimate Institute",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no NewClimate headlines found")
	}

	return out, nil
}

// collectHeadlinesCarbonKnowledgeHub collects headlines from Carbon Knowledge Hub
func collectHeadlinesCarbonKnowledgeHub(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.carbonknowledgehub.com"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool) // Track seen URLs to avoid duplicates

	// Primary selector: links with css-oxwq25 class (main article links)
	doc.Find("a.css-oxwq25, a[class*='css-']").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// Filter for content URLs
		// The site uses both singular and plural forms
		// We need actual article URLs, not category pages, so check for more than one slash
		isContentURL := (strings.Contains(href, "/factsheet") ||
			strings.Contains(href, "/story") ||
			strings.Contains(href, "/stories") ||
			strings.Contains(href, "/audio") ||
			strings.Contains(href, "/media") ||
			strings.Contains(href, "/news")) &&
			strings.Count(href, "/") > 1 // Ensure it's not just the category page

		if !isContentURL {
			return
		}

		// Build absolute URL
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			if strings.HasPrefix(href, "/") {
				articleURL = "https://www.carbonknowledgehub.com" + href
			} else {
				articleURL = "https://www.carbonknowledgehub.com/" + href
			}
		}

		// Skip if already seen
		if seen[articleURL] {
			return
		}

		// Get title
		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		// Skip common navigation text
		titleLower := strings.ToLower(title)
		if strings.Contains(titleLower, "read more") ||
		   strings.Contains(titleLower, "learn more") ||
		   strings.Contains(titleLower, "click here") ||
		   strings.Contains(titleLower, "view all") {
			return
		}

		// Extract date from parent container
		dateStr := time.Now().Format(time.RFC3339)
		container := link.ParentsFiltered("[class*='css-']").First()
		if container.Length() > 0 {
			// Look for date element with css-1fr5xea or similar classes
			dateElem := container.Find("[class*='css-1fr'], time, .date, [class*='date']")
			if dateElem.Length() > 0 {
				dateText := strings.TrimSpace(dateElem.First().Text())
				// Parse "14 Nov 2025" or similar formats
				for _, format := range []string{"2 Jan 2006", "_2 Jan 2006", "Jan 2, 2006", "2006-01-02"} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						break
					}
				}
			}
		}

		// Extract category/type
		category := ""
		if container.Length() > 0 {
			typeElem := container.Find("[class*='css-3aw'], .type, .category, [class*='tag']")
			if typeElem.Length() > 0 {
				category = strings.TrimSpace(typeElem.First().Text())
			}
		}

		// Build excerpt
		excerpt := ""
		if category != "" {
			excerpt = "Type: " + category
		}

		// Determine content type from URL
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
		case strings.Contains(href, "/data-tracker/"):
			contentType = "Data Tracker"
		}
		if contentType != "" && excerpt == "" {
			excerpt = "Type: " + contentType
		}

		seen[articleURL] = true

		out = append(out, Headline{
			Source:      "Carbon Knowledge Hub",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no Carbon Knowledge Hub headlines found")
	}

	return out, nil
}

// collectHeadlinesPwCJapan は PwC Japan のサステナビリティページから見出しを収集
//
// PwC Japanは最も複雑なスクレイピング実装の1つ。ページ内のJavaScriptから
// 3重エスケープされたJSONデータを抽出し、複数回のアンエスケープ処理を経て
// 記事情報をパースする。
//
// 実装の特殊性:
//   1. angular.loadFacetedNavigationスクリプトから埋め込みJSONを抽出
//   2. 16進エスケープされた引用符（\x22）をアンエスケープ
//   3. elements配列が3重エスケープされているため、2回のアンエスケープ処理
//   4. 正規表現で個別の記事オブジェクトを抽出
//   5. 日付フォーマット: YYYY-MM-DD
//
// 手法: HTML Scraping + 複雑なJSON抽出
//
// 引数:
//   limit: 収集する最大記事数
//   cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//   収集した見出しのスライス、エラー
func collectHeadlinesPwCJapan(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.pwc.com/jp/ja/knowledge/column/sustainability.html"

	client := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Follow redirects
		},
	}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
	// Do not set Accept-Encoding to receive uncompressed response
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Read the entire HTML content
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	bodyStr := string(bodyBytes)

	// Extract JSON data from angular.loadFacetedNavigation script
	// Pattern: angular.loadFacetedNavigation(..., "{...}")
	// The JSON object contains numberHits, elements, selectedTags, filterTags
	jsonPattern := regexp.MustCompile(`"(\{\\x22numberHits\\x22:\d+,\\x22elements\\x22:.*?\\x22filterTags\\x22:.*?\})"`)
	matches := jsonPattern.FindAllStringSubmatch(bodyStr, -1)

	out := make([]Headline, 0, limit)

	for _, match := range matches {
		if len(out) >= limit {
			break
		}

		if len(match) < 2 {
			continue
		}

		jsonStr := match[1]

		// Unescape hex-encoded quotes (\x22 -> ")
		jsonStr = strings.ReplaceAll(jsonStr, `\x22`, `"`)
		// Unescape other common escapes
		jsonStr = strings.ReplaceAll(jsonStr, `\/`, `/`)
		jsonStr = strings.ReplaceAll(jsonStr, `\u002D`, `-`)

		// Extract the elements field (it's a string-encoded JSON array)
		elementsPattern := regexp.MustCompile(`"elements":"(\[.*?\])(?:",|"}|"$)`)
		elementsMatch := elementsPattern.FindStringSubmatch(jsonStr)
		if len(elementsMatch) < 2 {
			continue
		}

		elementsStr := elementsMatch[1]

		// Unescape the triple-escaped elements array (needs to be done twice)
		for i := 0; i < 2; i++ {
			// Replace \\ with temporary placeholder
			elementsStr = strings.ReplaceAll(elementsStr, `\\`, "\x00")
			// Replace \" with "
			elementsStr = strings.ReplaceAll(elementsStr, `\"`, `"`)
			// Restore single backslash
			elementsStr = strings.ReplaceAll(elementsStr, "\x00", `\`)
		}

		// Parse individual article objects
		// Look for title, href, and publishDate fields
		titlePattern := regexp.MustCompile(`"title":"([^"]+)"`)
		hrefPattern := regexp.MustCompile(`"href":"([^"]+)"`)
		datePattern := regexp.MustCompile(`"publishDate":"([^"]*)"`)

		// Split by article objects (each starts with {"index":)
		articles := strings.Split(elementsStr, `{"index":`)

		for _, articleStr := range articles {
			if len(out) >= limit {
				break
			}

			if len(articleStr) < 50 {
				continue
			}

			// Extract title
			titleMatches := titlePattern.FindStringSubmatch(articleStr)
			if len(titleMatches) < 2 {
				continue
			}
			title := titleMatches[1]

			// Extract URL
			hrefMatches := hrefPattern.FindStringSubmatch(articleStr)
			if len(hrefMatches) < 2 {
				continue
			}
			url := hrefMatches[1]

			// Extract date
			dateStr := ""
			dateMatches := datePattern.FindStringSubmatch(articleStr)
			if len(dateMatches) >= 2 {
				dateStr = dateMatches[1]
			}

			// Build absolute URL
			articleURL := url
			if strings.HasPrefix(url, "/") {
				articleURL = "https://www.pwc.com" + url
			} else if !strings.HasPrefix(url, "http") {
				// Sometimes URLs come without leading slash
				continue
			}

			// Parse date (format: "YYYY-MM-DD")
			publishedAt := time.Now().Format(time.RFC3339)
			if dateStr != "" {
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					publishedAt = t.Format(time.RFC3339)
				}
			}

			out = append(out, Headline{
				Source:      "PwC Japan",
				Title:       title,
				URL:         articleURL,
				PublishedAt: publishedAt,
				Excerpt:     "",
				IsHeadline:  true,
			})
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no PwC Japan sustainability-related articles found in JSON data")
	}

	return out, nil
}

// collectHeadlinesMizuhoRT collects headlines from Mizuho Research & Technologies
func collectHeadlinesMizuhoRT(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// Use the 2025 publication page which lists recent reports
	newsURL := "https://www.mizuho-rt.co.jp/publication/2025/index.html"

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Keywords for carbon/GX-related reports
	carbonKeywords := []string{
		"カーボン", "脱炭素", "サステナビリティ", "GX", "カーボンニュートラル",
		"気候変動", "温室効果ガス", "CO2", "排出量", "GHG", "クレジット",
		"カーボンプライシング", "ETS", "排出量取引", "CSRD", "スコープ3",
		"再生可能エネルギー", "グリーン", "環境", "COP", "パリ協定",
		"carbon", "decarboniz", "sustainability", "climate", "emission",
	}

	out := make([]Headline, 0, limit)
	datePattern := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)

	// Extract articles from links
	doc.Find("a[href*='/business/'], a[href*='/publication/']").Each(func(i int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		title := strings.TrimSpace(s.Text())
		if title == "" {
			return
		}

		// Check if title contains carbon/sustainability keywords
		titleLower := strings.ToLower(title)
		containsKeyword := false
		for _, kw := range carbonKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				containsKeyword = true
				break
			}
		}

		if !containsKeyword {
			return
		}

		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Build absolute URL
		articleURL := href
		if strings.HasPrefix(href, "/") {
			articleURL = "https://www.mizuho-rt.co.jp" + href
		}

		// Extract date from surrounding text
		dateStr := time.Now().Format(time.RFC3339)
		parent := s.Parent()
		if parent != nil {
			parentText := parent.Text()
			if matches := datePattern.FindStringSubmatch(parentText); len(matches) == 4 {
				year := matches[1]
				month := matches[2]
				day := matches[3]
				if len(month) == 1 {
					month = "0" + month
				}
				if len(day) == 1 {
					day = "0" + day
				}
				dateStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day)
			}
		}

		out = append(out, Headline{
			Source:      "Mizuho Research & Technologies",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     "",
			IsHeadline:  true,
		})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("no Mizuho RT sustainability-related reports found")
	}

	return out, nil
}
