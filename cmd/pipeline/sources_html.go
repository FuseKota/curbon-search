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
//   8. Verra              - VCS規格運営団体
//   9. Gold Standard      - 高品質カーボンクレジット規格
//  10. ACR                - American Carbon Registry
//  11. CAR                - Climate Action Reserve
//  12. UNFCCC             - 国連気候変動枠組条約
//  13. IISD ENB           - 環境交渉速報
//  14. Climate Focus      - 気候政策コンサルティング
//  15. Puro.earth         - 炭素除去認証プラットフォーム
//  16. Isometric          - 炭素除去検証
//
// =============================================================================
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// Package-level compiled regex for performance (avoid recompiling in loops)
var reDatePublished = regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)

// collectHeadlinesICAP fetches articles from ICAP (Drupal site) using HTML scraping
func collectHeadlinesICAP(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://icapcarbonaction.com/en/news"

	client := cfg.Client
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
				if err == nil {
					if articleResp.StatusCode == http.StatusOK {
						articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
						if err == nil {
							var parts []string
							articleDoc.Find(".paragraph--type--text").Each(func(_ int, s *goquery.Selection) {
								t := strings.TrimSpace(s.Text())
								if t != "" {
									parts = append(parts, t)
								}
							})
							content = strings.Join(parts, "\n\n")
						}
					}
					articleResp.Body.Close() // Always close body when err == nil
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
func collectHeadlinesIETA(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	homeURL := "https://www.ieta.org/"

	client := cfg.Client
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
				if err == nil {
					if articleResp.StatusCode == http.StatusOK {
						articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
						if err == nil {
							// Extract intro + body text from news detail sections
							var parts []string
							articleDoc.Find(".section-news-detail .intro, .section-news-detail section.bg-white").Each(func(_ int, s *goquery.Selection) {
								t := strings.TrimSpace(s.Text())
								if t != "" {
									parts = append(parts, t)
								}
							})
							content = strings.Join(parts, "\n\n")
						}
					}
					articleResp.Body.Close() // Always close body when err == nil
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
func collectHeadlinesEnergyMonitor(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.energymonitor.ai/news/"

	client := cfg.Client
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
				if err == nil {
					if articleResp.StatusCode == http.StatusOK {
						articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
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
								if matches := reDatePublished.FindStringSubmatch(jsonText); len(matches) > 1 {
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
					articleResp.Body.Close() // Always close body when err == nil
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
func collectHeadlinesWorldBank(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// World Bank News Search APIでcarbon関連記事のURL・日付を取得し、
	// 各ページをスクレイピングしてタイトル・本文を取得する
	apiURL := fmt.Sprintf(
		"https://search.worldbank.org/api/v2/news?format=json&qterm=%%22carbon+pricing%%22+OR+%%22carbon+market%%22+OR+%%22carbon+credit%%22+OR+%%22emissions+trading%%22&rows=%d&os=0&srt=lnchdt&order=desc&fl=url,lnchdt,title,descr&lang_exact=English",
		limit,
	)

	var result struct {
		Documents map[string]json.RawMessage `json:"documents"`
	}
	if err := httpGetJSON(apiURL, cfg, &result); err != nil {
		return nil, fmt.Errorf("failed to fetch World Bank API: %w", err)
	}

	out := make([]Headline, 0, limit)
	for _, raw := range result.Documents {
		if len(out) >= limit {
			break
		}
		var doc struct {
			URL    string `json:"url"`
			Lnchdt string `json:"lnchdt"`
			Title  struct {
				Cdata string `json:"cdata!"`
			} `json:"title"`
			Descr struct {
				Cdata string `json:"cdata!"`
			} `json:"descr"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil || doc.URL == "" {
			continue
		}

		// HTTPをHTTPSに変換
		articleURL := strings.Replace(doc.URL, "http://", "https://", 1)

		dateStr := doc.Lnchdt // すでにRFC3339形式

		// APIにタイトルがあればそれを使用、なければページから取得
		title := doc.Title.Cdata
		excerpt := doc.Descr.Cdata

		// ページをスクレイピングしてタイトル・本文を補完
		pageDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			// タイトルが空の場合、ページから取得
			if title == "" {
				h1 := pageDoc.Find("h1").First()
				if h1.Length() > 0 {
					title = strings.TrimSpace(h1.Text())
				}
			}
			// 本文を<p>タグから取得
			var parts []string
			pageDoc.Find("p").Each(func(_ int, s *goquery.Selection) {
				text := strings.TrimSpace(s.Text())
				if len(text) > 50 {
					parts = append(parts, text)
				}
			})
			if bodyText := strings.Join(parts, "\n\n"); len(bodyText) > len(excerpt) {
				excerpt = bodyText
			}
		}

		if title == "" {
			continue
		}

		out = append(out, Headline{
			Source:      "World Bank",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}

// collectHeadlinesCarbonMarketWatch collects headlines from Carbon Market Watch
// NOTE: 2026-01: Currently disabled due to 403 Forbidden errors
func collectHeadlinesCarbonMarketWatch(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://carbonmarketwatch.org/publications/"

	client := cfg.Client
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

		// Extract date (empty string if not found)
		dateStr := ""
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
func collectHeadlinesNewClimate(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://newclimate.org/news"

	client := cfg.Client
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

		// Try to extract date from parent or sibling elements (empty string if not found)
		dateStr := ""
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

		// Fetch date and content from article page
		excerpt := ""
		articleDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			// Extract date from event-details calendar
			if dateStr == "" {
				articleDoc.Find(".event-details__name--calendar").Each(func(_ int, s *goquery.Selection) {
					if dateStr != "" {
						return
					}
					valElem := s.Parent().Find(".event-details__value")
					dateText := strings.TrimSpace(valElem.Text())
					if t, err := time.Parse("02 Jan 2006", dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
					}
				})
			}
			// Extract content from node__content
			nodeContent := articleDoc.Find(".node__content")
			if nodeContent.Length() > 0 {
				excerpt = strings.TrimSpace(nodeContent.Text())
			}
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
func collectHeadlinesCarbonKnowledgeHub(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.carbonknowledgehub.com"

	client := cfg.Client
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

		seen[articleURL] = true

		// 各記事ページから日付・本文を取得（Next.js SSR + __NEXT_DATA__）
		dateStr := ""
		excerpt := ""
		articleDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			// __NEXT_DATA__ JSONからfrontMatterを取得
			articleDoc.Find("script#__NEXT_DATA__").Each(func(_ int, s *goquery.Selection) {
				var nextData struct {
					Props struct {
						PageProps struct {
							Source struct {
								Frontmatter struct {
									Date        string `json:"date"`
									Description string `json:"description"`
								} `json:"frontmatter"`
							} `json:"source"`
						} `json:"pageProps"`
					} `json:"props"`
				}
				if err := json.Unmarshal([]byte(s.Text()), &nextData); err == nil {
					fm := nextData.Props.PageProps.Source.Frontmatter
					if fm.Date != "" {
						if t, err := time.Parse("2006-01-02", fm.Date); err == nil {
							dateStr = t.Format(time.RFC3339)
						}
					}
					if fm.Description != "" {
						excerpt = fm.Description
					}
				}
			})
			// SSRプリレンダリングされた本文からテキストを補完
			// Next.jsアプリのため<main>タグはなく、div#__nextにコンテンツがある
			mainContent := articleDoc.Find("div#__next")
			if mainContent.Length() > 0 {
				mainContent.Find("script, style, nav, header, footer, noscript").Remove()
				bodyText := strings.TrimSpace(mainContent.Text())
				if len(bodyText) > len(excerpt) {
					lines := strings.Split(bodyText, "\n")
					var cleaned []string
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line != "" {
							cleaned = append(cleaned, line)
						}
					}
					// 1行目はパンくずリスト・タイトル・メタ情報が結合しているのでスキップ
					if len(cleaned) > 1 {
						cleaned = cleaned[1:]
					}
					excerpt = strings.Join(cleaned, "\n")
				}
			}
		}

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

// =============================================================================
// VCM Certification Bodies
// =============================================================================

// collectHeadlinesVerra fetches news from Verra (VCS standard operator)
//
// Verra manages the Verified Carbon Standard (VCS), the world's most widely
// used voluntary GHG program.
func collectHeadlinesVerra(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://verra.org/news/feed/"

	client := cfg.Client
	req, err := http.NewRequest("GET", feedURL, nil)
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

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RSS parse failed: %w", err)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Verra RSS feed")
	}

	out := make([]Headline, 0, limit)

	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}

		articleURL := item.Link

		// Parse date (empty string if not available)
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// Get content from RSS
		excerpt := ""
		if item.Content != "" {
			excerpt = cleanHTMLTags(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		} else if item.Description != "" {
			excerpt = cleanHTMLTags(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}

		// If RSS content is short, fetch full article
		if len(excerpt) < 200 {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						// Try multiple content selectors
						selectors := []string{".entry-content", "article", ".post-content", "main"}
						for _, sel := range selectors {
							bodyElem := articleDoc.Find(sel)
							if bodyElem.Length() > 0 {
								content := strings.TrimSpace(bodyElem.Text())
								content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
								if len(content) > 100 {
									excerpt = content
									break
								}
							}
						}
					}
				}
			}
		}

		out = append(out, Headline{
			Source:      "Verra",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Verra: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesGoldStandard fetches news from Gold Standard
//
// Gold Standard is a certification standard for carbon offset projects
// focusing on sustainable development.
func collectHeadlinesGoldStandard(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.goldstandard.org/newsroom"

	client := cfg.Client
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

	// Gold Standard uses h4.title for article titles and time element for dates
	doc.Find("a[href*='/news/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip non-article links
		if href == "/news/" || href == "/newsroom" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		// Find title in h4.title within or near the link
		titleElem := link.Find("h4.title")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			// Try getting title from link text
			title = strings.TrimSpace(link.Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		seen[articleURL] = true

		// Find date from nearby time element (empty string if not found)
		dateStr := ""
		parent := link.Parent()
		for i := 0; i < 5; i++ {
			timeElem := parent.Find("time")
			if timeElem.Length() > 0 {
				if datetime, exists := timeElem.Attr("datetime"); exists {
					dateStr = datetime
					break
				}
			}
			parent = parent.Parent()
		}

		// Fetch full article content from individual page
		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Gold Standard uses <main> for article body
					bodyElem := articleDoc.Find("main")
					content = strings.TrimSpace(bodyElem.Text())
				}
			}
		}

		out = append(out, Headline{
			Source:      "Gold Standard",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     content,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Gold Standard: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesACR fetches news from American Carbon Registry
//
// ACR is a nonprofit enterprise of Winrock International that operates
// a voluntary carbon offset program.
func collectHeadlinesACR(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://acrcarbon.org/news/"

	client := cfg.Client
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

	doc.Find("article, .news-item, .post, div[class*='news'], div[class*='article']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		titleLink := article.Find("h2 a, h3 a, .title a, a[href*='/news/']").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		// Clean up title: normalize whitespace (remove newlines, multiple spaces)
		title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
		title = strings.TrimSpace(title)

		// ACR-specific: Remove "PUBLISHED ..." suffix from titles
		if idx := strings.Index(title, " PUBLISHED"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		// Remove category prefixes from titles
		acrPrefixes := []string{
			"Program Announcements ",
			"General ",
			"ACR in the News ",
			"Op-eds ",
			"Events ",
		}
		for _, prefix := range acrPrefixes {
			if strings.HasPrefix(title, prefix) {
				title = strings.TrimPrefix(title, prefix)
				title = strings.TrimSpace(title)
			}
		}

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		dateStr := ""
		foundDate := false
		dateElem := article.Find("time, .date, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"January 2, 2006",
					"Jan 2, 2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						foundDate = true
						break
					}
				}
			}
		}

		// Fetch full article content from individual page
		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// ACR uses WordPress block editor - remove unwanted sections first
					articleDoc.Find("header, footer, nav, .site-header, .site-footer, script, style, noscript").Remove()

					// Collect all paragraphs from the page and filter for article content
					var paragraphs []string
					articleDoc.Find("p").Each(func(_ int, p *goquery.Selection) {
						text := strings.TrimSpace(p.Text())
						textLower := strings.ToLower(text)

						// Skip short paragraphs
						if len(text) < 40 {
							return
						}

						// Skip known non-article content patterns
						skipPatterns := []string{
							"cookie", "gdpr", "privacy", "accept", "reject",
							"related news", "published", "home", "news",
							"we are using", "this website uses", "enable or disable",
							"strictly necessary", "3rd party", "save changes",
						}
						for _, pattern := range skipPatterns {
							if strings.Contains(textLower, pattern) {
								return
							}
						}

						// Skip if it looks like navigation/breadcrumb
						if strings.HasPrefix(text, "Home") || strings.HasPrefix(text, "News") {
							return
						}

						paragraphs = append(paragraphs, text)
					})

					if len(paragraphs) > 0 {
						content = strings.Join(paragraphs, "\n\n")
					}

					// Fallback: try full text from main content area
					if content == "" {
						mainElem := articleDoc.Find("main, article, .content, body")
						if mainElem.Length() > 0 {
							content = strings.TrimSpace(mainElem.First().Text())
							content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
						}
					}

					// Try to extract date from article page if not found
					if !foundDate {
						// Try JSON-LD schema first
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if foundDate {
								return
							}
							jsonText := script.Text()
							re := regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
							if matches := re.FindStringSubmatch(jsonText); len(matches) > 1 {
								dateStr = matches[1]
								foundDate = true
							}
						})
					}

					// ACR-specific: Try "PUBLISHED [date]" pattern from article text
					if !foundDate {
						articleText := articleDoc.Text()
						// Match "PUBLISHED January 5, 2026" or "PUBLISHED November 25, 2025"
						publishedRe := regexp.MustCompile(`PUBLISHED\s+((?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4})`)
						if match := publishedRe.FindStringSubmatch(articleText); len(match) > 1 {
							dateText := strings.ReplaceAll(match[1], ",", "")
							if t, err := time.Parse("January 2 2006", dateText); err == nil {
								dateStr = t.Format(time.RFC3339)
								foundDate = true
							}
						}
					}
				}
			}
		}

		// Fallback to current time if no date found
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "ACR",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     content,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ACR: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesCAR fetches news from Climate Action Reserve
//
// Climate Action Reserve is a carbon offset registry for the North American
// carbon market.
func collectHeadlinesCAR(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://climateactionreserve.org/updates/"

	client := cfg.Client
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

	doc.Find("article, .news-item, .post, div[class*='news'], div[class*='blog-post']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		titleLink := article.Find("h2 a, h3 a, .title a, .entry-title a").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title, .entry-title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		// Clean up title: normalize whitespace (remove newlines, multiple spaces)
		title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
		title = strings.TrimSpace(title)

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		// CAR-specific: Only allow internal blog posts
		// Skip external links and non-blog pages (program pages, etc.)
		if !strings.Contains(articleURL, "climateactionreserve.org") {
			return
		}
		// Only allow blog posts (URLs like /blog/YYYY/MM/DD/...)
		if !strings.Contains(articleURL, "/blog/") {
			return
		}

		seen[articleURL] = true

		dateStr := ""
		foundDate := false
		dateElem := article.Find("time, .date, .entry-date, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"January 2, 2006",
					"Jan 2, 2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						foundDate = true
						break
					}
				}
			}
		}

		// Fetch full article content from individual page
		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// CAR uses WordPress/Elementor with various content selectors
					// Try multiple selectors in order of preference
					selectors := []string{".entry-content", ".elementor-widget-theme-post-content", "article", ".post-content", "main"}
					for _, sel := range selectors {
						bodyElem := articleDoc.Find(sel)
						if bodyElem.Length() > 0 {
							content = strings.TrimSpace(bodyElem.Text())
							// Clean up content: normalize whitespace
							content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
							if len(content) > 50 {
								break
							}
						}
					}

					// Try to extract date from article page if not found
					if !foundDate {
						// Try JSON-LD schema
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if foundDate {
								return
							}
							jsonText := script.Text()
							re := regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
							if matches := re.FindStringSubmatch(jsonText); len(matches) > 1 {
								dateStr = matches[1]
								foundDate = true
							}
						})
					}
				}
			}
		}

		// Fallback to current time if no date found
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Climate Action Reserve",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     content,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Climate Action Reserve: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// International Organizations
// =============================================================================

// collectHeadlinesUNFCCC fetches news from United Nations Framework Convention on Climate Change
//
// UNFCCC is the international treaty on climate change that serves as the foundation
// for global climate negotiations.
func collectHeadlinesUNFCCC(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://unfccc.int/news"

	client := cfg.Client
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

	doc.Find("article, .news-item, .views-row, div[class*='teaser'], div[class*='card']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		titleLink := article.Find("h2 a, h3 a, .title a, a[href*='/news/']").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title, .field--name-title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Extract date (empty string if not found)
		dateStr := ""
		dateElem := article.Find("time, .date, .field--name-created, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"2 January 2006",
					"January 2, 2006",
					"02/01/2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						break
					}
				}
			}
		}

		excerpt := ""
		excerptElem := article.Find("p, .field--name-body, .summary").First()
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.Text())
		}

		out = append(out, Headline{
			Source:      "UNFCCC",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] UNFCCC: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIISD fetches news from IISD Earth Negotiations Bulletin
//
// IISD ENB provides reporting on international environmental negotiations,
// including climate change conferences and carbon market discussions.
// Note: IISD requires cookie-based session to access individual article pages.
func collectHeadlinesIISD(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://enb.iisd.org/"

	// Create a client with cookie jar to maintain session
	// IISD blocks requests without proper session cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	client := &http.Client{
		Timeout: cfg.Timeout,
		Jar:     jar,
	}

	// First, visit the homepage to get session cookies
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

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

	// Helper function to extract date from text
	extractDate := func(text string) string {
		datePatterns := []struct {
			regex  string
			format string
		}{
			{`(\d{1,2}\s+(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{4})`, "2 January 2006"},
			{`((?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4})`, "January 2, 2006"},
		}
		for _, dp := range datePatterns {
			re := regexp.MustCompile(dp.regex)
			if match := re.FindStringSubmatch(text); len(match) > 1 {
				dateText := strings.ReplaceAll(match[1], ",", "")
				if t, err := time.Parse(dp.format, dateText); err == nil {
					return t.Format(time.RFC3339)
				}
			}
		}
		return ""
	}

	// Helper function to fetch article page and extract full content
	// Returns: About (og:description) + full body content
	fetchArticleContent := func(articleURL string) string {
		time.Sleep(300 * time.Millisecond) // Delay between requests to avoid rate limiting

		var resp *http.Response
		// Retry up to 2 times on 403 (bot protection / rate limiting)
		for attempt := 0; attempt < 2; attempt++ {
			if attempt > 0 {
				time.Sleep(1 * time.Second)
			}
			req, err := http.NewRequest("GET", articleURL, nil)
			if err != nil {
				return ""
			}
			req.Header.Set("User-Agent", cfg.UserAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			req.Header.Set("Referer", newsURL)

			resp, err = client.Do(req)
			if err != nil {
				if os.Getenv("DEBUG_SCRAPING") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: failed to fetch %s: %v\n", articleURL, err)
				}
				return ""
			}
			if resp.StatusCode == http.StatusOK {
				break
			}
			resp.Body.Close()
			if resp.StatusCode != 403 {
				if os.Getenv("DEBUG_SCRAPING") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: status %d for %s\n", resp.StatusCode, articleURL)
				}
				return ""
			}
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: 403 for %s, retrying...\n", articleURL)
			}
		}
		if resp.StatusCode != http.StatusOK {
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: status %d for %s after retries\n", resp.StatusCode, articleURL)
			}
			return ""
		}
		defer resp.Body.Close()

		articleDoc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return ""
		}

		var parts []string

		// 1. Get About section from og:description
		if about := articleDoc.Find("meta[property='og:description']").AttrOr("content", ""); about != "" {
			parts = append(parts, "【About】\n"+strings.TrimSpace(about))
		}

		// 2. Get full body content from ALL .c-wysiwyg__content sections
		// Articles may have multiple sections separated by images
		var bodyParts []string
		seen := make(map[string]bool) // Avoid duplicates

		articleDoc.Find(".c-wysiwyg__content").Each(func(_ int, section *goquery.Selection) {
			// Get paragraphs from this section
			section.Find("p").Each(func(_ int, p *goquery.Selection) {
				text := strings.TrimSpace(p.Text())
				// Skip short texts, metadata, and newsletter subscription text
				if len(text) > 50 && !strings.Contains(text, "subscribe to the ENB") &&
					!strings.Contains(text, "Earth Negotiations Bulletin writers") &&
					!seen[text] {
					seen[text] = true
					bodyParts = append(bodyParts, text)
				}
			})

			// Also get list items (highlights, agenda items, etc.)
			section.Find("li").Each(func(_ int, li *goquery.Selection) {
				text := strings.TrimSpace(li.Text())
				if len(text) > 20 && !seen[text] {
					seen[text] = true
					bodyParts = append(bodyParts, "• "+text)
				}
			})
		})

		if len(bodyParts) > 0 {
			parts = append(parts, "\n【Content】\n"+strings.Join(bodyParts, "\n\n"))
		}

		if len(parts) == 0 {
			// Fallback to meta description if nothing found
			if desc := articleDoc.Find("meta[name='description']").AttrOr("content", ""); desc != "" {
				return strings.TrimSpace(desc)
			}
			return ""
		}

		return strings.Join(parts, "\n")
	}

	// First, collect from featured boxes (these have summaries on list page)
	doc.Find("a.c-featured-box, .c-featured-box").Each(func(_ int, box *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Get link - either from the element itself or from child
		var href string
		if h, exists := box.Attr("href"); exists {
			href = h
		} else {
			href, _ = box.Find("a").First().Attr("href")
		}
		if href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Get title
		title := strings.TrimSpace(box.Find(".c-featured-box__title, h3, h4").First().Text())
		if title == "" || len(title) < 10 {
			return
		}

		// Always fetch full content from article page (About + Content)
		excerpt := fetchArticleContent(articleURL)

		// Extract date from box text
		boxText := box.Text()
		dateStr := extractDate(boxText)
		if dateStr == "" {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "IISD ENB",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	// If we haven't reached limit, also collect from hero items (current events)
	if len(out) < limit {
		doc.Find(".c-hero-item").Each(func(_ int, hero *goquery.Selection) {
			if len(out) >= limit {
				return
			}

			link := hero.Find("a[href]").First()
			href, exists := link.Attr("href")
			if !exists || href == "" {
				return
			}

			articleURL := resolveURL(newsURL, href)
			if articleURL == "" || seen[articleURL] {
				return
			}
			seen[articleURL] = true

			title := strings.TrimSpace(hero.Find("h2, h3, .c-hero-item__title").First().Text())
			if title == "" || len(title) < 10 {
				return
			}

			// Hero items don't have descriptions on list page
			// Fetch content from individual article page using session cookies
			excerpt := fetchArticleContent(articleURL)

			dateStr := extractDate(hero.Text())
			if dateStr == "" {
				dateStr = time.Now().Format(time.RFC3339)
			}

			out = append(out, Headline{
				Source:      "IISD ENB",
				Title:       title,
				URL:         articleURL,
				PublishedAt: dateStr,
				Excerpt:     excerpt,
				IsHeadline:  true,
			})
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesClimateFocus fetches publications from Climate Focus
//
// Climate Focus is a climate policy advisory firm that publishes research
// on carbon markets and climate finance.
func collectHeadlinesClimateFocus(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	publicationsURL := "https://climatefocus.com/publications/"

	client := cfg.Client
	req, err := http.NewRequest("GET", publicationsURL, nil)
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

	// Climate Focus - find links to publications directly
	doc.Find("a[href*='/publications/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip if this is just the main publications page
		if href == publicationsURL || href == "https://climatefocus.com/publications/" {
			return
		}

		articleURL := resolveURL(publicationsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Get title from link text or from image alt
		title := strings.TrimSpace(link.Text())
		if title == "" {
			imgElem := link.Find("img")
			if imgElem.Length() > 0 {
				title, _ = imgElem.Attr("alt")
				title = strings.TrimSpace(title)
			}
		}

		// If still no title, extract from URL
		if title == "" || len(title) < 10 {
			// Extract title from URL path
			parts := strings.Split(href, "/")
			for i := len(parts) - 1; i >= 0; i-- {
				if parts[i] != "" {
					title = strings.ReplaceAll(parts[i], "-", " ")
					title = strings.Title(title)
					break
				}
			}
		}

		if title == "" || len(title) < 10 {
			return
		}

		// Get category from sibling/parent elements
		excerpt := ""
		parent := link.Parent()
		categoryElem := parent.Find(".category")
		if categoryElem.Length() > 0 {
			excerpt = "Category: " + strings.TrimSpace(categoryElem.Text())
		}

		// Fetch individual article page for date and content
		dateStr := ""
		foundDate := false
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Look for date in various locations
					// 1. Try JSON-LD schema (datePublished)
					articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
						if foundDate {
							return
						}
						text := script.Text()
						// Look for datePublished pattern
						if strings.Contains(text, "datePublished") {
							re := regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
							if match := re.FindStringSubmatch(text); len(match) > 1 {
								dateStr = match[1]
								foundDate = true
							}
						}
					})

					// 2. Try visible date text "Jan 2026" format
					if !foundDate {
						articleDoc.Find(".date, time, span[class*='date']").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							text := strings.TrimSpace(elem.Text())
							// Try "Jan 2026" format (short month + year)
							re := regexp.MustCompile(`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{4})`)
							if match := re.FindStringSubmatch(text); len(match) > 2 {
								// Parse as first day of month
								dateText := match[1] + " 1, " + match[2]
								if t, err := time.Parse("Jan 2, 2006", dateText); err == nil {
									dateStr = t.Format(time.RFC3339)
									foundDate = true
								}
							}
						})
					}

					// 3. Extract full article content from the page
					// Remove unwanted elements first
					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related-posts").Remove()

					// Try to find article content
					contentSelectors := []string{
						".entry-content",
						".article-content",
						".post-content",
						".content-area",
						"article .content",
						"main article",
						".elementor-widget-theme-post-content",
					}
					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							// Get text from paragraphs
							var paragraphs []string
							contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
								text := strings.TrimSpace(p.Text())
								if len(text) > 30 {
									paragraphs = append(paragraphs, text)
								}
							})
							if len(paragraphs) > 0 {
								excerpt = strings.Join(paragraphs, "\n\n")
								break
							}
						}
					}

					// Fallback to meta description if no content found
					if excerpt == "" {
						excerptElem := articleDoc.Find("meta[name='description'], meta[property='og:description']")
						if excerptElem.Length() > 0 {
							excerpt, _ = excerptElem.Attr("content")
							excerpt = strings.TrimSpace(excerpt)
						}
					}
				}
			}
		}

		// Fallback to current time if no date found
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Climate Focus",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Climate Focus: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// Additional Sources (Phase 5)
// =============================================================================

// collectHeadlinesPuroEarth fetches blog articles from Puro.earth
//
// Puro.earth is a carbon removal marketplace that provides certification
// for carbon removal projects and credits. Their blog contains news,
// methodology updates, and industry insights.
//
// 手法: Atom Feed (gofeed) + HTML scraping for full content
// URL: https://puro.earth/blog/our-blogs-1/feed
func collectHeadlinesPuroEarth(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://puro.earth/blog/our-blogs-1/feed"

	client := cfg.Client
	req, err := http.NewRequest("GET", feedURL, nil)
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

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Atom parse failed: %w", err)
	}

	out := make([]Headline, 0, limit)

	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}

		articleURL := item.Link
		if articleURL == "" {
			continue
		}

		// Parse date (empty string if not available)
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// Fetch article page to get full content
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Puro.earth uses Odoo CMS with specific class names
					// Content may be in <p> tags or as direct text nodes between <br> tags
					contentSelectors := []string{
						".o_wblog_post_content_field",
						".o_wblog_read_text",
					}

					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							// First try to get content from <p> tags
							var contentParts []string
							contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
								text := strings.TrimSpace(p.Text())
								if len(text) > 30 {
									contentParts = append(contentParts, text)
								}
							})

							// If <p> tags don't have enough content, get full text
							// (Puro.earth sometimes uses direct text with <br> separators)
							if len(strings.Join(contentParts, "")) < 200 {
								fullText := strings.TrimSpace(contentElem.Text())
								// Normalize whitespace (multiple spaces/newlines to single newline)
								fullText = regexp.MustCompile(`[\s]+`).ReplaceAllString(fullText, " ")
								// Split into paragraphs at logical breaks (sentences ending with period followed by capital)
								fullText = regexp.MustCompile(`\. ([A-Z])`).ReplaceAllString(fullText, ".\n\n$1")
								if len(fullText) > 100 {
									excerpt = fullText
									break
								}
							} else {
								excerpt = strings.Join(contentParts, "\n\n")
								break
							}
						}
					}
				}
			}
		}

		// Fallback to feed description if article fetch failed
		if excerpt == "" {
			if item.Description != "" {
				excerpt = item.Description
			} else if item.Content != "" {
				excerpt = item.Content
			}
			// Clean up HTML tags
			excerpt = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(excerpt, "")
			excerpt = regexp.MustCompile(`\s+`).ReplaceAllString(excerpt, " ")
			excerpt = strings.TrimSpace(excerpt)
		}

		out = append(out, Headline{
			Source:      "Puro.earth",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Puro.earth: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIsometric fetches resources from Isometric
//
// Isometric is a science-based carbon removal verification company
// that publishes research and resources on carbon removal.
//
// HTML structure:
// - Title: p.writing-card-title
// - Date: div.label-small.cc-date (format: "Oct 20, 2025")
// - Subtitle: div.u-text-grey80.u-hide
func collectHeadlinesIsometric(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	resourcesURL := "https://isometric.com/writing"

	client := cfg.Client
	req, err := http.NewRequest("GET", resourcesURL, nil)
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

	doc.Find("a[href*='/writing-articles/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(resourcesURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Find title from p.writing-card-title
		title := strings.TrimSpace(link.Find("p.writing-card-title").Text())
		if title == "" || len(title) < 10 {
			return
		}

		// Find date from div.cc-date (empty string if not found)
		dateStr := ""
		foundDate := false
		dateElem := link.Find("div.cc-date, .label-small.cc-date")
		if dateElem.Length() > 0 {
			dateText := strings.TrimSpace(dateElem.First().Text())
			// Format: "Oct 20, 2025" or "Jan 21, 2026"
			for _, format := range []string{
				"Jan 2, 2006",
				"Jan 02, 2006",
				"January 2, 2006",
			} {
				if t, err := time.Parse(format, dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
					foundDate = true
					break
				}
			}
		}

		// Get subtitle from listing page as initial excerpt
		subtitle := strings.TrimSpace(link.Find("div.u-text-grey80").Text())

		// Fetch article page to get full content
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Try to get date from article page if not found on listing
					if !foundDate {
						articleDoc.Find("div.cc-date, .label-small.cc-date, time").Each(func(_ int, dateEl *goquery.Selection) {
							if foundDate {
								return
							}
							dateText := strings.TrimSpace(dateEl.Text())
							for _, format := range []string{
								"Jan 2, 2006",
								"Jan 02, 2006",
								"January 2, 2006",
							} {
								if t, err := time.Parse(format, dateText); err == nil {
									dateStr = t.Format(time.RFC3339)
									foundDate = true
									break
								}
							}
						})
					}

					// Extract content from article body
					contentSelectors := []string{
						".rich-text",
						".w-richtext",
						"article",
						".article-content",
						".content",
					}

					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							var contentParts []string
							contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
								text := strings.TrimSpace(p.Text())
								if len(text) > 30 {
									contentParts = append(contentParts, text)
								}
							})
							if len(contentParts) > 0 {
								excerpt = strings.Join(contentParts, "\n\n")
								break
							}
						}
					}
				}
			}
		}

		// Fallback to subtitle if article fetch failed
		if excerpt == "" && subtitle != "" {
			excerpt = subtitle
		} else if excerpt == "" {
			excerpt = title
		}

		out = append(out, Headline{
			Source:      "Isometric",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Isometric: collected %d headlines\n", len(out))
	}

	return out, nil
}

