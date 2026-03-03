// =============================================================================
// sources_academic.go - 学術・研究ソース
// =============================================================================
//
// このファイルはカーボン関連コンテンツの学術・研究出版ソースを定義します。
// XML APIおよびRSSフィードを使用します。
//
// ソース一覧:
//   1. arXiv                      - プレプリントリポジトリ (XML API)
//   2. Nature Communications      - 科学ジャーナル (RSS + キーワードフィルタ)
//   3. OIES                       - Oxford Institute for Energy Studies (HTML)
//   4. IOP Science (ERL)          - Environmental Research Letters (RSS + キーワードフィルタ)
//   5. Nature Ecology & Evolution - 科学ジャーナル (RSS + キーワードフィルタ)
//   6. ScienceDirect              - Elsevierジャーナル (RSS + キーワードフィルタ)
//
// =============================================================================
package pipeline

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// reScienceDirectDate はScienceDirectのdescription HTMLから "Publication date: Month Year" を抽出する正規表現
var reScienceDirectDate = regexp.MustCompile(`Publication date:\s*(\w+ \d{4})`)

// =============================================================================
// arXiv ソース
// =============================================================================

// arXivFeed は arXiv APIから返されるAtomフィード構造を表す
type arXivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arXivEntry `xml:"entry"`
}

// arXivEntry は arXivの個別論文エントリを表す
type arXivEntry struct {
	Title     string        `xml:"title"`
	ID        string        `xml:"id"`
	Published string        `xml:"published"`
	Updated   string        `xml:"updated"`
	Summary   string        `xml:"summary"`
	Authors   []arXivAuthor `xml:"author"`
	Links     []arXivLink   `xml:"link"`
}

// arXivAuthor は arXivエントリの著者を表す
type arXivAuthor struct {
	Name string `xml:"name"`
}

// arXivLink は arXivエントリのリンクを表す
type arXivLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// carbonKeywordsArXiv は arXiv論文の関連性を確認するためのキーワードリスト
// 物理学論文での誤検知を避けるため複合フレーズを使用
// （例: "emission" 単体では "positron emission", "light emission" 等にマッチしてしまう）
var carbonKeywordsArXiv = []string{
	// 気候変動に特化した複合用語
	"carbon emission", "carbon dioxide", "co2 emission", "greenhouse gas",
	"carbon pricing", "carbon tax", "carbon market", "carbon credit",
	"emissions trading", "cap and trade", "carbon trading",
	"climate change", "climate policy", "global warming",
	"decarbonization", "decarbonisation", "net-zero", "net zero", "carbon neutral",
	"renewable energy", "clean energy", "energy transition",
	"carbon capture", "carbon storage", "carbon sequestration",
	"carbon footprint", "carbon intensity",
	// 国際協定
	"paris agreement", "kyoto protocol",
}

