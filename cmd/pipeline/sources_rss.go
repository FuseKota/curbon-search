// =============================================================================
// sources_rss.go - RSSフィードソース（非日本語）
// =============================================================================
//
// このファイルはRSSフィードを使用する非日本語ニュースソースを定義します。
// gofeed ライブラリを使用してRSS/Atomフィードを解析します。
//
// 【含まれるソース】
//   1. Politico EU - EU政策・エネルギー・気候変動ニュース
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
