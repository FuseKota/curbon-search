// =============================================================================
// sources_academic.go - Academic/Research Sources
// =============================================================================
//
// This file defines academic and research publication sources for carbon-related
// content using XML APIs and RSS feeds.
//
// Sources:
//   1. arXiv         - Pre-print repository (XML API)
//   2. Nature Communications - Scientific journal (RSS + keyword filter)
//   3. OIES          - Oxford Institute for Energy Studies (HTML)
//
// =============================================================================
package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// =============================================================================
// arXiv Source
// =============================================================================

// arXivFeed represents the Atom feed structure from arXiv API
type arXivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arXivEntry `xml:"entry"`
}

// arXivEntry represents a single paper entry from arXiv
type arXivEntry struct {
	Title     string       `xml:"title"`
	ID        string       `xml:"id"`
	Published string       `xml:"published"`
	Updated   string       `xml:"updated"`
	Summary   string       `xml:"summary"`
	Authors   []arXivAuthor `xml:"author"`
	Links     []arXivLink  `xml:"link"`
}

// arXivAuthor represents an author in arXiv entry
type arXivAuthor struct {
	Name string `xml:"name"`
}

// arXivLink represents a link in arXiv entry
type arXivLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// carbonKeywordsArXiv contains keywords for filtering arXiv papers to ensure relevance
// Using compound phrases to avoid false positives from physics papers
// (e.g., "emission" alone matches "positron emission", "light emission", etc.)
var carbonKeywordsArXiv = []string{
	// Climate-specific compound terms
	"carbon emission", "carbon dioxide", "co2 emission", "greenhouse gas",
	"carbon pricing", "carbon tax", "carbon market", "carbon credit",
	"emissions trading", "cap and trade", "carbon trading",
	"climate change", "climate policy", "global warming",
	"decarbonization", "decarbonisation", "net-zero", "net zero", "carbon neutral",
	"renewable energy", "clean energy", "energy transition",
	"carbon capture", "carbon storage", "carbon sequestration",
	"carbon footprint", "carbon intensity",
	// International agreements
	"paris agreement", "kyoto protocol",
}

// collectHeadlinesArXiv fetches carbon-related papers from arXiv using their API
//
// API Documentation: https://info.arxiv.org/help/api/index.html
// Rate limit: 3 seconds between requests (enforced)
//
// Search query targets papers in q-fin (Quantitative Finance), econ (Economics),
// and physics (specifically environmental economics topics)
func collectHeadlinesArXiv(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// Search specifically for climate/carbon economics papers
	// Using category restrictions to avoid physics papers
	// Categories:
	//   econ.GN - Economics (General Economics)
	//   q-fin.* - Quantitative Finance
	//   physics.soc-ph - Physics and Society (includes some climate policy papers)
	//   physics.ao-ph - Atmospheric and Oceanic Physics
	//   stat.AP - Statistics Applications

	// Build search query with category filter AND keyword filter
	// Format: (cat:econ.* OR cat:q-fin.*) AND (keyword1 OR keyword2)
	categories := "cat:econ.GN+OR+cat:q-fin.GN+OR+cat:q-fin.PM+OR+cat:stat.AP"
	keywords := "carbon+OR+climate+OR+emission+OR+environmental+policy"

	// Combined query: papers in relevant categories that mention carbon/climate terms
	searchQuery := fmt.Sprintf("(%s)+AND+(%s)", categories, keywords)

	// arXiv API URL with search parameters
	// max_results limits results, sortBy=submittedDate gets newest first
	apiURL := fmt.Sprintf(
		"http://export.arxiv.org/api/query?search_query=%s&start=0&max_results=%d&sortBy=submittedDate&sortOrder=descending",
		searchQuery,
		limit*10, // Request more to account for keyword filtering
	)

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

	// Parse XML response
	var feed arXivFeed
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("XML parse failed: %w", err)
	}

	out := make([]Headline, 0, limit)

	for _, entry := range feed.Entries {
		if len(out) >= limit {
			break
		}

		// Clean title (remove newlines that arXiv adds)
		title := strings.TrimSpace(entry.Title)
		title = strings.ReplaceAll(title, "\n", " ")
		title = strings.Join(strings.Fields(title), " ")
		if title == "" {
			continue
		}

		// Clean summary for keyword check
		summaryClean := strings.TrimSpace(entry.Summary)
		summaryClean = strings.ReplaceAll(summaryClean, "\n", " ")
		summaryClean = strings.Join(strings.Fields(summaryClean), " ")

		// Apply keyword filter to ensure paper is actually about carbon/climate
		titleLower := strings.ToLower(title)
		summaryLower := strings.ToLower(summaryClean)
		hasKeyword := false
		for _, kw := range carbonKeywordsArXiv {
			if strings.Contains(titleLower, kw) || strings.Contains(summaryLower, kw) {
				hasKeyword = true
				break
			}
		}
		if !hasKeyword {
			continue
		}

		// Get the abstract page URL (the ID is the URL)
		articleURL := entry.ID

		// Find PDF link if available
		for _, link := range entry.Links {
			if link.Type == "application/pdf" {
				// We prefer the abstract page, but PDF is available
				break
			}
		}

		// Parse date (arXiv uses RFC3339)
		dateStr := entry.Published
		if dateStr == "" {
			dateStr = entry.Updated
		}

		// Use already cleaned summary
		summary := summaryClean

		// Build author string
		var authors []string
		for _, author := range entry.Authors {
			authors = append(authors, author.Name)
		}
		authorStr := strings.Join(authors, ", ")
		if len(authorStr) > 100 {
			authorStr = authorStr[:100] + "..."
		}

		excerpt := summary
		if authorStr != "" {
			excerpt = "Authors: " + authorStr + "\n\n" + summary
		}

		out = append(out, Headline{
			Source:      "arXiv",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] arXiv: collected %d headlines\n", len(out))
	}

	// Respect rate limit - sleep 3 seconds after request
	time.Sleep(3 * time.Second)

	return out, nil
}