// collectHeadlinesArXiv は arXiv APIを使用してカーボン関連論文を取得する
//
// APIドキュメント: https://info.arxiv.org/help/api/index.html
// レート制限: リクエスト間3秒（強制）
//
// 検索クエリはq-fin（定量ファイナンス）、econ（経済学）、
// physics（特に環境経済学トピック）の論文を対象とする
func collectHeadlinesArXiv(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// 気候/カーボン経済学論文を特定的に検索
	// 物理学論文を避けるためカテゴリ制限を使用
	// カテゴリ:
	//   econ.GN - 経済学（一般経済学）
	//   q-fin.* - 定量ファイナンス
	//   physics.soc-ph - 物理学と社会（気候政策論文を含む）
	//   physics.ao-ph - 大気海洋物理学
	//   stat.AP - 統計応用

	// カテゴリフィルタとキーワードフィルタで検索クエリを構築
	// 形式: (cat:econ.* OR cat:q-fin.*) AND (keyword1 OR keyword2)
	categories := "cat:econ.GN+OR+cat:q-fin.GN+OR+cat:q-fin.PM+OR+cat:stat.AP"
	keywords := "carbon+OR+climate+OR+emission+OR+environmental+policy"

	// 結合クエリ: 関連カテゴリ内でカーボン/気候用語に言及する論文
	searchQuery := fmt.Sprintf("(%s)+AND+(%s)", categories, keywords)

	// arXiv API URLと検索パラメータ
	// max_resultsで結果数を制限、sortBy=submittedDateで新しい順に取得
	apiURL := fmt.Sprintf(
		"http://export.arxiv.org/api/query?search_query=%s&start=0&max_results=%d&sortBy=submittedDate&sortOrder=descending",
		searchQuery,
		limit*10, // キーワードフィルタリングを考慮して多めにリクエスト
	)

	client := cfg.Client
	req, err := http.NewRequest("GET", apiURL, nil)
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

	// XMLレスポンスをパース
	var feed arXivFeed
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("XML parse failed: %w", err)
	}

	out := make([]Headline, 0, limit)

	for _, entry := range feed.Entries {
		if len(out) >= limit {
			break
		}

		// タイトルをクリーンアップ（arXivが追加する改行を除去）
		title := strings.TrimSpace(entry.Title)
		title = strings.ReplaceAll(title, "\n", " ")
		title = strings.Join(strings.Fields(title), " ")
		if title == "" {
			continue
		}

		// キーワードチェック用にサマリーをクリーンアップ
		summaryClean := strings.TrimSpace(entry.Summary)
		summaryClean = strings.ReplaceAll(summaryClean, "\n", " ")
		summaryClean = strings.Join(strings.Fields(summaryClean), " ")

		// キーワードフィルタを適用して論文が実際にカーボン/気候関連か確認
		if !matchesKeywords(title, summaryClean, carbonKeywordsArXiv) {
			continue
		}

		// アブストラクトページのURLを取得（IDがURLになっている）
		articleURL := entry.ID

		// PDFリンクがあれば取得
		for _, link := range entry.Links {
			if link.Type == "application/pdf" {
				// アブストラクトページを優先するが、PDFも利用可能
				break
			}
		}

		// 日付をパース（arXivはRFC3339形式を使用）
		dateStr := entry.Published
		if dateStr == "" {
			dateStr = entry.Updated
		}

		// クリーンアップ済みのサマリーを使用
		summary := summaryClean

		// 著者文字列を構築
		var authors []string
		for _, author := range entry.Authors {
			authors = append(authors, author.Name)
		}
		authorStr := strings.Join(authors, ", ")
		if len(authorStr) > 100 {
			authorStr = authorStr[:100] + "..."
		}

		excerpt := summary
		if authorStr != "" {
			excerpt = "Authors: " + authorStr + "\n\n" + summary
		}

		out = append(out, Headline{
			Source:      "arXiv",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] arXiv: collected %d headlines\n", len(out))
	}

	// レート制限を遵守 - リクエスト後3秒待機
	time.Sleep(3 * time.Second)

	return out, nil
}

// =============================================================================
// Nature Communications ソース
// =============================================================================

// carbonKeywordsNature は Nature Communications記事のフィルタリング用キーワードリスト
var carbonKeywordsNature = []string{
	"carbon", "emission", "greenhouse", "climate change", "net zero",
	"decarbonization", "decarbonisation", "carbon dioxide", "CO2",
	"carbon pricing", "carbon tax", "cap and trade", "emissions trading",
	"carbon market", "carbon credit", "offset", "sequestration",
	"carbon capture", "CCS", "CCUS", "negative emissions",
}

