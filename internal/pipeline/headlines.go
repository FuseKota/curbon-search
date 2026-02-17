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
//   - sources_html.go             - HTMLスクレイピングソース
//   - sources_japan.go            - 日本語ソース
//   - sources_rss.go              - RSSフィードソース
//   - sources_academic.go         - 学術・研究機関ソース
//   - sources_regional_ets.go     - 地域ETSソース
//
// =============================================================================
// 【実装ソース一覧】（全38ソース、有効35ソース）
// =============================================================================
//
// ▼ 無料ソース - WordPress REST API（8ソース）- sources_wordpress.go
//  1. CarbonCredits.jp    - 日本のカーボンクレジット情報
//  2. Carbon Herald       - CDR技術ニュース
//  3. Climate Home News   - 国際気候政策
//  4. CarbonCredits.com   - 教育・啓発コンテンツ
//  5. Sandbag             - EU ETSアナリスト
//  6. Ecosystem Marketplace - 自然気候ソリューション
//  7. Carbon Brief        - 気候科学・政策
//  8. RMI                 - エネルギー転換シンクタンク
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
//  29. Politico EU          - EU政策・エネルギー・気候変動ニュース
//  38. Euractiv             - EU政策ニュース（メインフィード + キーワードフィルタ）
//  42. Carbon Market Watch  - カーボン市場監視NGO（RSSフィード）
//
// ▼ 学術・研究機関ソース（6ソース）- sources_academic.go
//  30. arXiv              - プレプリントリポジトリ
//  31. Nature Communications - 科学ジャーナル（キーワードフィルタ）
//  37. OIES               - オックスフォードエネルギー研究所（プログラムページ経由）
//  39. IOP Science (ERL)  - 環境研究レター（RSSフィード + キーワードフィルタ）
//  40. Nature Eco&Evo     - 生態学・進化学（RSSフィード + キーワードフィルタ）
//  41. ScienceDirect      - Elsevier学術誌（RSSフィード + キーワードフィルタ）
//
// ▼ 地域ETSソース（4ソース）- sources_regional_ets.go
//  32. EU ETS             - 欧州委員会ETSニュース
//  33. California CARB    - カリフォルニア大気資源局
//  34. RGGI               - 北東部州温室効果ガスイニシアティブ
//  35. Australia CER      - オーストラリア・クリーンエネルギー規制局
//
// ▼ 一時無効化中のソース
//   - UNFCCC              - Incapsula (Imperva) 保護により全エンドポイントブロック
//   - UN News Climate     - コンテンツ抽出の改善が必要
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
// =============================================================================
package pipeline

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// Package-level compiled regex for performance (avoid recompiling on every call)
var reScriptTags = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
var reHTMLTags = regexp.MustCompile(`<[^>]*>`)
var reShortcodes = regexp.MustCompile(`\[/?[a-z_]+[^\]]*\]`)
var reWhitespace = regexp.MustCompile(`\s+`)
var reDatePublishedJSON = regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
var reJapaneseDateYMD = regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
var reMultipleNewlines = regexp.MustCompile(`\n{3,}`) // 3つ以上の連続改行

// =============================================================================
// ソースレジストリ（Source Registry）
// =============================================================================
//
// 各ソースの収集関数をマップで管理することで、main.goのif文を削減し、
// 新規ソース追加時の変更を最小化します。
//
// 【使用方法】
//
//	collector, ok := sourceCollectors["carbonherald"]
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
type HeadlineCollector func(limit int, cfg HeadlineSourceConfig) ([]Headline, error)

