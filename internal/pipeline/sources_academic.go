// =============================================================================
// sources_academic.go - Academic/Research Sources
// =============================================================================
//
// This file defines academic and research publication sources for carbon-related
// content using XML APIs and RSS feeds.
//
// Sources:
//   1. arXiv                      - Pre-print repository (XML API)
//   2. Nature Communications      - Scientific journal (RSS + keyword filter)
//   3. OIES                       - Oxford Institute for Energy Studies (HTML)
//   4. IOP Science (ERL)          - Environmental Research Letters (RSS + keyword filter)
//   5. Nature Ecology & Evolution - Scientific journal (RSS + keyword filter)
//   6. ScienceDirect              - Elsevier journal (RSS + keyword filter)
//
// =============================================================================
package pipeline

import (
	"encoding/xml"
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

// reScienceDirectDate extracts "Publication date: Month Year" from ScienceDirect description HTML
var reScienceDirectDate = regexp.MustCompile(`Publication date:\s*(\w+ \d{4})`)

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
	Title     string        `xml:"title"`
	ID        string        `xml:"id"`
	Published string        `xml:"published"`
	Updated   string        `xml:"updated"`
	Summary   string        `xml:"summary"`
	Authors   []arXivAuthor `xml:"author"`
	Links     []arXivLink   `xml:"link"`
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
func collectHeadlinesArXiv(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
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

	client := cfg.Client
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
		if !matchesKeywords(title, summaryClean, carbonKeywordsArXiv) {
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
// of natural sciences. We use the climate-change subject feed which is pre-filtered
// by Nature's subject taxonomy, so no additional keyword filtering is needed.
//
// The subject feed provides titles, URLs, and dates but no descriptions.
// Abstracts are fetched from each article page (id="Abs1" section).
//
// Note: nature.com uses Fastly bot protection that detects Go's TLS fingerprint
// and returns a JavaScript challenge page. We use curl as a workaround since
// curl's TLS fingerprint is accepted by the server.
//
// URL: https://www.nature.com/subjects/climate-change/ncomms.rss
func collectHeadlinesNatureComms(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.nature.com/subjects/climate-change/ncomms.rss"

	// Nature.com blocks Go's TLS fingerprint with a JS challenge page.
	// Use curl to fetch the RSS feed instead.
	body, err := fetchViaCurl(feedURL, cfg.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("curl fetch failed: %w", err)
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseString(body)
	if err != nil {
		return nil, fmt.Errorf("RSS parse failed: %w", err)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Nature Communications RSS feed")
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

		// Parse date
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// Fetch abstract from article page via curl
		excerpt := fetchNatureAbstract(articleURL, cfg.UserAgent)

		out = append(out, Headline{
			Source:      "Nature Communications",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}

// fetchNatureAbstract fetches the abstract from a Nature article page.
// Uses curl to bypass TLS fingerprint detection.
func fetchNatureAbstract(articleURL string, userAgent string) string {
	body, err := fetchViaCurl(articleURL, userAgent)
	if err != nil {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}

	// Extract abstract from Abs1 section
	abstract := doc.Find("#Abs1-content p, #Abs1 p").First().Text()
	return strings.TrimSpace(abstract)
}

// =============================================================================
// OIES (Oxford Institute for Energy Studies) Source
// =============================================================================

// collectHeadlinesOIES fetches publications from Oxford Institute for Energy Studies
//
// OIES publishes research papers on energy and environmental economics,
// including carbon markets and climate policy.
//
// Strategy: The main /publications/ page uses JavaScript rendering, so we fetch
// publications from multiple programme pages that render content server-side:
//   - Carbon Management Programme (primary - carbon/climate focused)
//   - Energy Transition Research Initiative
//   - Gas, Electricity, and other programmes
func collectHeadlinesOIES(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// Programme pages that render publications in HTML (not JavaScript)
	programmeURLs := []string{
		"https://www.oxfordenergy.org/carbon-management-programme/",
		"https://www.oxfordenergy.org/energy-transition-research-initiative/",
		"https://www.oxfordenergy.org/gas-programme/",
		"https://www.oxfordenergy.org/electricity-programme/",
	}

	client := cfg.Client
	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	for _, programmeURL := range programmeURLs {
		if len(out) >= limit {
			break
		}

		headlines, err := fetchOIESProgrammePage(client, programmeURL, cfg.UserAgent)
		if err != nil {
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] OIES: error fetching %s: %v\n", programmeURL, err)
			}
			continue
		}

		for _, h := range headlines {
			if len(out) >= limit {
				break
			}
			if seen[h.URL] {
				continue
			}
			seen[h.URL] = true

			// Fetch article page to get excerpt/content
			excerpt, date := fetchOIESArticleContent(client, h.URL, cfg.UserAgent)
			if excerpt != "" {
				h.Excerpt = excerpt
			}
			if date != "" {
				h.PublishedAt = date
			}

			out = append(out, h)
		}
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] OIES: collected %d headlines from %d programmes\n", len(out), len(programmeURLs))
	}

	return out, nil
}

// fetchOIESArticleContent fetches the excerpt and date from an individual article page
func fetchOIESArticleContent(client *http.Client, articleURL, userAgent string) (excerpt, date string) {
	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", ""
	}

	// Remove noise elements before extracting content
	doc.Find("nav, header, footer, aside, .sidebar, script, style, .menu, .navigation, form, .related, .related-products, .upsells, section.products").Remove()

	// Collect substantial paragraphs from the page
	var paragraphs []string
	doc.Find("p").Each(func(_ int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())

		// Skip short paragraphs
		if len(text) < 50 {
			return
		}

		// Skip paragraphs with noise patterns
		if strings.Count(text, "\t") > 2 || strings.Count(text, "\n") > 3 {
			return
		}

		// Skip truncated previews from related articles (end with [...] or ...)
		if strings.HasSuffix(text, "[…]") || strings.HasSuffix(text, "…]") {
			return
		}

		// Skip navigation/boilerplate text
		lowerText := strings.ToLower(text)
		if strings.Contains(lowerText, "cookie") ||
			strings.Contains(lowerText, "privacy policy") ||
			strings.Contains(lowerText, "sign up") ||
			strings.Contains(lowerText, "subscribe") ||
			strings.Contains(lowerText, "register your email") ||
			strings.Contains(lowerText, "notification of new") ||
			strings.HasPrefix(text, "By:") {
			return
		}

		paragraphs = append(paragraphs, text)
	})

	// Use paragraphs if found, otherwise fall back to meta description
	if len(paragraphs) > 0 {
		excerpt = strings.Join(paragraphs, "\n\n")
	} else {
		// Fallback to meta description
		if metaDesc, exists := doc.Find("meta[name='description']").Attr("content"); exists && metaDesc != "" {
			excerpt = strings.TrimSpace(metaDesc)
		}
		if excerpt == "" {
			if ogDesc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists && ogDesc != "" {
				excerpt = strings.TrimSpace(ogDesc)
			}
		}
	}

	// Clean up truncation markers
	excerpt = strings.TrimSuffix(excerpt, "[…]")
	excerpt = strings.TrimSuffix(excerpt, "…")
	excerpt = strings.TrimSuffix(excerpt, " [")
	excerpt = strings.TrimSpace(excerpt)

	// Truncate very long excerpts (2000 chars max)
	if len(excerpt) > 2000 {
		excerpt = excerpt[:1997] + "..."
	}

	// Try to get date from JSON-LD
	doc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
		text := script.Text()
		if dateMatch := reDatePublishedJSON.FindStringSubmatch(text); len(dateMatch) > 1 {
			if t, err := time.Parse("2006-01-02", dateMatch[1]); err == nil {
				date = t.Format(time.RFC3339)
			} else if t, err := time.Parse(time.RFC3339, dateMatch[1]); err == nil {
				date = t.Format(time.RFC3339)
			}
		}
	})

	return excerpt, date
}