// collectHeadlinesNatureComms は Nature Communications RSSから気候関連記事を取得する
//
// Nature Communicationsは自然科学全分野をカバーする査読付きオープンアクセスジャーナル。
// Natureの主題分類で事前フィルタされたclimate-changeサブジェクトフィードを使用するため、
// 追加のキーワードフィルタリングは不要。
//
// サブジェクトフィードはタイトル、URL、日付を提供するがdescriptionは含まない。
// アブストラクトは各記事ページ（id="Abs1"セクション）から取得する。
//
// 注意: nature.comはFastly bot保護を使用し、GoのTLSフィンガープリントを検出して
// JavaScriptチャレンジページを返す。curlのTLSフィンガープリントはサーバーに
// 受け入れられるため、回避策としてcurlを使用する。
//
// URL: https://www.nature.com/subjects/climate-change/ncomms.rss
func collectHeadlinesNatureComms(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.nature.com/subjects/climate-change/ncomms.rss"

	// Nature.comはGoのTLSフィンガープリントをJSチャレンジページでブロックする。
	// 代わりにcurlでRSSフィードを取得する。
	body, err := fetchViaCurl(feedURL, cfg.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("curl fetch failed: %w", err)
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseString(body)
	if err != nil {
		return nil, fmt.Errorf("RSS parse failed: %w", err)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Nature Communications RSS feed")
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

		// 日付をパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// 記事ページからcurl経由でアブストラクトを取得
		excerpt := fetchNatureAbstract(articleURL, cfg.UserAgent)

		out = append(out, Headline{
			Source:      "Nature Communications",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	return out, nil
}

// fetchNatureAbstract は Nature記事ページからアブストラクトを取得する。
// TLSフィンガープリント検出を回避するためcurlを使用する。
func fetchNatureAbstract(articleURL string, userAgent string) string {
	body, err := fetchViaCurl(articleURL, userAgent)
	if err != nil {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}

	// Abs1セクションからアブストラクトを抽出
	abstract := doc.Find("#Abs1-content p, #Abs1 p").First().Text()
	return strings.TrimSpace(abstract)
}

// =============================================================================
// OIES (Oxford Institute for Energy Studies) ソース
// =============================================================================

// collectHeadlinesOIES は Oxford Institute for Energy Studiesから出版物を取得する
//
// OIESはエネルギー・環境経済学の研究論文を出版しており、
// カーボン市場や気候政策を含む。
//
// 戦略: メインの/publications/ページはJavaScriptレンダリングを使用するため、
// サーバーサイドでコンテンツをレンダリングする複数のプログラムページから取得する:
//   - Carbon Management Programme（主要 - カーボン/気候に特化）
//   - Energy Transition Research Initiative
//   - Gas、Electricity、その他のプログラム
func collectHeadlinesOIES(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// HTMLで出版物をレンダリングするプログラムページ（JavaScriptではない）
	programmeURLs := []string{
		"https://www.oxfordenergy.org/carbon-management-programme/",
		"https://www.oxfordenergy.org/energy-transition-research-initiative/",
		"https://www.oxfordenergy.org/gas-programme/",
		"https://www.oxfordenergy.org/electricity-programme/",
	}

	client := cfg.Client
	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	for _, programmeURL := range programmeURLs {
		if len(out) >= limit {
			break
		}

		headlines, err := fetchOIESProgrammePage(client, programmeURL, cfg.UserAgent)
		if err != nil {
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] OIES: error fetching %s: %v\n", programmeURL, err)
			}
			continue
		}

		for _, h := range headlines {
			if len(out) >= limit {
				break
			}
			if seen[h.URL] {
				continue
			}
			seen[h.URL] = true

			// 記事ページからExcerpt/コンテンツを取得
			excerpt, date := fetchOIESArticleContent(client, h.URL, cfg.UserAgent)
			if excerpt != "" {
				h.Excerpt = excerpt
			}
			if date != "" {
				h.PublishedAt = date
			}

			out = append(out, h)
		}
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] OIES: collected %d headlines from %d programmes\n", len(out), len(programmeURLs))
	}

	return out, nil
}

// fetchOIESArticleContent は個別記事ページからExcerptと日付を取得する
func fetchOIESArticleContent(client *http.Client, articleURL, userAgent string) (excerpt, date string) {
	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", ""
	}

	// コンテンツ抽出前にノイズ要素を除去
	doc.Find("nav, header, footer, aside, .sidebar, script, style, .menu, .navigation, form, .related, .related-products, .upsells, section.products").Remove()

	// ページから実質的な段落を収集
	var paragraphs []string
	doc.Find("p").Each(func(_ int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())

		// 短い段落をスキップ
		if len(text) < 50 {
			return
		}

		// ノイズパターンを含む段落をスキップ
		if strings.Count(text, "\t") > 2 || strings.Count(text, "\n") > 3 {
			return
		}

		// 関連記事からの切り詰めプレビューをスキップ（[...] や ... で終わるもの）
		if strings.HasSuffix(text, "[…]") || strings.HasSuffix(text, "…]") {
			return
		}

		// ナビゲーション/定型文テキストをスキップ
		lowerText := strings.ToLower(text)
		if strings.Contains(lowerText, "cookie") ||
			strings.Contains(lowerText, "privacy policy") ||
			strings.Contains(lowerText, "sign up") ||
			strings.Contains(lowerText, "subscribe") ||
			strings.Contains(lowerText, "register your email") ||
			strings.Contains(lowerText, "notification of new") ||
			strings.HasPrefix(text, "By:") {
			return
		}

		paragraphs = append(paragraphs, text)
	})

	// 段落が見つかればそれを使用、なければmeta descriptionにフォールバック
	if len(paragraphs) > 0 {
		excerpt = strings.Join(paragraphs, "\n\n")
	} else {
		// meta descriptionにフォールバック
		if metaDesc, exists := doc.Find("meta[name='description']").Attr("content"); exists && metaDesc != "" {
			excerpt = strings.TrimSpace(metaDesc)
		}
		if excerpt == "" {
			if ogDesc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists && ogDesc != "" {
				excerpt = strings.TrimSpace(ogDesc)
			}
		}
	}

	// 切り詰めマーカーをクリーンアップ
	excerpt = strings.TrimSuffix(excerpt, "[…]")
	excerpt = strings.TrimSuffix(excerpt, "…")
	excerpt = strings.TrimSuffix(excerpt, " [")
	excerpt = strings.TrimSpace(excerpt)

	// 非常に長いExcerptを切り詰め（最大2000文字）
	if len(excerpt) > 2000 {
		excerpt = excerpt[:1997] + "..."
	}

	// JSON-LDから日付の取得を試行
	doc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
		text := script.Text()
		if dateMatch := reDatePublishedJSON.FindStringSubmatch(text); len(dateMatch) > 1 {
			if t, err := time.Parse("2006-01-02", dateMatch[1]); err == nil {
				date = t.Format(time.RFC3339)
			} else if t, err := time.Parse(time.RFC3339, dateMatch[1]); err == nil {
				date = t.Format(time.RFC3339)
			}
		}
	})

	return excerpt, date
}

