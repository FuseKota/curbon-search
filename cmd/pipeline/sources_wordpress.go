// =============================================================================
// sources_wordpress.go - WordPress REST API ソース
// =============================================================================
//
// このファイルはWordPress REST APIを使用するニュースソースを定義します。
// 全てのソースは headlines.go の collectWordPressHeadlines() を使用します。
//
// 【含まれるソース】
//   1. CarbonCredits.jp    - 日本のカーボンクレジット情報
//   2. Carbon Herald       - CDR技術ニュース
//   3. Climate Home News   - 国際気候政策
//   4. CarbonCredits.com   - 教育・啓発コンテンツ
//   5. Sandbag             - EU ETSアナリスト
//   6. Ecosystem Marketplace - 自然気候ソリューション
//   7. Carbon Brief        - 気候科学・政策
//   8. RMI                 - エネルギー転換シンクタンク
//
// =============================================================================
package main

import (
	"fmt"
	"os"
	"strings"
)

// collectHeadlinesCarbonCreditsJP collects headlines from carboncredits.jp using WordPress REST API
func collectHeadlinesCarbonCreditsJP(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlines("https://carboncredits.jp", "CarbonCredits.jp", limit, cfg)
}

// collectHeadlinesCarbonHerald collects headlines from carbonherald.com using WordPress REST API
func collectHeadlinesCarbonHerald(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlines("https://carbonherald.com", "Carbon Herald", limit, cfg)
}

// collectHeadlinesClimateHomeNews collects headlines from climatechangenews.com using WordPress REST API
func collectHeadlinesClimateHomeNews(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlines("https://www.climatechangenews.com", "Climate Home News", limit, cfg)
}

// collectHeadlinesCarbonCreditscom collects headlines from carboncredits.com using WordPress REST API
func collectHeadlinesCarbonCreditscom(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlines("https://carboncredits.com", "CarbonCredits.com", limit, cfg)
}

// collectHeadlinesSandbag fetches articles from Sandbag using WordPress REST API
func collectHeadlinesSandbag(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlines("https://sandbag.be", "Sandbag", limit, cfg)
}

// collectHeadlinesEcosystemMarketplace fetches articles from Ecosystem Marketplace using WordPress REST API
//
// 注意: Ecosystem Marketplaceは記事をカスタム投稿タイプ「featured-articles」に保存している。
// 標準の「posts」エンドポイントには2011-2017年の古いアーカイブしかないため、
// featured-articlesエンドポイントを使用する。
func collectHeadlinesEcosystemMarketplace(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlinesCustomType(
		"https://www.ecosystemmarketplace.com",
		"Ecosystem Marketplace",
		"featured-articles", // カスタム投稿タイプ
		limit,
		cfg,
	)
}

// collectHeadlinesCarbonBrief fetches articles from Carbon Brief using WordPress REST API
func collectHeadlinesCarbonBrief(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	return collectWordPressHeadlines("https://www.carbonbrief.org", "Carbon Brief", limit, cfg)
}

// collectHeadlinesRMI fetches articles from RMI (Rocky Mountain Institute)
//
// RMIはGutenbergブロック（Datawrapperチャート等）を多用しており、
// WordPress REST APIのcontent.renderedでは記事本文が途中で切れる。
// そのため、APIで記事一覧を取得し、各ページをスクレイピングして全文を取得する。
func collectHeadlinesRMI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// WordPress APIで記事一覧（タイトル・URL・日付）を取得
	apiURL := fmt.Sprintf("https://rmi.org/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date_gmt", limit)

	var posts []WPPost
	if err := httpGetJSON(apiURL, cfg, &posts); err != nil {
		return nil, fmt.Errorf("failed to fetch RMI API: %w", err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		publishedAt := ""
		if p.DateGMT != "" {
			publishedAt = p.DateGMT + "Z"
		}

		// 各記事ページをスクレイピングして全文取得
		// RMIには2つのテンプレートがある:
		//   新: div.my-12.single_news_content-wrapper に本文
		//   旧: div.single_news_content-wrapper 内に直接 <p> タグ
		excerpt := ""
		doc, err := fetchDoc(p.Link, cfg)
		if err == nil {
			sel := doc.Find("div.my-12.single_news_content-wrapper")
			if sel.Length() == 0 {
				// 旧テンプレート: 不要要素を除去して本文のみ取得
				sel = doc.Find("div.single_news_content-wrapper")
				sel.Find("div.blog_social").Remove()
				sel.Find("div.single_news_content_meta").Remove()
				sel.Find("h3").First().Remove()
				sel.Find("h6").First().Remove()
			}
			if sel.Length() > 0 {
				sel.Find("script, style, iframe, svg").Remove()
				excerpt = cleanExtractedText(sel.Text())
			}
		}
		if os.Getenv("DEBUG_SCRAPING") != "" && err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] RMI: failed to fetch page %s: %v\n", p.Link, err)
		}

		out = append(out, Headline{
			Source:      "RMI",
			Title:       title,
			URL:         p.Link,
			PublishedAt: publishedAt,
			Excerpt:     excerpt,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] RMI: collected %d headlines\n", len(out))
	}

	return out, nil
}
