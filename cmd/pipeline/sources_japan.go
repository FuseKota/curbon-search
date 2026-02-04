// =============================================================================
// sources_japan.go - 日本語ソース
// =============================================================================
//
// このファイルは日本語のニュースソースからの記事収集関数を定義します。
// RSS、HTMLスクレイピング、複雑なJSON抽出など様々な手法を使用します。
//
// 【含まれるソース】
//   1. JRI（日本総研）    - RSSフィード
//   2. 環境省             - プレスリリース（HTMLスクレイピング）
//   3. JPX（日本取引所）  - RSSフィード
//   4. METI Shingikai     - 審議会リスト（HTMLスクレイピング）
//   5. PwC Japan          - 複雑なJSON抽出
//   6. Mizuho R&T         - HTMLスクレイピング
//
// =============================================================================
package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// carbonKeywordsJapan: カーボン/気候変動関連記事のフィルタリング用キーワードリスト
//
// 日本語ソース（JRI、環境省、METI、Mizuho R&T）で使用。
// 幅広いトピックをカバーするため、以下のカテゴリを含む：
//   - カーボン関連: カーボン、炭素、脱炭素、カーボンニュートラル
//   - 温室効果ガス: CO2、温室効果ガス、GHG
//   - 市場/取引: 排出量取引、ETS、カーボンプライシング、カーボンクレジット
//   - 気候変動: 気候変動、クライメート
//   - 英語キーワード: carbon, climate（英語混在記事用）
var carbonKeywordsJapan = []string{
	"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
	"気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
	"カーボンクレジット", "クレジット市場", "carbon", "climate",
	"JCM", "二国間クレジット", "カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
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
//
//	limit: 収集する最大記事数
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
func collectHeadlinesJRI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	rssURL := "https://www.jri.co.jp/xml.jsp?id=12966" // JRI の RSSフィードURL

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

	out := make([]Headline, 0, limit)
	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		// Check if title contains carbon-related keywords
		title := item.Title
		_ = carbonKeywordsJapan // unused for now - filtering disabled
		// titleLower := strings.ToLower(title)
		// containsKeyword := false
		// for _, kw := range carbonKeywordsJapan {
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

	// Return empty slice if no articles found (not an error)
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

	// Return empty slice if no articles found (not an error)
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

	// Return empty slice if no carbon-related articles found (not an error)
	// JPX feed is working but may not always have carbon-specific content
	return out, nil
}