// =============================================================================
// Nature Communications Source
// =============================================================================

// carbonKeywordsNature contains keywords for filtering Nature Communications articles
var carbonKeywordsNature = []string{
	"carbon", "emission", "greenhouse", "climate change", "net zero",
	"decarbonization", "decarbonisation", "carbon dioxide", "CO2",
	"carbon pricing", "carbon tax", "cap and trade", "emissions trading",
	"carbon market", "carbon credit", "offset", "sequestration",
	"carbon capture", "CCS", "CCUS", "negative emissions",
}

// collectHeadlinesNatureComms fetches climate-related articles from Nature Communications RSS
//
// Nature Communications is a peer-reviewed open-access journal covering all areas
// of natural sciences. We filter for carbon/climate related articles using keywords.
func collectHeadlinesNatureComms(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.nature.com/ncomms.rss"

	client := &http.Client{Timeout: cfg.Timeout}
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

	out := make([]Headline, 0, limit)

	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}

		// Apply keyword filter
		titleLower := strings.ToLower(title)
		descLower := strings.ToLower(item.Description)

		hasKeyword := false
		for _, kw := range carbonKeywordsNature {
			if strings.Contains(titleLower, kw) || strings.Contains(descLower, kw) {
				hasKeyword = true
				break
			}
		}

		if !hasKeyword {
			continue
		}

		articleURL := item.Link

		// Parse date
		dateStr := time.Now().Format(time.RFC3339)
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// Get description/abstract
		excerpt := cleanHTMLTags(item.Description)
		excerpt = strings.TrimSpace(excerpt)

		out = append(out, Headline{
			Source:      "Nature Communications",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Nature Communications: collected %d headlines (filtered from RSS)\n", len(out))
	}

	return out, nil
}

// =============================================================================
// OIES (Oxford Institute for Energy Studies) Source
// =============================================================================

// collectHeadlinesOIES fetches publications from Oxford Institute for Energy Studies
//
// OIES publishes research papers on energy and environmental economics,
// including carbon markets and climate policy.
func collectHeadlinesOIES(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	publicationsURL := "https://www.oxfordenergy.org/publications/"

	client := &http.Client{Timeout: cfg.Timeout}
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

	// OIES uses article cards for publications
	doc.Find("article, .publication-item, .post, div[class*='publication']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Find title and link
		titleLink := article.Find("h2 a, h3 a, .title a, a.title").First()
		if titleLink.Length() == 0 {
			titleLink = article.Find("a[href*='/publications/']").First()
		}

		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			// Try getting title from heading directly
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(publicationsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Extract date
		dateStr := time.Now().Format(time.RFC3339)
		dateElem := article.Find("time, .date, .published, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				// Try various date formats
				for _, format := range []string{
					"2 January 2006",
					"January 2, 2006",
					"02/01/2006",
					"2006-01-02",
					"Jan 2, 2006",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						break
					}
				}
			}
		}

		// Extract excerpt/summary
		excerpt := ""
		excerptElem := article.Find("p, .excerpt, .summary, .description").First()
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.Text())
		}

		out = append(out, Headline{
			Source:      "OIES",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] OIES: collected %d headlines\n", len(out))
	}

	return out, nil
}
