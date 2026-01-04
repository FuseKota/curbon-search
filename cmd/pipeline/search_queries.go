// =============================================================================
// search_queries.go - 検索クエリ生成モジュール
// =============================================================================
//
// このファイルは有料記事の見出しと要約から、関連する無料記事を検索するための
// クエリを生成します。OpenAI Web検索で使用されます。
//
// =============================================================================
// 【7段階のクエリ生成戦略】
// =============================================================================
//
// ① 完全一致検索:     "見出し全文"（引用符付き）
// ② 要約からの抽出:   要約の最初の1文、固有名詞、数値情報
// ③ 市場キーワード:   VCM、ETS、CORSIA等の略語を正式名称に展開
// ④ 地域別サイト検索: site:europa.eu、site:go.jp等
// ⑤ PDF優先検索:      filetype:pdf（公式文書を優先）
// ⑥ 公式発表検索:     official announcement
// ⑦ 国際機関サイト:   site:unfccc.int OR site:iea.org等
//
// =============================================================================
// 【生成されるクエリの例】
// =============================================================================
//
// 入力:
//   title:   "EU carbon price hits €100 for first time"
//   excerpt: "The European Union Allowance price reached €100.34 on February 21..."
//
// 出力:
//   1. "EU carbon price hits €100 for first time"     ← 完全一致
//   2. "The European Union Allowance price reached"   ← 要約から
//   3. EU carbon price hits €100 European Union       ← 固有名詞
//   4. EU carbon price hits €100 emissions trading    ← キーワード展開
//   5. EU carbon price hits €100 site:europa.eu       ← 地域サイト
//   6. EU carbon price hits €100 filetype:pdf         ← PDF優先
//   7. EU carbon price hits €100 site:unfccc.int...   ← 国際機関
//
// =============================================================================
// 【初心者向けポイント】
// =============================================================================
//
// - 検索クエリは多様性が重要（同じ情報を異なる角度から検索）
// - site: 演算子は特定ドメインに限定して検索
// - filetype: 演算子は特定のファイル形式に限定
// - 引用符 "" は完全一致検索を意味
// - 正規表現を使って固有名詞や数値を抽出
//
// =============================================================================
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// メイン関数: 検索クエリ生成
// =============================================================================

