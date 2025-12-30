package main

import (
	"fmt"
	"strings"
)

// buildSearchQueries generates search queries from a paid headline.
// 目的：
// - 見出しそのもの
// - 国・制度・キーワードを含めた再検索
// - site: / filetype: 演算子で一次情報を優先
func buildSearchQueries(title string) []string {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	queries := []string{}
	lower := strings.ToLower(title)

	// ① 完全一致（引用符付き）
	queries = append(queries, fmt.Sprintf(`"%s"`, title))

	// ② カーボン市場系キーワード補助
	if strings.Contains(lower, "vcm") {
		queries = append(queries, title+" voluntary carbon market")
	}
	if strings.Contains(lower, "ets") || strings.Contains(lower, "eua") || strings.Contains(lower, "uka") {
		queries = append(queries, title+" emissions trading system")
	}
	if strings.Contains(lower, "corsia") {
		queries = append(queries, title+" CORSIA ICAO")
	}
	if strings.Contains(lower, "ccer") {
		queries = append(queries, title+" CCER China")
	}
	if strings.Contains(lower, "biochar") {
		queries = append(queries, title+" biochar project")
	}

	// ③ 地域別 site: 演算子（政府・規制当局を優先）
	if strings.Contains(lower, "south korea") || strings.Contains(lower, "korea") {
		queries = append(queries, title+" site:go.kr")
	}
	if strings.Contains(lower, "eu") || strings.Contains(lower, "europe") {
		queries = append(queries, title+" site:europa.eu")
	}
	if strings.Contains(lower, "japan") {
		queries = append(queries, title+" site:go.jp")
	}
	if strings.Contains(lower, "uk") || strings.Contains(lower, "united kingdom") {
		queries = append(queries, title+" site:gov.uk")
	}
	if strings.Contains(lower, "china") {
		queries = append(queries, title+" site:gov.cn")
	}
	if strings.Contains(lower, "australia") {
		queries = append(queries, title+" site:gov.au")
	}

	// ④ PDF 優先（一次資料・規制文書）
	queries = append(queries, title+" filetype:pdf")

	// ⑤ 国・地域が含まれていそうなら official を足す
	countries := []string{
		"south korea", "korea", "china", "japan",
		"eu", "europe", "uk", "united states", "us",
		"australia", "new zealand", "taiwan",
	}

	for _, c := range countries {
		if strings.Contains(lower, c) {
			queries = append(queries, title+" official announcement")
			break
		}
	}

	// ⑥ NGO/国際機関の一次情報サイトを優先
	if strings.Contains(lower, "carbon") || strings.Contains(lower, "climate") || strings.Contains(lower, "emissions") {
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
