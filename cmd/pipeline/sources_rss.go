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
//
// =============================================================================
package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
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
func collectHeadlinesPoliticoEU(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.politico.eu/section/energy/feed/"

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

	// RSSフィードをパース
	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RSS parse failed: %w", err)
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

		// 日付のパース
		dateStr := time.Now().Format(time.RFC3339)
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// 記事の全文を取得（content:encoded から、なければ description から）
		// Notionに保存する際に全文が必要なため、切り詰めない
		excerpt := ""
		if item.Content != "" {
			// content:encoded から全文を取得（HTMLタグを除去）
			excerpt = cleanHTMLTags(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		} else if item.Description != "" {
			// content がない場合は description を使用
			excerpt = cleanHTMLTags(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}

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
// to focus on carbon/climate-related content
var carbonKeywordsEuractiv = []string{
	"carbon", "emission", "ets", "climate", "co2", "greenhouse",
	"net zero", "net-zero", "decarbonisation", "decarbonization",
	"green deal", "fit for 55", "cbam", "carbon border",
	"renewable", "energy transition", "paris agreement",
	"methane", "carbon market", "carbon price", "carbon tax",
}

// collectHeadlinesEuractiv fetches articles from Euractiv main RSS feed
// and filters for carbon/climate-related content
//
// Euractiv is a European news site focusing on EU policy.
// Note: Section-specific feeds (like /section/emissions-trading-scheme/feed/)
// are protected by Cloudflare, so we use the main feed with keyword filtering.
//
// URL: https://www.euractiv.com/feed/
func collectHeadlinesEuractiv(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// Use main feed (section feeds are Cloudflare-protected)
	feedURL := "https://www.euractiv.com/feed/"

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

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Euractiv RSS feed")
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
		excerpt := ""
		if item.Content != "" {
			excerpt = cleanHTMLTags(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		} else if item.Description != "" {
			excerpt = cleanHTMLTags(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}

		// Filter by keywords - main feed has all topics
		titleLower := strings.ToLower(title)
		excerptLower := strings.ToLower(excerpt)
		hasKeyword := false
		for _, kw := range carbonKeywordsEuractiv {
			if strings.Contains(titleLower, kw) || strings.Contains(excerptLower, kw) {
				hasKeyword = true
				break
			}
		}
		if !hasKeyword {
			continue
		}

		// Remove UTM tracking parameters
		articleURL := item.Link
		if idx := strings.Index(articleURL, "?utm_"); idx > 0 {
			articleURL = articleURL[:idx]
		}

		// Parse date
		dateStr := time.Now().Format(time.RFC3339)
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

// collectHeadlinesUKETS fetches articles from UK Government ETS Atom feed
//
// The UK Emissions Trading Scheme page provides official government updates
// on the UK ETS policy and implementation.
//
// URL: https://www.gov.uk/government/publications.atom?topics%5B%5D=uk-emissions-trading-scheme
func collectHeadlinesUKETS(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.gov.uk/government/publications.atom?topics%5B%5D=uk-emissions-trading-scheme"

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

		// Parse date
		dateStr := time.Now().Format(time.RFC3339)
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
func collectHeadlinesUNNews(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://news.un.org/feed/subscribe/en/news/topic/climate-change/feed/rss.xml"

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
		dateStr := time.Now().Format(time.RFC3339)
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// Get content - UN News usually has good descriptions
		excerpt := ""
		if item.Content != "" {
			excerpt = cleanHTMLTags(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		} else if item.Description != "" {
			excerpt = cleanHTMLTags(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}

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
