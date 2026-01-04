// matcher.go
// IDF（逆文書頻度）ベースのマッチングとスコアリングエンジン
//
// このモジュールは、有料記事の見出しと無料記事候補の類似度を計算し、
// 最も関連性の高い無料記事をマッチングします。
//
// 主要な機能:
//   - トークン化と正規化（ストップワード除去、マーケット/トピックキーワード正規化）
//   - IDF（逆文書頻度）計算
//   - シグナル抽出（Markets, Topics, Geos）
//   - 多次元スコアリング（IDF加重リコール、Jaccard、Market/Topic/Geo一致、新しさ）
//   - ドメイン品質ブースト（.gov, .pdf, NGOドメイン）
//   - 厳格フィルタリング（マーケットマッチ必須、地域マッチ必須）
//
// スコアリング重み配分:
//   - IDF加重リコール類似度: 56%
//   - IDF加重Jaccard類似度: 28%
//   - マーケットマッチ: 6%
//   - トピックマッチ: 4%
//   - 地理的マッチ: 2%
//   - 新しさ: 4%
//   - 品質ブースト: 最大+0.18
package main

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// -------------------- トークン化 / 正規化 --------------------

// reTok はトークン抽出用の正規表現（英数字とハイフン区切り）
var reTok = regexp.MustCompile(`[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*`)

// stop はストップワード（除外する一般的な単語）のマップ
// ノイズの多いマッチを避けるため、一般的な冠詞や前置詞を除外
var stop = map[string]bool{
	"the": true, "a": true, "an": true,
	"to": true, "of": true, "in": true, "on": true, "for": true, "with": true, "by": true,
	"and": true, "or": true, "as": true, "at": true, "from": true,
	"after": true, "before": true, "amid": true, "over": true, "under": true,
	"into": true, "out": true, "up": true, "down": true,
	"new": true, "fresh": true, "year": true, "yr": true,
}

// normToken はマーケット/トピックキーワードの正規化マップ
//
// 異なる表記を統一することで、マッチング精度を向上:
//   - 複数形→単数形（credits→credit, offsets→offset）
//   - 大文字小文字の統一（EUA→eua, UKA→uka）
//   - 略語の統一（I-RECs→irec）
//
// カバーするカテゴリ:
//   - カーボン市場: EUA, UKA, RGGI, CCA, ACCU, NZU, I-REC, CORSIA, CCER
//   - トピック: VCM, CDR, DAC, BECCS, biochar, methane, forest
//   - 一般用語: credit, offset
var normToken = map[string]string{
	"euas": "eua", "eua": "eua",
	"ukas": "uka", "uka": "uka",
	"rggi": "rggi",
	"ccas": "cca", "cca": "cca",
	"accus": "accu", "accu": "accu",
	"nzus": "nzu", "nzu": "nzu",
	"i-rec": "irec", "i-recs": "irec", "irec": "irec",
	"corsia": "corsia",
	"ccer": "ccer",
	"vcm": "vcm",
	"cdr": "cdr",
	"dac": "dac",
	"beccs": "beccs",
	"biochar": "biochar",
	"methane": "methane",
	"forest": "forest",
	"credits": "credit",
	"credit": "credit",
	"offset": "offset",
	"offsets": "offset",
}

func tokenize(raw string) []string {
	parts := reTok.FindAllString(raw, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.ToLower(p)
		if v, ok := normToken[t]; ok {
			t = v
		}
		if stop[t] {
			continue
		}
		if len(t) <= 1 {
			continue
		}
		out = append(out, t)
	}
	return uniqPreserveOrder(out)
}