// collectHeadlinesMETI collects headlines from METI Shingikai (Council/Committee) list page
//
// This function fetches the METI shingikai index page and extracts energy/carbon-related
// council meetings using a two-stage fetch approach similar to collectHeadlinesEnvMinistry().
//
// HTML structure:
// - METI uses <dl class="date_sp"> with <dd> elements containing article links
// - Date appears alongside each entry in Japanese format (YYYY年MM月DD日)
//
// Filter logic:
// - URL path filter: /shingikai/enecho/ (Agency for Natural Resources and Energy) or
//   /shingikai/sankoshin/ (Industrial Structure Council including GX-related subcommittees)
// - Keyword filter: energy, power, gas, carbon, decarbonization, GX, hydrogen, etc.
// - If URL path matches -> collect (even without keyword match)
// - If keyword matches -> collect (even without URL path match)
//
// URL: https://www.meti.go.jp/shingikai/index.html
func collectHeadlinesMETI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	baseURL := "https://www.meti.go.jp"
	indexURL := baseURL + "/shingikai/index.html"

	// Use longer timeout for METI (government site can be slow)
	timeout := cfg.Timeout
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	// Use standard browser User-Agent (METI may block custom agents)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

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

	// URL path filters (energy-related departments)
	energyPaths := []string{
		"/shingikai/enecho/",    // Agency for Natural Resources and Energy
		"/shingikai/sankoshin/", // Industrial Structure Council (GX-related)
	}

	// Keyword filters for energy/carbon-related content
	energyKeywords := []string{
		"エネルギー", "電力", "ガス", "資源", "燃料",
		"カーボン", "脱炭素", "GX", "グリーン",
		"水素", "アンモニア", "原子力", "再生可能",
		"排出", "温暖化", "気候", "蓄電", "電池",
	}

	// Date regex for Japanese date format
	dateRe := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)

	out := make([]Headline, 0, limit)

	// Find all dd elements containing shingikai links (METI uses dl > dd structure for updates)
	doc.Find("dd").Each(func(i int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		link := s.Find("a[href*='/shingikai/']").First()
		if link.Length() == 0 {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip index pages
		if strings.Contains(href, "index") {
			return
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 5 {
			return
		}

		// Check URL path filter
		isEnergyPath := false
		for _, path := range energyPaths {
			if strings.Contains(href, path) {
				isEnergyPath = true
				break
			}
		}

		// Check keyword filter
		hasKeyword := false
		titleLower := strings.ToLower(title)
		for _, kw := range energyKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				hasKeyword = true
				break
			}
		}

		// Apply filter logic:
		// - Path match -> collect (regardless of keyword)
		// - Keyword match -> collect (regardless of path)
		if !isEnergyPath && !hasKeyword {
			return
		}

		// Build absolute URL
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			articleURL = baseURL + href
		}

		// Extract date from li text
		liText := s.Text()
		dateStr := ""
		if dateMatch := dateRe.FindStringSubmatch(liText); dateMatch != nil {
			year := dateMatch[1]
			month := fmt.Sprintf("%02d", atoi(dateMatch[2]))
			day := fmt.Sprintf("%02d", atoi(dateMatch[3]))
			dateStr = fmt.Sprintf("%s-%s-%sT00:00:00+09:00", year, month, day)
		}

		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] METI Shingikai: %s (path=%v, keyword=%v)\n", title[:min(50, len(title))], isEnergyPath, hasKeyword)
		}

		// Fetch article page for excerpt (2nd stage fetch)
		excerpt := fetchMETIArticleExcerpt(client, articleURL, cfg.UserAgent)

		out = append(out, Headline{
			Source:      "METI Shingikai",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	return out, nil
}

// fetchMETIArticleExcerpt fetches the article page and extracts text content
// Returns empty string if the page only contains PDFs or fetch fails
func fetchMETIArticleExcerpt(client *http.Client, url string, userAgent string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	// Use standard browser User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	// Try to find main content area
	var excerpt string

	// Common content selectors for METI pages
	contentSelectors := []string{
		"#main_content",
		".contents",
		"#contents",
		"main",
		"article",
	}

	for _, sel := range contentSelectors {
		content := doc.Find(sel)
		if content.Length() > 0 {
			excerpt = strings.TrimSpace(content.Text())
			break
		}
	}

	// Fallback to body if no content area found
	if excerpt == "" {
		excerpt = strings.TrimSpace(doc.Find("body").Text())
	}

	// Clean up whitespace
	excerpt = regexp.MustCompile(`\s+`).ReplaceAllString(excerpt, " ")
	excerpt = strings.TrimSpace(excerpt)

	// Truncate to 2000 characters
	if len(excerpt) > 2000 {
		excerpt = excerpt[:2000]
	}

	return excerpt
}

// atoi converts string to int, returns 0 on error
func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// collectHeadlinesPwCJapan は PwC Japan のサステナビリティページから見出しを収集
//
// PwC Japanは最も複雑なスクレイピング実装の1つ。ページ内のJavaScriptから
// 3重エスケープされたJSONデータを抽出し、複数回のアンエスケープ処理を経て
// 記事情報をパースする。
//
// 実装の特殊性:
//  1. angular.loadFacetedNavigationスクリプトから埋め込みJSONを抽出
//  2. 16進エスケープされた引用符（\x22）をアンエスケープ
//  3. elements配列が3重エスケープされているため、2回のアンエスケープ処理
//  4. 正規表現で個別の記事オブジェクトを抽出
//  5. 日付フォーマット: YYYY-MM-DD
//
// 手法: HTML Scraping + 複雑なJSON抽出
//
// 引数:
//
//	limit: 収集する最大記事数
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
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

	// Return empty slice if no articles found (not an error)
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

	// Return empty slice if no articles found (not an error)
	return out, nil
}
