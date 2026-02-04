// =============================================================================
// headlines.go - ニュースソース共通ロジック
// =============================================================================
//
// このファイルはニュースソースからの記事収集に関する共通ロジックを提供します。
// 個別のソース実装は以下のファイルに分割されています：
//
// 【ファイル構成】
//   - headlines.go (このファイル)  - 共通ロジック
//   - sources_wordpress.go        - WordPress REST APIソース
//   - sources_paid.go             - 有料ソース（Carbon Pulse, QCI）
//   - sources_html.go             - HTMLスクレイピングソース
//   - sources_japan.go            - 日本語ソース
//   - sources_rss.go              - RSSフィードソース
//   - sources_academic.go         - 学術・研究機関ソース
//   - sources_regional_ets.go     - 地域ETSソース
//
// =============================================================================
// 【実装ソース一覧】（全36ソース、有効33ソース）
// =============================================================================
//
// ▼ 有料ソース（見出しのみ取得）- sources_paid.go
//  1. Carbon Pulse    - カーボン市場専門ニュース（業界最大手）
//  2. QCI             - Quantum Commodity Intelligence
//
// ▼ 無料ソース - WordPress REST API（7ソース）- sources_wordpress.go
//  3. CarbonCredits.jp    - 日本のカーボンクレジット情報
//  4. Carbon Herald       - CDR技術ニュース
//  5. Climate Home News   - 国際気候政策
//  6. CarbonCredits.com   - 教育・啓発コンテンツ
//  7. Sandbag             - EU ETSアナリスト
//  8. Ecosystem Marketplace - 自然気候ソリューション
//  9. Carbon Brief        - 気候科学・政策
//
// ▼ 無料ソース - HTMLスクレイピング（12ソース）- sources_html.go
//  10. ICAP               - 国際カーボンアクションパートナーシップ
//  11. IETA               - 国際排出量取引協会
//  12. Energy Monitor     - エネルギー転換ニュース
//  13. World Bank         - 世界銀行気候変動
//  14. NewClimate Institute - 気候研究機関
//  15. Carbon Knowledge Hub - 教育プラットフォーム
//  16. Verra              - VCS規格運営団体
//  17. Gold Standard      - 高品質カーボンクレジット規格
//  18. ACR                - American Carbon Registry
//  19. CAR                - Climate Action Reserve
//  20. IISD ENB           - 環境交渉速報
//  21. Climate Focus      - 気候政策コンサルティング
//  22. Isometric          - 炭素除去検証
//
// ▼ 無料ソース - 日本語ソース（5ソース）- sources_japan.go
//  23. JRI（日本総研）    - RSSフィード
//  24. 環境省             - プレスリリース
//  25. METI（経産省）     - SME Agency RSS
//  26. PwC Japan          - コンサルティングレポート
//  27. Mizuho R&T         - 金融調査レポート
//
// ▼ その他 - sources_japan.go
//  28. JPX（日本取引所）  - カーボン関連株式ニュース
//
// ▼ RSS/Atomフィードソース（3ソース）- sources_rss.go
//  29. Politico EU        - EU政策・エネルギー・気候変動ニュース
//  36. UN News            - 国連ニュース気候変動セクション（UNFCCC代替）
//  38. Euractiv           - EU政策ニュース（メインフィード + キーワードフィルタ）
//
// ▼ 学術・研究機関ソース（3ソース）- sources_academic.go
//  30. arXiv              - プレプリントリポジトリ
//  31. Nature Communications - 科学ジャーナル（キーワードフィルタ）
//  37. OIES               - オックスフォードエネルギー研究所（プログラムページ経由）
//
// ▼ 地域ETSソース（4ソース）- sources_regional_ets.go
//  32. EU ETS             - 欧州委員会ETSニュース
//  33. California CARB    - カリフォルニア大気資源局
//  34. RGGI               - 北東部州温室効果ガスイニシアティブ
//  35. Australia CER      - オーストラリア・クリーンエネルギー規制局
//
// ▼ 一時無効化中のソース
//  - Carbon Market Watch  - 403 Forbiddenエラー
//  - UNFCCC              - Incapsula保護 (UN Newsで代替)
//
// =============================================================================
// 【デバッグ方法】
// =============================================================================
//
// 環境変数でデバッグ情報を出力:
//
//	DEBUG_SCRAPING=1  - スクレイピング処理の詳細ログ
//	DEBUG_HTML=1      - 取得したHTMLの構造を出力
//
// 使用例:
//
//	DEBUG_SCRAPING=1 ./pipeline -sources=carbonpulse -perSource=1 -queriesPerHeadline=0
//
// =============================================================================
package main

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// =============================================================================
// ソースレジストリ（Source Registry）
// =============================================================================
//
// 各ソースの収集関数をマップで管理することで、main.goのif文を削減し、
// 新規ソース追加時の変更を最小化します。
//
// 【使用方法】
//
//	collector, ok := sourceCollectors["carbonpulse"]
//	if ok {
//	    headlines, err := collector(10, cfg)
//	}
//
// =============================================================================