// fetchOIESProgrammePage extracts publications from a single OIES programme page
func fetchOIESProgrammePage(client *http.Client, programmeURL, userAgent string) ([]Headline, error) {
	req, err := http.NewRequest("GET", programmeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

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

	var headlines []Headline

	// OIES programme pages list publications as links with dates
	// Look for links to /publications/ and /research/ URLs
	doc.Find("a[href*='/publications/'], a[href*='/research/']").Each(func(_ int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip navigation and category links
		if strings.Contains(href, "/publication-topic/") ||
			strings.Contains(href, "/publication-category/") ||
			strings.HasSuffix(href, "/publications/") ||
			strings.HasSuffix(href, "/research/") {
			return
		}

		articleURL := resolveURL(programmeURL, href)
		if articleURL == "" {
			return
		}

		// Get title from link text
		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		// Skip PDF download links (we want the article page)
		if strings.HasSuffix(strings.ToLower(href), ".pdf") {
			return
		}

		// Look for date near the link
		// OIES uses format like "22.01.26" (DD.MM.YY)
		dateStr := ""

		// Check parent elements for date
		parent := link.Parent()
		for i := 0; i < 3 && parent.Length() > 0; i++ {
			parentText := parent.Text()
			if d := parseOIESDate(parentText); d != "" {
				dateStr = d
				break
			}
			parent = parent.Parent()
		}

		// Filter out entries older than 2 years (only if date was found)
		if dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				if time.Since(t) > 2*365*24*time.Hour {
					return
				}
			}
		}

		headlines = append(headlines, Headline{
			Source:      "OIES",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			IsHeadline:  true,
		})
	})

	return headlines, nil
}

