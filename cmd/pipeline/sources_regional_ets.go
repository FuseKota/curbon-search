// sources_regional_ets.go - 地域別排出権取引ソース
// =============================================================================
//
// 地域別排出権取引制度および規制機関のソースを定義する。
//
// ソース一覧:
//   1. EU ETS (EC)      - 欧州委員会ETSニュース
//   2. California CARB  - カリフォルニア大気資源局
//   3. RGGI             - 地域温室効果ガスイニシアティブ
//   4. Australia CER    - オーストラリア クリーンエネルギー規制当局
//   5. UK ETS           - 英国政府ETS出版物（HTMLスクレイピング）
//
// =============================================================================
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ledongthuc/pdf"
)

// =============================================================================
// PDFテキスト抽出ヘルパー
// =============================================================================

// extractTextFromPDF は指定URLからPDFをダウンロードしてテキストを抽出する
func extractTextFromPDF(pdfURL string, client *http.Client, userAgent string) (string, error) {
	req, err := http.NewRequest("GET", pdfURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// PDFコンテンツをメモリに読み込む
	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	// PDFデータからリーダーを作成
	reader := bytes.NewReader(pdfData)
	pdfReader, err := pdf.NewReader(reader, int64(len(pdfData)))
	if err != nil {
		return "", fmt.Errorf("failed to parse PDF: %w", err)
	}

	// 全ページからテキストを抽出
	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	// 抽出テキストをクリーンアップ
	result := textBuilder.String()
	result = strings.TrimSpace(result)
	// 空白を正規化
	result = strings.Join(strings.Fields(result), " ")

	return result, nil
}

// =============================================================================
// EU ETS（欧州委員会）ソース
// =============================================================================

// collectHeadlinesEUETS は欧州委員会ETSページからニュースを取得する
//
// 欧州委員会の気候変動対策サイトから、EU排出権取引制度に関する
// 公式ニュースと更新情報を提供する。
func collectHeadlinesEUETS(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://climate.ec.europa.eu/news-other-reads/news_en"

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

	// ECサイトはニュースアイテムカードを使用
	doc.Find("article, .ecl-card, .news-item, div[class*='news'], div[class*='listing-item']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// タイトルリンクを取得
		titleLink := article.Find("h2 a, h3 a, .ecl-card__title a, .title a, a[class*='title']").First()
		if titleLink.Length() == 0 {
			titleLink = article.Find("a").First()
		}

		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
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

		// 一覧ページから日付を抽出
		dateStr := ""
		foundDate := false
		dateElem := article.Find("time, .date, .ecl-date-block, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"2 January 2006",
					"02/01/2006",
					"2006-01-02",
					"02 January 2006",
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
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// 不要な要素を除去
					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related").Remove()

					// 日付が未取得の場合、記事ページから抽出を試行
					if !foundDate {
						articleDoc.Find("time, .date, meta[property='article:published_time']").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							if datetime, exists := elem.Attr("datetime"); exists {
								dateStr = datetime
								foundDate = true
							} else if content, exists := elem.Attr("content"); exists {
								dateStr = content
								foundDate = true
							}
						})
					}

					// 記事本文からコンテンツを抽出
					contentSelectors := []string{
						".ecl-editor",
						".ecl-page-content",
						"article .content",
						".field--name-body",
						"main article",
						".page-content",
					}
					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
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

					// フォールバック: メインコンテンツから全段落を取得
					if excerpt == "" {
						var paragraphs []string
						articleDoc.Find("main p, article p").Each(func(_ int, p *goquery.Selection) {
							text := strings.TrimSpace(p.Text())
							if len(text) > 40 {
								paragraphs = append(paragraphs, text)
							}
						})
						if len(paragraphs) > 0 {
							excerpt = strings.Join(paragraphs, "\n\n")
						}
					}
				}
			}
		}

		// 日付が見つからない場合は現在時刻をフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "EU ETS",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] EU ETS: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// カリフォルニア CARBソース
// =============================================================================