// buildSearchQueries は有料記事の見出しと要約から検索クエリを生成する
//
// 【処理の流れ】
//  1. 完全一致クエリ（引用符付き）を追加
//  2. 要約がある場合、追加情報を抽出してクエリに追加
//  3. カーボン市場系キーワードを検出して展開
//  4. 地域情報を検出してサイト演算子を追加
//  5. PDF優先、公式発表、国際機関サイトのクエリを追加
//  6. 重複を除去して返す
//
// 引数:
//
//	title:   有料記事のタイトル
//	excerpt: 有料記事の要約（無料で見える部分）
//
// 戻り値:
//
//	検索クエリのスライス（重複なし）
func buildSearchQueries(title, excerpt string) []string {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	queries := []string{}
	lower := strings.ToLower(title)
	excerptLower := strings.ToLower(excerpt)

	// =========================================================================
	// ① 完全一致検索（引用符付き）
	// =========================================================================
	// 見出し全体をそのまま検索。最も精度が高いが、完全一致する記事は少ない
	queries = append(queries, fmt.Sprintf(`"%s"`, title))

	// =========================================================================
	// ② 要約からの情報抽出
	// =========================================================================
	if excerpt != "" {
		// 最初の1-2文を抽出（最大150文字）
		// 要約の冒頭は最も重要な情報を含むことが多い
		firstSentence := extractFirstSentence(excerpt, 150)
		if firstSentence != "" && firstSentence != title {
			queries = append(queries, fmt.Sprintf(`"%s"`, firstSentence))
		}

		// 固有名詞の抽出（大文字始まりの2語以上の連続）
		// 例: "European Commission", "Climate Action Network"
		properNouns := extractProperNouns(excerpt)
		for _, noun := range properNouns {
			// タイトルに含まれていない固有名詞を追加
			if !strings.Contains(lower, strings.ToLower(noun)) {
				queries = append(queries, title+" "+noun)
			}
		}

		// 数値情報の抽出
		// 例: "$50 million", "30%", "2025"
		numbers := extractNumbersWithContext(excerpt)
		for _, num := range numbers {
			if !strings.Contains(title, num) {
				queries = append(queries, title+" "+num)
			}
		}
	}

	// =========================================================================
	// ③ カーボン市場キーワードの展開
	// =========================================================================
	// 略語を正式名称に展開することで、検索範囲を広げる
	combinedText := lower + " " + excerptLower

	// VCM = Voluntary Carbon Market（自主的カーボン市場）
	if strings.Contains(combinedText, "vcm") {
		queries = append(queries, title+" voluntary carbon market")
	}

	// ETS = Emissions Trading System, EUA = EU Allowance, UKA = UK Allowance
	if strings.Contains(combinedText, "ets") || strings.Contains(combinedText, "eua") || strings.Contains(combinedText, "uka") {
		queries = append(queries, title+" emissions trading system")
	}

	// CORSIA = Carbon Offsetting and Reduction Scheme for International Aviation
	if strings.Contains(combinedText, "corsia") {
		queries = append(queries, title+" CORSIA ICAO")
	}

	// CCER = China Certified Emission Reduction（中国認証排出削減量）
	if strings.Contains(combinedText, "ccer") {
		queries = append(queries, title+" CCER China")
	}

	// バイオ炭プロジェクト
	if strings.Contains(combinedText, "biochar") {
		queries = append(queries, title+" biochar project")
	}

	// =========================================================================
	// ④ 地域別サイト検索
	// =========================================================================
	// 政府・規制当局の公式サイトを優先的に検索

	// 韓国
	if strings.Contains(combinedText, "south korea") || strings.Contains(combinedText, "korea") {
		queries = append(queries, title+" site:go.kr")
	}
	// EU
	if strings.Contains(combinedText, "eu") || strings.Contains(combinedText, "europe") {
		queries = append(queries, title+" site:europa.eu")
	}
	// 日本
	if strings.Contains(combinedText, "japan") {
		queries = append(queries, title+" site:go.jp")
	}
	// イギリス
	if strings.Contains(combinedText, "uk") || strings.Contains(combinedText, "united kingdom") {
		queries = append(queries, title+" site:gov.uk")
	}
	// 中国
	if strings.Contains(combinedText, "china") {
		queries = append(queries, title+" site:gov.cn")
	}
	// オーストラリア
	if strings.Contains(combinedText, "australia") {
		queries = append(queries, title+" site:gov.au")
	}

	// =========================================================================
	// ⑤ PDF優先検索
	// =========================================================================
	// 公式文書、規制文書、レポートはPDF形式が多い
	queries = append(queries, title+" filetype:pdf")

	// =========================================================================
	// ⑥ 公式発表検索
	// =========================================================================
	// 国・地域が含まれている場合、公式発表を検索
	countries := []string{
		"south korea", "korea", "china", "japan",
		"eu", "europe", "uk", "united states", "us",
		"australia", "new zealand", "taiwan",
	}

	for _, c := range countries {
		if strings.Contains(combinedText, c) {
			queries = append(queries, title+" official announcement")
			break // 1回だけ追加
		}
	}

	// =========================================================================
	// ⑦ 国際機関サイト検索
	// =========================================================================
	// カーボン・気候関連のキーワードがある場合、主要国際機関を検索
	if strings.Contains(combinedText, "carbon") || strings.Contains(combinedText, "climate") || strings.Contains(combinedText, "emissions") {
		queries = append(queries, title+" site:unfccc.int OR site:icvcm.org OR site:iea.org")
	}

	// =========================================================================
	// 重複除去
	// =========================================================================
	seen := map[string]bool{}
	out := []string{}
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" || seen[q] {
			continue
		}
		seen[q] = true
		out = append(out, q)
	}

	return out
}

// =============================================================================
// ヘルパー関数: テキスト抽出
// =============================================================================

