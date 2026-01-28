// =============================================================================
// sources_paid.go - 有料ソース（見出しのみ取得）
// =============================================================================
//
// このファイルは有料ニュースソースから見出しのみを収集する関数を定義します。
// 有料記事の本文は取得しません（プロジェクトの基本原則）。
//
// 【含まれるソース】
//   1. Carbon Pulse - カーボン市場専門ニュース（業界最大手）
//   2. QCI          - Quantum Commodity Intelligence
//
// =============================================================================
package pipeline

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// グローバル正規表現パターン（有料ソース用）
var (
	// reCarbonPulseID は Carbon Pulse の記事ID形式をマッチ（例: /12345/）
	reCarbonPulseID = regexp.MustCompile(`^/\d+/$`)

	// reQCIArticle は QCI の記事URLパターンをマッチ
	reQCIArticle = regexp.MustCompile(`/carbon/article/`)
)

// collectHeadlinesCarbonPulse は Carbon Pulse（有料ソース）から見出しと要約を収集
//
// Carbon Pulse は有料サブスクリプションサービスですが、以下のページは無料でアクセス可能：
//   - トップページ: 記事の要約（excerpt）付き
//   - デイリータイムライン: 見出しのみ
//   - ニュースレターカテゴリ: 見出しのみ
//
// 引数:
//
//	limit: 収集する最大記事数
//	cfg: スクレイピング設定（タイムアウト、User-Agent等）
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
func collectHeadlinesCarbonPulse(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	// 無料でアクセス可能な3つのページを巡回
	// トップページのみ記事の要約（excerpt）が取得可能
	pages := []string{
		"https://carbon-pulse.com/",       // トップページ（excerpt付き）
		cfg.CarbonPulseTimelineURL,        // デイリータイムライン
		cfg.CarbonPulseNewsletters,        // ニュースレターカテゴリ
	}
	out := []Headline{}       // 収集結果を格納するスライス
	seen := map[string]bool{} // URL重複チェック用マップ

	// デバッグモード時: スクレイピング対象ページを出力
	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Scraping Carbon Pulse from:\n")
		for _, p := range pages {
			fmt.Fprintf(os.Stderr, "  - %s\n", p)
		}
	}

	for pageIdx, pageURL := range pages {
		doc, err := fetchDoc(pageURL, cfg)
		if err != nil {
			// Do not fail hard; just continue.
			if os.Getenv("DEBUG_SCRAPING") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Failed to fetch %s: %v\n", pageURL, err)
			}
			continue
		}

		// Debug: Save HTML to inspect structure
		if os.Getenv("DEBUG_HTML") != "" {
			html, _ := doc.Html()
			if len(html) > 0 {
				length := len(html)
				if length > 2000 {
					length = 2000
				}
				fmt.Fprintf(os.Stderr, "[DEBUG] First 2000 chars of HTML:\n%s\n\n", html[:length])
			}
		}

		// Special handling for top page: extract from main content area (.post divs)
		// Always extract excerpts from the top page
		if pageIdx == 0 {
			doc.Find("div.post").Each(func(_ int, postDiv *goquery.Selection) {
				if limit > 0 && len(out) >= limit {
					return
				}

				// Find the title link in this post
				titleLink := postDiv.Find("h2.posttitle a").First()
				if titleLink.Length() == 0 {
					return
				}

				href, ok := titleLink.Attr("href")
				if !ok {
					return
				}

				txt := strings.TrimSpace(titleLink.Text())
				if txt == "" || len(txt) < 10 {
					return
				}

				abs := resolveURL(pageURL, href)
				if abs == "" {
					return
				}
				u, err := url.Parse(abs)
				if err != nil || u.Host == "" {
					return
				}

				if !strings.HasSuffix(u.Host, "carbon-pulse.com") {
					return
				}

				if seen[abs] {
					return
				}
				seen[abs] = true

				// Extract excerpt from this post div
				var excerpt string
				fullText := postDiv.Text()

				if os.Getenv("DEBUG_HTML") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Post full text (first 2000 chars):\n%s\n\n", fullText[:min2(2000, len(fullText))])
				}

				// Remove "Read More" and everything after
				readMoreIdx := strings.Index(fullText, "Read More")
				if readMoreIdx > 0 {
					fullText = fullText[:readMoreIdx]
				}

				// Split into lines and find the excerpt
				lines := strings.Split(fullText, "\n")
				var excerptBuilder strings.Builder
				maxChars := 500

				for i, line := range lines {
					line = strings.TrimSpace(line)

					// Skip empty lines, metadata, tags, and navigation
					if line == "" || len(line) < 30 {
						continue
					}

					// Skip the title itself (exact match)
					if line == txt {
						continue
					}

					// Skip metadata, tags, CSS, and navigation
					if strings.Contains(line, "Published") ||
						strings.Contains(line, "Last updated") ||
						strings.Contains(line, "Carbon Pulse Premium") ||
						strings.Contains(line, "Nature & Biodiversity") ||
						strings.Contains(line, "Net Zero Pulse") ||
						strings.HasPrefix(line, "Top") ||
						strings.HasPrefix(line, "#") ||
						strings.Contains(line, "{") ||
						strings.Contains(line, "}") ||
						strings.Contains(line, "padding:") ||
						strings.Contains(line, "border:") ||
						strings.Contains(line, "/") && strings.Contains(line, "2025") || // Skip date lines
						strings.Count(line, "/") >= 2 || // Skip author/category lines like "Name / Category / Subcategory"
						strings.Contains(line, "Asia Pacific") ||
						strings.Contains(line, "LATAM") ||
						strings.Contains(line, "EMEA") ||
						strings.Contains(line, "VCM Developments") {
						continue
					}

					// This should be the excerpt
					if len(line) > 30 && !strings.HasPrefix(line, "http") {
						excerptBuilder.WriteString(line)

						// Also check next line if it's a continuation
						if i+1 < len(lines) {
							nextLine := strings.TrimSpace(lines[i+1])
							if len(nextLine) > 20 && !strings.Contains(nextLine, "Read More") && !strings.Contains(nextLine, "Published") {
								excerptBuilder.WriteString(" ")
								excerptBuilder.WriteString(nextLine)
							}
						}
						break
					}
				}

				excerpt = strings.TrimSpace(excerptBuilder.String())
				if len(excerpt) > maxChars {
					excerpt = excerpt[:maxChars] + "..."
				}

				if os.Getenv("DEBUG_SCRAPING") != "" && excerpt != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Extracted excerpt from post (%d chars): %s...\n", len(excerpt), excerpt[:min2(100, len(excerpt))])
				}

				out = append(out, Headline{Source: "Carbon Pulse", Title: txt, URL: abs, Excerpt: excerpt, IsHeadline: true})
			})

			// Skip the regular link extraction for top page since we already processed it
			if limit > 0 && len(out) >= limit {
				break
			}
			continue
		}

		// Regular link extraction for other pages
		doc.Find("a").Each(func(_ int, s *goquery.Selection) {
			if limit > 0 && len(out) >= limit {
				return
			}

			href, ok := s.Attr("href")
			if !ok {
				return
			}
			txt := strings.TrimSpace(s.Text())
			if txt == "" {
				return
			}

			// 無意味なリンクテキストを除外
			txtLower := strings.ToLower(txt)
			if txtLower == "read more" || txtLower == "continue reading" || txtLower == "click here" || len(txt) < 10 {
				return
			}

			abs := resolveURL(pageURL, href)
			if abs == "" {
				return
			}
			u, err := url.Parse(abs)
			if err != nil || u.Host == "" {
				return
			}

			if !strings.HasSuffix(u.Host, "carbon-pulse.com") {
				return
			}

			// Keep only numeric article URLs like /470597/ (avoid nav like /register/).
			if !reCarbonPulseID.MatchString(u.Path) {
				return
			}

			if seen[abs] {
				return
			}
			seen[abs] = true

			// Try to extract excerpt from nearby text (parent, siblings, or following paragraphs)
			// Note: Timeline/newsletter pages typically don't have excerpts in the HTML structure
			excerpt := extractExcerptFromContext(s)

			out = append(out, Headline{Source: "Carbon Pulse", Title: txt, URL: abs, Excerpt: excerpt, IsHeadline: true})
		})

		if limit > 0 && len(out) >= limit {
			break
		}
	}

	// Note: If consistently empty, site may be blocking or layout changed
	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Collected %d Carbon Pulse headlines\n", len(out))
		if len(out) > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Latest: %s\n", out[0].Title)
			fmt.Fprintf(os.Stderr, "[DEBUG] URL: %s\n", out[0].URL)
		}
	}

	return out, nil
}