// fetchOIESProgrammePage は単一のOIESプログラムページから出版物を抽出する
func fetchOIESProgrammePage(client *http.Client, programmeURL, userAgent string) ([]Headline, error) {
	req, err := http.NewRequest("GET", programmeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

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

	var headlines []Headline

	// OIESプログラムページは出版物をリンクと日付で一覧表示する
	// /publications/ と /research/ URLへのリンクを探す
	doc.Find("a[href*='/publications/'], a[href*='/research/']").Each(func(_ int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// ナビゲーションとカテゴリリンクをスキップ
		if strings.Contains(href, "/publication-topic/") ||
			strings.Contains(href, "/publication-category/") ||
			strings.HasSuffix(href, "/publications/") ||
			strings.HasSuffix(href, "/research/") {
			return
		}

		articleURL := resolveURL(programmeURL, href)
		if articleURL == "" {
			return
		}

		// リンクテキストからタイトルを取得
		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		// PDFダウンロードリンクをスキップ（記事ページが必要）
		if strings.HasSuffix(strings.ToLower(href), ".pdf") {
			return
		}

		// リンク付近で日付を探す
		// OIESは "22.01.26" (DD.MM.YY) 形式を使用
		dateStr := ""

		// 親要素で日付を確認
		parent := link.Parent()
		for i := 0; i < 3 && parent.Length() > 0; i++ {
			parentText := parent.Text()
			if d := parseOIESDate(parentText); d != "" {
				dateStr = d
				break
			}
			parent = parent.Parent()
		}

		// 2年以上古いエントリを除外（日付が見つかった場合のみ）
		if dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				if time.Since(t) > 2*365*24*time.Hour {
					return
				}
			}
		}

		headlines = append(headlines, Headline{
			Source:      "OIES",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,

		})
	})

	return headlines, nil
}

