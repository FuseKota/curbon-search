// =============================================================================
// sources_rss.go - RSSフィードソース（非日本語）
// =============================================================================
//
// このファイルはRSSフィードを使用する非日本語ニュースソースを定義します。
// gofeed ライブラリを使用してRSS/Atomフィードを解析します。
//
// 【含まれるソース】
//   1. Politico EU - EU政策・エネルギー・気候変動ニュース
//   2. Euractiv ETS - EU ETS関連ニュース
//   3. UK ETS - UK政府ETS関連ニュース（Atom Feed）
//   4. UN News Climate - 国連ニュース気候変動セクション
//   5. Carbon Market Watch - カーボン市場監視NGO
//
// =============================================================================
package pipeline

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// collectHeadlinesPoliticoEU は Politico EU の Energy & Climate セクションから記事を収集
//
// Politico EU は欧州の政治・政策ニュースを専門とするメディアで、
// エネルギー政策、気候変動対策、EU規制などを詳細にカバーしている。
// RSSフィードからEnergy and Climateセクションの記事を取得。
//
// 手法: RSS Feed (gofeed)
// URL: https://www.politico.eu/section/energy/feed/
//
// 引数:
//
//	limit: 収集する最大記事数
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
func collectHeadlinesPoliticoEU(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.politico.eu/section/energy/feed/"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Politico EU RSS feed")
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

		// URLからトラッキングパラメータを除去
		articleURL := item.Link
		if idx := strings.Index(articleURL, "?utm_"); idx > 0 {
			articleURL = articleURL[:idx]
		}

		// 日付のパース（取得できない場合は空文字列）
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// 記事の全文を取得（content:encoded から、なければ description から）
		excerpt := extractRSSExcerpt(item)

		out = append(out, Headline{
			Source:      "Politico EU",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	// Return empty slice if no articles found (not an error)
	return out, nil
}

// carbonKeywordsEuractiv contains keywords for filtering Euractiv articles
// to focus on carbon/climate-related content.
// Categories from RSS items are also checked against these keywords.
// Note: "ets" is avoided as a standalone keyword because it matches substrings
// like "Markets", "bets", "Metsola" etc. Use specific forms instead.
var carbonKeywordsEuractiv = []string{
	"carbon", "emission", "climate", "co2", "greenhouse",
	"net zero", "net-zero", "decarbonisation", "decarbonization",
	"green deal", "fit for 55", "cbam", "carbon border",
	"renewable", "energy transition", "paris agreement",
	"methane", "carbon market", "carbon price", "carbon tax",
	"energy", "environment", "sustainability",
	"eu ets", "ets2", "emissions trading", "uk ets",
}

// reEuactivSpaces normalizes whitespace in scraped Euractiv article text
var reEuractiveSpaces = regexp.MustCompile(`\s+`)

// collectHeadlinesEuractiv fetches articles from Euractiv main RSS feed,
// filters for carbon/climate-related content using title+description+categories,
// and scrapes article pages for full-text excerpts.
//
// Euractiv is a European news site focusing on EU policy.
// Note: Section-specific feeds (like /section/emissions-trading-scheme/feed/)
// are protected by Cloudflare, so we use the main feed with keyword filtering.
// Article pages are accessible via Go's http.Client for content extraction.
//
// URL: https://www.euractiv.com/feed/
func collectHeadlinesEuractiv(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// Use main feed (section feeds are Cloudflare-protected)
	feedURL := "https://www.euractiv.com/feed/"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Euractiv RSS feed")
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

		// Get RSS description for keyword filtering
		rssExcerpt := extractRSSExcerpt(item)

		// Include categories in keyword matching for better recall
		catStr := strings.Join(item.Categories, " ")

		// Filter by keywords using title + description + categories
		if !matchesKeywords(title, rssExcerpt+" "+catStr, carbonKeywordsEuractiv) {
			continue
		}

		// Remove UTM tracking parameters
		articleURL := item.Link
		if idx := strings.Index(articleURL, "?utm_"); idx > 0 {
			articleURL = articleURL[:idx]
		}

		// Scrape article page for full-text excerpt
		excerpt := fetchEuractivArticleExcerpt(client, articleURL, cfg.UserAgent)
		if excerpt == "" {
			// Fall back to RSS description if scraping fails
			excerpt = rssExcerpt
		}

		// Parse date
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Euractiv",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}

// fetchEuractivArticleExcerpt scrapes an Euractiv article page for body text.
// Returns empty string if the page is paywalled (Euractiv Pro) or inaccessible.
func fetchEuractivArticleExcerpt(client *http.Client, articleURL, userAgent string) string {
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

	// Detect paywall (Euractiv Pro articles have Lorem ipsum placeholder)
	content := doc.Find("div.c-news-detail__content")
	if content.Length() == 0 {
		return ""
	}

	// Remove unwanted elements (ads, scripts, etc.)
	content.Find("script, style, .ad-container, aside, .c-news-detail__subscribe").Remove()

	text := strings.TrimSpace(content.Text())
	text = reEuractiveSpaces.ReplaceAllString(text, " ")

	// Check for paywall markers
	if strings.Contains(text, "Lorem ipsum") || strings.Contains(text, "…Subscribe now") {
		return ""
	}

	return text
}

// collectHeadlinesUKETS fetches articles from UK Government ETS Atom feed
//
// The UK Emissions Trading Scheme page provides official government updates
// on the UK ETS policy and implementation.
//
// URL: https://www.gov.uk/government/publications.atom?topics%5B%5D=uk-emissions-trading-scheme
func collectHeadlinesUKETS(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.gov.uk/government/publications.atom?topics%5B%5D=uk-emissions-trading-scheme"

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

		articleURL := item.Link

		// Parse date
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// Get summary/content
		excerpt := ""
		if item.Description != "" {
			excerpt = cleanHTMLTags(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}

		out = append(out, Headline{
			Source:      "UK ETS",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}

// collectHeadlinesUNNews fetches articles from UN News Climate Change RSS feed
//
// UN News is the official news service of the United Nations, covering
// global climate change news, UNFCCC meetings, and international climate policy.
// This serves as an alternative to scraping unfccc.int directly.
//
// URL: https://news.un.org/feed/subscribe/en/news/topic/climate-change/feed/rss.xml
func collectHeadlinesUNNews(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://news.un.org/feed/subscribe/en/news/topic/climate-change/feed/rss.xml"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in UN News RSS feed")
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
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// Get content - UN News usually has good descriptions
		excerpt := extractRSSExcerpt(item)

		out = append(out, Headline{
			Source:      "UN News",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}

// collectHeadlinesCarbonBrief collects headlines from Carbon Brief RSS feed
//
// Carbon Brief is a UK-based website covering the latest developments in
// climate science, climate policy and energy policy. Previously used WordPress
// REST API, but switched to RSS for content:encoded full article extraction.
//
// URL: https://www.carbonbrief.org/feed/
func collectHeadlinesCarbonBrief(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.carbonbrief.org/feed/"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Carbon Brief RSS feed")
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

		// Full article content from content:encoded, fallback to description.
		excerpt := extractRSSExcerpt(item)
		// Truncate to 1000 chars for Notion AI summarization
		excerpt = truncateString(excerpt, 1000)

		out = append(out, Headline{
			Source:      "Carbon Brief",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}

// collectHeadlinesCarbonMarketWatch collects headlines from Carbon Market Watch RSS feed
//
// Carbon Market Watch is a Brussels-based NGO that monitors carbon markets
// and advocates for fair and effective climate policy. Their website blocks
// direct HTML scraping (403), but the WordPress RSS feed is accessible.
// The feed includes full article content via content:encoded.
//
// URL: https://carbonmarketwatch.org/feed/
func collectHeadlinesCarbonMarketWatch(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://carbonmarketwatch.org/feed/"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Carbon Market Watch RSS feed")
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

		// Full article content from content:encoded, fallback to description
		excerpt := extractRSSExcerpt(item)

		out = append(out, Headline{
			Source:      "Carbon Market Watch",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	return out, nil
}