// HeadlineCollector は見出し収集関数のシグネチャを定義する型
//
// 全てのcollectHeadlines*関数はこのシグネチャに従う:
//   - limit: 取得する記事の最大数
//   - cfg:   HTTP設定（User-Agent、タイムアウト）
//   - 戻り値: 収集した見出しとエラー
type HeadlineCollector func(limit int, cfg headlineSourceConfig) ([]Headline, error)

// sourceCollectors は全ソースの収集関数を格納するレジストリ
//
// キー: ソース識別子（CLIの-sourcesで指定する文字列）
// 値:  対応する収集関数
var sourceCollectors = map[string]HeadlineCollector{
	// 有料ソース（見出しのみ）- sources_paid.go
	"carbonpulse": collectHeadlinesCarbonPulse,
	"qci":         collectHeadlinesQCI,

	// WordPress REST APIソース - sources_wordpress.go
	"carboncredits.jp":      collectHeadlinesCarbonCreditsJP,
	"carbonherald":          collectHeadlinesCarbonHerald,
	"climatehomenews":       collectHeadlinesClimateHomeNews,
	"carboncredits.com":     collectHeadlinesCarbonCreditscom,
	"sandbag":               collectHeadlinesSandbag,
	"ecosystem-marketplace": collectHeadlinesEcosystemMarketplace,
	"carbon-brief":          collectHeadlinesCarbonBrief,

	// HTMLスクレイピングソース - sources_html.go
	"icap":                 collectHeadlinesICAP,
	"ieta":                 collectHeadlinesIETA,
	"energy-monitor":       collectHeadlinesEnergyMonitor,
	"world-bank":           collectHeadlinesWorldBank,
	"newclimate":           collectHeadlinesNewClimate,
	"carbon-knowledge-hub": collectHeadlinesCarbonKnowledgeHub,
	// "carbon-market-watch": collectHeadlinesCarbonMarketWatch, // 2026-01: 403 Forbidden エラーのため一時無効化

	// 日本語ソース - sources_japan.go
	"jri":          collectHeadlinesJRI,
	"env-ministry": collectHeadlinesEnvMinistry,
	"meti":         collectHeadlinesMETI,
	"pwc-japan":    collectHeadlinesPwCJapan,
	"mizuho-rt":    collectHeadlinesMizuhoRT,

	// その他 - sources_japan.go
	"jpx": collectHeadlinesJPX,

	// 欧州政策ソース（RSSフィード）- sources_rss.go
	"politico-eu": collectHeadlinesPoliticoEU,
	"euractiv":    collectHeadlinesEuractiv,

	// 学術・研究機関ソース - sources_academic.go
	"arxiv":        collectHeadlinesArXiv,
	"nature-comms": collectHeadlinesNatureComms,
	"oies":         collectHeadlinesOIES,

	// VCM認証団体 - sources_html.go
	"verra":         collectHeadlinesVerra,
	"gold-standard": collectHeadlinesGoldStandard,
	"acr":           collectHeadlinesACR,
	"car":           collectHeadlinesCAR,

	// 国際機関 - sources_html.go, sources_rss.go
	// "unfccc":        collectHeadlinesUNFCCC, // 2026-01: Incapsula protection, temporarily disabled
	// "un-news":       collectHeadlinesUNNews, // 2026-02: Pending - need to improve content extraction
	"iisd":          collectHeadlinesIISD,
	"climate-focus": collectHeadlinesClimateFocus,

	// 地域ETS - sources_regional_ets.go
	"eu-ets":        collectHeadlinesEUETS,
	"uk-ets":        collectHeadlinesUKETSHTML, // HTML scraping version (Atom feed was empty)
	"carb":          collectHeadlinesCARB,
	"rggi":          collectHeadlinesRGGI,
	"australia-cer": collectHeadlinesAustraliaCER,

	// 追加ソース（CDR関連）- sources_html.go
	"puro-earth": collectHeadlinesPuroEarth, // Blog page now working
	"isometric":  collectHeadlinesIsometric,
}

