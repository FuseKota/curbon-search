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
//   5. NewClimate         - 気候研究機関
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
package pipeline

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
)

// collectHeadlinesICAP fetches articles from ICAP (Drupal site) using HTML scraping
func collectHeadlinesICAP(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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
					articleResp.Body.Close()
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

	doc.Find("div.card-body").Each(func(_ int, cardBody *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		titleElem := cardBody.Find("h3.news-title")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			return
		}

		dateElem := cardBody.Find("div.resource-date")
		dateStr := strings.TrimSpace(dateElem.Text())

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

		publishedAt := ""
		if dateStr != "" {
			t, err := time.Parse("Jan 2, 2006", dateStr)
			if err == nil {
				publishedAt = t.Format(time.RFC3339)
			}
		}

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
							articleDoc.Find(".section-news-detail .intro, .section-news-detail section.bg-white").Each(func(_ int, s *goquery.Selection) {
								t := strings.TrimSpace(s.Text())
								if t != "" {
									parts = append(parts, t)
								}
							})
							content = strings.Join(parts, "\n\n")
						}
					}
					articleResp.Body.Close()
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

	doc.Find("article").Each(func(_ int, article *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

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
							bodyElem := articleDoc.Find("article .entry-content, article .article-content, .post-content, .content").First()
							content = strings.TrimSpace(bodyElem.Text())

							articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
								if publishedAt != "" {
									return
								}
								jsonText := script.Text()
								if matches := reDatePublishedJSON.FindStringSubmatch(jsonText); len(matches) > 1 {
									publishedAt = matches[1]
								}
							})

							if publishedAt == "" {
								timeElem := articleDoc.Find("time")
								datetime, exists := timeElem.Attr("datetime")
								if exists {
									publishedAt = datetime
								}
							}
						}
					}
					articleResp.Body.Close()
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

		articleURL := strings.Replace(doc.URL, "http://", "https://", 1)
		dateStr := doc.Lnchdt
		title := doc.Title.Cdata
		excerpt := doc.Descr.Cdata

		pageDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			if title == "" {
				h1 := pageDoc.Find("h1").First()
				if h1.Length() > 0 {
					title = strings.TrimSpace(h1.Text())
				}
			}
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

