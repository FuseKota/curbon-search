// =============================================================================
// sources_japan.go - 日本語ソース
// =============================================================================
//
// このファイルは日本語のニュースソースからの記事収集関数を定義します。
// RSS、HTMLスクレイピング、複雑なJSON抽出など様々な手法を使用します。
//
// 【含まれるソース】
//   1. JRI（日本総研）    - RSSフィード
//   2. 環境省             - プレスリリース（HTMLスクレイピング）
//   3. JPX（日本取引所）  - RSSフィード
//   4. METI Shingikai     - 審議会リスト（HTMLスクレイピング）
//   5. PwC Japan          - 複雑なJSON抽出
//   6. Mizuho R&T         - HTMLスクレイピング
//
// =============================================================================
package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// carbonKeywordsJapan: カーボン/気候変動関連記事のフィルタリング用キーワードリスト
//
// 日本語ソース（JRI、環境省、METI、Mizuho R&T）で使用。
// 幅広いトピックをカバーするため、以下のカテゴリを含む：
//   - カーボン関連: カーボン、炭素、脱炭素、カーボンニュートラル
//   - 温室効果ガス: CO2、温室効果ガス、GHG
//   - 市場/取引: 排出量取引、ETS、カーボンプライシング、カーボンクレジット
//   - 気候変動: 気候変動、クライメート
//   - 英語キーワード: carbon, climate（英語混在記事用）
var carbonKeywordsJapan = []string{
	"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
	"気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
	"カーボンクレジット", "クレジット市場", "carbon", "climate",
	"JCM", "二国間クレジット", "カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
	"サステナビリティ", "エネルギー転換", "再生可能エネルギー", "グリーン",
}