// =============================================================================
// 設定と構造体
// =============================================================================

// headlineSourceConfig は見出し収集時の設定を保持
type headlineSourceConfig struct {
	CarbonPulseTimelineURL string        // Carbon Pulse タイムラインページURL
	CarbonPulseNewsletters string        // Carbon Pulse ニュースレターカテゴリURL
	QCIHomeURL             string        // QCI ホームページURL
	UserAgent              string        // HTTPリクエスト時のUser-Agentヘッダー
	Timeout                time.Duration // HTTPリクエストのタイムアウト時間
}

// defaultHeadlineConfig はデフォルトの見出し収集設定を返す
func defaultHeadlineConfig() headlineSourceConfig {
	return headlineSourceConfig{
		CarbonPulseTimelineURL: "https://carbon-pulse.com/daily-timeline/",
		CarbonPulseNewsletters: "https://carbon-pulse.com/category/newsletters/",
		QCIHomeURL:             "https://www.qcintel.com/carbon/",
		UserAgent:              "Mozilla/5.0 (compatible; carbon-relay/1.0; +https://example.invalid)",
		Timeout:                20 * time.Second, // デフォルト20秒タイムアウト
	}
}

// =============================================================================
// 共通収集関数
// =============================================================================

// CollectFromSources は指定されたソースから見出しを収集する
//
// 【引数】
//   - sources:   収集するソースのリスト（例: ["carbonpulse", "qci"]）
//   - perSource: ソースあたりの最大記事数
//   - cfg:       HTTP設定
//
// 【戻り値】
//   - 収集した見出し（重複除去済み）
//   - エラー（未知のソースが指定された場合など）
//
// 【使用例】
//
//	headlines, err := CollectFromSources([]string{"carbonherald", "carbon-brief"}, 10, cfg)
func CollectFromSources(sources []string, perSource int, cfg headlineSourceConfig) ([]Headline, error) {
	var headlines []Headline

	for _, src := range sources {
		collector, ok := sourceCollectors[src]
		if !ok {
			return nil, fmt.Errorf("unknown source: %s", src)
		}

		hs, err := collector(perSource, cfg)
		if err != nil {
			return nil, fmt.Errorf("collecting %s: %w", src, err)
		}

		headlines = append(headlines, hs...)
	}

	return uniqueHeadlinesByURL(headlines), nil
}

// FilterHeadlinesByHours は指定された時間以内に公開された記事のみをフィルタリングする
//
// 【引数】
//   - headlines: フィルタリング対象の記事リスト
//   - hours:     何時間以内の記事を残すか（例: 24 = 過去24時間）
//
// 【戻り値】
//   - 指定時間以内に公開された記事のリスト
//
// 【注意】
//   - PublishedAtが空または解析できない記事は除外される
//   - PublishedAtはRFC3339形式を想定（例: "2026-01-05T12:00:00Z"）
//
// 【使用例】
//
//	headlines, _ := CollectFromSources(sources, perSource, cfg)
//	filtered := FilterHeadlinesByHours(headlines, 24) // 過去24時間の記事のみ
func FilterHeadlinesByHours(headlines []Headline, hours int) []Headline {
	if hours <= 0 {
		return headlines // 0以下の場合はフィルタリングしない
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var filtered []Headline

	for _, h := range headlines {
		if h.PublishedAt == "" {
			continue // 日付がない記事は除外
		}

		// RFC3339形式でパース試行
		pubTime, err := time.Parse(time.RFC3339, h.PublishedAt)
		if err != nil {
			// RFC3339以外の形式も試行（例: "2006-01-02T15:04:05"）
			pubTime, err = time.Parse("2006-01-02T15:04:05", h.PublishedAt)
			if err != nil {
				// 日付のみの形式も試行（例: "2006-01-02"）
				pubTime, err = time.Parse("2006-01-02", h.PublishedAt)
				if err != nil {
					if os.Getenv("DEBUG_SCRAPING") != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG] FilterHeadlinesByHours: cannot parse date '%s' for '%s'\n", h.PublishedAt, h.Title)
					}
					continue // パースできない場合は除外
				}
			}
		}

		if pubTime.After(cutoff) {
			filtered = append(filtered, h)
		}
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] FilterHeadlinesByHours: %d -> %d headlines (last %d hours)\n", len(headlines), len(filtered), hours)
	}

	return filtered
}

