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
package main

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
func collectHeadlinesPoliticoEU(limit int, cfg headlineSourceConfig) ([]Headline, error) {
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

		})
	}

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// carbonKeywordsEuractiv は Euractiv 記事をカーボン・気候関連コンテンツに
// 絞り込むためのキーワード一覧。
// RSSアイテムのカテゴリもこのキーワードで照合する。
// 注意: "ets" は単独キーワードとして使わない。"Markets", "bets", "Metsola" 等の
// 部分文字列に一致してしまうため、具体的な形式を使用する。
var carbonKeywordsEuractiv = []string{
	"carbon", "emission", "climate", "co2", "greenhouse",
	"net zero", "net-zero", "decarbonisation", "decarbonization",
	"green deal", "fit for 55", "cbam", "carbon border",
	"renewable", "energy transition", "paris agreement",
	"methane", "carbon market", "carbon price", "carbon tax",
	"energy", "environment", "sustainability",
	"eu ets", "ets2", "emissions trading", "uk ets",
}

// reEuractiveSpaces は Euractiv 記事テキストの空白文字を正規化する
var reEuractiveSpaces = regexp.MustCompile(`\s+`)

// collectHeadlinesEuractiv は Euractiv のメインRSSフィードから記事を取得し、
// タイトル+説明文+カテゴリでカーボン・気候関連コンテンツをフィルタリングし、
// 記事ページをスクレイピングして全文Excerptを取得する。
//
// Euractiv はEU政策に特化した欧州ニュースサイト。
// 注意: セクション別フィード（/section/emissions-trading-scheme/feed/ 等）は
// Cloudflareで保護されているため、メインフィード+キーワードフィルタリングを使用。
// 記事ページはGoの http.Client でアクセス可能。
//
// URL: https://www.euractiv.com/feed/
func collectHeadlinesEuractiv(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// メインフィードを使用（セクション別フィードはCloudflare保護あり）
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

		// キーワードフィルタリング用にRSS descriptionを取得
		rssExcerpt := extractRSSExcerpt(item)

		// カテゴリもキーワードマッチングに含めて再現率を向上
		catStr := strings.Join(item.Categories, " ")

		// タイトル+説明文+カテゴリでキーワードフィルタリング
		if !matchesKeywords(title, rssExcerpt+" "+catStr, carbonKeywordsEuractiv) {
			continue
		}

		// UTMトラッキングパラメータを除去
		articleURL := item.Link
		if idx := strings.Index(articleURL, "?utm_"); idx > 0 {
			articleURL = articleURL[:idx]
		}

		// 記事ページをスクレイピングして全文Excerptを取得
		excerpt := fetchEuractivArticleExcerpt(client, articleURL, cfg.UserAgent)
		if excerpt == "" {
			// スクレイピング失敗時はRSS descriptionにフォールバック
			excerpt = rssExcerpt
		}

		// 日付のパース
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

		})
	}

	return out, nil
}

// fetchEuractivArticleExcerpt は Euractiv 記事ページから本文をスクレイピングする。
// ペイウォール（Euractiv Pro）またはアクセス不可の場合は空文字列を返す。
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

	// ペイウォール検出（Euractiv Pro記事はLorem ipsumプレースホルダーを含む）
	content := doc.Find("div.c-news-detail__content")
	if content.Length() == 0 {
		return ""
	}

	// 不要な要素を除去（広告、スクリプト等）
	content.Find("script, style, .ad-container, aside, .c-news-detail__subscribe").Remove()

	text := strings.TrimSpace(content.Text())
	text = reEuractiveSpaces.ReplaceAllString(text, " ")

	// ペイウォールマーカーを確認
	if strings.Contains(text, "Lorem ipsum") || strings.Contains(text, "…Subscribe now") {
		return ""
	}

	return text
}

// collectHeadlinesUKETS は UK政府 ETS Atomフィードから記事を取得する
//
// UK排出量取引制度ページは、UK ETSの政策と実施に関する
// 公式な政府アップデートを提供する。
//
// URL: https://www.gov.uk/government/publications.atom?topics%5B%5D=uk-emissions-trading-scheme
func collectHeadlinesUKETS(limit int, cfg headlineSourceConfig) ([]Headline, error) {
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

		// 日付のパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// サマリー/コンテンツを取得
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

		})
	}

	return out, nil
}

// collectHeadlinesUNNews は UN News 気候変動RSSフィードから記事を取得する
//
// UN Newsは国連の公式ニュースサービスで、
// 世界の気候変動ニュース、UNFCCC会議、国際気候政策をカバーしている。
// unfccc.int の直接スクレイピングの代替として機能する。
//
// URL: https://news.un.org/feed/subscribe/en/news/topic/climate-change/feed/rss.xml
func collectHeadlinesUNNews(limit int, cfg headlineSourceConfig) ([]Headline, error) {
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

		// 日付のパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// コンテンツを取得（UN Newsは通常良質なdescriptionを持つ）
		excerpt := extractRSSExcerpt(item)

		out = append(out, Headline{
			Source:      "UN News",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	return out, nil
}

// collectHeadlinesCarbonBrief は Carbon Brief RSSフィードから見出しを収集する
//
// Carbon Briefは気候科学、気候政策、エネルギー政策の最新動向を
// カバーする英国拠点のウェブサイト。以前はWordPress REST APIを使用していたが、
// content:encoded による全文取得のためRSSに切り替えた。
//
// URL: https://www.carbonbrief.org/feed/
func collectHeadlinesCarbonBrief(limit int, cfg headlineSourceConfig) ([]Headline, error) {
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

		// 日付のパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// content:encoded から全文取得、なければ description にフォールバック
		excerpt := extractRSSExcerpt(item)
		// Notion AI要約用に3000文字で切り詰め
		excerpt = truncateString(excerpt, 3000)

		out = append(out, Headline{
			Source:      "Carbon Brief",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	return out, nil
}

// collectHeadlinesCarbonMarketWatch は Carbon Market Watch RSSフィードから見出しを収集する
//
// Carbon Market Watchはブリュッセル拠点のNGOで、カーボン市場を監視し
// 公正で効果的な気候政策を提唱している。ウェブサイトは直接のHTMLスクレイピングを
// ブロック（403）するが、WordPress RSSフィードはアクセス可能。
// フィードは content:encoded 経由で全文を含む。
//
// URL: https://carbonmarketwatch.org/feed/
func collectHeadlinesCarbonMarketWatch(limit int, cfg headlineSourceConfig) ([]Headline, error) {
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

		// 日付のパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// content:encoded から全文取得、なければ description にフォールバック
		excerpt := extractRSSExcerpt(item)

		out = append(out, Headline{
			Source:      "Carbon Market Watch",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	return out, nil
}
