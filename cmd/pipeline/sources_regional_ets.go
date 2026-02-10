// =============================================================================
// sources_regional_ets.go - Regional Emissions Trading System Sources
// =============================================================================
//
// This file defines sources for regional emissions trading systems and
// regulatory bodies.
//
// Sources:
//   1. EU ETS (EC)      - European Commission ETS news
//   2. California CARB  - California Air Resources Board
//   3. RGGI             - Regional Greenhouse Gas Initiative
//   4. Australia CER    - Clean Energy Regulator
//   5. UK ETS           - UK Government ETS publications (HTML scraping)
//
// =============================================================================
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ledongthuc/pdf"
)

// =============================================================================
// PDF Text Extraction Helper
// =============================================================================

// extractTextFromPDF downloads a PDF from the given URL and extracts its text content
func extractTextFromPDF(pdfURL string, client *http.Client, userAgent string) (string, error) {
	req, err := http.NewRequest("GET", pdfURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Read PDF content into memory
	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	// Create a reader from the PDF data
	reader := bytes.NewReader(pdfData)
	pdfReader, err := pdf.NewReader(reader, int64(len(pdfData)))
	if err != nil {
		return "", fmt.Errorf("failed to parse PDF: %w", err)
	}

	// Extract text from all pages
	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	// Clean up the extracted text
	result := textBuilder.String()
	result = strings.TrimSpace(result)
	// Normalize whitespace
	result = strings.Join(strings.Fields(result), " ")

	return result, nil
}

// =============================================================================
// EU ETS (European Commission) Source
// =============================================================================

// collectHeadlinesEUETS fetches news from European Commission ETS page
//
// The European Commission's climate action site provides official news and
// updates about the EU Emissions Trading System.
func collectHeadlinesEUETS(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://climate.ec.europa.eu/news-other-reads/news_en"

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

	// EC site uses news item cards
	doc.Find("article, .ecl-card, .news-item, div[class*='news'], div[class*='listing-item']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Find title link
		titleLink := article.Find("h2 a, h3 a, .ecl-card__title a, .title a, a[class*='title']").First()
		if titleLink.Length() == 0 {
			titleLink = article.Find("a").First()
		}

		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
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

		// Extract date from listing page first
		dateStr := ""
		foundDate := false
		dateElem := article.Find("time, .date, .ecl-date-block, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"2 January 2006",
					"02/01/2006",
					"2006-01-02",
					"02 January 2006",
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
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Remove unwanted elements
					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related").Remove()

					// Try to extract date from article page if not found
					if !foundDate {
						articleDoc.Find("time, .date, meta[property='article:published_time']").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							if datetime, exists := elem.Attr("datetime"); exists {
								dateStr = datetime
								foundDate = true
							} else if content, exists := elem.Attr("content"); exists {
								dateStr = content
								foundDate = true
							}
						})
					}

					// Extract content from article body
					contentSelectors := []string{
						".ecl-editor",
						".ecl-page-content",
						"article .content",
						".field--name-body",
						"main article",
						".page-content",
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

					// Fallback: get all paragraphs from main content
					if excerpt == "" {
						var paragraphs []string
						articleDoc.Find("main p, article p").Each(func(_ int, p *goquery.Selection) {
							text := strings.TrimSpace(p.Text())
							if len(text) > 40 {
								paragraphs = append(paragraphs, text)
							}
						})
						if len(paragraphs) > 0 {
							excerpt = strings.Join(paragraphs, "\n\n")
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
			Source:      "EU ETS",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] EU ETS: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// California CARB Source
// =============================================================================

// collectHeadlinesCARB fetches news from California Air Resources Board
//
// CARB manages California's cap-and-trade program and publishes news about
// emissions regulations and climate policy.
func collectHeadlinesCARB(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://ww2.arb.ca.gov/news"

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

	// CARB news listing
	doc.Find("article, .news-item, .views-row, div[class*='node--type-news']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Find title
		titleLink := article.Find("h2 a, h3 a, .field--name-title a, a[href*='/news/']").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
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

		// Extract date
		dateStr := ""
		dateElem := article.Find("time, .date, .field--name-created, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"January 2, 2006",
					"Jan 2, 2006",
					"01/02/2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						break
					}
				}
			}
		}

		// Fetch individual article page for full body content
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Remove nav, header, footer, sidebar elements
					articleDoc.Find("header, footer, nav, aside, .sidebar, script, style, .breadcrumb").Remove()

					// Extract content from main element
					mainContent := articleDoc.Find("main#main-content, main, article, .content")
					if mainContent.Length() > 0 {
						// Get all paragraph text
						var paragraphs []string
						mainContent.Find("p").Each(func(_ int, p *goquery.Selection) {
							text := strings.TrimSpace(p.Text())
							if len(text) > 20 {
								paragraphs = append(paragraphs, text)
							}
						})
						excerpt = strings.Join(paragraphs, "\n\n")
					}
				}
			}
		}

		// Fallback to listing page excerpt if article fetch failed
		if excerpt == "" {
			excerptElem := article.Find("p, .field--name-body, .summary, .teaser").First()
			if excerptElem.Length() > 0 {
				excerpt = strings.TrimSpace(excerptElem.Text())
			}
		}

		out = append(out, Headline{
			Source:      "CARB",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] CARB: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// RGGI Source
// =============================================================================

// collectHeadlinesRGGI fetches news from Regional Greenhouse Gas Initiative
//
// RGGI is a cooperative effort among Eastern US states to cap and reduce
// power sector CO2 emissions.
func collectHeadlinesRGGI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.rggi.org/news-releases/rggi-releases"

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

	// RGGI uses a table structure with rows for each news item
	doc.Find("table.table tbody tr").Each(func(_ int, row *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Find link and title in the body cell
		bodyCell := row.Find("td.views-field-body")
		link := bodyCell.Find("a").First()

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Extract description from body cell (text after the link)
		listingDescription := ""
		bodyCellText := strings.TrimSpace(bodyCell.Text())
		if bodyCellText != "" && bodyCellText != title {
			// Remove the title from the body cell text to get the description
			listingDescription = strings.TrimSpace(strings.TrimPrefix(bodyCellText, title))
		}

		// Extract date from time element
		dateStr := ""
		foundDate := false
		timeElem := row.Find("time")
		if timeElem.Length() > 0 {
			if datetime, exists := timeElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			}
		}

		// Extract type from type cell
		typeCell := row.Find("td.views-field-field-item-type")
		itemType := strings.TrimSpace(typeCell.Text())

		// Fetch content from article page or PDF
		excerpt := ""
		isPDF := strings.HasSuffix(strings.ToLower(articleURL), ".pdf")

		if isPDF {
			// Extract text from PDF
			pdfText, err := extractTextFromPDF(articleURL, client, cfg.UserAgent)
			if err == nil && len(pdfText) > 50 {
				// Limit PDF text to reasonable length
				if len(pdfText) > 2000 {
					pdfText = pdfText[:2000] + "..."
				}
				excerpt = pdfText
			}
		} else {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						// Remove unwanted elements
						articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar").Remove()

						// Try to extract date if not found
						if !foundDate {
							articleDoc.Find("time").Each(func(_ int, elem *goquery.Selection) {
								if foundDate {
									return
								}
								if datetime, exists := elem.Attr("datetime"); exists {
									dateStr = datetime
									foundDate = true
								}
							})
						}

						// Extract content from main content area
						contentSelectors := []string{
							".field--name-body",
							".content",
							"article",
							"main",
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
					}
				}
			}
		}

		// Fallback: use listing description or type as excerpt
		if excerpt == "" {
			if listingDescription != "" {
				// Use description from listing page
				excerpt = listingDescription
				if strings.HasSuffix(strings.ToLower(articleURL), ".pdf") {
					excerpt = "[PDF] " + excerpt
				}
			} else if strings.HasSuffix(strings.ToLower(articleURL), ".pdf") {
				excerpt = "PDF Document - Type: " + itemType
			} else {
				excerpt = "Type: " + itemType
			}
		}

		// Fallback to current time if no date found
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "RGGI",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] RGGI: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// Australia CER Source
// =============================================================================