// sourceCollectors は全ソースの収集関数を格納するレジストリ
//
// キー: ソース識別子（CLIの-sourcesで指定する文字列）
// 値:  対応する収集関数
var sourceCollectors = map[string]HeadlineCollector{
	// =========================================================================
	// sources_wordpress.go - WordPress REST API ソース (8)
	// =========================================================================
	"carboncredits.jp":      collectHeadlinesCarbonCreditsJP,
	"carbonherald":          collectHeadlinesCarbonHerald,
	"climatehomenews":       collectHeadlinesClimateHomeNews,
	"carboncredits.com":     collectHeadlinesCarbonCreditscom,
	"sandbag":               collectHeadlinesSandbag,
	"ecosystem-marketplace": collectHeadlinesEcosystemMarketplace,
	"carbon-brief":          collectHeadlinesCarbonBrief,
	"rmi":                   collectHeadlinesRMI,

	// =========================================================================
	// sources_japan.go - 日本語ソース (5)
	// =========================================================================
	"jri":          collectHeadlinesJRI,
	"env-ministry": collectHeadlinesEnvMinistry,
	"jpx":          collectHeadlinesJPX,
	"meti":         collectHeadlinesMETI,
	"pwc-japan":    collectHeadlinesPwCJapan,
	"mizuho-rt":    collectHeadlinesMizuhoRT,

	// =========================================================================
	// sources_rss.go - RSS/Atom フィードソース (3)
	// =========================================================================
	"politico-eu":         collectHeadlinesPoliticoEU,
	"euractiv":            collectHeadlinesEuractiv,
	"carbon-market-watch": collectHeadlinesCarbonMarketWatch,
	// "un-news": collectHeadlinesUNNews, // 2026-02: コンテンツ抽出の改善が必要
	// "unfccc": collectHeadlinesUNFCCC, // 2026-01: Incapsula (Imperva) 保護 - 全エンドポイントブロック

	// =========================================================================
	// sources_academic.go - 学術・研究機関ソース (6)
	// =========================================================================
	"arxiv":         collectHeadlinesArXiv,
	"nature-comms":  collectHeadlinesNatureComms,
	"oies":          collectHeadlinesOIES,
	"iopscience":    collectHeadlinesIOPScience,
	// "nature-ecoevo": collectHeadlinesNatureEcoEvo, // 2026-02: 有料記事のため停止
	"sciencedirect": collectHeadlinesScienceDirect,

	// =========================================================================
	// sources_regional_ets.go - 地域排出量取引システム (5)
	// =========================================================================
	"eu-ets":        collectHeadlinesEUETS,
	"uk-ets":        collectHeadlinesUKETSHTML, // HTML版（Atom feedが空のため）
	"carb":          collectHeadlinesCARB,
	"rggi":          collectHeadlinesRGGI,
	"australia-cer": collectHeadlinesAustraliaCER,

	// =========================================================================
	// sources_html.go - HTMLスクレイピングソース (14)
	// =========================================================================
	// ニュースメディア・シンクタンク
	"icap":                 collectHeadlinesICAP,
	"ieta":                 collectHeadlinesIETA,
	"energy-monitor":       collectHeadlinesEnergyMonitor,
	"world-bank":           collectHeadlinesWorldBank,
	"newclimate":           collectHeadlinesNewClimate,
	"carbon-knowledge-hub": collectHeadlinesCarbonKnowledgeHub,

	// VCM認証団体
	"verra":         collectHeadlinesVerra,
	"gold-standard": collectHeadlinesGoldStandard,
	"acr":           collectHeadlinesACR,
	"car":           collectHeadlinesCAR,

	// 国際機関
	"iisd":          collectHeadlinesIISD,
	"climate-focus": collectHeadlinesClimateFocus,
	// CDR関連
	"puro-earth": collectHeadlinesPuroEarth,
	"isometric":  collectHeadlinesIsometric,
}

// =============================================================================
// 設定と構造体
// =============================================================================

// HeadlineSourceConfig は見出し収集時の設定を保持
type HeadlineSourceConfig struct {
	UserAgent string        // HTTPリクエスト時のUser-Agentヘッダー
	Timeout   time.Duration // HTTPリクエストのタイムアウト時間
	Client    *http.Client  // 共有HTTPクライアント（コネクションプーリング有効）
}