// collectHeadlinesNewClimate collects headlines from NewClimate Institute
func collectHeadlinesNewClimate(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	doc.Find("a[href^='/news/'], a[href^='/resources/publications/']").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists {
			return
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		articleURL := "https://newclimate.org" + href

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

		excerpt := ""
		articleDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
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

	return out, nil
}

// collectHeadlinesCarbonKnowledgeHub collects headlines from Carbon Knowledge Hub
func collectHeadlinesCarbonKnowledgeHub(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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
	seen := make(map[string]bool)

	doc.Find("a.css-oxwq25, a[class*='css-']").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		isContentURL := (strings.Contains(href, "/factsheet") ||
			strings.Contains(href, "/story") ||
			strings.Contains(href, "/stories") ||
			strings.Contains(href, "/audio") ||
			strings.Contains(href, "/media") ||
			strings.Contains(href, "/news")) &&
			strings.Count(href, "/") > 1

		if !isContentURL {
			return
		}

		articleURL := href
		if !strings.HasPrefix(href, "http") {
			if strings.HasPrefix(href, "/") {
				articleURL = "https://www.carbonknowledgehub.com" + href
			} else {
				articleURL = "https://www.carbonknowledgehub.com/" + href
			}
		}

		if seen[articleURL] {
			return
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		titleLower := strings.ToLower(title)
		if strings.Contains(titleLower, "read more") ||
			strings.Contains(titleLower, "learn more") ||
			strings.Contains(titleLower, "click here") ||
			strings.Contains(titleLower, "view all") {
			return
		}

		seen[articleURL] = true

		dateStr := ""
		excerpt := ""
		articleDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
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

	return out, nil
}

// =============================================================================
// VCM Certification Bodies
// =============================================================================

// collectHeadlinesVerra fetches news from Verra (VCS standard operator)
func collectHeadlinesVerra(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://verra.org/news/feed/"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Verra RSS feed")
	}

	client := cfg.Client
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

		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		excerpt := extractRSSExcerpt(item)

		if len(excerpt) < 200 {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						selectors := []string{".entry-content", "article", ".post-content", "main"}
						for _, sel := range selectors {
							bodyElem := articleDoc.Find(sel)
							if bodyElem.Length() > 0 {
								content := strings.TrimSpace(bodyElem.Text())
								content = reWhitespace.ReplaceAllString(content, " ")
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
func collectHeadlinesGoldStandard(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	doc.Find("a[href*='/news/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		if href == "/news/" || href == "/newsroom" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		titleElem := link.Find("h4.title")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			title = strings.TrimSpace(link.Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		seen[articleURL] = true

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

		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
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
func collectHeadlinesACR(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

		title = reWhitespace.ReplaceAllString(title, " ")
		title = strings.TrimSpace(title)

		if idx := strings.Index(title, " PUBLISHED"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
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

		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					articleDoc.Find("header, footer, nav, .site-header, .site-footer, script, style, noscript").Remove()

					var paragraphs []string
					articleDoc.Find("p").Each(func(_ int, p *goquery.Selection) {
						text := strings.TrimSpace(p.Text())
						textLower := strings.ToLower(text)
						if len(text) < 40 {
							return
						}
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
						if strings.HasPrefix(text, "Home") || strings.HasPrefix(text, "News") {
							return
						}
						paragraphs = append(paragraphs, text)
					})

					if len(paragraphs) > 0 {
						content = strings.Join(paragraphs, "\n\n")
					}

					if content == "" {
						mainElem := articleDoc.Find("main, article, .content, body")
						if mainElem.Length() > 0 {
							content = strings.TrimSpace(mainElem.First().Text())
							content = reWhitespace.ReplaceAllString(content, " ")
						}
					}

					if !foundDate {
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if foundDate {
								return
							}
							jsonText := script.Text()
							if matches := reDatePublishedJSON.FindStringSubmatch(jsonText); len(matches) > 1 {
								dateStr = matches[1]
								foundDate = true
							}
						})
					}

					if !foundDate {
						articleText := articleDoc.Text()
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
func collectHeadlinesCAR(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

		title = reWhitespace.ReplaceAllString(title, " ")
		title = strings.TrimSpace(title)

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		if !strings.Contains(articleURL, "climateactionreserve.org") {
			return
		}
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

		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					selectors := []string{".entry-content", ".elementor-widget-theme-post-content", "article", ".post-content", "main"}
					for _, sel := range selectors {
						bodyElem := articleDoc.Find(sel)
						if bodyElem.Length() > 0 {
							content = strings.TrimSpace(bodyElem.Text())
							content = reWhitespace.ReplaceAllString(content, " ")
							if len(content) > 50 {
								break
							}
						}
					}

					if !foundDate {
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if foundDate {
								return
							}
							jsonText := script.Text()
							if matches := reDatePublishedJSON.FindStringSubmatch(jsonText); len(matches) > 1 {
								dateStr = matches[1]
								foundDate = true
							}
						})
					}
				}
			}
		}

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

// collectHeadlinesUNFCCC fetches news from UNFCCC
func collectHeadlinesUNFCCC(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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
func collectHeadlinesIISD(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	newsURL := "https://enb.iisd.org/"

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	client := &http.Client{
		Timeout: cfg.Timeout,
		Jar:     jar,
	}

	// Retry up to 3 times with exponential backoff (AWS IPs often get 403)
	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * 2 * time.Second
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: homepage 403, retrying in %v (attempt %d/3)...\n", delay, attempt+1)
			}
			time.Sleep(delay)
		}
		req, err := http.NewRequest("GET", newsURL, nil)
		if err != nil {
			return nil, fmt.Errorf("request creation failed: %w", err)
		}
		req.Header.Set("User-Agent", cfg.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d (after 3 retries)", resp.StatusCode)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

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

	fetchArticleContent := func(articleURL string) string {
		time.Sleep(500 * time.Millisecond)

		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
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
				return ""
			}
			if resp.StatusCode == http.StatusOK {
				break
			}
			resp.Body.Close()
			if resp.StatusCode != 403 {
				return ""
			}
		}
		if resp.StatusCode != http.StatusOK {
			return ""
		}
		defer resp.Body.Close()

		articleDoc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return ""
		}

		var parts []string

		if about := articleDoc.Find("meta[property='og:description']").AttrOr("content", ""); about != "" {
			parts = append(parts, "【About】\n"+strings.TrimSpace(about))
		}

		var bodyParts []string
		seenText := make(map[string]bool)

		articleDoc.Find(".c-wysiwyg__content").Each(func(_ int, section *goquery.Selection) {
			section.Find("p").Each(func(_ int, p *goquery.Selection) {
				text := strings.TrimSpace(p.Text())
				if len(text) > 50 && !strings.Contains(text, "subscribe to the ENB") &&
					!strings.Contains(text, "Earth Negotiations Bulletin writers") &&
					!seenText[text] {
					seenText[text] = true
					bodyParts = append(bodyParts, text)
				}
			})
			section.Find("li").Each(func(_ int, li *goquery.Selection) {
				text := strings.TrimSpace(li.Text())
				if len(text) > 20 && !seenText[text] {
					seenText[text] = true
					bodyParts = append(bodyParts, "• "+text)
				}
			})
		})

		if len(bodyParts) > 0 {
			parts = append(parts, "\n【Content】\n"+strings.Join(bodyParts, "\n\n"))
		}

		if len(parts) == 0 {
			if desc := articleDoc.Find("meta[name='description']").AttrOr("content", ""); desc != "" {
				return strings.TrimSpace(desc)
			}
			return ""
		}

		return strings.Join(parts, "\n")
	}

	doc.Find("a.c-featured-box, .c-featured-box").Each(func(_ int, box *goquery.Selection) {
		if len(out) >= limit {
			return
		}

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

		title := strings.TrimSpace(box.Find(".c-featured-box__title, h3, h4").First().Text())
		if title == "" || len(title) < 10 {
			return
		}

		excerpt := fetchArticleContent(articleURL)

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
func collectHeadlinesClimateFocus(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	doc.Find("a[href*='/publications/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		if href == publicationsURL || href == "https://climatefocus.com/publications/" {
			return
		}
		if strings.Contains(href, "sf_paged=") || strings.Contains(href, "wpengine.com") {
			return
		}

		articleURL := resolveURL(publicationsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		title := strings.TrimSpace(link.Text())
		if title == "" {
			imgElem := link.Find("img")
			if imgElem.Length() > 0 {
				title, _ = imgElem.Attr("alt")
				title = strings.TrimSpace(title)
			}
		}

		if title == "" || len(title) < 10 {
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

		excerpt := ""
		parent := link.Parent()
		categoryElem := parent.Find(".category")
		if categoryElem.Length() > 0 {
			excerpt = "Category: " + strings.TrimSpace(categoryElem.Text())
		}

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
					articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
						if foundDate {
							return
						}
						text := script.Text()
						if strings.Contains(text, "datePublished") {
							if match := reDatePublishedJSON.FindStringSubmatch(text); len(match) > 1 {
								dateStr = match[1]
								foundDate = true
							}
						}
					})

					if !foundDate {
						articleDoc.Find(".date, time, span[class*='date']").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							text := strings.TrimSpace(elem.Text())
							re := regexp.MustCompile(`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{4})`)
							if match := re.FindStringSubmatch(text); len(match) > 2 {
								dateText := match[1] + " 1, " + match[2]
								if t, err := time.Parse("Jan 2, 2006", dateText); err == nil {
									dateStr = t.Format(time.RFC3339)
									foundDate = true
								}
							}
						})
					}

					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related-posts").Remove()

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
// Additional Sources
// =============================================================================

// collectHeadlinesPuroEarth fetches blog articles from Puro.earth
func collectHeadlinesPuroEarth(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://puro.earth/blog/our-blogs-1/feed"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	client := cfg.Client
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

		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					contentSelectors := []string{
						".o_wblog_post_content_field",
						".o_wblog_read_text",
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

							if len(strings.Join(contentParts, "")) < 200 {
								fullText := strings.TrimSpace(contentElem.Text())
								fullText = reWhitespace.ReplaceAllString(fullText, " ")
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

		if excerpt == "" {
			if item.Description != "" {
				excerpt = item.Description
			} else if item.Content != "" {
				excerpt = item.Content
			}
			excerpt = reHTMLTags.ReplaceAllString(excerpt, "")
			excerpt = reWhitespace.ReplaceAllString(excerpt, " ")
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
func collectHeadlinesIsometric(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

		title := strings.TrimSpace(link.Find("p.writing-card-title").Text())
		if title == "" || len(title) < 10 {
			return
		}

		dateStr := ""
		foundDate := false
		dateElem := link.Find("div.cc-date, .label-small.cc-date")
		if dateElem.Length() > 0 {
			dateText := strings.TrimSpace(dateElem.First().Text())
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

		subtitle := strings.TrimSpace(link.Find("div.u-text-grey80").Text())

		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
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