// collectHeadlinesAustraliaCER fetches news from Australia Clean Energy Regulator
//
// The CER is the Australian Government agency responsible for administering
// climate change laws including the Emissions Reduction Fund.
func collectHeadlinesAustraliaCER(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://cer.gov.au/news-and-media/news"

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

	// Australia CER uses cer-card class for news items
	doc.Find("div.cer-card.news, article.cer-card").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Find title in cer-card__heading
		headingElem := article.Find(".cer-card__heading a, h2 a, h3 a").First()
		title := strings.TrimSpace(headingElem.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find(".cer-card__heading, h2, h3").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := headingElem.Attr("href")
		if !exists || href == "" {
			// Try finding any link
			anyLink := article.Find("a[href]").First()
			href, exists = anyLink.Attr("href")
		}
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// Extract date from cer-card__changed
		dateStr := ""
		foundDate := false
		dateElem := article.Find(".cer-card__changed, time, .date")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"2 January 2006",
					"02/01/2006",
					"2 Jan 2006",
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
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Remove unwanted elements
					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related").Remove()

					// Try to extract date from article page if not found
					if !foundDate {
						articleDoc.Find("time, .date").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							if datetime, exists := elem.Attr("datetime"); exists {
								dateStr = datetime
								foundDate = true
							}
						})
					}

					// Extract content from article body (paragraphs and list items)
					contentSelectors := []string{
						".field--name-body",
						".content",
						"article .body",
						"main article",
						".page-content",
					}
					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							var contentParts []string
							// Extract paragraphs and list items
							contentElem.Find("p, li").Each(func(_ int, elem *goquery.Selection) {
								text := strings.TrimSpace(elem.Text())
								if len(text) > 20 {
									// Add bullet for list items
									if goquery.NodeName(elem) == "li" {
										text = "• " + text
									}
									contentParts = append(contentParts, text)
								}
							})
							if len(contentParts) > 0 {
								excerpt = strings.Join(contentParts, "\n\n")
								break
							}
						}
					}

					// Fallback: try all paragraphs and list items from main
					if excerpt == "" {
						var contentParts []string
						articleDoc.Find("main p, main li, article p, article li").Each(func(_ int, elem *goquery.Selection) {
							text := strings.TrimSpace(elem.Text())
							if len(text) > 30 {
								if goquery.NodeName(elem) == "li" {
									text = "• " + text
								}
								contentParts = append(contentParts, text)
							}
						})
						if len(contentParts) > 0 {
							excerpt = strings.Join(contentParts, "\n\n")
						}
					}
				}
			}
		}

		// Fallback to listing page excerpt if no content found
		if excerpt == "" {
			bodyElem := article.Find(".cer-card__body, p, .summary")
			if bodyElem.Length() > 0 {
				excerpt = strings.TrimSpace(bodyElem.First().Text())
			}
		}

		// Fallback to current time if no date found
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Australia CER",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Australia CER: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// UK ETS Source
// =============================================================================

