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
//
// =============================================================================
package main

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