// collectHeadlinesJRI は JRI（日本総合研究所）の RSSフィードから見出しを収集
//
// JRI は日本のシンクタンクで、カーボンニュートラルや気候変動に関する
// レポートを公開している。RSSフィードから記事を取得し、carbonKeywordsで
// カーボン関連記事をフィルタリング（現在はフィルタ無効化中）。
//
// 手法: RSS Feed (gofeed)
//
// 引数:
//
//	limit: 収集する最大記事数
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
func collectHeadlinesJRI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	rssURL := "https://www.jri.co.jp/xml.jsp?id=12966" // JRI の RSSフィードURL

	client := cfg.Client
	req, err := http.NewRequest("GET", rssURL, nil)
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

	// RSSフィードをパース（gofeedライブラリを使用）
	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RSS parse failed: %w", err)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in RSS feed")
	}

	out := make([]Headline, 0, limit)
	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		title := item.Title

		publishedAt := ""
		if item.PublishedParsed != nil {
			publishedAt = item.PublishedParsed.Format(time.RFC3339)
		}

		// 記事ページを取得してコンテンツを抽出
		excerpt := ""
		if item.Link != "" && !strings.HasSuffix(item.Link, ".pdf") {
			doc, err := fetchDoc(item.Link, cfg)
			if err == nil {
				// JRI ページ構造:
				//   - div.cont03: レポートページ（全文を含む）
				//   - article#main: オピニオン/コラムページ（メインコンテンツ領域）
				sel := doc.Find("div.cont03")
				if sel.Length() == 0 {
					sel = doc.Find("article#main")
				}
				sel.Find("script, style, div.content-utility").Remove()
				text := sel.Text()
				// ノイズ行（ナビゲーション、カテゴリラベル等）を除去
				lines := strings.Split(text, "\n")
				var contentLines []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// 短いナビゲーション/ラベル行をスキップ
					if len([]rune(line)) < 20 {
						continue
					}
					contentLines = append(contentLines, line)
				}
				if len(contentLines) > 0 {
					excerpt = strings.Join(contentLines, "\n")
				}
			}
		}

		// Excerpt を取得できなかった場合、RSS の description を使用
		if excerpt == "" && item.Description != "" {
			excerpt = cleanHTMLTags(item.Description)
		}

		// キーワードフィルタ: カーボン/気候変動関連の記事のみを収集
		if !matchesKeywords(title, excerpt, carbonKeywordsJapan) {
			continue
		}

		out = append(out, Headline{
			Source:      "Japan Research Institute",
			Title:       title,
			URL:         item.Link,
			PublishedAt: publishedAt,
			Excerpt:     excerpt,

		})
	}

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// collectHeadlinesEnvMinistry は 環境省のプレスリリースから見出しを収集する
func collectHeadlinesEnvMinistry(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	pressURL := "https://www.env.go.jp/press/"

	client := cfg.Client
	req, err := http.NewRequest("GET", pressURL, nil)
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
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// カーボン/気候変動関連記事のフィルタリング用キーワード
	carbonKeywords := []string{
		"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
		"気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
		"カーボンクレジット", "クレジット市場", "JCM", "二国間クレジット",
		"カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
	}

	out := make([]Headline, 0, limit)
	currentDate := ""

	// プレスリリースをパース
	doc.Find("span.p-press-release-list__heading, li.c-news-link__item").Each(func(i int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// 日付見出しかどうかを確認
		if s.Is("span.p-press-release-list__heading") {
			dateText := strings.TrimSpace(s.Text())
			// "2025年12月26日発表" を "2025-12-26" に変換
			dateText = strings.Replace(dateText, "発表", "", 1)
			dateText = strings.TrimSpace(dateText)

			// 日本語の日付フォーマットをパース
			var year, month, day int
			if _, parseErr := fmt.Sscanf(dateText, "%d年%d月%d日", &year, &month, &day); parseErr == nil {
				currentDate = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			}
			return
		}

		// 記事アイテムを処理
		if !s.Is("li.c-news-link__item") {
			return
		}

		// タイトルとURLを抽出
		link := s.Find("a.c-news-link__link")
		title := strings.TrimSpace(link.Text())
		href, exists := link.Attr("href")
		if !exists || title == "" {
			return
		}

		// タイトルにカーボン関連キーワードが含まれるか確認
		if !matchesKeywords(title, "", carbonKeywords) {
			return
		}

		// 絶対URLを構築
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			articleURL = "https://www.env.go.jp" + href
		}

		// 記事の全文コンテンツを取得
		excerpt := ""
		contentResp, err := client.Get(articleURL)
		if err == nil {
			if contentResp.StatusCode == http.StatusOK {
				contentDoc, err := goquery.NewDocumentFromReader(contentResp.Body)
				if err == nil {
					// 記事ページからメインコンテンツを抽出
					contentDoc.Find("div.l-content, div.c-content, article, main").Each(func(_ int, cs *goquery.Selection) {
						if excerpt == "" {
							text := strings.TrimSpace(cs.Text())
							if len(text) > 100 {
								excerpt = text
							}
						}
					})
				}
			}
			contentResp.Body.Close() // ループ内では defer を使わず即座にクローズ
		}

		// 公開日をフォーマット（他ソースとの一貫性のためUTCを使用）
		publishedAt := ""
		if currentDate != "" {
			publishedAt = currentDate + "T00:00:00Z"
		}

		out = append(out, Headline{
			Source:      "Japan Environment Ministry",
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAt,
			Excerpt:     excerpt,

		})
	})

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// collectHeadlinesJPX は JPX（日本取引所グループ）の RSSフィードから見出しを収集する
func collectHeadlinesJPX(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// JPX の RSSフィードを使用
	feedURL := "https://www.jpx.co.jp/rss/jpx-news.xml"

	fp := gofeed.NewParser()
	fp.Client = cfg.Client

	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JPX RSS: %w", err)
	}

	// カーボンクレジット関連記事のフィルタリング用キーワード
	carbonKeywords := []string{
		"カーボン", "炭素", "クレジット", "排出", "GX", "グリーン",
		"脱炭素", "CO2", "温室効果ガス", "取引", "市場", "環境",
	}

	out := make([]Headline, 0, limit)

	for _, item := range feed.Items {
		if len(out) >= limit {
			break
		}

		// タイトルまたはリンクにカーボン関連キーワードが含まれるか確認
		titleLower := strings.ToLower(item.Title)
		linkLower := strings.ToLower(item.Link)
		containsKeyword := false
		for _, kw := range carbonKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) ||
				strings.Contains(linkLower, "carbon") ||
				strings.Contains(linkLower, "クレジット") {
				containsKeyword = true
				break
			}
		}

		if !containsKeyword {
			continue
		}

		// 日付をパース（利用不可の場合は空文字列）
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// コンテンツを取得: RSS の description/content が空のため、記事ページをスクレイピング
		excerpt := ""
		if item.Description != "" {
			excerpt = html.UnescapeString(item.Description)
			excerpt = strings.TrimSpace(excerpt)
		}
		if item.Content != "" && excerpt == "" {
			excerpt = html.UnescapeString(item.Content)
			excerpt = strings.TrimSpace(excerpt)
		}
		if excerpt == "" && item.Link != "" {
			doc, err := fetchDoc(item.Link, cfg)
			if err == nil {
				sel := doc.Find("p.component-text")
				if sel.Length() > 0 {
					excerpt = strings.TrimSpace(sel.Text())
				}
			}
		}

		out = append(out, Headline{
			Source:      "Japan Exchange Group (JPX)",
			Title:       item.Title,
			URL:         item.Link,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	// カーボン関連記事が見つからない場合は空スライスを返す（エラーではない）
	// JPX フィードは動作しているが、常にカーボン関連コンテンツがあるとは限らない
	return out, nil
}

// collectHeadlinesMETI は METI 審議会リストページから見出しを収集する
//
// METI 審議会のインデックスページを取得し、エネルギー/カーボン関連の
// 審議会情報を collectHeadlinesEnvMinistry() と同様の2段階取得方式で抽出する。
//
// HTML 構造:
// - METI は <dl class="date_sp"> の <dd> 要素に記事リンクを配置
// - 各エントリに日本語形式の日付（YYYY年MM月DD日）が付随
//
// フィルタロジック:
// - URL パスフィルタ: /shingikai/enecho/（資源エネルギー庁）または
//   /shingikai/sankoshin/（産業構造審議会、GX関連部会を含む）
// - キーワードフィルタ: エネルギー、電力、ガス、カーボン、脱炭素、GX、水素等
// - URL パスが一致 → 収集（キーワード一致なしでも）
// - キーワードが一致 → 収集（URL パス一致なしでも）
//
// URL: https://www.meti.go.jp/shingikai/index.html
func collectHeadlinesMETI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	baseURL := "https://www.meti.go.jp"
	indexURL := baseURL + "/shingikai/index.html"

	// METI 用に長めのタイムアウトを設定（政府サイトはレスポンスが遅い場合がある）
	timeout := cfg.Timeout
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	// 標準的なブラウザ User-Agent を使用（METI はカスタムエージェントをブロックする場合がある）
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

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
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// URL パスフィルタ（エネルギー関連部門）
	energyPaths := []string{
		"/shingikai/enecho/",    // 資源エネルギー庁
		"/shingikai/sankoshin/", // 産業構造審議会（GX関連）
	}

	// エネルギー/カーボン関連コンテンツのキーワードフィルタ
	energyKeywords := []string{
		"エネルギー", "電力", "ガス", "資源", "燃料",
		"カーボン", "脱炭素", "GX", "グリーン",
		"水素", "アンモニア", "原子力", "再生可能",
		"排出", "温暖化", "気候", "蓄電", "電池",
	}

	// 日本語日付フォーマット用の正規表現（パッケージレベル）
	dateRe := reJapaneseDateYMD

	out := make([]Headline, 0, limit)

	// 審議会リンクを含む全 dd 要素を検索（METI は dl > dd 構造で更新情報を表示）
	doc.Find("dd").Each(func(i int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		link := s.Find("a[href*='/shingikai/']").First()
		if link.Length() == 0 {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// インデックスページをスキップ
		if strings.Contains(href, "index") {
			return
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 5 {
			return
		}

		// URL パスフィルタを確認
		isEnergyPath := false
		for _, path := range energyPaths {
			if strings.Contains(href, path) {
				isEnergyPath = true
				break
			}
		}

		// キーワードフィルタを確認
		hasKeyword := false
		titleLower := strings.ToLower(title)
		for _, kw := range energyKeywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				hasKeyword = true
				break
			}
		}

		// フィルタロジックを適用:
		// - パス一致 → 収集（キーワードに関係なく）
		// - キーワード一致 → 収集（パスに関係なく）
		if !isEnergyPath && !hasKeyword {
			return
		}

		// 絶対URLを構築
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			articleURL = baseURL + href
		}

		// li テキストから日付を抽出（他ソースとの一貫性のためUTCを使用）
		liText := s.Text()
		dateStr := ""
		if dateMatch := dateRe.FindStringSubmatch(liText); dateMatch != nil {
			year := dateMatch[1]
			month := fmt.Sprintf("%02d", atoi(dateMatch[2]))
			day := fmt.Sprintf("%02d", atoi(dateMatch[3]))
			dateStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day)
		}

		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] METI Shingikai: %s (path=%v, keyword=%v)\n", title[:min(50, len(title))], isEnergyPath, hasKeyword)
		}

		// 記事ページから Excerpt と日付を取得（2段階目の取得）
		excerpt, articleDate := fetchMETIArticleExcerpt(client, articleURL, cfg.UserAgent, title)
		if articleDate != "" {
			dateStr = articleDate
		}

		out = append(out, Headline{
			Source:      "METI Shingikai",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	return out, nil
}

// fetchMETIArticleExcerpt は 記事ページを取得し、テキストコンテンツと日付を抽出する
// (excerpt, dateStr) を返す。ページがPDFのみ、または取得失敗の場合は空文字列を返す
func fetchMETIArticleExcerpt(client *http.Client, url string, userAgent string, title string) (string, string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", ""
	}
	// 標準的なブラウザ User-Agent を使用
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

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

	// "最終更新日：YYYY年MM月DD日" から日付を抽出
	dateStr := ""
	bodyText := doc.Find("body").Text()
	if dateMatch := reJapaneseDateYMD.FindStringSubmatch(bodyText); dateMatch != nil {
		// 最後の一致を使用（最終更新日は通常ページ下部にある）
		allMatches := reJapaneseDateYMD.FindAllStringSubmatch(bodyText, -1)
		if len(allMatches) > 0 {
			last := allMatches[len(allMatches)-1]
			year := last[1]
			month := fmt.Sprintf("%02d", atoi(last[2]))
			day := fmt.Sprintf("%02d", atoi(last[3]))
			dateStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day)
		}
	}

	// 不要な要素を除去（JS通知、パンくずリスト、印刷ボタン、ナビゲーション）
	doc.Find("script, style, noscript, header, footer, nav, .jsOn, #topicpath, .topicpath, .breadcrumb, .breadcrumbs, #pankuzu, .print, .print-area").Remove()

	// メインコンテンツ領域を探索
	var excerpt string

	// METI ページで一般的なコンテンツセレクタ
	contentSelectors := []string{
		"#main_content",
		".contents",
		"#contents",
		"main",
		"article",
	}

	for _, sel := range contentSelectors {
		content := doc.Find(sel)
		if content.Length() > 0 {
			excerpt = strings.TrimSpace(content.Text())
			break
		}
	}

	// コンテンツ領域が見つからない場合は body にフォールバック
	if excerpt == "" {
		excerpt = strings.TrimSpace(doc.Find("body").Text())
	}

	// HTMLタグと空白を除去
	excerpt = cleanHTMLTags(excerpt)
	excerpt = reWhitespace.ReplaceAllString(excerpt, " ")
	excerpt = strings.TrimSpace(excerpt)

	// 先頭の「印刷」ボタンテキストを除去
	excerpt = strings.TrimPrefix(excerpt, "印刷")
	excerpt = strings.TrimSpace(excerpt)

	// パンくずリストテキストを除去（例: "ホーム 審議会・研究会 ... タイトル タイトル"）
	// "開催日" を実際のコンテンツの開始位置として使用
	if idx := strings.Index(excerpt, "開催日"); idx > 0 {
		excerpt = excerpt[idx:]
	} else if title != "" {
		// フォールバック: タイトルの最後の出現位置までを除去
		if idx := strings.LastIndex(excerpt, title); idx >= 0 {
			excerpt = strings.TrimSpace(excerpt[idx+len(title):])
		}
	}

	// 2000文字に切り詰め
	if len(excerpt) > 2000 {
		excerpt = excerpt[:2000]
	}

	return excerpt, dateStr
}