// parseOIESDate はOIESの日付形式 (DD.MM.YY) を含むテキストから日付を抽出する
func parseOIESDate(text string) string {
	// OIESは "22.01.26" のような形式を使用（2026年1月22日）
	// DD.MM.YYパターンを探す
	for i := 0; i < len(text)-7; i++ {
		if text[i] >= '0' && text[i] <= '9' &&
			text[i+1] >= '0' && text[i+1] <= '9' &&
			text[i+2] == '.' &&
			text[i+3] >= '0' && text[i+3] <= '9' &&
			text[i+4] >= '0' && text[i+4] <= '9' &&
			text[i+5] == '.' &&
			text[i+6] >= '0' && text[i+6] <= '9' &&
			text[i+7] >= '0' && text[i+7] <= '9' {

			dateCandidate := text[i : i+8]
			// DD.MM.YY形式でパース
			t, err := time.Parse("02.01.06", dateCandidate)
			if err == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	return ""
}

// =============================================================================
// 学術ジャーナル共通キーワードリスト
// =============================================================================

// carbonKeywordsAcademic は学術ジャーナル記事のフィルタリング用キーワードリスト。
// カーボン/気候トピックへの関連性を確認する。IOP Science (ERL)、
// Nature Ecology & Evolution、ScienceDirectソースで共有。
var carbonKeywordsAcademic = []string{
	"carbon", "emission", "greenhouse", "climate change", "net zero",
	"decarbonization", "decarbonisation", "carbon dioxide", "CO2",
	"carbon pricing", "carbon tax", "cap and trade", "emissions trading",
	"carbon market", "carbon credit", "offset", "sequestration",
	"carbon capture", "CCS", "CCUS", "negative emissions",
	"global warming", "climate policy", "paris agreement",
	"renewable energy", "energy transition", "fossil fuel",
}

// =============================================================================
// IOP Science (Environmental Research Letters) ソース
// =============================================================================

// collectHeadlinesIOPScience は IOP Science Environmental Research Letters RSSから記事を取得する
//
// Environmental Research Letters (ERL) は環境科学をカバーするオープンアクセスジャーナル。
// ジャーナルのRSSフィードにキーワードフィルタリングを適用して
// カーボン/気候関連記事を抽出する。
//
// フィード形式: RDF/RSS 1.0（gofeedが自動処理）
// URL: https://iopscience.iop.org/journal/rss/1748-9326
func collectHeadlinesIOPScience(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://iopscience.iop.org/journal/rss/1748-9326"

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

		// キーワードフィルタリング用にコンテンツを取得
		excerpt := extractRSSExcerpt(item)

		// キーワードフィルタ - ERLは幅広い環境科学をカバー
		if !matchesKeywords(title, excerpt, carbonKeywordsAcademic) {
			continue
		}

		articleURL := item.Link

		// 日付をパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "IOP Science (ERL)",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IOP Science (ERL): collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// Nature Ecology & Evolution ソース
// =============================================================================

// collectHeadlinesNatureEcoEvo は Nature Ecology & Evolution RSSから記事を取得する
//
// Nature Ecology & Evolutionは生態学と進化生物学をカバーする。
// キーワードフィルタリングでカーボン/気候関連記事を抽出する。
//
// 注意: Nature.comにはRSSリクエストをブロックする可能性のあるbot保護がある（Nature Comms参照）。
// ブロックされた場合、空スライスを正常に返す。
//
// URL: https://www.nature.com/natecolevol.rss
func collectHeadlinesNatureEcoEvo(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://www.nature.com/natecolevol.rss"

	// Nature.comはCookieベースの認証リダイレクトを使用（303 -> idp.nature.com -> 戻り）。
	// リダイレクト間でCookieを保持するためcookie jar付きクライアントが必要。
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: cfg.Client.Timeout,
		Jar:     jar,
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		// Nature.comがbot保護でブロックする可能性あり - 空を正常に返す
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: request failed (possible bot protection): %v\n", err)
		}
		return []Headline{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: unexpected status %d (possible bot protection)\n", resp.StatusCode)
		}
		return []Headline{}, nil
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: RSS parse failed (possible HTML challenge page): %v\n", err)
		}
		return []Headline{}, nil
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

		// キーワードフィルタリング用にコンテンツを取得
		excerpt := extractRSSExcerpt(item)

		// キーワードフィルタ
		if !matchesKeywords(title, excerpt, carbonKeywordsAcademic) {
			continue
		}

		articleURL := item.Link

		// 日付をパース
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Nature Eco&Evo",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Nature Eco&Evo: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// ScienceDirect ソース
// =============================================================================

// collectHeadlinesScienceDirect は ScienceDirect RSSフィードから記事を取得する
//
// ScienceDirect (Elsevier) は学術ジャーナルをホストする。対象ジャーナルは
// "Resources, Conservation & Recycling Advances" (ISSN 2950-631X) で、
// サステナビリティと資源管理トピックをカバーする。
//
// URL: https://rss.sciencedirect.com/publication/science/2950631X
func collectHeadlinesScienceDirect(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	feedURL := "https://rss.sciencedirect.com/publication/science/2950631X"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
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

		// キーワードフィルタリング用にコンテンツを取得
		excerpt := extractRSSExcerpt(item)

		// キーワードフィルタ
		if !matchesKeywords(title, excerpt, carbonKeywordsAcademic) {
			continue
		}

		articleURL := item.Link

		// 日付をパース - ScienceDirect RSSには標準的な日付フィールドがないが、
		// descriptionに "Publication date: Month Year" が含まれる
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		} else if item.Description != "" {
			dateStr = parseScienceDirectDate(item.Description)
		}

		// 記事ページからアブストラクトを取得（RSSにはメタデータのみ）
		if articleURL != "" {
			if abs := fetchScienceDirectAbstract(articleURL, client, cfg.UserAgent); abs != "" {
				excerpt = abs
			}
		}

		out = append(out, Headline{
			Source:      "ScienceDirect",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ScienceDirect: collected %d headlines\n", len(out))
	}

	return out, nil
}

// fetchScienceDirectAbstract は記事ページを取得してアブストラクトテキストを抽出する。
func fetchScienceDirectAbstract(articleURL string, client *http.Client, userAgent string) string {
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

	// アブストラクトは <div class="abstract author"> 内にある
	// （"abstract author-highlights" や "abstract graphical" ではない）
	abs := doc.Find("div.abstract.author").Not(".author-highlights").First().Text()
	abs = strings.TrimSpace(abs)
	// 先頭の "Abstract" ラベルを除去
	abs = strings.TrimPrefix(abs, "Abstract")
	abs = strings.TrimSpace(abs)

	return abs
}

// parseScienceDirectDate はScienceDirectのdescription HTMLから日付を抽出する。
// 入力例: "<p>Publication date: March 2026</p>..." -> "2026-03-01T00:00:00Z"
func parseScienceDirectDate(desc string) string {
	m := reScienceDirectDate.FindStringSubmatch(desc)
	if m == nil {
		return ""
	}
	t, err := time.Parse("January 2006", m[1])
	if err != nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
