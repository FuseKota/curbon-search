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
//
// =============================================================================
package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// =============================================================================
// EU ETS (European Commission) Source
// =============================================================================

// collectHeadlinesEUETS fetches news from European Commission ETS page
//
// The European Commission's climate action site provides official news and
// updates about the EU Emissions Trading System.
func collectHeadlinesEUETS(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://climate.ec.europa.eu/news-other-reads/news_en"

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

		// Extract date
		dateStr := time.Now().Format(time.RFC3339)
		dateElem := article.Find("time, .date, .ecl-date-block, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
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
						break
					}
				}
			}
		}

		// Extract excerpt
		excerpt := ""
		excerptElem := article.Find("p, .ecl-card__description, .description, .summary").First()
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.Text())
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
		dateStr := time.Now().Format(time.RFC3339)
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

		// Extract excerpt
		excerpt := ""
		excerptElem := article.Find("p, .field--name-body, .summary, .teaser").First()
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.Text())
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

		// Extract date from time element
		dateStr := time.Now().Format(time.RFC3339)
		timeElem := row.Find("time")
		if timeElem.Length() > 0 {
			if datetime, exists := timeElem.Attr("datetime"); exists {
				dateStr = datetime
			}
		}

		// Extract type from type cell
		typeCell := row.Find("td.views-field-field-item-type")
		itemType := strings.TrimSpace(typeCell.Text())

		out = append(out, Headline{
			Source:      "RGGI",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     "Type: " + itemType,
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
		dateStr := time.Now().Format(time.RFC3339)
		dateElem := article.Find(".cer-card__changed, time, .date")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
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
						break
					}
				}
			}
		}

		// Extract excerpt from cer-card__body
		excerpt := ""
		bodyElem := article.Find(".cer-card__body, p, .summary")
		if bodyElem.Length() > 0 {
			excerpt = strings.TrimSpace(bodyElem.First().Text())
		}

		// Extract tags
		tagsElem := article.Find(".cer-card__tags")
		if tagsElem.Length() > 0 {
			tags := strings.TrimSpace(tagsElem.Text())
			if tags != "" && excerpt == "" {
				excerpt = "Tags: " + tags
			}
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
