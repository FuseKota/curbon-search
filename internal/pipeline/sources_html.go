// =============================================================================
// sources_html.go - HTMLスクレイピングソース
// =============================================================================
//
// このファイルはHTMLスクレイピングを使用するニュースソースを定義します。
// goquery ライブラリを使用してHTML構造から記事情報を抽出します。
//
// 【含まれるソース】
//   1. ICAP               - 国際カーボンアクションパートナーシップ
//   2. IETA               - 国際排出量取引協会
//   3. Energy Monitor     - エネルギー転換ニュース
//   4. World Bank         - 世界銀行気候変動
//   5. Carbon Market Watch - NGO監視団体（一時無効化中）
//   6. NewClimate         - 気候研究機関
//   7. Carbon Knowledge Hub - 教育プラットフォーム
//
// =============================================================================
package pipeline

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// collectHeadlinesICAP fetches articles from ICAP (Drupal site) using HTML scraping
func collectHeadlinesICAP(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ICAP: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIETA fetches articles from IETA using HTML scraping
func collectHeadlinesIETA(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IETA: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesEnergyMonitor fetches articles from Energy Monitor using HTML scraping
func collectHeadlinesEnergyMonitor(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

						// Try to find published date from JSON-LD structured data
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if publishedAt != "" {
								return
							}
							jsonText := script.Text()
							// Extract datePublished from JSON-LD
							re := regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
							if matches := re.FindStringSubmatch(jsonText); len(matches) > 1 {
								publishedAt = matches[1]
							}
						})

						// Fallback: try time element
						if publishedAt == "" {
							timeElem := articleDoc.Find("time")
							datetime, exists := timeElem.Attr("datetime")
							if exists {
								publishedAt = datetime
							}
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

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Energy Monitor: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesWorldBank collects headlines from World Bank Climate Change publications
func collectHeadlinesWorldBank(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	// Return empty slice if no articles found (not an error)
	return out, nil
}

// collectHeadlinesCarbonMarketWatch collects headlines from Carbon Market Watch
// NOTE: 2026-01: Currently disabled due to 403 Forbidden errors
func collectHeadlinesCarbonMarketWatch(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	// Return empty slice if no articles found (not an error)
	return out, nil
}

// collectHeadlinesNewClimate collects headlines from NewClimate Institute
func collectHeadlinesNewClimate(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	// Return empty slice if no articles found (not an error)
	return out, nil
}

// collectHeadlinesCarbonKnowledgeHub collects headlines from Carbon Knowledge Hub
func collectHeadlinesCarbonKnowledgeHub(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	// Return empty slice if no articles found (not an error)
	return out, nil
}