// DefaultHeadlineConfig はデフォルトの見出し収集設定を返す
func DefaultHeadlineConfig() HeadlineSourceConfig {
	timeout := 30 * time.Second // 30秒タイムアウト（一部のサイトは遅い）
	return HeadlineSourceConfig{
		UserAgent: "Mozilla/5.0 (compatible; carbon-relay/1.0; +https://example.invalid)",
		Timeout:   timeout,
		Client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// =============================================================================
// 共通収集関数
// =============================================================================

// CollectFromSources は指定されたソースから見出しを収集する
//
// 【引数】
//   - sources:   収集するソースのリスト（例: ["carbonherald", "carbon-brief"]）
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
// CollectResult は収集結果とエラー情報を保持する
type CollectResult struct {
	Headlines []Headline
	Errors    []string
}

func CollectFromSources(sources []string, perSource int, cfg HeadlineSourceConfig) (*CollectResult, error) {
	result := &CollectResult{}

	for _, src := range sources {
		collector, ok := sourceCollectors[src]
		if !ok {
			errMsg := fmt.Sprintf("[ERROR] unknown source: %s", src)
			fmt.Fprintln(os.Stderr, errMsg)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		hs, err := collector(perSource, cfg)
		if err != nil {
			errMsg := fmt.Sprintf("[ERROR] collecting %s: %v", src, err)
			fmt.Fprintln(os.Stderr, errMsg)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		result.Headlines = append(result.Headlines, hs...)
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\n[WARN] %d source(s) failed (collected %d headlines from %d sources):\n",
			len(result.Errors), len(result.Headlines), len(sources)-len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
	}

	result.Headlines = uniqueHeadlinesByURL(result.Headlines)
	return result, nil
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
//   - PublishedAtが空の記事はフィルタをスキップして保持される（日付不明のため除外しない）
//   - PublishedAtが解析できない記事は除外される
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
			// 日付が不明な記事はフィルタをスキップして保持
			// （time.Now()フォールバックを廃止したため、古い記事が誤って含まれることはない）
			filtered = append(filtered, h)
			continue
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

		// cutoff〜現在の範囲内の記事のみ保持（未来日付の記事を除外）
		now := time.Now()
		if pubTime.After(cutoff) && !pubTime.After(now) {
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
// WordPress REST APIを使用するソース（8ソース）の共通処理を提供します。
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
func collectWordPressHeadlines(baseURL, sourceName string, limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// WordPress REST API endpoint - get full content for free articles
	// Use date_gmt for consistent UTC timestamps across all WordPress sources
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts?per_page=%d&_fields=title,link,date_gmt,content", baseURL, limit)

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

		// Convert date_gmt to RFC3339 format with UTC timezone indicator
		// WordPress date_gmt format: "2026-01-05T14:42:50"
		publishedAt := ""
		if p.DateGMT != "" {
			publishedAt = p.DateGMT + "Z" // Add Z suffix to indicate UTC
		}

		out = append(out, Headline{
			Source:      sourceName,
			Title:       title,
			URL:         p.Link,
			PublishedAt: publishedAt,
			Excerpt:     content, // Store full content in Excerpt field for free articles
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s: collected %d headlines\n", sourceName, len(out))
	}

	return out, nil
}

// collectWordPressHeadlinesCustomType はカスタム投稿タイプからWordPress記事を収集する
//
// 一部のWordPressサイトは標準の「posts」ではなくカスタム投稿タイプを使用している。
// 例: Ecosystem Marketplaceは「featured-articles」を使用。
//
// 【引数】
//   - baseURL:    WordPressサイトのベースURL
//   - sourceName: ソース名
//   - postType:   カスタム投稿タイプ（例: "featured-articles"）
//   - limit:      取得する記事の最大数
//   - cfg:        HTTP設定
func collectWordPressHeadlinesCustomType(baseURL, sourceName, postType string, limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// WordPress REST API endpoint with custom post type
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/%s?per_page=%d&_fields=title,link,date_gmt,content", baseURL, postType, limit)

	var posts []WPPost
	if err := httpGetJSON(apiURL, cfg, &posts); err != nil {
		return nil, fmt.Errorf("failed to fetch %s API: %w", sourceName, err)
	}

	out := make([]Headline, 0, len(posts))
	for _, p := range posts {
		title := cleanHTMLTags(p.Title.Rendered)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		content := cleanHTMLTags(p.Content.Rendered)
		content = strings.TrimSpace(content)

		publishedAt := ""
		if p.DateGMT != "" {
			publishedAt = p.DateGMT + "Z"
		}

		out = append(out, Headline{
			Source:      sourceName,
			Title:       title,
			URL:         p.Link,
			PublishedAt: publishedAt,
			Excerpt:     content,
			IsHeadline:  true,
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s: collected %d headlines from custom type '%s'\n", sourceName, len(out), postType)
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
func fetchDoc(u string, cfg HeadlineSourceConfig) (*goquery.Document, error) {
	client := cfg.Client // 共有HTTPクライアントを使用
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

	// Strategy 1: Check for <p> tags in parent elements
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

	// Strategy 2: Check for <div class="excerpt"> or similar
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

// matchesKeywords は title または excerpt が keywords のいずれかを含むかチェック
//
// キーワードフィルタリングが必要なソース（arXiv, IOP Science, Nature Eco&Evo,
// ScienceDirect, Euractiv, JRI, Env Ministry, Mizuho R&T）で共通使用。
func matchesKeywords(title, excerpt string, keywords []string) bool {
	titleLower := strings.ToLower(title)
	excerptLower := strings.ToLower(excerpt)
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(titleLower, kwLower) || strings.Contains(excerptLower, kwLower) {
			return true
		}
	}
	return false
}

// fetchRSSFeed は指定URLからRSS/Atomフィードを取得してパース
//
// 共有HTTPクライアントを使用してフィードをフェッチし、gofeedでパースする。
// sources_rss.go, sources_html.go, sources_academic.go の8箇所で共通使用。
func fetchRSSFeed(feedURL string, cfg HeadlineSourceConfig) (*gofeed.Feed, error) {
	client := cfg.Client
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

	return feed, nil
}

// extractRSSExcerpt は gofeed.Item から Content/Description を優先取得して整形
//
// Content フィールドが空でなければ Content を、なければ Description を使用。
// HTMLタグを除去してトリム。
func extractRSSExcerpt(item *gofeed.Item) string {
	raw := item.Content
	if raw == "" {
		raw = item.Description
	}
	if raw == "" {
		return ""
	}
	text := cleanHTMLTags(raw)
	return strings.TrimSpace(text)
}

// fetchViaCurl fetches a URL using curl to bypass TLS fingerprint detection.
// Some sites (e.g., nature.com with Fastly) block Go's net/http TLS fingerprint
// but allow curl. This function shells out to curl as a workaround.
func fetchViaCurl(targetURL string, userAgent string) (string, error) {
	cmd := exec.Command("curl", "-sL",
		"-H", "User-Agent: "+userAgent,
		"--max-time", "30",
		targetURL,
	)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("curl failed for %s: %w", targetURL, err)
	}
	return string(output), nil
}

// cleanHTMLTags removes HTML tags and decodes HTML entities
func cleanHTMLTags(htmlStr string) string {
	// Remove <script>...</script> blocks entirely (content included)
	text := reScriptTags.ReplaceAllString(htmlStr, "")
	// Remove HTML tags (using pre-compiled regex for performance)
	text = reHTMLTags.ReplaceAllString(text, "")
	// Remove WordPress/Divi shortcodes like [et_pb_section ...] [/et_pb_section]
	text = reShortcodes.ReplaceAllString(text, "")
	// Decode HTML entities (including Japanese characters)
	text = html.UnescapeString(text)
	return text
}

// cleanExtractedText は goquery .Text() の出力を整理する
// タブ・連続空白・空行を除去し、きれいなテキストにする
func cleanExtractedText(raw string) string {
	lines := strings.Split(raw, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	result := strings.Join(cleaned, "\n")
	return strings.TrimSpace(result)
}
