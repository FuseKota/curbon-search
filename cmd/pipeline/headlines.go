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

	"github.com/PuerkitoBio/goquery"
)

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var (
	reCarbonPulseID = regexp.MustCompile(`^/\d+/$`)
	reQCIArticle    = regexp.MustCompile(`/carbon/article/`)
)

type headlineSourceConfig struct {
	CarbonPulseTimelineURL string
	CarbonPulseNewsletters string
	QCIHomeURL             string
	UserAgent              string
	Timeout                time.Duration
}

func defaultHeadlineConfig() headlineSourceConfig {
	return headlineSourceConfig{
		CarbonPulseTimelineURL: "https://carbon-pulse.com/daily-timeline/",
		CarbonPulseNewsletters: "https://carbon-pulse.com/category/newsletters/",
		QCIHomeURL:             "https://www.qcintel.com/carbon/",
		UserAgent:              "Mozilla/5.0 (compatible; carbon-relay/1.0; +https://example.invalid)",
		Timeout:                20 * time.Second,
	}
}

func collectHeadlinesCarbonPulse(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// We try 3 pages that are typically visible without subscription.
	// The top page has article excerpts in the main content area.
	pages := []string{
		"https://carbon-pulse.com/", // Top page with excerpts
		cfg.CarbonPulseTimelineURL,
		cfg.CarbonPulseNewsletters,
	}
	out := []Headline{}
	seen := map[string]bool{}

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

func fetchDoc(u string, cfg headlineSourceConfig) (*goquery.Document, error) {
	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: status %s", u, resp.Status)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

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
	return base.ResolveReference(u).String()
}

// extractExcerptFromContext extracts text content near a headline link
// from the timeline/listing page itself, without fetching the article page.
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