func uniqPreserveOrder(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// -------------------- Signals --------------------

type Signals struct {
	Markets map[string]bool
	Topics  map[string]bool
	Geos    map[string]bool
}

var (
	reUS = regexp.MustCompile(`\bUS\b|(?i)\b(u\.s\.a\.|u\.s\.|usa|united states)\b`)
	reUK = regexp.MustCompile(`\bUK\b|(?i)\b(u\.k\.|united kingdom)\b`)
	reEU = regexp.MustCompile(`\bEU\b|(?i)\b(european union)\b`)
)

func tokenizeForSignals(raw string) []string {
	parts := reTok.FindAllString(raw, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.ToLower(p)
		if v, ok := normToken[t]; ok {
			t = v
		}
		out = append(out, t)
	}
	return uniqPreserveOrder(out)
}

var marketTerms = []string{"eua", "uka", "rggi", "cca", "accu", "nzu", "irec", "ccer", "corsia"}

var topicTerms = []string{
	"vcm", "cdr", "dac", "beccs", "biochar", "methane", "forest",
	"offset", "credit",
	"voluntary carbon market",
	"carbon border adjustment mechanism",
	"emissions trading system",
}

func extractSignals(raw string) Signals {
	lower := strings.ToLower(raw)
	s := Signals{Markets: map[string]bool{}, Topics: map[string]bool{}, Geos: map[string]bool{}}

	tokSet := map[string]bool{}
	for _, t := range tokenizeForSignals(raw) {
		tokSet[t] = true
	}

	// --- geo ---
	if reUS.MatchString(raw) {
		s.Geos["united_states"] = true
	}
	if reUK.MatchString(raw) {
		s.Geos["united_kingdom"] = true
	}
	if reEU.MatchString(raw) {
		s.Geos["eu"] = true
	}
	if strings.Contains(lower, "europe") {
		s.Geos["europe"] = true
	}
	if strings.Contains(lower, "south korea") {
		s.Geos["south_korea"] = true
	}
	if strings.Contains(lower, "new zealand") {
		s.Geos["new_zealand"] = true
	}

	singleGeos := []string{"taiwan", "malaysia", "india", "china", "australia", "alberta", "guyana", "brazil", "indonesia", "vietnam", "south africa"}
	for _, g := range singleGeos {
		if tokSet[g] {
			s.Geos[g] = true
		}
	}

	// --- market synonyms ---
	if strings.Contains(lower, "eu ets") || strings.Contains(lower, "eu emissions trading") || strings.Contains(lower, "emissions trading system") {
		tokSet["eua"] = true
	}
	if strings.Contains(lower, "uk ets") {
		tokSet["uka"] = true
	}
	if strings.Contains(lower, "regional greenhouse gas initiative") {
		tokSet["rggi"] = true
	}
	if strings.Contains(lower, "california carbon allowance") || strings.Contains(lower, "california cap-and-trade") {
		tokSet["cca"] = true
	}
	if strings.Contains(lower, "australian carbon credit unit") || strings.Contains(lower, "safeguard mechanism") {
		tokSet["accu"] = true
	}
	if strings.Contains(lower, "new zealand ets") {
		tokSet["nzu"] = true
	}

	// --- markets (token exact) ---
	for _, m := range marketTerms {
		if tokSet[m] {
			s.Markets[m] = true
		}
	}

	// --- topics ---
	for _, k := range topicTerms {
		if strings.Contains(k, " ") {
			if strings.Contains(lower, k) {
				s.Topics[k] = true
			}
		} else {
			if tokSet[k] {
				s.Topics[k] = true
			}
		}
	}

	return s
}

func hasSpecificGeo(sig Signals) bool {
	for g := range sig.Geos {
		switch g {
		case "eu", "europe", "united_states", "united_kingdom":
			continue
		default:
			return true
		}
	}
	return false
}

// -------------------- IDF --------------------

func buildIDF(docs [][]string) map[string]float64 {
	df := map[string]int{}
	N := len(docs)
	for _, d := range docs {
		seen := map[string]bool{}
		for _, t := range d {
			if seen[t] {
				continue
			}
			seen[t] = true
			df[t]++
		}
	}

	idf := map[string]float64{}
	for t, f := range df {
		idf[t] = math.Log(1.0 + float64(N)/float64(1+f))
	}

	max := 0.0
	for _, v := range idf {
		if v > max {
			max = v
		}
	}
	idf["__DEFAULT__"] = max
	return idf
}

func idfValue(idf map[string]float64, t string) float64 {
	if v, ok := idf[t]; ok {
		return v
	}
	if v, ok := idf["__DEFAULT__"]; ok && v > 0 {
		return v
	}
	return 1.0
}

// -------------------- Similarities --------------------

func idfWeightedRecallOverlap(aTokens, bTokens []string, idf map[string]float64) (overlap float64, shared int) {
	aSet := map[string]bool{}
	for _, t := range aTokens {
		aSet[t] = true
	}
	bSet := map[string]bool{}
	for _, t := range bTokens {
		bSet[t] = true
	}

	den := 0.0
	num := 0.0
	for t := range aSet {
		w := idfValue(idf, t)
		den += w
		if bSet[t] {
			num += w
			shared++
		}
	}
	if den == 0 {
		return 0, 0
	}
	return clamp01(num / den), shared
}

func idfWeightedJaccard(aTokens, bTokens []string, idf map[string]float64) float64 {
	aSet := map[string]bool{}
	for _, t := range aTokens {
		aSet[t] = true
	}
	bSet := map[string]bool{}
	for _, t := range bTokens {
		bSet[t] = true
	}

	unionW := 0.0
	interW := 0.0
	for t := range aSet {
		unionW += idfValue(idf, t)
	}
	for t := range bSet {
		if aSet[t] {
			continue
		}
		unionW += idfValue(idf, t)
	}
	for t := range aSet {
		if bSet[t] {
			interW += idfValue(idf, t)
		}
	}
	if unionW == 0 {
		return 0
	}
	return clamp01(interW / unionW)
}

func intersectScore(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	matched := 0
	for k := range a {
		if b[k] {
			matched++
		}
	}
	den := len(a)
	if den == 0 {
		return 0
	}
	return clamp01(float64(matched) / float64(den))
}

func recencyScoreRFC3339(publishedAt string, now time.Time, daysBack int) float64 {
	if publishedAt == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		return 0
	}
	age := now.Sub(t).Hours() / 24.0
	if age < 0 {
		age = 0
	}
	if daysBack > 0 && age > float64(daysBack) {
		return 0
	}
	return clamp01(math.Exp(-age / 14.0))
}

// -------------------- Quality heuristics --------------------