// collectHeadlinesQCI は QCI（Quantum Commodity Intelligence）から見出しを収集
//
// QCI は有料のカーボン市場情報サービス。トップページから記事リンクを抽出。
//
// 引数:
//
//	limit: 収集する最大記事数
//	cfg: タイムアウトとUser-Agent設定
//
// 戻り値:
//
//	収集した見出しのスライス、エラー
func collectHeadlinesQCI(limit int, cfg HeadlineSourceConfig) ([]Headline, error) {
	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Scraping QCI from: %s\n", cfg.QCIHomeURL)
	}

	doc, err := fetchDoc(cfg.QCIHomeURL, cfg)
	if err != nil {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to fetch QCI: %v\n", err)
		}
		return nil, err
	}

	out := []Headline{}
	seen := map[string]bool{}

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if limit > 0 && len(out) >= limit {
			return
		}

		href, ok := s.Attr("href")
		if !ok {
			return
		}
		txt := strings.TrimSpace(s.Text())
		if txt == "" {
			return
		}

		// 無意味なリンクテキストを除外
		txtLower := strings.ToLower(txt)
		if txtLower == "read more" || txtLower == "continue reading" || txtLower == "click here" || len(txt) < 10 {
			return
		}

		abs := resolveURL(cfg.QCIHomeURL, href)
		if abs == "" {
			return
		}
		u, err := url.Parse(abs)
		if err != nil || u.Host == "" {
			return
		}
		if !strings.HasSuffix(u.Host, "qcintel.com") {
			return
		}
		if !reQCIArticle.MatchString(u.Path) {
			return
		}

		if seen[abs] {
			return
		}
		seen[abs] = true

		out = append(out, Headline{Source: "QCI", Title: txt, URL: abs, IsHeadline: true})
	})

	// Note: If consistently empty, site may be blocking or layout changed
	if os.Getenv("DEBUG_SCRAPING") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Collected %d QCI headlines\n", len(out))
		if len(out) > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Latest: %s\n", out[0].Title)
			fmt.Fprintf(os.Stderr, "[DEBUG] URL: %s\n", out[0].URL)
		}
	}

	return out, nil
}
