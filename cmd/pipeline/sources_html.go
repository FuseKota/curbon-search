// =============================================================================
// sources_html.go - HTMLスクレイピングソース
// =============================================================================
//
// このファイルはHTMLスクレイピングを使用するニュースソースを定義します。
// goquery ライブラリを使用してHTML構造から記事情報を抽出します。
//
// 【含まれるソース】
//   1. ICAP               - 国際カーボンアクションパートナーシップ
//   2. IETA               - 国際排出量取引協会
//   3. Energy Monitor     - エネルギー転換ニュース
//   4. World Bank         - 世界銀行気候変動
//   5. NewClimate         - 気候研究機関
//   7. Carbon Knowledge Hub - 教育プラットフォーム
//   8. Verra              - VCS規格運営団体
//   9. Gold Standard      - 高品質カーボンクレジット規格
//  10. ACR                - American Carbon Registry
//  11. CAR                - Climate Action Reserve
//  12. UNFCCC             - 国連気候変動枠組条約
//  13. IISD ENB           - 環境交渉速報
//  14. Climate Focus      - 気候政策コンサルティング
//  15. Puro.earth         - 炭素除去認証プラットフォーム
//  16. Isometric          - 炭素除去検証
//
// =============================================================================
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// collectHeadlinesICAP は ICAP（Drupalサイト）からHTMLスクレイピングで記事を取得する
func collectHeadlinesICAP(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://icapcarbonaction.com/en/news"

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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	doc.Find("article.news-embed-grid").Each(func(_ int, article *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		// タイトルを抽出
		titleElem := article.Find("h3.content-title a.link-title span")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			return
		}

		// URLを抽出
		linkElem := article.Find("a.link-title")
		href, exists := linkElem.Attr("href")
		if !exists || href == "" {
			return
		}
		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// 日付を抽出
		timeElem := article.Find("time")
		datetime, _ := timeElem.Attr("datetime")

		// 記事ページから全文を取得
		content := ""
		if articleURL != "" {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil {
					if articleResp.StatusCode == http.StatusOK {
						articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
						if err == nil {
							var parts []string
							articleDoc.Find(".paragraph--type--text").Each(func(_ int, s *goquery.Selection) {
								t := strings.TrimSpace(s.Text())
								if t != "" {
									parts = append(parts, t)
								}
							})
							content = strings.Join(parts, "\n\n")
						}
					}
					articleResp.Body.Close() // err == nilの場合は必ずBodyをClose
				}
			}
		}

		out = append(out, Headline{
			Source:      "ICAP",
			Title:       title,
			URL:         articleURL,
			PublishedAt: datetime,
			Excerpt:     content,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ICAP: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIETA は IETAからHTMLスクレイピングで記事を取得する
func collectHeadlinesIETA(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	homeURL := "https://www.ieta.org/"

	client := cfg.Client
	req, err := http.NewRequest("GET", homeURL, nil)
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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// ニュース項目を検索 - card-bodyコンテナを探す
	doc.Find("div.card-body").Each(func(_ int, cardBody *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		// タイトルを抽出
		titleElem := cardBody.Find("h3.news-title")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			return
		}

		// 日付を抽出（同じcard-body内）
		dateElem := cardBody.Find("div.resource-date")
		dateStr := strings.TrimSpace(dateElem.Text())

		// URLを抽出（兄弟要素のa.link-cover - 親コンテナまで遡る必要あり）
		parent := cardBody.Parent()
		linkElem := parent.Find("a.link-cover")
		href, exists := linkElem.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(homeURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// 日付をRFC3339形式にパース
		publishedAt := ""
		if dateStr != "" {
			// "Dec 18, 2025"形式をパース
			t, err := time.Parse("Jan 2, 2006", dateStr)
			if err == nil {
				publishedAt = t.Format(time.RFC3339)
			}
		}

		// 記事ページから全文を取得
		content := ""
		if articleURL != "" {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil {
					if articleResp.StatusCode == http.StatusOK {
						articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
						if err == nil {
							// ニュース詳細セクションからイントロ+本文テキストを抽出
							var parts []string
							articleDoc.Find(".section-news-detail .intro, .section-news-detail section.bg-white").Each(func(_ int, s *goquery.Selection) {
								t := strings.TrimSpace(s.Text())
								if t != "" {
									parts = append(parts, t)
								}
							})
							content = strings.Join(parts, "\n\n")
						}
					}
					articleResp.Body.Close() // err == nilの場合は必ずBodyをClose
				}
			}
		}

		out = append(out, Headline{
			Source:      "IETA",
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAt,
			Excerpt:     content,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IETA: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesEnergyMonitor は Energy MonitorからHTMLスクレイピングで記事を取得する
func collectHeadlinesEnergyMonitor(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.energymonitor.ai/news/"

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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// 記事要素を検索
	doc.Find("article").Each(func(_ int, article *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		// h3 > a からタイトルとURLを抽出
		linkElem := article.Find("h3 a")
		title := strings.TrimSpace(linkElem.Text())
		if title == "" {
			return
		}

		href, exists := linkElem.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// 記事ページから全文を取得
		content := ""
		publishedAt := ""
		if articleURL != "" {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil {
					if articleResp.StatusCode == http.StatusOK {
						articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
						if err == nil {
							// コンテンツを検索
							bodyElem := articleDoc.Find("article .entry-content, article .article-content, .post-content, .content").First()
							content = strings.TrimSpace(bodyElem.Text())

							// JSON-LD構造化データから公開日を検索
							articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
								if publishedAt != "" {
									return
								}
								jsonText := script.Text()
								// JSON-LDからdatePublishedを抽出
								if matches := reDatePublishedJSON.FindStringSubmatch(jsonText); len(matches) > 1 {
									publishedAt = matches[1]
								}
							})

							// フォールバック: time要素を試行
							if publishedAt == "" {
								timeElem := articleDoc.Find("time")
								datetime, exists := timeElem.Attr("datetime")
								if exists {
									publishedAt = datetime
								}
							}
						}
					}
					articleResp.Body.Close() // err == nilの場合は必ずBodyをClose
				}
			}
		}

		out = append(out, Headline{
			Source:      "Energy Monitor",
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAt,
			Excerpt:     content,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Energy Monitor: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesWorldBank は 世界銀行の気候変動関連出版物からヘッドラインを収集する
func collectHeadlinesWorldBank(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// World Bank News Search APIでcarbon関連記事のURL・日付を取得し、
	// 各ページをスクレイピングしてタイトル・本文を取得する
	apiURL := fmt.Sprintf(
		"https://search.worldbank.org/api/v2/news?format=json&qterm=%%22carbon+pricing%%22+OR+%%22carbon+market%%22+OR+%%22carbon+credit%%22+OR+%%22emissions+trading%%22&rows=%d&os=0&srt=lnchdt&order=desc&fl=url,lnchdt,title,descr&lang_exact=English",
		limit,
	)

	var result struct {
		Documents map[string]json.RawMessage `json:"documents"`
	}
	if err := httpGetJSON(apiURL, cfg, &result); err != nil {
		return nil, fmt.Errorf("failed to fetch World Bank API: %w", err)
	}

	out := make([]Headline, 0, limit)
	for _, raw := range result.Documents {
		if len(out) >= limit {
			break
		}
		var doc struct {
			URL    string `json:"url"`
			Lnchdt string `json:"lnchdt"`
			Title  struct {
				Cdata string `json:"cdata!"`
			} `json:"title"`
			Descr struct {
				Cdata string `json:"cdata!"`
			} `json:"descr"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil || doc.URL == "" {
			continue
		}

		// HTTPをHTTPSに変換
		articleURL := strings.Replace(doc.URL, "http://", "https://", 1)

		dateStr := doc.Lnchdt // すでにRFC3339形式

		// APIにタイトルがあればそれを使用、なければページから取得
		title := doc.Title.Cdata
		excerpt := doc.Descr.Cdata

		// ページをスクレイピングしてタイトル・本文を補完
		pageDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			// タイトルが空の場合、ページから取得
			if title == "" {
				h1 := pageDoc.Find("h1").First()
				if h1.Length() > 0 {
					title = strings.TrimSpace(h1.Text())
				}
			}
			// 本文を<p>タグから取得
			var parts []string
			pageDoc.Find("p").Each(func(_ int, s *goquery.Selection) {
				text := strings.TrimSpace(s.Text())
				if len(text) > 50 {
					parts = append(parts, text)
				}
			})
			if bodyText := strings.Join(parts, "\n\n"); len(bodyText) > len(excerpt) {
				excerpt = bodyText
			}
		}

		if title == "" {
			continue
		}

		out = append(out, Headline{
			Source:      "World Bank",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	return out, nil
}

// collectHeadlinesNewClimate は NewClimate Instituteからヘッドラインを収集する
func collectHeadlinesNewClimate(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://newclimate.org/news"

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

	// ニュース項目をパース
	doc.Find("a[href^='/news/'], a[href^='/resources/publications/']").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists {
			return
		}

		// タイトルを取得 - リンクテキストまたは子要素内にある場合がある
		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		// 絶対URLを構築
		articleURL := "https://newclimate.org" + href

		// 親要素または兄弟要素から日付を抽出（見つからない場合は空文字列）
		dateStr := ""
		parent := link.Parent().Parent()
		dateElem := parent.Find("time, .date, span[class*='date']")
		if dateElem.Length() > 0 {
			if dateAttr, exists := dateElem.Attr("datetime"); exists {
				dateStr = dateAttr
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				if t, err := time.Parse("2 January 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				} else if t, err := time.Parse("January 2, 2006", dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
				}
			}
		}

		// 記事ページから日付とコンテンツを取得
		excerpt := ""
		articleDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			// event-detailsカレンダーから日付を抽出
			if dateStr == "" {
				articleDoc.Find(".event-details__name--calendar").Each(func(_ int, s *goquery.Selection) {
					if dateStr != "" {
						return
					}
					valElem := s.Parent().Find(".event-details__value")
					dateText := strings.TrimSpace(valElem.Text())
					if t, err := time.Parse("02 Jan 2006", dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
					}
				})
			}
			// node__contentからコンテンツを抽出
			nodeContent := articleDoc.Find(".node__content")
			if nodeContent.Length() > 0 {
				excerpt = strings.TrimSpace(nodeContent.Text())
			}
		}

		out = append(out, Headline{
			Source:      "NewClimate Institute",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// collectHeadlinesCarbonKnowledgeHub は Carbon Knowledge Hubからヘッドラインを収集する
func collectHeadlinesCarbonKnowledgeHub(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.carbonknowledgehub.com"

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
	seen := make(map[string]bool) // 重複回避のためURLを追跡

	// メインセレクタ: css-oxwq25クラスのリンク（メイン記事リンク）
	doc.Find("a.css-oxwq25, a[class*='css-']").Each(func(i int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// コンテンツURLでフィルタリング
		// サイトは単数形・複数形の両方を使用
		// カテゴリページではなく実際の記事URLが必要なので、スラッシュが複数あることを確認
		isContentURL := (strings.Contains(href, "/factsheet") ||
			strings.Contains(href, "/story") ||
			strings.Contains(href, "/stories") ||
			strings.Contains(href, "/audio") ||
			strings.Contains(href, "/media") ||
			strings.Contains(href, "/news")) &&
			strings.Count(href, "/") > 1 // カテゴリページだけでないことを確認

		if !isContentURL {
			return
		}

		// 絶対URLを構築
		articleURL := href
		if !strings.HasPrefix(href, "http") {
			if strings.HasPrefix(href, "/") {
				articleURL = "https://www.carbonknowledgehub.com" + href
			} else {
				articleURL = "https://www.carbonknowledgehub.com/" + href
			}
		}

		// 既に処理済みならスキップ
		if seen[articleURL] {
			return
		}

		// タイトルを取得
		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		// 一般的なナビゲーションテキストをスキップ
		titleLower := strings.ToLower(title)
		if strings.Contains(titleLower, "read more") ||
			strings.Contains(titleLower, "learn more") ||
			strings.Contains(titleLower, "click here") ||
			strings.Contains(titleLower, "view all") {
			return
		}

		seen[articleURL] = true

		// 各記事ページから日付・本文を取得（Next.js SSR + __NEXT_DATA__）
		dateStr := ""
		excerpt := ""
		articleDoc, err := fetchDoc(articleURL, cfg)
		if err == nil {
			// __NEXT_DATA__ JSONからfrontMatterを取得
			articleDoc.Find("script#__NEXT_DATA__").Each(func(_ int, s *goquery.Selection) {
				var nextData struct {
					Props struct {
						PageProps struct {
							Source struct {
								Frontmatter struct {
									Date        string `json:"date"`
									Description string `json:"description"`
								} `json:"frontmatter"`
							} `json:"source"`
						} `json:"pageProps"`
					} `json:"props"`
				}
				if err := json.Unmarshal([]byte(s.Text()), &nextData); err == nil {
					fm := nextData.Props.PageProps.Source.Frontmatter
					if fm.Date != "" {
						if t, err := time.Parse("2006-01-02", fm.Date); err == nil {
							dateStr = t.Format(time.RFC3339)
						}
					}
					if fm.Description != "" {
						excerpt = fm.Description
					}
				}
			})
			// SSRプリレンダリングされた本文からテキストを補完
			// Next.jsアプリのため<main>タグはなく、div#__nextにコンテンツがある
			mainContent := articleDoc.Find("div#__next")
			if mainContent.Length() > 0 {
				mainContent.Find("script, style, nav, header, footer, noscript").Remove()
				bodyText := strings.TrimSpace(mainContent.Text())
				if len(bodyText) > len(excerpt) {
					lines := strings.Split(bodyText, "\n")
					var cleaned []string
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line != "" {
							cleaned = append(cleaned, line)
						}
					}
					// 1行目はパンくずリスト・タイトル・メタ情報が結合しているのでスキップ
					if len(cleaned) > 1 {
						cleaned = cleaned[1:]
					}
					excerpt = strings.Join(cleaned, "\n")
				}
			}
		}

		out = append(out, Headline{
			Source:      "Carbon Knowledge Hub",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	// 記事が見つからない場合は空スライスを返す（エラーではない）
	return out, nil
}

// =============================================================================
// VCM認証機関
// =============================================================================

// collectHeadlinesVerra は Verra（VCS規格運営団体）からニュースを取得する
//
// Verraは世界で最も広く利用されている自主的GHGプログラムである
// Verified Carbon Standard (VCS) を管理している。
func collectHeadlinesVerra(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://verra.org/news/feed/"

	feed, err := fetchRSSFeed(feedURL, cfg)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in Verra RSS feed")
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

		articleURL := item.Link

		// 日付をパース（取得できない場合は空文字列）
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		}

		// RSSからコンテンツを取得
		excerpt := extractRSSExcerpt(item)

		// RSSコンテンツが短い場合は記事全文を取得
		if len(excerpt) < 200 {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						// 複数のコンテンツセレクタを試行
						selectors := []string{".entry-content", "article", ".post-content", "main"}
						for _, sel := range selectors {
							bodyElem := articleDoc.Find(sel)
							if bodyElem.Length() > 0 {
								content := strings.TrimSpace(bodyElem.Text())
								content = reWhitespace.ReplaceAllString(content, " ")
								if len(content) > 100 {
									excerpt = content
									break
								}
							}
						}
					}
				}
			}
		}

		out = append(out, Headline{
			Source:      "Verra",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Verra: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesGoldStandard は Gold Standardからニュースを取得する
//
// Gold Standardは持続可能な開発に重点を置いた
// カーボンオフセットプロジェクトの認証規格である。
func collectHeadlinesGoldStandard(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.goldstandard.org/newsroom"

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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// Gold Standardはh4.titleで記事タイトル、time要素で日付を使用
	doc.Find("a[href*='/news/'], a[href*='/events/'], a[href*='/consultations/'], a[href*='/publications/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// 記事以外のリンクをスキップ
		if href == "/news/" || href == "/events/" || href == "/consultations/" || href == "/publications/" || href == "/newsroom" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		// リンク内またはその近くのh4.titleからタイトルを検索
		titleElem := link.Find("h4.title")
		title := strings.TrimSpace(titleElem.Text())
		if title == "" {
			// リンクテキストからタイトルを取得
			title = strings.TrimSpace(link.Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		seen[articleURL] = true

		// 親article要素内のtime要素から日付を検索
		dateStr := ""
		article := link.Closest("article")
		if article.Length() > 0 {
			timeElem := article.Find("time")
			if timeElem.Length() > 0 {
				// datetime属性を優先（ISO形式）
				if dt, exists := timeElem.Attr("datetime"); exists && dt != "" {
					if t, err := time.Parse("2006-01-02T15:04:05-0700", dt); err == nil {
						dateStr = t.UTC().Format(time.RFC3339)
					} else if t, err := time.Parse("2006-01-02T15:04:05", dt); err == nil {
						dateStr = t.Format(time.RFC3339)
					}
				}
				// フォールバック: テキストコンテンツ
				if dateStr == "" {
					rawDate := strings.TrimSpace(timeElem.Text())
					if idx := strings.Index(rawDate, " | "); idx > 0 {
						rawDate = rawDate[:idx]
					} else if idx := strings.Index(rawDate, " - "); idx > 0 {
						rawDate = rawDate[:idx]
					}
					rawDate = strings.TrimSpace(rawDate)
					if t, err := time.Parse("Jan 2, 2006", rawDate); err == nil {
						dateStr = t.Format(time.RFC3339)
					}
				}
			}
		}

		// 個別記事ページから全文を取得
		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Gold Standardは記事本文に<main>を使用
					bodyElem := articleDoc.Find("main")
					content = strings.TrimSpace(bodyElem.Text())

					// フォールバック: 一覧ページで日付が見つからない場合、記事ページから取得
					if dateStr == "" {
						if pgTime := articleDoc.Find("time[datetime]"); pgTime.Length() > 0 {
							if dt, exists := pgTime.Attr("datetime"); exists && dt != "" {
								if t, err := time.Parse("2006-01-02T15:04:05-0700", dt); err == nil {
									dateStr = t.UTC().Format(time.RFC3339)
								} else if t, err := time.Parse("2006-01-02T15:04:05", dt); err == nil {
									dateStr = t.Format(time.RFC3339)
								}
							}
						}
					}
				}
			}
		}

		out = append(out, Headline{
			Source:      "Gold Standard",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     content,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Gold Standard: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesACR は American Carbon Registryからニュースを取得する
//
// ACRはWinrock Internationalの非営利事業体で、
// 自主的カーボンオフセットプログラムを運営している。
func collectHeadlinesACR(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://acrcarbon.org/news/"

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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	doc.Find("article, .news-item, .post, div[class*='news'], div[class*='article']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		titleLink := article.Find("h2 a, h3 a, .title a, a[href*='/news/']").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		// タイトルの整形: 空白を正規化（改行・複数スペースを除去）
		title = reWhitespace.ReplaceAllString(title, " ")
		title = strings.TrimSpace(title)

		// ACR固有: タイトルから"PUBLISHED ..."サフィックスを除去
		if idx := strings.Index(title, " PUBLISHED"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
		// タイトルからカテゴリプレフィックスを除去
		acrPrefixes := []string{
			"Program Announcements ",
			"General ",
			"ACR in the News ",
			"Op-eds ",
			"Events ",
		}
		for _, prefix := range acrPrefixes {
			if strings.HasPrefix(title, prefix) {
				title = strings.TrimPrefix(title, prefix)
				title = strings.TrimSpace(title)
			}
		}

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		dateStr := ""
		foundDate := false
		dateElem := article.Find("time, .date, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"January 2, 2006",
					"Jan 2, 2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						foundDate = true
						break
					}
				}
			}
		}

		// 個別記事ページから全文を取得
		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// ACRはWordPressブロックエディタを使用 - 不要なセクションを先に除去
					articleDoc.Find("header, footer, nav, .site-header, .site-footer, script, style, noscript").Remove()

					// ページ内の全段落を収集し、記事コンテンツをフィルタリング
					var paragraphs []string
					articleDoc.Find("p").Each(func(_ int, p *goquery.Selection) {
						text := strings.TrimSpace(p.Text())
						textLower := strings.ToLower(text)

						// 短い段落をスキップ
						if len(text) < 40 {
							return
						}

						// 既知の非記事コンテンツパターンをスキップ
						skipPatterns := []string{
							"cookie", "gdpr", "privacy", "accept", "reject",
							"related news", "published", "home", "news",
							"we are using", "this website uses", "enable or disable",
							"strictly necessary", "3rd party", "save changes",
						}
						for _, pattern := range skipPatterns {
							if strings.Contains(textLower, pattern) {
								return
							}
						}

						// ナビゲーション/パンくずリストに見える場合はスキップ
						if strings.HasPrefix(text, "Home") || strings.HasPrefix(text, "News") {
							return
						}

						paragraphs = append(paragraphs, text)
					})

					if len(paragraphs) > 0 {
						content = strings.Join(paragraphs, "\n\n")
					}

					// フォールバック: メインコンテンツエリアの全テキストを試行
					if content == "" {
						mainElem := articleDoc.Find("main, article, .content, body")
						if mainElem.Length() > 0 {
							content = strings.TrimSpace(mainElem.First().Text())
							content = reWhitespace.ReplaceAllString(content, " ")
						}
					}

					// 日付が見つからない場合は記事ページから抽出を試行
					if !foundDate {
						// まずJSON-LDスキーマを試行
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if foundDate {
								return
							}
							jsonText := script.Text()
							if matches := reDatePublishedJSON.FindStringSubmatch(jsonText); len(matches) > 1 {
								dateStr = matches[1]
								foundDate = true
							}
						})
					}

					// ACR固有: 記事テキストから"PUBLISHED [date]"パターンを試行
					if !foundDate {
						articleText := articleDoc.Text()
						// "PUBLISHED January 5, 2026"や"PUBLISHED November 25, 2025"にマッチ
						publishedRe := regexp.MustCompile(`PUBLISHED\s+((?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4})`)
						if match := publishedRe.FindStringSubmatch(articleText); len(match) > 1 {
							dateText := strings.ReplaceAll(match[1], ",", "")
							if t, err := time.Parse("January 2 2006", dateText); err == nil {
								dateStr = t.Format(time.RFC3339)
								foundDate = true
							}
						}
					}
				}
			}
		}

		// 日付が見つからない場合は現在時刻にフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "ACR",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     content,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] ACR: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesCAR は Climate Action Reserveからニュースを取得する
//
// Climate Action Reserveは北米カーボン市場向けの
// カーボンオフセットレジストリである。
func collectHeadlinesCAR(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://climateactionreserve.org/updates/"

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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	doc.Find("article, .news-item, .post, div[class*='news'], div[class*='blog-post']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		titleLink := article.Find("h2 a, h3 a, .title a, .entry-title a").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title, .entry-title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		// タイトルの整形: 空白を正規化（改行・複数スペースを除去）
		title = reWhitespace.ReplaceAllString(title, " ")
		title = strings.TrimSpace(title)

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		// CAR固有: 内部ブログ記事のみ許可
		// 外部リンクや非ブログページ（プログラムページ等）をスキップ
		if !strings.Contains(articleURL, "climateactionreserve.org") {
			return
		}
		// ブログ記事のみ許可（/blog/YYYY/MM/DD/...形式のURL）
		if !strings.Contains(articleURL, "/blog/") {
			return
		}

		seen[articleURL] = true

		dateStr := ""
		foundDate := false
		dateElem := article.Find("time, .date, .entry-date, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"January 2, 2006",
					"Jan 2, 2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						foundDate = true
						break
					}
				}
			}
		}

		// 個別記事ページから全文を取得
		content := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// CARはWordPress/Elementorで様々なコンテンツセレクタを使用
					// 優先順位の順に複数のセレクタを試行
					selectors := []string{".entry-content", ".elementor-widget-theme-post-content", "article", ".post-content", "main"}
					for _, sel := range selectors {
						bodyElem := articleDoc.Find(sel)
						if bodyElem.Length() > 0 {
							content = strings.TrimSpace(bodyElem.Text())
							// コンテンツの整形: 空白を正規化
							content = reWhitespace.ReplaceAllString(content, " ")
							if len(content) > 50 {
								break
							}
						}
					}

					// 日付が見つからない場合は記事ページから抽出を試行
					if !foundDate {
						// JSON-LDスキーマを試行
						articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
							if foundDate {
								return
							}
							jsonText := script.Text()
							if matches := reDatePublishedJSON.FindStringSubmatch(jsonText); len(matches) > 1 {
								dateStr = matches[1]
								foundDate = true
							}
						})
					}
				}
			}
		}

		// 日付が見つからない場合は現在時刻にフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Climate Action Reserve",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     content,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Climate Action Reserve: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// 国際機関
// =============================================================================

// collectHeadlinesUNFCCC は 国連気候変動枠組条約（UNFCCC）からニュースを取得する
//
// UNFCCCは気候変動に関する国際条約であり、
// 世界的な気候交渉の基盤となっている。
func collectHeadlinesUNFCCC(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://unfccc.int/news"

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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	doc.Find("article, .news-item, .views-row, div[class*='teaser'], div[class*='card']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		titleLink := article.Find("h2 a, h3 a, .title a, a[href*='/news/']").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title, .field--name-title").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := titleLink.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// 日付を抽出（見つからない場合は空文字列）
		dateStr := ""
		dateElem := article.Find("time, .date, .field--name-created, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"2 January 2006",
					"January 2, 2006",
					"02/01/2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						break
					}
				}
			}
		}

		excerpt := ""
		excerptElem := article.Find("p, .field--name-body, .summary").First()
		if excerptElem.Length() > 0 {
			excerpt = strings.TrimSpace(excerptElem.Text())
		}

		out = append(out, Headline{
			Source:      "UNFCCC",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] UNFCCC: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIISD は IISD Earth Negotiations Bulletinからニュースを取得する
//
// IISD ENBは気候変動会議やカーボン市場の議論を含む
// 国際環境交渉に関するレポートを提供している。
// 注意: IISDは個別記事ページへのアクセスにCookieベースのセッションが必要。
func collectHeadlinesIISD(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://enb.iisd.org/"

	// セッション維持のためCookie jar付きクライアントを作成
	// IISDは適切なセッションCookieなしのリクエストをブロックする
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	client := &http.Client{
		Timeout: cfg.Timeout,
		Jar:     jar,
	}

	// まずホームページにアクセスしてセッションCookieを取得
	// 指数バックオフで最大3回リトライ（AWS IPは403を受けやすい）
	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * 2 * time.Second
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: homepage 403, retrying in %v (attempt %d/3)...\n", delay, attempt+1)
			}
			time.Sleep(delay)
		}
		req, err := http.NewRequest("GET", newsURL, nil)
		if err != nil {
			return nil, fmt.Errorf("request creation failed: %w", err)
		}
		req.Header.Set("User-Agent", cfg.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d (after 3 retries)", resp.StatusCode)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// テキストから日付を抽出するヘルパー関数
	extractDate := func(text string) string {
		datePatterns := []struct {
			regex  string
			format string
		}{
			{`(\d{1,2}\s+(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{4})`, "2 January 2006"},
			{`((?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4})`, "January 2, 2006"},
		}
		for _, dp := range datePatterns {
			re := regexp.MustCompile(dp.regex)
			if match := re.FindStringSubmatch(text); len(match) > 1 {
				dateText := strings.ReplaceAll(match[1], ",", "")
				if t, err := time.Parse(dp.format, dateText); err == nil {
					return t.Format(time.RFC3339)
				}
			}
		}
		return ""
	}

	// 記事ページを取得して全文を抽出するヘルパー関数
	// 戻り値: About (og:description) + 本文全文
	fetchArticleContent := func(articleURL string) string {
		time.Sleep(500 * time.Millisecond) // レート制限回避のためリクエスト間に遅延

		var resp *http.Response
		// 403時に最大3回リトライ（bot保護/レート制限）
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
			req, err := http.NewRequest("GET", articleURL, nil)
			if err != nil {
				return ""
			}
			req.Header.Set("User-Agent", cfg.UserAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			req.Header.Set("Referer", newsURL)

			resp, err = client.Do(req)
			if err != nil {
				if os.Getenv("DEBUG_SCRAPING") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: failed to fetch %s: %v\n", articleURL, err)
				}
				return ""
			}
			if resp.StatusCode == http.StatusOK {
				break
			}
			resp.Body.Close()
			if resp.StatusCode != 403 {
				if os.Getenv("DEBUG_SCRAPING") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: status %d for %s\n", resp.StatusCode, articleURL)
				}
				return ""
			}
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: 403 for %s, retrying...\n", articleURL)
			}
		}
		if resp.StatusCode != http.StatusOK {
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: status %d for %s after retries\n", resp.StatusCode, articleURL)
			}
			return ""
		}
		defer resp.Body.Close()

		articleDoc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return ""
		}

		var parts []string

		// 1. og:descriptionからAboutセクションを取得
		if about := articleDoc.Find("meta[property='og:description']").AttrOr("content", ""); about != "" {
			parts = append(parts, "【About】\n"+strings.TrimSpace(about))
		}

		// 2. 全ての.c-wysiwyg__contentセクションから本文を取得
		// 記事は画像で区切られた複数セクションを持つ場合がある
		var bodyParts []string
		seen := make(map[string]bool) // 重複を回避

		articleDoc.Find(".c-wysiwyg__content").Each(func(_ int, section *goquery.Selection) {
			// このセクションから段落を取得
			section.Find("p").Each(func(_ int, p *goquery.Selection) {
				text := strings.TrimSpace(p.Text())
				// 短いテキスト、メタデータ、ニュースレター購読テキストをスキップ
				if len(text) > 50 && !strings.Contains(text, "subscribe to the ENB") &&
					!strings.Contains(text, "Earth Negotiations Bulletin writers") &&
					!seen[text] {
					seen[text] = true
					bodyParts = append(bodyParts, text)
				}
			})

			// リスト項目も取得（ハイライト、議題等）
			section.Find("li").Each(func(_ int, li *goquery.Selection) {
				text := strings.TrimSpace(li.Text())
				if len(text) > 20 && !seen[text] {
					seen[text] = true
					bodyParts = append(bodyParts, "• "+text)
				}
			})
		})

		if len(bodyParts) > 0 {
			parts = append(parts, "\n【Content】\n"+strings.Join(bodyParts, "\n\n"))
		}

		if len(parts) == 0 {
			// 何も見つからない場合はmeta descriptionにフォールバック
			if desc := articleDoc.Find("meta[name='description']").AttrOr("content", ""); desc != "" {
				return strings.TrimSpace(desc)
			}
			return ""
		}

		return strings.Join(parts, "\n")
	}

	// まずfeatured boxから収集（一覧ページにサマリーあり）
	doc.Find("a.c-featured-box, .c-featured-box").Each(func(_ int, box *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// リンクを取得 - 要素自体または子要素から
		var href string
		if h, exists := box.Attr("href"); exists {
			href = h
		} else {
			href, _ = box.Find("a").First().Attr("href")
		}
		if href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// タイトルを取得
		title := strings.TrimSpace(box.Find(".c-featured-box__title, h3, h4").First().Text())
		if title == "" || len(title) < 10 {
			return
		}

		// 常に記事ページから全文を取得（About + Content）
		excerpt := fetchArticleContent(articleURL)

		// boxテキストから日付を抽出
		boxText := box.Text()
		dateStr := extractDate(boxText)
		if dateStr == "" {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "IISD ENB",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	// 上限に達していない場合、hero items（現在のイベント）からも収集
	if len(out) < limit {
		doc.Find(".c-hero-item").Each(func(_ int, hero *goquery.Selection) {
			if len(out) >= limit {
				return
			}

			link := hero.Find("a[href]").First()
			href, exists := link.Attr("href")
			if !exists || href == "" {
				return
			}

			articleURL := resolveURL(newsURL, href)
			if articleURL == "" || seen[articleURL] {
				return
			}
			seen[articleURL] = true

			title := strings.TrimSpace(hero.Find("h2, h3, .c-hero-item__title").First().Text())
			if title == "" || len(title) < 10 {
				return
			}

			// hero itemsには一覧ページに説明がない
			// セッションCookieを使用して個別記事ページからコンテンツを取得
			excerpt := fetchArticleContent(articleURL)

			dateStr := extractDate(hero.Text())
			if dateStr == "" {
				dateStr = time.Now().Format(time.RFC3339)
			}

			out = append(out, Headline{
				Source:      "IISD ENB",
				Title:       title,
				URL:         articleURL,
				PublishedAt: dateStr,
				Excerpt:     excerpt,
	
			})
		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] IISD ENB: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesClimateFocus は Climate Focusから出版物を取得する
//
// Climate Focusはカーボン市場や気候ファイナンスに関する
// 調査を発行する気候政策アドバイザリー企業である。
func collectHeadlinesClimateFocus(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	publicationsURL := "https://climatefocus.com/publications/"

	client := cfg.Client
	req, err := http.NewRequest("GET", publicationsURL, nil)
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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	// Climate Focus - 出版物へのリンクを直接検索
	doc.Find("a[href*='/publications/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		// メイン出版物ページ、ページネーション、またはステージングURLの場合はスキップ
		if href == publicationsURL || href == "https://climatefocus.com/publications/" {
			return
		}
		if strings.Contains(href, "sf_paged=") || strings.Contains(href, "wpengine.com") {
			return
		}

		articleURL := resolveURL(publicationsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// リンクテキストまたは画像altからタイトルを取得
		title := strings.TrimSpace(link.Text())
		if title == "" {
			imgElem := link.Find("img")
			if imgElem.Length() > 0 {
				title, _ = imgElem.Attr("alt")
				title = strings.TrimSpace(title)
			}
		}

		// まだタイトルがない場合、URLから抽出
		if title == "" || len(title) < 10 {
			// URLパスからタイトルを抽出
			parts := strings.Split(href, "/")
			for i := len(parts) - 1; i >= 0; i-- {
				if parts[i] != "" {
					title = strings.ReplaceAll(parts[i], "-", " ")
					title = strings.Title(title)
					break
				}
			}
		}

		if title == "" || len(title) < 10 {
			return
		}

		// 兄弟/親要素からカテゴリを取得
		excerpt := ""
		parent := link.Parent()
		categoryElem := parent.Find(".category")
		if categoryElem.Length() > 0 {
			excerpt = "Category: " + strings.TrimSpace(categoryElem.Text())
		}

		// 個別記事ページから日付とコンテンツを取得
		dateStr := ""
		foundDate := false
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// 様々な場所から日付を検索
					// 1. JSON-LDスキーマ（datePublished）を試行
					articleDoc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
						if foundDate {
							return
						}
						text := script.Text()
						// datePublishedパターンを検索
						if strings.Contains(text, "datePublished") {
							if match := reDatePublishedJSON.FindStringSubmatch(text); len(match) > 1 {
								dateStr = match[1]
								foundDate = true
							}
						}
					})

					// 2. 表示されている日付テキスト "Jan 2026" 形式を試行
					if !foundDate {
						articleDoc.Find(".date, time, span[class*='date']").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							text := strings.TrimSpace(elem.Text())
							// "Jan 2026" 形式を試行（短縮月名 + 年）
							re := regexp.MustCompile(`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{4})`)
							if match := re.FindStringSubmatch(text); len(match) > 2 {
								// 月の1日としてパース
								dateText := match[1] + " 1, " + match[2]
								if t, err := time.Parse("Jan 2, 2006", dateText); err == nil {
									dateStr = t.Format(time.RFC3339)
									foundDate = true
								}
							}
						})
					}

					// 3. ページから記事全文を抽出
					// まず不要な要素を除去
					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related-posts").Remove()

					// 記事コンテンツを検索
					contentSelectors := []string{
						".entry-content",
						".article-content",
						".post-content",
						".content-area",
						"article .content",
						"main article",
						".elementor-widget-theme-post-content",
					}
					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							// 段落からテキストを取得
							var paragraphs []string
							contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
								text := strings.TrimSpace(p.Text())
								if len(text) > 30 {
									paragraphs = append(paragraphs, text)
								}
							})
							if len(paragraphs) > 0 {
								excerpt = strings.Join(paragraphs, "\n\n")
								break
							}
						}
					}

					// コンテンツが見つからない場合はmeta descriptionにフォールバック
					if excerpt == "" {
						excerptElem := articleDoc.Find("meta[name='description'], meta[property='og:description']")
						if excerptElem.Length() > 0 {
							excerpt, _ = excerptElem.Attr("content")
							excerpt = strings.TrimSpace(excerpt)
						}
					}
				}
			}
		}

		// 日付が見つからない場合は現在時刻にフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Climate Focus",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Climate Focus: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// 追加ソース（フェーズ5）
// =============================================================================

// collectHeadlinesPuroEarth は Puro.earthからブログ記事を取得する
//
// Puro.earthは炭素除去プロジェクトとクレジットの認証を提供する
// 炭素除去マーケットプレイスである。ブログにはニュース、
// 方法論の更新、業界のインサイトが掲載されている。
//
// 手法: Atom Feed (gofeed) + 全文取得用HTMLスクレイピング
// URL: https://puro.earth/blog/our-blogs-1/feed
func collectHeadlinesPuroEarth(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	feedURL := "https://puro.earth/blog/our-blogs-1/feed"

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

		articleURL := item.Link
		if articleURL == "" {
			continue
		}

		// 日付をパース（取得できない場合は空文字列）
		dateStr := ""
		if item.PublishedParsed != nil {
			dateStr = item.PublishedParsed.Format(time.RFC3339)
		} else if item.UpdatedParsed != nil {
			dateStr = item.UpdatedParsed.Format(time.RFC3339)
		}

		// 記事ページから全文を取得
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// Puro.earthは特定のクラス名を持つOdoo CMSを使用
					// コンテンツは<p>タグまたは<br>タグ間の直接テキストノードにある場合がある
					contentSelectors := []string{
						".o_wblog_post_content_field",
						".o_wblog_read_text",
					}

					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							// まず<p>タグからコンテンツを取得
							var contentParts []string
							contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
								text := strings.TrimSpace(p.Text())
								if len(text) > 30 {
									contentParts = append(contentParts, text)
								}
							})

							// <p>タグに十分なコンテンツがない場合、全テキストを取得
							// （Puro.earthは<br>区切りの直接テキストを使うことがある）
							if len(strings.Join(contentParts, "")) < 200 {
								fullText := strings.TrimSpace(contentElem.Text())
								// 空白を正規化（複数スペース/改行を単一改行に）
								fullText = reWhitespace.ReplaceAllString(fullText, " ")
								// 論理的な区切り（ピリオド+大文字の文頭）で段落に分割
								fullText = regexp.MustCompile(`\. ([A-Z])`).ReplaceAllString(fullText, ".\n\n$1")
								if len(fullText) > 100 {
									excerpt = fullText
									break
								}
							} else {
								excerpt = strings.Join(contentParts, "\n\n")
								break
							}
						}
					}
				}
			}
		}

		// 記事取得に失敗した場合はフィードのdescriptionにフォールバック
		if excerpt == "" {
			if item.Description != "" {
				excerpt = item.Description
			} else if item.Content != "" {
				excerpt = item.Content
			}
			// HTMLタグを除去
			excerpt = reHTMLTags.ReplaceAllString(excerpt, "")
			excerpt = reWhitespace.ReplaceAllString(excerpt, " ")
			excerpt = strings.TrimSpace(excerpt)
		}

		out = append(out, Headline{
			Source:      "Puro.earth",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	}

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Puro.earth: collected %d headlines\n", len(out))
	}

	return out, nil
}

// collectHeadlinesIsometric は Isometricからリソースを取得する
//
// Isometricは炭素除去に関する調査・リソースを発行する
// 科学ベースの炭素除去検証企業である。
//
// HTML構造:
// - タイトル: p.writing-card-title
// - 日付: div.label-small.cc-date（形式: "Oct 20, 2025"）
// - サブタイトル: div.u-text-grey80.u-hide
func collectHeadlinesIsometric(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	resourcesURL := "https://isometric.com/writing"

	client := cfg.Client
	req, err := http.NewRequest("GET", resourcesURL, nil)
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
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	out := make([]Headline, 0, limit)
	seen := make(map[string]bool)

	doc.Find("a[href*='/writing-articles/']").Each(func(_ int, link *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(resourcesURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// p.writing-card-titleからタイトルを検索
		title := strings.TrimSpace(link.Find("p.writing-card-title").Text())
		if title == "" || len(title) < 10 {
			return
		}

		// div.cc-dateから日付を検索（見つからない場合は空文字列）
		dateStr := ""
		foundDate := false
		dateElem := link.Find("div.cc-date, .label-small.cc-date")
		if dateElem.Length() > 0 {
			dateText := strings.TrimSpace(dateElem.First().Text())
			// 形式: "Oct 20, 2025" または "Jan 21, 2026"
			for _, format := range []string{
				"Jan 2, 2006",
				"Jan 02, 2006",
				"January 2, 2006",
			} {
				if t, err := time.Parse(format, dateText); err == nil {
					dateStr = t.Format(time.RFC3339)
					foundDate = true
					break
				}
			}
		}

		// 一覧ページからサブタイトルを初期excerptとして取得
		subtitle := strings.TrimSpace(link.Find("div.u-text-grey80").Text())

		// 記事ページから全文を取得
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// 一覧ページで日付が見つからない場合、記事ページから取得を試行
					if !foundDate {
						articleDoc.Find("div.cc-date, .label-small.cc-date, time").Each(func(_ int, dateEl *goquery.Selection) {
							if foundDate {
								return
							}
							dateText := strings.TrimSpace(dateEl.Text())
							for _, format := range []string{
								"Jan 2, 2006",
								"Jan 02, 2006",
								"January 2, 2006",
							} {
								if t, err := time.Parse(format, dateText); err == nil {
									dateStr = t.Format(time.RFC3339)
									foundDate = true
									break
								}
							}
						})
					}

					// 記事本文からコンテンツを抽出
					contentSelectors := []string{
						".rich-text",
						".w-richtext",
						"article",
						".article-content",
						".content",
					}

					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							var contentParts []string
							contentElem.Find("p").Each(func(_ int, p *goquery.Selection) {
								text := strings.TrimSpace(p.Text())
								if len(text) > 30 {
									contentParts = append(contentParts, text)
								}
							})
							if len(contentParts) > 0 {
								excerpt = strings.Join(contentParts, "\n\n")
								break
							}
						}
					}
				}
			}
		}

		// 記事取得に失敗した場合はサブタイトルにフォールバック
		if excerpt == "" && subtitle != "" {
			excerpt = subtitle
		} else if excerpt == "" {
			excerpt = title
		}

		out = append(out, Headline{
			Source:      "Isometric",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Isometric: collected %d headlines\n", len(out))
	}

	return out, nil
}