// atoi は 文字列を int に変換する。エラー時は 0 を返す
func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// min は 2つの整数のうち小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// collectHeadlinesPwCJapan は PwC Japan のサステナビリティページから見出しを収集
//
// PwC Japanは最も複雑なスクレイピング実装の1つ。ページ内のJavaScriptから
// 3重エスケープされたJSONデータを抽出し、複数回のアンエスケープ処理を経て
// 記事情報をパースする。
//
// 実装の特殊性:
//  1. angular.loadFacetedNavigationスクリプトから埋め込みJSONを抽出
//  2. 16進エスケープされた引用符（\x22）をアンエスケープ
//  3. elements配列が3重エスケープされているため、2回のアンエスケープ処理
//  4. 正規表現で個別の記事オブジェクトを抽出
//  5. 日付フォーマット: YYYY-MM-DD
//
// 手法: HTML Scraping + 複雑なJSON抽出
//
// 引数:
//
//	limit: 収集する最大記事数
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
func collectHeadlinesPwCJapan(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.pwc.com/jp/ja/knowledge/column/sustainability.html"

	client := cfg.Client // 共有クライアントを使用（デフォルトでリダイレクトに追従）
	req, err := http.NewRequest("GET", newsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
	// 非圧縮レスポンスを受信するため Accept-Encoding は設定しない
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// HTMLコンテンツ全体を読み込み
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	bodyStr := string(bodyBytes)

	// angular.loadFacetedNavigation スクリプトからJSONデータを抽出
	// パターン: angular.loadFacetedNavigation(..., "{...}")
	// JSONオブジェクトには numberHits, elements, selectedTags, filterTags が含まれる
	jsonPattern := regexp.MustCompile(`"(\{\\x22numberHits\\x22:\d+,\\x22elements\\x22:.*?\\x22filterTags\\x22:.*?\})"`)
	matches := jsonPattern.FindAllStringSubmatch(bodyStr, -1)

	out := make([]Headline, 0, limit)

	for _, match := range matches {
		if len(out) >= limit {
			break
		}

		if len(match) < 2 {
			continue
		}

		jsonStr := match[1]

		// 16進エスケープされた引用符をアンエスケープ (\x22 -> ")
		jsonStr = strings.ReplaceAll(jsonStr, `\x22`, `"`)
		// その他の一般的なエスケープをアンエスケープ
		jsonStr = strings.ReplaceAll(jsonStr, `\/`, `/`)
		jsonStr = strings.ReplaceAll(jsonStr, `\u002D`, `-`)

		// elements フィールドを抽出（文字列エンコードされたJSON配列）
		elementsPattern := regexp.MustCompile(`"elements":"(\[.*?\])(?:",|"}|"$)`)
		elementsMatch := elementsPattern.FindStringSubmatch(jsonStr)
		if len(elementsMatch) < 2 {
			continue
		}

		elementsStr := elementsMatch[1]

		// 3重エスケープされた elements 配列をアンエスケープ（2回実行が必要）
		for i := 0; i < 2; i++ {
			// \\ を一時プレースホルダに置換
			elementsStr = strings.ReplaceAll(elementsStr, `\\`, "\x00")
			// \" を " に置換
			elementsStr = strings.ReplaceAll(elementsStr, `\"`, `"`)
			// 単一バックスラッシュを復元
			elementsStr = strings.ReplaceAll(elementsStr, "\x00", `\`)
		}

		// 個別の記事オブジェクトをパース
		// title, href, publishDate フィールドを探索
		titlePattern := regexp.MustCompile(`"title":"([^"]+)"`)
		hrefPattern := regexp.MustCompile(`"href":"([^"]+)"`)
		datePattern := regexp.MustCompile(`"publishDate":"([^"]*)"`)

		// 記事オブジェクトで分割（各オブジェクトは {"index": で開始）
		articles := strings.Split(elementsStr, `{"index":`)

		for _, articleStr := range articles {
			if len(out) >= limit {
				break
			}

			if len(articleStr) < 50 {
				continue
			}

			// タイトルを抽出
			titleMatches := titlePattern.FindStringSubmatch(articleStr)
			if len(titleMatches) < 2 {
				continue
			}
			title := titleMatches[1]

			// URLを抽出
			hrefMatches := hrefPattern.FindStringSubmatch(articleStr)
			if len(hrefMatches) < 2 {
				continue
			}
			url := hrefMatches[1]

			// 日付を抽出
			dateStr := ""
			dateMatches := datePattern.FindStringSubmatch(articleStr)
			if len(dateMatches) >= 2 {
				dateStr = dateMatches[1]
			}

			// 絶対URLを構築
			articleURL := url
			if strings.HasPrefix(url, "/") {
				articleURL = "https://www.pwc.com" + url
			} else if !strings.HasPrefix(url, "http") {
				// 先頭スラッシュなしのURLの場合がある
				continue
			}

			// 日付をパース（形式: "YYYY-MM-DD"、利用不可の場合は空文字列）
			publishedAt := ""
			if dateStr != "" {
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					publishedAt = t.Format(time.RFC3339)
				}
			}

			// 記事ページから Excerpt を取得
			excerpt := ""
			if doc, err := fetchDoc(articleURL, cfg); err == nil {
				doc.Find("script, style").Remove()
				sel := doc.Find("div.text-component")
				if sel.Length() > 0 {
					var parts []string
					sel.Each(func(_ int, s *goquery.Selection) {
						t := strings.TrimSpace(s.Text())
						if t != "" {
							parts = append(parts, t)
						}
					})
					excerpt = strings.Join(parts, "\n")
				}
			}

			out = append(out, Headline{
				Source:      "PwC Japan",
				Title:       title,
				URL:         articleURL,
				PublishedAt: publishedAt,
				Excerpt:     excerpt,
	
			})
		}
	}

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// collectHeadlinesMizuhoRT は みずほリサーチ&テクノロジーズから見出しを収集する
func collectHeadlinesMizuhoRT(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// 今年のパブリケーションページ（最新レポート一覧）を使用
	currentYear := time.Now().Year()
	newsURL := fmt.Sprintf("https://www.mizuho-rt.co.jp/publication/%d/index.html", currentYear)

	client := cfg.Client
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
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	out := make([]Headline, 0, limit)
	datePattern := reJapaneseDateYMD

	// リストアイテムから記事を抽出
	doc.Find(".section__news-list-item").Each(func(i int, item *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// リンクからタイトルとURLを取得
		link := item.Find(".section__news-link")
		title := strings.TrimSpace(link.Text())
		if title == "" {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// 絶対URLを構築
		articleURL := href
		if strings.HasPrefix(href, "/") {
			articleURL = "https://www.mizuho-rt.co.jp" + href
		}

		// .section__news-date から日付を抽出
		dateStr := ""
		dateText := strings.TrimSpace(item.Find(".section__news-date").Text())
		if matches := datePattern.FindStringSubmatch(dateText); len(matches) == 4 {
			year := matches[1]
			month := matches[2]
			day := matches[3]
			if len(month) == 1 {
				month = "0" + month
			}
			if len(day) == 1 {
				day = "0" + day
			}
			dateStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day)
		}

		// 記事ページから Excerpt と日付を取得
		excerpt, pageDate := fetchMizuhoArticleDetail(articleURL, client, cfg.UserAgent)
		if pageDate != "" && dateStr == "" {
			dateStr = pageDate
		}

		out = append(out, Headline{
			Source:      "Mizuho Research & Technologies",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// fetchMizuhoArticleDetail は Mizuho の記事ページから Excerpt と日付を取得する
func fetchMizuhoArticleDetail(articleURL string, client *http.Client, userAgent string) (excerpt string, dateStr string) {
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

	// report-detail_post から Excerpt を抽出
	post := doc.Find(".report-detail_post")
	if post.Length() > 0 {
		excerpt = strings.TrimSpace(post.Text())
	}

	// <time> タグから日付を抽出
	datePattern := reJapaneseDateYMD
	doc.Find("time").Each(func(i int, s *goquery.Selection) {
		if dateStr != "" {
			return
		}
		t := strings.TrimSpace(s.Text())
		if matches := datePattern.FindStringSubmatch(t); len(matches) == 4 {
			month := matches[2]
			day := matches[3]
			if len(month) == 1 {
				month = "0" + month
			}
			if len(day) == 1 {
				day = "0" + day
			}
			dateStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", matches[1], month, day)
		}
	})

	return excerpt, dateStr
}