// collectHeadlinesCARB はカリフォルニア大気資源局からニュースを取得する
//
// CARBはカリフォルニア州のキャップ・アンド・トレード制度を管理し、
// 排出規制と気候政策に関するニュースを公開している。
func collectHeadlinesCARB(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://ww2.arb.ca.gov/news"

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

	// CARBニュース一覧
	doc.Find("article, .news-item, .views-row, div[class*='node--type-news']").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// タイトルを取得
		titleLink := article.Find("h2 a, h3 a, .field--name-title a, a[href*='/news/']").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find("h2, h3, .title").First().Text())
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

		// 日付を抽出
		dateStr := ""
		dateElem := article.Find("time, .date, .field--name-created, span[class*='date']")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"January 2, 2006",
					"Jan 2, 2006",
					"01/02/2006",
					"2006-01-02",
				} {
					if t, err := time.Parse(format, dateText); err == nil {
						dateStr = t.Format(time.RFC3339)
						break
					}
				}
			}
		}

		// 個別記事ページから本文全体を取得
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// ナビ・ヘッダー・フッター・サイドバー要素を除去
					articleDoc.Find("header, footer, nav, aside, .sidebar, script, style, .breadcrumb").Remove()

					// メイン要素からコンテンツを抽出
					mainContent := articleDoc.Find("main#main-content, main, article, .content")
					if mainContent.Length() > 0 {
						// 全段落テキストを取得
						var paragraphs []string
						mainContent.Find("p").Each(func(_ int, p *goquery.Selection) {
							text := strings.TrimSpace(p.Text())
							if len(text) > 20 {
								paragraphs = append(paragraphs, text)
							}
						})
						excerpt = strings.Join(paragraphs, "\n\n")
					}
				}
			}
		}

		// 記事取得失敗時は一覧ページの抜粋をフォールバック
		if excerpt == "" {
			excerptElem := article.Find("p, .field--name-body, .summary, .teaser").First()
			if excerptElem.Length() > 0 {
				excerpt = strings.TrimSpace(excerptElem.Text())
			}
		}

		out = append(out, Headline{
			Source:      "CARB",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] CARB: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// RGGIソース
// =============================================================================

// collectHeadlinesRGGI は地域温室効果ガスイニシアティブからニュースを取得する
//
// RGGIは米国東部の州による協力的取り組みで、電力部門の
// CO2排出量の上限設定と削減を目的としている。
func collectHeadlinesRGGI(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://www.rggi.org/news-releases/rggi-releases"

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

	// RGGIは各ニュースアイテムをテーブル行で表示
	doc.Find("table.table tbody tr").Each(func(_ int, row *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// 本文セルからリンクとタイトルを取得
		bodyCell := row.Find("td.views-field-body")
		link := bodyCell.Find("a").First()

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// 本文セルから説明を抽出（リンク後のテキスト）
		listingDescription := ""
		bodyCellText := strings.TrimSpace(bodyCell.Text())
		if bodyCellText != "" && bodyCellText != title {
			// 本文セルテキストからタイトルを除去して説明を取得
			listingDescription = strings.TrimSpace(strings.TrimPrefix(bodyCellText, title))
		}

		// time要素から日付を抽出
		dateStr := ""
		foundDate := false
		timeElem := row.Find("time")
		if timeElem.Length() > 0 {
			if datetime, exists := timeElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			}
		}

		// タイプセルからアイテム種別を抽出
		typeCell := row.Find("td.views-field-field-item-type")
		itemType := strings.TrimSpace(typeCell.Text())

		// 記事ページまたはPDFからコンテンツを取得
		excerpt := ""
		isPDF := strings.HasSuffix(strings.ToLower(articleURL), ".pdf")

		if isPDF {
			// PDFからテキストを抽出
			pdfText, err := extractTextFromPDF(articleURL, client, cfg.UserAgent)
			if err == nil && len(pdfText) > 50 {
				// PDFテキストを適切な長さに制限
				if len(pdfText) > 2000 {
					pdfText = pdfText[:2000] + "..."
				}
				excerpt = pdfText
			}
		} else {
			articleReq, err := http.NewRequest("GET", articleURL, nil)
			if err == nil {
				articleReq.Header.Set("User-Agent", cfg.UserAgent)
				articleResp, err := client.Do(articleReq)
				if err == nil && articleResp.StatusCode == http.StatusOK {
					articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
					articleResp.Body.Close()
					if err == nil {
						// 不要な要素を除去
						articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar").Remove()

						// 日付が未取得の場合、抽出を試行
						if !foundDate {
							articleDoc.Find("time").Each(func(_ int, elem *goquery.Selection) {
								if foundDate {
									return
								}
								if datetime, exists := elem.Attr("datetime"); exists {
									dateStr = datetime
									foundDate = true
								}
							})
						}

						// メインコンテンツエリアからコンテンツを抽出
						contentSelectors := []string{
							".field--name-body",
							".content",
							"article",
							"main",
						}
						for _, sel := range contentSelectors {
							contentElem := articleDoc.Find(sel)
							if contentElem.Length() > 0 {
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
					}
				}
			}
		}

		// フォールバック: 一覧の説明またはタイプをExcerptとして使用
		if excerpt == "" {
			if listingDescription != "" {
				// 一覧ページの説明を使用
				excerpt = listingDescription
				if strings.HasSuffix(strings.ToLower(articleURL), ".pdf") {
					excerpt = "[PDF] " + excerpt
				}
			} else if strings.HasSuffix(strings.ToLower(articleURL), ".pdf") {
				excerpt = "PDF Document - Type: " + itemType
			} else {
				excerpt = "Type: " + itemType
			}
		}

		// 日付が見つからない場合は現在時刻をフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "RGGI",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] RGGI: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// オーストラリア CERソース
// =============================================================================

// collectHeadlinesAustraliaCER はオーストラリア クリーンエネルギー規制当局からニュースを取得する
//
// CERは排出削減基金を含む気候変動法を管理する
// オーストラリア政府機関である。
func collectHeadlinesAustraliaCER(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	newsURL := "https://cer.gov.au/news-and-media/news"

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

	// オーストラリアCERはニュースアイテムにcer-cardクラスを使用
	doc.Find("div.cer-card.news, article.cer-card").Each(func(_ int, article *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// cer-card__headingからタイトルを取得
		headingElem := article.Find(".cer-card__heading a, h2 a, h3 a").First()
		title := strings.TrimSpace(headingElem.Text())
		if title == "" {
			title = strings.TrimSpace(article.Find(".cer-card__heading, h2, h3").First().Text())
		}
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := headingElem.Attr("href")
		if !exists || href == "" {
			// 任意のリンクを探す
			anyLink := article.Find("a[href]").First()
			href, exists = anyLink.Attr("href")
		}
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(newsURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}
		seen[articleURL] = true

		// cer-card__changedから日付を抽出
		dateStr := ""
		foundDate := false
		dateElem := article.Find(".cer-card__changed, time, .date")
		if dateElem.Length() > 0 {
			if datetime, exists := dateElem.Attr("datetime"); exists {
				dateStr = datetime
				foundDate = true
			} else {
				dateText := strings.TrimSpace(dateElem.Text())
				for _, format := range []string{
					"2 January 2006",
					"02/01/2006",
					"2 Jan 2006",
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
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// 不要な要素を除去
					articleDoc.Find("header, footer, nav, script, style, noscript, .sidebar, .related").Remove()

					// 日付が未取得の場合、記事ページから抽出を試行
					if !foundDate {
						articleDoc.Find("time, .date").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							if datetime, exists := elem.Attr("datetime"); exists {
								dateStr = datetime
								foundDate = true
							}
						})
					}

					// 記事本文からコンテンツを抽出（段落とリストアイテム）
					contentSelectors := []string{
						".field--name-body",
						".content",
						"article .body",
						"main article",
						".page-content",
					}
					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
							var contentParts []string
							// 段落とリストアイテムを抽出
							contentElem.Find("p, li").Each(func(_ int, elem *goquery.Selection) {
								text := strings.TrimSpace(elem.Text())
								if len(text) > 20 {
									// リストアイテムにはバレットを付加
									if goquery.NodeName(elem) == "li" {
										text = "• " + text
									}
									contentParts = append(contentParts, text)
								}
							})
							if len(contentParts) > 0 {
								excerpt = strings.Join(contentParts, "\n\n")
								break
							}
						}
					}

					// フォールバック: mainから全段落とリストアイテムを取得
					if excerpt == "" {
						var contentParts []string
						articleDoc.Find("main p, main li, article p, article li").Each(func(_ int, elem *goquery.Selection) {
							text := strings.TrimSpace(elem.Text())
							if len(text) > 30 {
								if goquery.NodeName(elem) == "li" {
									text = "• " + text
								}
								contentParts = append(contentParts, text)
							}
						})
						if len(contentParts) > 0 {
							excerpt = strings.Join(contentParts, "\n\n")
						}
					}
				}
			}
		}

		// コンテンツが取得できない場合は一覧ページの抜粋をフォールバック
		if excerpt == "" {
			bodyElem := article.Find(".cer-card__body, p, .summary")
			if bodyElem.Length() > 0 {
				excerpt = strings.TrimSpace(bodyElem.First().Text())
			}
		}

		// 日付が見つからない場合は現在時刻をフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "Australia CER",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Australia CER: collected %d headlines\n", len(out))
	}

	return out, nil
}

// =============================================================================
// UK ETSソース
// =============================================================================

// collectHeadlinesUKETSHTML は英国政府ETS出版物からニュースを取得する
//
// UK排出権取引制度はUK ETS Authority（英国、スコットランド、ウェールズ
// 政府および北アイルランド行政府の合同機関）が管理している。
// gov.ukの検索結果からUK ETS関連の出版物をスクレイピングする。
func collectHeadlinesUKETSHTML(limit int, cfg headlineSourceConfig) ([]Headline, error) {
	// gov.ukでUK ETSの出版物とニュースを検索
	searchURL := "https://www.gov.uk/search/all?keywords=%22UK+Emissions+Trading+Scheme%22&order=updated-newest"

	client := cfg.Client
	req, err := http.NewRequest("GET", searchURL, nil)
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

	// gov.ukの検索結果は各結果にgem-c-document-list__itemを使用
	doc.Find("li.gem-c-document-list__item, .gem-c-document-list__item, div.finder-results li").Each(func(_ int, item *goquery.Selection) {
		if len(out) >= limit {
			return
		}

		// タイトルリンクを取得
		link := item.Find("a.gem-c-document-list__item-title, a[data-track-category='navFinderLinkClicked']").First()
		if link.Length() == 0 {
			link = item.Find("a").First()
		}

		title := strings.TrimSpace(link.Text())
		if title == "" || len(title) < 10 {
			return
		}

		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}

		articleURL := resolveURL(searchURL, href)
		if articleURL == "" || seen[articleURL] {
			return
		}

		// フィルタ: UK ETS関連コンテンツのみ含める
		titleLower := strings.ToLower(title)
		if !strings.Contains(titleLower, "ets") &&
			!strings.Contains(titleLower, "emissions trading") &&
			!strings.Contains(titleLower, "carbon") {
			return
		}

		seen[articleURL] = true

		// メタデータから日付を抽出
		dateStr := ""
		foundDate := false
		metaElem := item.Find(".gem-c-document-list__attribute, .document-list-item-metadata")
		if metaElem.Length() > 0 {
			metaText := strings.TrimSpace(metaElem.Text())
			// "Updated: DD Month YYYY"等の形式を探す
			if strings.Contains(metaText, "Updated:") {
				dateText := strings.TrimPrefix(metaText, "Updated:")
				dateText = strings.TrimSpace(dateText)
				for _, format := range []string{
					"2 January 2006",
					"02 January 2006",
					"January 2, 2006",
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
		excerpt := ""
		articleReq, err := http.NewRequest("GET", articleURL, nil)
		if err == nil {
			articleReq.Header.Set("User-Agent", cfg.UserAgent)
			articleResp, err := client.Do(articleReq)
			if err == nil && articleResp.StatusCode == http.StatusOK {
				articleDoc, err := goquery.NewDocumentFromReader(articleResp.Body)
				articleResp.Body.Close()
				if err == nil {
					// 不要な要素を除去
					articleDoc.Find("header, footer, nav, script, style, noscript, .gem-c-contextual-sidebar").Remove()

					// 日付が未取得の場合、記事ページから抽出を試行
					if !foundDate {
						articleDoc.Find("time, .gem-c-metadata__definition").Each(func(_ int, elem *goquery.Selection) {
							if foundDate {
								return
							}
							if datetime, exists := elem.Attr("datetime"); exists {
								dateStr = datetime
								foundDate = true
							} else {
								text := strings.TrimSpace(elem.Text())
								for _, format := range []string{
									"2 January 2006",
									"02 January 2006",
									"2006-01-02",
								} {
									if t, err := time.Parse(format, text); err == nil {
										dateStr = t.Format(time.RFC3339)
										foundDate = true
										break
									}
								}
							}
						})
					}

					// gov.ukのページ構造からコンテンツを抽出
					contentSelectors := []string{
						".gem-c-govspeak",
						".govuk-govspeak",
						".publication-content",
						"main .content",
						"article",
					}
					for _, sel := range contentSelectors {
						contentElem := articleDoc.Find(sel)
						if contentElem.Length() > 0 {
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

					// フォールバック: metaディスクリプションを試行
					if excerpt == "" {
						metaDesc := articleDoc.Find("meta[name='description']")
						if metaDesc.Length() > 0 {
							excerpt, _ = metaDesc.Attr("content")
							excerpt = strings.TrimSpace(excerpt)
						}
					}
				}
			}
		}

		// コンテンツが取得できない場合は一覧ページの説明をフォールバック
		if excerpt == "" {
			descElem := item.Find(".gem-c-document-list__item-description, p")
			if descElem.Length() > 0 {
				excerpt = strings.TrimSpace(descElem.First().Text())
			}
		}

		// 日付が見つからない場合は現在時刻をフォールバック
		if !foundDate {
			dateStr = time.Now().Format(time.RFC3339)
		}

		out = append(out, Headline{
			Source:      "UK ETS",
			Title:       title,
			URL:         articleURL,
			PublishedAt: dateStr,
			Excerpt:     excerpt,

		})
	})

	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] UK ETS: collected %d headlines\n", len(out))
	}

	return out, nil
}