// extractFirstSentence は最初の文を抽出する（最大maxLen文字）
//
// 【文の区切り】
//
//	". " "! " "? " の後にスペースがある位置を文の終わりとみなす
//
// 使用例:
//
//	extractFirstSentence("Hello world. This is test.", 20)
//	// => "Hello world."
func extractFirstSentence(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// 文末記号（後にスペースが続くもの）を探す
	sentenceEnders := []string{". ", "! ", "? "}
	minIdx := len(text)

	for _, ender := range sentenceEnders {
		idx := strings.Index(text, ender)
		if idx > 0 && idx < minIdx {
			minIdx = idx + 1 // 句読点を含める
		}
	}

	if minIdx < len(text) {
		text = text[:minIdx]
	}

	// 最大文字数で制限
	if len(text) > maxLen {
		text = text[:maxLen]
		// 単語の途中で切れないよう、最後のスペースで切る
		if lastSpace := strings.LastIndex(text, " "); lastSpace > 0 {
			text = text[:lastSpace]
		}
	}

	return strings.TrimSpace(text)
}

// extractProperNouns は固有名詞らしき語句を抽出する（最大3個）
//
// 【判定基準】
//
//	大文字で始まる単語が2語以上連続している場合、固有名詞とみなす
//	例: "European Commission", "New York Times", "Climate Action Network"
//
// 【除外するパターン】
//
//	"The ..." "A ..." で始まるものは一般的な表現なので除外
func extractProperNouns(text string) []string {
	// 大文字始まりの単語が2語以上連続するパターン
	// \b は単語境界、[A-Z][a-z]+ は大文字始まりの単語
	re := regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)+\b`)
	matches := re.FindAllString(text, -1)

	seen := map[string]bool{}
	result := []string{}

	for _, match := range matches {
		match = strings.TrimSpace(match)
		if match == "" || seen[match] {
			continue
		}

		// "The ..." や "A ..." で始まるものは除外
		lower := strings.ToLower(match)
		if strings.HasPrefix(lower, "the ") || strings.HasPrefix(lower, "a ") {
			continue
		}

		seen[match] = true
		result = append(result, match)

		// 最大3個まで（クエリが多すぎると効率が下がる）
		if len(result) >= 3 {
			break
		}
	}

	return result
}

// extractNumbersWithContext は文脈付きの数値を抽出する（最大2個）
//
// 【抽出パターン】
//  1. 通貨付き: "$50 million", "€1.2 billion"
//  2. パーセント: "30%", "1.5%"
//  3. 単位付き: "1.5 billion tons", "50 million credits"
//  4. 年号: "2024", "2025"
//
// 【なぜ数値が重要か】
//
//	ニュース記事では具体的な数値が差別化要因になることが多い
//	"$50 million carbon credit deal" のような数値は記事を特定しやすい
func extractNumbersWithContext(text string) []string {
	patterns := []*regexp.Regexp{
		// 通貨: $50 million, €1.2 billion
		regexp.MustCompile(`[$€£¥]\s*[\d,]+(?:\.\d+)?\s*(?:million|billion|trillion|thousand|mn|bn)`),
		// パーセント: 30%, 1.5%
		regexp.MustCompile(`\d+(?:\.\d+)?%`),
		// 単位付き数値: 1.5 billion tons, 50 million credits
		regexp.MustCompile(`\d+(?:\.\d+)?\s*(?:million|billion|trillion|thousand)\s+(?:tons?|credits?|tonnes?|dollars?|euros?)`),
		// 年号: 2020〜2029
		regexp.MustCompile(`\b20[12]\d\b`),
	}

	seen := map[string]bool{}
	result := []string{}

	for _, re := range patterns {
		matches := re.FindAllString(text, -1)
		for _, match := range matches {
			match = strings.TrimSpace(match)
			if match == "" || seen[match] {
				continue
			}
			seen[match] = true
			result = append(result, match)

			// 最大2個まで
			if len(result) >= 2 {
				return result
			}
		}
	}

	return result
}