// collectHeadlinesUKETSHTML fetches news from UK Government ETS publications
//
// The UK Emissions Trading Scheme is managed by the UK ETS Authority
// (a joint body of UK, Scottish, Welsh governments and NI Executive).
// This scrapes gov.uk search results for UK ETS related publications.
func collectHeadlinesUKETSHTML(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// Search gov.uk for UK ETS publications and news
	searchURL := "https://www.gov.uk/search/all?keywords=%22UK+Emissions+Trading+Scheme%22&order=updated-newest"

	client := cfg.Client
	req, err := http.NewRequest("GET", searchURL, nil)
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

	// gov.uk search results use gem-c-document-list__item for each result
	doc.Find("li.gem-c-document-list__item, .gem-c-document-list__item, div.finder-results li").Each(func(_ int, item *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// Find title link
		link := item.Find("a.gem-c-document-list__item-title, a[data-track-category='navFinderLinkClicked']").First()
		if link.Length() == 0 {
			link = item.Find("a").First()
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(searchURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		// Filter: only include UK ETS related content
		titleLower := strings.ToLower(title)
		if !strings.Contains(titleLower, "ets") &&
			!strings.Contains(titleLower, "emissions trading") &&
			!strings.Contains(titleLower, "carbon") {
			return
		}

		seen[articleURL] = true

		// Extract date from metadata
		dateStr := ""
		foundDate := false
		metaElem := item.Find(".gem-c-document-list__attribute, .document-list-item-metadata")
		if metaElem.Length() > 0 {
			metaText := strings.TrimSpace(metaElem.Text())
			// Look for "Updated: DD Month YYYY" or similar
			if strings.Contains(metaText, "Updated:") {
				dateText := strings.TrimPrefix(metaText, "Updated:")
				dateText = strings.TrimSpace(dateText)
				for _, format := range []string{
					"2 January 2006",
					"02 January 2006",
					"January 2, 2006",
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
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Remove unwanted elements
					articleDoc.Find("header, footer, nav, script, style, noscript, .gem-c-contextual-sidebar").Remove()

					// Try to extract date from article page if not found
					if !foundDate {
						articleDoc.Find("time, .gem-c-metadata__definition").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							if datetime, exists := elem.Attr("datetime"); exists {
								dateStr = datetime
								foundDate = true
							} else {
								text := strings.TrimSpace(elem.Text())
								for _, format := range []string{
									"2 January 2006",
									"02 January 2006",
									"2006-01-02",
								} {
									if t, err := time.Parse(format, text); err == nil {
										dateStr = t.Format(time.RFC3339)
										foundDate = true
										break
									}
								}
							}
						})
					}

					// Extract content from gov.uk page structure
					contentSelectors := []string{
						".gem-c-govspeak",
						".govuk-govspeak",
						".publication-content",
						"main .content",
						"article",
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

					// Fallback: try meta description
					if excerpt == "" {
						metaDesc := articleDoc.Find("meta[name='description']")
						if metaDesc.Length() > 0 {
							excerpt, _ = metaDesc.Attr("content")
							excerpt = strings.TrimSpace(excerpt)
						}
					}
				}
			}
		}

		// Fallback to listing page description if no content found
		if excerpt == "" {
			descElem := item.Find(".gem-c-document-list__item-description, p")
			if descElem.Length() > 0 {
				excerpt = strings.TrimSpace(descElem.First().Text())
			}
		}

		// Fallback to current time if no date found
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "UK ETS",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] UK ETS: collected %d headlines\n", len(out))
	}

	return out, nil
}