// =============================================================================
// WordPress REST API 共通処理
// =============================================================================
//
// WordPress REST APIを使用するソース（7ソース）の共通処理を提供します。
//
// =============================================================================

// collectWordPressHeadlines はWordPress REST APIから記事を収集する共通関数
//
// 【引数】
//   - baseURL:    WordPressサイトのベースURL（例: "https://carbonherald.com"）
//   - sourceName: ソース名（例: "Carbon Herald"）
//   - limit:      取得する記事の最大数
//   - cfg:        HTTP設定
//
// 【使用例】
//
//	headlines, err := collectWordPressHeadlines(
//	    "https://carbonherald.com",
//	    "Carbon Herald",
//	    10,
//	    cfg,
//	)
func collectWordPressHeadlines(baseURL, sourceName string, limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// WordPress REST API endpoint - get full content for free articles
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date,content", baseURL, limit)

	// httpGetJSON is defined in utils.go
	var posts []WPPost
	if err := httpGetJSON(apiURL, cfg, &posts); err != nil {
		return nil, fmt.Errorf("failed to fetch %s API: %w", sourceName, err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		// Clean up HTML entities from title
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		// Clean up HTML from full content (free article)
		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		out = append(out, Headline{
			Source:      sourceName,
			Title:       title,
			URL:         p.Link,
			PublishedAt: p.Date,  // WordPress API returns RFC3339 format
			Excerpt:     content, // Store full content in Excerpt field for free articles
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s: collected %d headlines\n", sourceName, len(out))
	}

	return out, nil
}

// =============================================================================
// ヘルパー関数
// =============================================================================

// min2 は2つの整数のうち小さい方を返すヘルパー関数
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fetchDoc は指定URLからHTMLドキュメントを取得してgoqueryでパース
//
// タイムアウト設定と適切なHTTPヘッダー（User-Agent, Accept）を含めて
// HTTPリクエストを送信し、レスポンスをgoquery.Documentとして返す
//
// 引数:
//
//	u: 取得するURL
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	パースされたHTMLドキュメント、エラー
func fetchDoc(u string, cfg headlineSourceConfig) (*goquery.Document, error) {
	client := &http.Client{Timeout: cfg.Timeout} // タイムアウト付きHTTPクライアント
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	// ブロッキング回避のため、ブラウザ風のヘッダーを設定
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// HTTPステータスコードチェック（200番台以外はエラー）
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: status %s", u, resp.Status)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// resolveURL は相対URLを絶対URLに変換
//
// ベースURLと相対URL（href）から完全な絶対URLを生成する。
// 既に絶対URLの場合はそのまま返す。
//
// 引数:
//
//	baseURL: 基準となるページのURL
//	href: 相対または絶対URL
//
// 戻り値:
//
//	解決された絶対URL（エラー時は空文字列）
func resolveURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	// 相対URLを絶対URLに解決
	return base.ResolveReference(u).String()
}

// extractExcerptFromContext はリンク周辺のテキストから記事要約を抽出
//
// 個別の記事ページをフェッチせず、タイムライン/一覧ページ内で
// リンク要素の周辺にあるテキストコンテンツを要約として抽出する。
//
// 引数:
//
//	linkSel: 記事リンクのgoquery Selection
//
// 戻り値:
//
//	抽出された要約テキスト（最大500文字）
func extractExcerptFromContext(linkSel *goquery.Selection) string {
	var excerpt strings.Builder
	maxChars := 500

	// Debug: show link context - check multiple parent levels
	if os.Getenv("DEBUG_HTML") != "" {
		linkHTML, _ := linkSel.Html()
		parent1HTML, _ := linkSel.Parent().Html()
		parent2HTML, _ := linkSel.Parent().Parent().Html()
		parent3HTML, _ := linkSel.Parent().Parent().Parent().Html()

		fmt.Fprintf(os.Stderr, "\n[DEBUG] ========== Link Context ==========\n")
		fmt.Fprintf(os.Stderr, "[DEBUG] Link HTML: %s\n", linkHTML[:min2(200, len(linkHTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] Parent Level 1: %s\n", parent1HTML[:min2(800, len(parent1HTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] Parent Level 2: %s\n", parent2HTML[:min2(1500, len(parent2HTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] Parent Level 3 (FULL): %s\n", parent3HTML[:min2(5000, len(parent3HTML))])
		fmt.Fprintf(os.Stderr, "[DEBUG] =====================================\n\n")
	}

	// Strategy 1: Carbon Pulse top page article structure
	// The excerpt is a text node between <a class="thumbLink"> and <a class="readMore">
	articleContainer := linkSel.Parent().Parent().Parent()

	if os.Getenv("DEBUG_SCRAPING") != "" {
		classes, _ := articleContainer.Attr("class")
		fmt.Fprintf(os.Stderr, "[DEBUG] Article container classes: %s\n", classes)
		fmt.Fprintf(os.Stderr, "[DEBUG] Has 'post' class: %v\n", articleContainer.HasClass("post"))
	}

	// Check if this is a Carbon Pulse article container (has class "post")
	if articleContainer.HasClass("post") {
		// Get all text from the container
		fullText := articleContainer.Text()

		// Remove metadata section (Published ... / Last updated ... / Author ... / Categories ...)
		// Find "Read More" and take text before it
		readMoreIdx := strings.Index(fullText, "Read More")
		if readMoreIdx > 0 {
			fullText = fullText[:readMoreIdx]
		}

		// Split into lines and find the excerpt (typically after tags like "Carbon Pulse Premium")
		lines := strings.Split(fullText, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)

			// Skip empty lines, metadata, tags, and navigation
			if line == "" || len(line) < 30 {
				continue
			}
			if strings.Contains(line, "Published") ||
				strings.Contains(line, "Last updated") ||
				strings.Contains(line, "Carbon Pulse Premium") ||
				strings.Contains(line, "Nature & Biodiversity") ||
				strings.Contains(line, "Net Zero Pulse") ||
				strings.HasPrefix(line, "Top") {
				continue
			}

			// This should be the excerpt
			if len(line) > 30 && !strings.HasPrefix(line, "http") {
				excerpt.WriteString(line)

				// Also check next line if it's a continuation
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if len(nextLine) > 20 && !strings.Contains(nextLine, "Read More") {
						excerpt.WriteString(" ")
						excerpt.WriteString(nextLine)
					}
				}
				break
			}
		}
	}

	// Strategy 2: Fallback - Check for <p> tags (for other page structures)
	if excerpt.Len() == 0 {
		parent := linkSel.Parent()
		parent.Find("p:not(.metaStuff)").Each(func(i int, s *goquery.Selection) {
			if excerpt.Len() >= maxChars {
				return
			}
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				if excerpt.Len() > 0 {
					excerpt.WriteString(" ")
				}
				excerpt.WriteString(text)
			}
		})
	}

	// Strategy 3: Check for <div class="excerpt"> or similar
	if excerpt.Len() == 0 {
		linkSel.Parent().Parent().Find(".excerpt, .summary, .description").Each(func(i int, s *goquery.Selection) {
			if excerpt.Len() >= maxChars {
				return
			}
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				if excerpt.Len() > 0 {
					excerpt.WriteString(" ")
				}
				excerpt.WriteString(text)
			}
		})
	}

	result := strings.TrimSpace(excerpt.String())

	// Truncate if too long
	if len(result) > maxChars {
		result = result[:maxChars] + "..."
	}

	if os.Getenv("DEBUG_SCRAPING") != "" && result != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Extracted excerpt from context (%d chars)\n", len(result))
	}

	return result
}

// cleanHTMLTags removes HTML tags and decodes HTML entities
func cleanHTMLTags(htmlStr string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(htmlStr, "")
	// Decode HTML entities (including Japanese characters)
	text = html.UnescapeString(text)
	return text
}
