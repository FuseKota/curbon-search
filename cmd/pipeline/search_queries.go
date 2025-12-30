package main

import (
	"fmt"
	"regexp"
	"strings"
)

// buildSearchQueries generates search queries from a paid headline and excerpt.
// 目的：
// - 見出しそのもの
// - excerptから抽出した具体的な情報（企業名、数値、組織名など）
// - 国・制度・キーワードを含めた再検索
// - site: / filetype: 演算子で一次情報を優先
func buildSearchQueries(title, excerpt string) []string {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	queries := []string{}
	lower := strings.ToLower(title)
	excerptLower := strings.ToLower(excerpt)

	// ① 完全一致（引用符付き）
	queries = append(queries, fmt.Sprintf(`"%s"`, title))

	// ② excerptを活用した検索クエリ（excerptがある場合）
	if excerpt != "" {
		// excerptの最初の1-2文を抽出（最大150文字）
		firstSentence := extractFirstSentence(excerpt, 150)
		if firstSentence != "" && firstSentence != title {
			queries = append(queries, fmt.Sprintf(`"%s"`, firstSentence))
		}

		// excerptから組織名・企業名らしき固有名詞を抽出（大文字始まりの2語以上）
		properNouns := extractProperNouns(excerpt)
		for _, noun := range properNouns {
			if !strings.Contains(lower, strings.ToLower(noun)) {
				// タイトルに含まれていない固有名詞をタイトルと組み合わせて検索
				queries = append(queries, title+" "+noun)
			}
		}

		// excerptから数値情報を抽出（例: "1.5 billion", "$50 million", "30%"）
		numbers := extractNumbersWithContext(excerpt)
		for _, num := range numbers {
			if !strings.Contains(title, num) {
				queries = append(queries, title+" "+num)
			}
		}
	}

	// ③ カーボン市場系キーワード補助（タイトルまたはexcerptから）
	combinedText := lower + " " + excerptLower
	if strings.Contains(combinedText, "vcm") {
		queries = append(queries, title+" voluntary carbon market")
	}
	if strings.Contains(combinedText, "ets") || strings.Contains(combinedText, "eua") || strings.Contains(combinedText, "uka") {
		queries = append(queries, title+" emissions trading system")
	}
	if strings.Contains(combinedText, "corsia") {
		queries = append(queries, title+" CORSIA ICAO")
	}
	if strings.Contains(combinedText, "ccer") {
		queries = append(queries, title+" CCER China")
	}
	if strings.Contains(combinedText, "biochar") {
		queries = append(queries, title+" biochar project")
	}

	// ④ 地域別 site: 演算子（政府・規制当局を優先、タイトルまたはexcerptから検出）
	if strings.Contains(combinedText, "south korea") || strings.Contains(combinedText, "korea") {
		queries = append(queries, title+" site:go.kr")
	}
	if strings.Contains(combinedText, "eu") || strings.Contains(combinedText, "europe") {
		queries = append(queries, title+" site:europa.eu")
	}
	if strings.Contains(combinedText, "japan") {
		queries = append(queries, title+" site:go.jp")
	}
	if strings.Contains(combinedText, "uk") || strings.Contains(combinedText, "united kingdom") {
		queries = append(queries, title+" site:gov.uk")
	}
	if strings.Contains(combinedText, "china") {
		queries = append(queries, title+" site:gov.cn")
	}
	if strings.Contains(combinedText, "australia") {
		queries = append(queries, title+" site:gov.au")
	}

	// ⑤ PDF 優先（一次資料・規制文書）
	queries = append(queries, title+" filetype:pdf")

	// ⑥ 国・地域が含まれていそうなら official を足す
	countries := []string{
		"south korea", "korea", "china", "japan",
		"eu", "europe", "uk", "united states", "us",
		"australia", "new zealand", "taiwan",
	}

	for _, c := range countries {
		if strings.Contains(combinedText, c) {
			queries = append(queries, title+" official announcement")
			break
		}
	}

	// ⑦ NGO/国際機関の一次情報サイトを優先
	if strings.Contains(combinedText, "carbon") || strings.Contains(combinedText, "climate") || strings.Contains(combinedText, "emissions") {
		queries = append(queries, title+" site:unfccc.int OR site:icvcm.org OR site:iea.org")
	}

	// 重複除去
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

// extractFirstSentence extracts the first sentence from text, up to maxLen characters
func extractFirstSentence(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Find first sentence ending (. ! ?)
	sentenceEnders := []string{". ", "! ", "? "}
	minIdx := len(text)

	for _, ender := range sentenceEnders {
		idx := strings.Index(text, ender)
		if idx > 0 && idx < minIdx {
			minIdx = idx + 1 // Include the punctuation
		}
	}

	if minIdx < len(text) {
		text = text[:minIdx]
	}

	// Limit to maxLen
	if len(text) > maxLen {
		text = text[:maxLen]
		// Cut at last space to avoid cutting words
		if lastSpace := strings.LastIndex(text, " "); lastSpace > 0 {
			text = text[:lastSpace]
		}
	}

	return strings.TrimSpace(text)
}

// extractProperNouns extracts likely proper nouns (capitalized multi-word phrases)
// from text, returning up to maxNouns results
func extractProperNouns(text string) []string {
	// Match sequences of capitalized words (2+ words)
	// Example: "New York Times", "European Commission", "Climate Action Network"
	re := regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)+\b`)
	matches := re.FindAllString(text, -1)

	seen := map[string]bool{}
	result := []string{}

	for _, match := range matches {
		match = strings.TrimSpace(match)
		// Skip common words that aren't proper nouns
		if match == "" || seen[match] {
			continue
		}
		// Skip generic phrases
		lower := strings.ToLower(match)
		if strings.HasPrefix(lower, "the ") || strings.HasPrefix(lower, "a ") {
			continue
		}
		seen[match] = true
		result = append(result, match)

		// Limit to top 3 proper nouns to avoid query bloat
		if len(result) >= 3 {
			break
		}
	}

	return result
}

// extractNumbersWithContext extracts numbers with their context (units, currency, percentages)
// Example: "$50 million", "1.5 billion", "30%", "2025"
func extractNumbersWithContext(text string) []string {
	patterns := []*regexp.Regexp{
		// Currency: $50 million, €1.2 billion
		regexp.MustCompile(`[$€£¥]\s*[\d,]+(?:\.\d+)?\s*(?:million|billion|trillion|thousand|mn|bn)`),
		// Percentages: 30%, 1.5%
		regexp.MustCompile(`\d+(?:\.\d+)?%`),
		// Numbers with units: 1.5 billion tons, 50 million credits
		regexp.MustCompile(`\d+(?:\.\d+)?\s*(?:million|billion|trillion|thousand)\s+(?:tons?|credits?|tonnes?|dollars?|euros?)`),
		// Years: 2024, 2025
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

			// Limit to top 2 number contexts
			if len(result) >= 2 {
				return result
			}
		}
	}

	return result
}