// parseOIESDate extracts date from text containing OIES date format (DD.MM.YY)
func parseOIESDate(text string) string {
	// OIES uses format like "22.01.26" for 22 January 2026
	// Look for pattern DD.MM.YY
	for i := 0; i < len(text)-7; i++ {
		if text[i] >= '0' && text[i] <= '9' &&
			text[i+1] >= '0' && text[i+1] <= '9' &&
			text[i+2] == '.' &&
			text[i+3] >= '0' && text[i+3] <= '9' &&
			text[i+4] >= '0' && text[i+4] <= '9' &&
			text[i+5] == '.' &&
			text[i+6] >= '0' && text[i+6] <= '9' &&
			text[i+7] >= '0' && text[i+7] <= '9' {

			dateCandidate := text[i : i+8]
			// Parse as DD.MM.YY
			t, err := time.Parse("02.01.06", dateCandidate)
			if err == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	return ""
}

// =============================================================================
// 学術ジャーナル共通キーワードリスト
// =============================================================================

// carbonKeywordsAcademic contains keywords for filtering academic journal articles
// to ensure relevance to carbon/climate topics. Shared by IOP Science (ERL),
// Nature Ecology & Evolution, and ScienceDirect sources.
var carbonKeywordsAcademic = []string{
	"carbon", "emission", "greenhouse", "climate change", "net zero",
	"decarbonization", "decarbonisation", "carbon dioxide", "CO2",
	"carbon pricing", "carbon tax", "cap and trade", "emissions trading",
	"carbon market", "carbon credit", "offset", "sequestration",
	"carbon capture", "CCS", "CCUS", "negative emissions",
	"global warming", "climate policy", "paris agreement",
	"renewable energy", "energy transition", "fossil fuel",
}

// =============================================================================
// IOP Science (Environmental Research Letters) Source
// =============================================================================

// collectHeadlinesIOPScience fetches articles from IOP Science Environmental Research Letters RSS
//
// Environmental Research Letters (ERL) is an open-access journal covering
// environmental science. We use the journal's RSS feed with keyword filtering
// to extract carbon/climate-related articles.
//
// Feed format: RDF/RSS 1.0 (gofeed handles automatically)
// URL: https://iopscience.iop.org/journal/rss/1748-9326
func collectHeadlinesIOPScience(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://iopscience.iop.org/journal/rss/1748-9326"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
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

		// Get content for keyword filtering
		excerpt := extractRSSExcerpt(item)

		// Keyword filter - ERL covers broad environmental science
		if !matchesKeywords(title, excerpt, carbonKeywordsAcademic) {
			continue
		}

		articleURL := item.Link

		// Parse date
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "IOP Science (ERL)",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IOP Science (ERL): collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// Nature Ecology & Evolution Source
// =============================================================================

// collectHeadlinesNatureEcoEvo fetches articles from Nature Ecology & Evolution RSS
//
// Nature Ecology & Evolution covers ecology and evolutionary biology.
// We use keyword filtering to extract carbon/climate-related articles.
//
// NOTE: Nature.com has bot protection that may block RSS requests (see Nature Comms).
// If blocked, this will return an empty slice gracefully.
//
// URL: https://www.nature.com/natecolevol.rss
func collectHeadlinesNatureEcoEvo(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.nature.com/natecolevol.rss"

	// Nature.com uses a cookie-based auth redirect (303 -> idp.nature.com -> back).
	// We need a client with a cookie jar to persist cookies across redirects.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: cfg.Client.Timeout,
		Jar:     jar,
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		// Nature.com may block with bot protection - return empty gracefully
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: request failed (possible bot protection): %v\n", err)
		}
		return []Headline{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: unexpected status %d (possible bot protection)\n", resp.StatusCode)
		}
		return []Headline{}, nil
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: RSS parse failed (possible HTML challenge page): %v\n", err)
		}
		return []Headline{}, nil
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

		// Get content for keyword filtering
		excerpt := extractRSSExcerpt(item)

		// Keyword filter
		if !matchesKeywords(title, excerpt, carbonKeywordsAcademic) {
			continue
		}

		articleURL := item.Link

		// Parse date
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Nature Eco&Evo",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// ScienceDirect Source
// =============================================================================

// collectHeadlinesScienceDirect fetches articles from ScienceDirect RSS feed
//
// ScienceDirect (Elsevier) hosts academic journals. We target the journal
// "Resources, Conservation & Recycling Advances" (ISSN 2950-631X) which covers
// sustainability and resource management topics.
//
// URL: https://rss.sciencedirect.com/publication/science/2950631X
func collectHeadlinesScienceDirect(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://rss.sciencedirect.com/publication/science/2950631X"

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

		// Get content for keyword filtering
		excerpt := extractRSSExcerpt(item)

		// Keyword filter
		if !matchesKeywords(title, excerpt, carbonKeywordsAcademic) {
			continue
		}

		articleURL := item.Link

		// Parse date - ScienceDirect RSS has no standard date fields,
		// but description contains "Publication date: Month Year"
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		} else if item.Description != "" {
			dateStr = parseScienceDirectDate(item.Description)
		}

		// Fetch abstract from article page (RSS only has metadata)
		if articleURL != "" {
			if abs := fetchScienceDirectAbstract(articleURL, client, cfg.UserAgent); abs != "" {
				excerpt = abs
			}
		}

		out = append(out, Headline{
			Source:      "ScienceDirect",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ScienceDirect: collected %d headlines\n", len(out))
	}

	return out, nil
}

// fetchScienceDirectAbstract fetches the article page and extracts the abstract text.
func fetchScienceDirectAbstract(articleURL string, client *http.Client, userAgent string) string {
	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", userAgent)

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

	// Abstract is inside <div class="abstract author">
	// (not "abstract author-highlights" or "abstract graphical")
	abs := doc.Find("div.abstract.author").Not(".author-highlights").First().Text()
	abs = strings.TrimSpace(abs)
	// Remove leading "Abstract" label
	abs = strings.TrimPrefix(abs, "Abstract")
	abs = strings.TrimSpace(abs)

	return abs
}

// parseScienceDirectDate extracts date from ScienceDirect description HTML.
// Input like: "<p>Publication date: March 2026</p>..." -> "2026-03-01T00:00:00Z"
func parseScienceDirectDate(desc string) string {
	m := reScienceDirectDate.FindStringSubmatch(desc)
	if m == nil {
		return ""
	}
	t, err := time.Parse("January 2006", m[1])
	if err != nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