// Returns 0..0.18 boost for "more primary" sources.
func sourceQualityBoost(u string) float64 {
	pu, err := url.Parse(u)
	if err != nil {
		return 0
	}
	host := strings.ToLower(pu.Host)
	path := strings.ToLower(pu.Path)

	// PDFs / official docs
	if strings.HasSuffix(path, ".pdf") {
		return 0.18
	}

	// Government / regulators
	if strings.HasSuffix(host, ".gov") || strings.HasSuffix(host, ".gov.uk") || strings.HasSuffix(host, ".gouv.fr") || strings.HasSuffix(host, ".go.jp") {
		return 0.18
	}
	if strings.Contains(host, "europa.eu") {
		return 0.16
	}
	if strings.Contains(host, "sec.gov") || strings.Contains(host, "epa.gov") || strings.Contains(host, "energy.gov") || strings.Contains(host, "ec.europa.eu") {
		return 0.18
	}

	// Corporate IR
	if strings.Contains(path, "/investor") || strings.Contains(path, "/investors") || strings.Contains(path, "/ir") || strings.Contains(host, "investor") {
		return 0.12
	}

	// NGOs / policy orgs frequently used in this domain
	ngos := []string{"carbonmarketwatch.org", "forest-trends.org", "ecosystemmarketplace.com", "icvcm.org", "unfccc.int", "iea.org"}
	for _, d := range ngos {
		if strings.HasSuffix(host, d) {
			return 0.12
		}
	}

	// Press release wires
	pr := []string{"prnewswire.com", "businesswire.com", "globenewswire.com"}
	for _, d := range pr {
		if strings.HasSuffix(host, d) {
			return 0.08
		}
	}

	return 0
}

// -------------------- Matching --------------------

type scored struct {
	item   FreeArticle
	score  float64
	reason string
}

// Main scoring rule.
func scoreHeadlineCandidate(h Headline, cand FreeArticle, idf map[string]float64, now time.Time, daysBack int, strictMarket bool, minScore float64) (scored, bool) {
	hTok := tokenize(h.Title)
	cTok := tokenize(cand.Title)

	hs := extractSignals(h.Title)
	cs := extractSignals(cand.Title)

	overlap, sharedTokens := idfWeightedRecallOverlap(hTok, cTok, idf)
	titleSim := idfWeightedJaccard(hTok, cTok, idf)
	rec := recencyScoreRFC3339(cand.PublishedAt, now, daysBack)

	marketMatch := intersectScore(hs.Markets, cs.Markets)
	topicMatch := intersectScore(hs.Topics, cs.Topics)
	geoMatch := intersectScore(hs.Geos, cs.Geos)

	if strictMarket && len(hs.Markets) > 0 && marketMatch == 0 {
		return scored{}, false
	}

	if hasSpecificGeo(hs) && geoMatch == 0 {
		return scored{}, false
	}

	// Avoid matching purely on broad geos.
	if marketMatch == 0 && topicMatch == 0 && geoMatch > 0 && overlap < 0.50 && titleSim < 0.84 {
		return scored{}, false
	}

	// Require some lexical substance unless titleSim is extremely high.
	if sharedTokens < 2 && titleSim < 0.90 {
		return scored{}, false
	}

	qBoost := sourceQualityBoost(cand.URL)

	// Similarity-focused, with small signal + quality nudges.
	score := 0.56*overlap + 0.28*titleSim + 0.06*marketMatch + 0.04*topicMatch + 0.02*geoMatch + 0.04*rec + qBoost
	if score < minScore {
		return scored{}, false
	}

	reason := fmt.Sprintf("overlap=%.2f titleSim=%.2f recency=%.2f market=%.2f topic=%.2f geo=%.2f quality=%.2f sharedTokens=%d",
		overlap, titleSim, rec, marketMatch, topicMatch, geoMatch, qBoost, sharedTokens)

	return scored{item: cand, score: score, reason: reason}, true
}

func topKRelated(h Headline, candidates []FreeArticle, idf map[string]float64, now time.Time, daysBack int, strictMarket bool, topK int, minScore float64) []RelatedFree {
	best := make([]scored, 0, topK)
	seenURL := map[string]bool{}

	for _, cand := range candidates {
		if cand.URL == "" || cand.Title == "" {
			continue
		}
		if seenURL[cand.URL] {
			continue
		}
		seenURL[cand.URL] = true

		if s, ok := scoreHeadlineCandidate(h, cand, idf, now, daysBack, strictMarket, minScore); ok {
			best = append(best, s)
		}
	}

	sort.Slice(best, func(i, j int) bool {
		if best[i].score == best[j].score {
			return best[i].item.Title < best[j].item.Title
		}
		return best[i].score > best[j].score
	})

	if len(best) > topK {
		best = best[:topK]
	}

	out := make([]RelatedFree, 0, len(best))
	for _, b := range best {
		out = append(out, RelatedFree{
			Source:      b.item.Source,
			Title:       b.item.Title,
			URL:         b.item.URL,
			PublishedAt: b.item.PublishedAt,
			Excerpt:     b.item.Excerpt,
			Score:       b.score,
			Reason:      b.reason,
		})
	}
	return out
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
