// =============================================================================
// matcher.go - IDF（逆文書頻度）ベースのマッチング・スコアリングエンジン
// =============================================================================
//
// このファイルは、有料記事の見出しと無料記事候補の類似度を計算し、
// 最も関連性の高い無料記事をマッチングします。
// Carbon Relayのモード2（有料記事マッチング）の中核機能です。
//
// =============================================================================
// 【スコアリング重み配分】
// =============================================================================
//
// 最終スコア = 以下の合計（0.0〜1.0+品質ブースト）
//
//   ┌─────────────────────────────┬────────┐
//   │ 評価項目                    │ 重み   │
//   ├─────────────────────────────┼────────┤
//   │ IDF加重リコール類似度       │   56%  │
//   │ IDF加重Jaccard類似度        │   28%  │
//   │ マーケットマッチ            │    6%  │
//   │ トピックマッチ              │    4%  │
//   │ 地理的マッチ                │    2%  │
//   │ 新しさ（Recency）           │    4%  │
//   │ 品質ブースト                │ +0.18  │
//   └─────────────────────────────┴────────┘
//
// =============================================================================
// 【主要な機能】
// =============================================================================
//
// 1. トークン化と正規化
//    - ストップワード除去（the, a, of, in 等）
//    - マーケット/トピックキーワードの正規化（EUAs→eua等）
//
// 2. IDF（逆文書頻度）計算
//    - 珍しい単語ほど高いスコア
//    - 公式: IDF(t) = log(1 + N / (1 + df(t)))
//
// 3. シグナル抽出
//    - Markets: EUA, UKA, RGGI, CCA 等のカーボン市場
//    - Topics: VCM, CDR, DAC, biochar 等のトピック
//    - Geos: EU, US, UK, Japan, China 等の地域
//
// 4. 多次元スコアリング
//    - IDF加重リコール: 見出しの単語がどれだけ候補に含まれるか
//    - Jaccard類似度: 両方の単語セットの重なり具合
//
// 5. ドメイン品質ブースト
//    - .gov ドメイン: +0.18
//    - .pdf ファイル: +0.18
//    - NGOサイト: +0.12
//    - プレスリリース: +0.08
//
// =============================================================================
// 【フィルタリングルール】
// =============================================================================
//
// 以下の条件を満たさない候補は除外：
//
// 1. strictMarket モードで市場シグナルがある場合、候補も同じ市場を持つ必要
// 2. 特定の地域（台湾、韓国等）が見出しにある場合、候補も同じ地域を持つ必要
// 3. 共有トークンが2未満かつタイトル類似度0.90未満の場合は除外
// 4. 広い地域のみでマッチし、語彙的重複が少ない場合は除外
//
// =============================================================================
// 【初心者向けポイント】
// =============================================================================
//
// - IDF: 「珍しい単語は重要」という考え方
//   例: "carbon" は多くの記事に出現するので重要度低い
//       "CORSIA" は珍しいので重要度高い
//
// - Jaccard類似度: 2つの集合の重なり具合を測る
//   公式: |A ∩ B| / |A ∪ B|
//
// - リコール: 見出しの単語が候補にどれだけ含まれるか
//   公式: |A ∩ B| / |A|
//
// =============================================================================
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

// =============================================================================
// トークン化 / 正規化
// =============================================================================

// reTok はトークン抽出用の正規表現
// 英数字とハイフンで構成される単語をマッチ（例: "I-REC", "gpt-4o"）
var reTok = regexp.MustCompile(`[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*`)

// stop はストップワード（除外する一般的な単語）のマップ
//
// これらの単語は多くの記事に出現するため、マッチングのノイズになる。
// 例: "the carbon credit" と "the EUA price" は "the" で一致してしまう
var stop = map[string]bool{
	// 冠詞
	"the": true, "a": true, "an": true,
	// 前置詞
	"to": true, "of": true, "in": true, "on": true, "for": true, "with": true, "by": true,
	// 接続詞
	"and": true, "or": true, "as": true, "at": true, "from": true,
	// 時間関連
	"after": true, "before": true, "amid": true, "over": true, "under": true,
	// 方向
	"into": true, "out": true, "up": true, "down": true,
	// 汎用形容詞
	"new": true, "fresh": true, "year": true, "yr": true,
}

// normToken はマーケット/トピックキーワードの正規化マップ
//
// 【正規化の目的】
//
//	異なる表記を統一することで、マッチング精度を向上させる
//
// 【正規化ルール】
//   - 複数形→単数形: credits→credit, offsets→offset
//   - 大文字→小文字: EUA→eua, UKA→uka
//   - 略語の統一: I-RECs→irec
//
// 【カバーするカテゴリ】
//   - カーボン市場: EUA, UKA, RGGI, CCA, ACCU, NZU, I-REC, CORSIA, CCER
//   - トピック: VCM, CDR, DAC, BECCS, biochar, methane, forest
//   - 一般用語: credit, offset
var normToken = map[string]string{
	// EUカーボン市場（EU Allowance）
	"euas": "eua", "eua": "eua",
	// UKカーボン市場（UK Allowance）
	"ukas": "uka", "uka": "uka",
	// 米国北東部の排出権取引（Regional Greenhouse Gas Initiative）
	"rggi": "rggi",
	// カリフォルニア排出権（California Carbon Allowance）
	"ccas": "cca", "cca": "cca",
	// オーストラリアカーボンクレジット（Australian Carbon Credit Unit）
	"accus": "accu", "accu": "accu",
	// ニュージーランド排出権（NZ Unit）
	"nzus": "nzu", "nzu": "nzu",
	// 再生可能エネルギー証書（International Renewable Energy Certificate）
	"i-rec": "irec", "i-recs": "irec", "irec": "irec",
	// 国際航空カーボンオフセット（Carbon Offsetting and Reduction Scheme for International Aviation）
	"corsia": "corsia",
	// 中国認証排出削減量（China Certified Emission Reduction）
	"ccer": "ccer",
	// 自主的カーボン市場（Voluntary Carbon Market）
	"vcm": "vcm",
	// 二酸化炭素除去（Carbon Dioxide Removal）
	"cdr": "cdr",
	// 直接空気回収（Direct Air Capture）
	"dac": "dac",
	// バイオエネルギーCCS
	"beccs": "beccs",
	// バイオ炭
	"biochar": "biochar",
	// メタン
	"methane": "methane",
	// 森林
	"forest": "forest",
	// クレジット（複数形→単数形）
	"credits": "credit", "credit": "credit",
	// オフセット（複数形→単数形）
	"offset": "offset", "offsets": "offset",
}

// tokenize はテキストをトークン（単語）のスライスに分割する
//
// 【処理の流れ】
//  1. 正規表現でトークンを抽出
//  2. 小文字に変換
//  3. 正規化マップで表記を統一
//  4. ストップワードを除外
//  5. 1文字以下のトークンを除外
//  6. 重複を除去（出現順は保持）
func tokenize(raw string) []string {
	parts := reTok.FindAllString(raw, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.ToLower(p)
		// 正規化マップで表記を統一
		if v, ok := normToken[t]; ok {
			t = v
		}
		// ストップワードは除外
		if stop[t] {
			continue
		}
		// 1文字以下は除外
		if len(t) <= 1 {
			continue
		}
		out = append(out, t)
	}
	return uniqPreserveOrder(out)
}

// uniqPreserveOrder は重複を除去しつつ出現順を保持する
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

// =============================================================================
// シグナル抽出（Markets, Topics, Geos）
// =============================================================================

// Signals は記事から抽出されたシグナル情報を保持する
//
// 【シグナルの種類】
//   - Markets: カーボン市場の種類（EUA, UKA, RGGI等）
//   - Topics: トピック（VCM, CDR, biochar等）
//   - Geos: 地域（EU, US, Japan等）
type Signals struct {
	Markets map[string]bool // カーボン市場シグナル
	Topics  map[string]bool // トピックシグナル
	Geos    map[string]bool // 地理的シグナル
}

// 国名を検出するための正規表現パターン
var (
	// US: "US", "USA", "U.S.", "United States" など
	reUS = regexp.MustCompile(`\bUS\b|(?i)\b(u\.s\.a\.|u\.s\.|usa|united states)\b`)
	// UK: "UK", "U.K.", "United Kingdom" など
	reUK = regexp.MustCompile(`\bUK\b|(?i)\b(u\.k\.|united kingdom)\b`)
	// EU: "EU", "European Union" など
	reEU = regexp.MustCompile(`\bEU\b|(?i)\b(european union)\b`)
)

// tokenizeForSignals はシグナル抽出用のトークン化（ストップワード除去なし）
//
// シグナル抽出ではストップワードも含めて全トークンを確認する必要がある
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

// marketTerms は検出するカーボン市場の種類
var marketTerms = []string{
	"eua",     // EU Allowance
	"uka",     // UK Allowance
	"rggi",    // Regional Greenhouse Gas Initiative
	"cca",     // California Carbon Allowance
	"accu",    // Australian Carbon Credit Unit
	"nzu",     // NZ Unit
	"irec",    // I-REC
	"ccer",    // China Certified Emission Reduction
	"corsia",  // ICAO Carbon Offset
}

// topicTerms は検出するトピック
var topicTerms = []string{
	"vcm",                                  // Voluntary Carbon Market
	"cdr",                                  // Carbon Dioxide Removal
	"dac",                                  // Direct Air Capture
	"beccs",                                // Bioenergy with CCS
	"biochar",                              // バイオ炭
	"methane",                              // メタン
	"forest",                               // 森林
	"offset",                               // オフセット
	"credit",                               // クレジット
	"voluntary carbon market",              // 複数語トピック
	"carbon border adjustment mechanism",   // CBAM
	"emissions trading system",             // ETS
}

// extractSignals はテキストからシグナル（Markets, Topics, Geos）を抽出する
//
// 【抽出方法】
//  1. 正規表現で国名を検出（US, UK, EU）
//  2. 文字列検索で複数語の地名を検出（South Korea, New Zealand）
//  3. トークンから市場シグナルを検出
//  4. フレーズマッチでトピックシグナルを検出
func extractSignals(raw string) Signals {
	lower := strings.ToLower(raw)
	s := Signals{
		Markets: map[string]bool{},
		Topics:  map[string]bool{},
		Geos:    map[string]bool{},
	}

	// トークンセットを作成
	tokSet := map[string]bool{}
	for _, t := range tokenizeForSignals(raw) {
		tokSet[t] = true
	}

	// =========================================================================
	// 地理的シグナルの抽出
	// =========================================================================

	// 正規表現でUS, UK, EUを検出
	if reUS.MatchString(raw) {
		s.Geos["united_states"] = true
	}
	if reUK.MatchString(raw) {
		s.Geos["united_kingdom"] = true
	}
	if reEU.MatchString(raw) {
		s.Geos["eu"] = true
	}

	// 文字列検索で複数語の地名を検出
	if strings.Contains(lower, "europe") {
		s.Geos["europe"] = true
	}
	if strings.Contains(lower, "south korea") {
		s.Geos["south_korea"] = true
	}
	if strings.Contains(lower, "new zealand") {
		s.Geos["new_zealand"] = true
	}

	// 単一語の国名を検出
	singleGeos := []string{
		"taiwan", "malaysia", "india", "china", "australia",
		"alberta", "guyana", "brazil", "indonesia", "vietnam", "south africa",
	}
	for _, g := range singleGeos {
		if tokSet[g] {
			s.Geos[g] = true
		}
	}

	// =========================================================================
	// マーケット同義語の展開
	// =========================================================================
	// フレーズから市場シグナルを推測

	// "EU ETS" → EUA
	if strings.Contains(lower, "eu ets") || strings.Contains(lower, "eu emissions trading") || strings.Contains(lower, "emissions trading system") {
		tokSet["eua"] = true
	}
	// "UK ETS" → UKA
	if strings.Contains(lower, "uk ets") {
		tokSet["uka"] = true
	}
	// "Regional Greenhouse Gas Initiative" → RGGI
	if strings.Contains(lower, "regional greenhouse gas initiative") {
		tokSet["rggi"] = true
	}
	// "California Carbon Allowance" or "cap-and-trade" → CCA
	if strings.Contains(lower, "california carbon allowance") || strings.Contains(lower, "california cap-and-trade") {
		tokSet["cca"] = true
	}
	// "Australian Carbon Credit Unit" or "Safeguard Mechanism" → ACCU
	if strings.Contains(lower, "australian carbon credit unit") || strings.Contains(lower, "safeguard mechanism") {
		tokSet["accu"] = true
	}
	// "New Zealand ETS" → NZU
	if strings.Contains(lower, "new zealand ets") {
		tokSet["nzu"] = true
	}

	// =========================================================================
	// マーケットシグナルの抽出（トークン完全一致）
	// =========================================================================
	for _, m := range marketTerms {
		if tokSet[m] {
			s.Markets[m] = true
		}
	}

	// =========================================================================
	// トピックシグナルの抽出
	// =========================================================================
	for _, k := range topicTerms {
		if strings.Contains(k, " ") {
			// 複数語トピックはフレーズマッチ
			if strings.Contains(lower, k) {
				s.Topics[k] = true
			}
		} else {
			// 単一語トピックはトークンマッチ
			if tokSet[k] {
				s.Topics[k] = true
			}
		}
	}

	return s
}

// hasSpecificGeo は特定の地域シグナルを持つかどうかを判定する
//
// EU, Europe, US, UK は広い地域なので「特定」とはみなさない
// 台湾、韓国、オーストラリア等は「特定」とみなす
func hasSpecificGeo(sig Signals) bool {
	for g := range sig.Geos {
		switch g {
		case "eu", "europe", "united_states", "united_kingdom":
			// 広い地域は特定とみなさない
			continue
		default:
			// それ以外は特定の地域
			return true
		}
	}
	return false
}

// =============================================================================
// IDF（逆文書頻度）計算
// =============================================================================

// buildIDF はドキュメントコーパスからIDFマップを構築する
//
// 【IDF（逆文書頻度）とは】
//
//	珍しい単語ほど高いスコアを与える重み付け手法
//	例: "carbon" は多くの記事に出現 → IDF低い
//	    "CORSIA" は珍しい → IDF高い
//
// 【計算式】
//
//	IDF(t) = log(1 + N / (1 + df(t)))
//	  N: ドキュメント総数
//	  df(t): トークンtを含むドキュメント数
//
// 【__DEFAULT__キー】
//
//	未知のトークンに対するデフォルト値として、最大IDF値を使用
func buildIDF(docs [][]string) map[string]float64 {
	// df: 各トークンがいくつのドキュメントに出現するか
	df := map[string]int{}
	N := len(docs)

	for _, d := range docs {
		seen := map[string]bool{} // 同じドキュメント内での重複を除去
		for _, t := range d {
			if seen[t] {
				continue
			}
			seen[t] = true
			df[t]++
		}
	}

	// IDFを計算
	idf := map[string]float64{}
	for t, f := range df {
		idf[t] = math.Log(1.0 + float64(N)/float64(1+f))
	}

	// 最大IDF値をデフォルト値として保存
	max := 0.0
	for _, v := range idf {
		if v > max {
			max = v
		}
	}
	idf["__DEFAULT__"] = max

	return idf
}

// idfValue はトークンのIDF値を取得する（未知トークンはデフォルト値）
func idfValue(idf map[string]float64, t string) float64 {
	if v, ok := idf[t]; ok {
		return v
	}
	if v, ok := idf["__DEFAULT__"]; ok && v > 0 {
		return v
	}
	return 1.0
}

// =============================================================================
// 類似度計算
// =============================================================================

// idfWeightedRecallOverlap はIDF加重リコール類似度を計算する
//
// 【リコールとは】
//
//	見出し（A）の単語のうち、候補（B）にどれだけ含まれるか
//	公式: Σ(IDF(t) for t in A∩B) / Σ(IDF(t) for t in A)
//
// 【戻り値】
//   - overlap: 重なり率（0.0〜1.0）
//   - shared: 共有トークン数
func idfWeightedRecallOverlap(aTokens, bTokens []string, idf map[string]float64) (overlap float64, shared int) {
	aSet := map[string]bool{}
	for _, t := range aTokens {
		aSet[t] = true
	}
	bSet := map[string]bool{}
	for _, t := range bTokens {
		bSet[t] = true
	}

	den := 0.0 // 分母: Aのトークンの重み合計
	num := 0.0 // 分子: A∩Bのトークンの重み合計
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

// idfWeightedJaccard はIDF加重Jaccard類似度を計算する
//
// 【Jaccard類似度とは】
//
//	2つの集合の重なり具合を測る指標
//	公式: |A ∩ B| / |A ∪ B|
//
// 【IDF加重版】
//
//	各トークンにIDF重みを付けて計算
//	公式: Σ(IDF(t) for t in A∩B) / Σ(IDF(t) for t in A∪B)
func idfWeightedJaccard(aTokens, bTokens []string, idf map[string]float64) float64 {
	aSet := map[string]bool{}
	for _, t := range aTokens {
		aSet[t] = true
	}
	bSet := map[string]bool{}
	for _, t := range bTokens {
		bSet[t] = true
	}

	// 和集合の重み合計
	unionW := 0.0
	for t := range aSet {
		unionW += idfValue(idf, t)
	}
	for t := range bSet {
		if aSet[t] {
			continue // 重複は除く
		}
		unionW += idfValue(idf, t)
	}

	// 積集合の重み合計
	interW := 0.0
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

// intersectScore はシグナルマップの重なり度を計算する
//
// 公式: |A ∩ B| / |A|（Aに対するリコール）
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

// recencyScoreRFC3339 は記事の新しさスコアを計算する
//
// 【計算式】
//
//	score = exp(-age / 14.0)
//	  age: 記事の経過日数
//	  14日で約36%に減衰、30日で約12%に減衰
//
// 【daysBack制限】
//
//	指定日数より古い記事は0点
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
	// 指定日数より古い場合は0点
	if daysBack > 0 && age > float64(daysBack) {
		return 0
	}
	// 指数関数的減衰（半減期約10日）
	return clamp01(math.Exp(-age / 14.0))
}

// =============================================================================
// ドメイン品質ヒューリスティック
// =============================================================================

// sourceQualityBoost はURLに基づいて品質ブースト（0〜0.18）を返す
//
// 【品質ブーストの考え方】
//
//	一次情報源や公式文書は、ニュース記事より信頼性が高い
//	これらにブーストを与えることで、より良いマッチングを実現
//
// 【ブースト値】
//   - PDF / 政府ドメイン: +0.18
//   - EU公式サイト: +0.16
//   - 企業IR: +0.12
//   - NGO / 政策機関: +0.12
//   - プレスリリース: +0.08
func sourceQualityBoost(u string) float64 {
	pu, err := url.Parse(u)
	if err != nil {
		return 0
	}
	host := strings.ToLower(pu.Host)
	path := strings.ToLower(pu.Path)

	// PDF文書（公式レポート、規制文書等）
	if strings.HasSuffix(path, ".pdf") {
		return 0.18
	}

	// 政府 / 規制当局
	if strings.HasSuffix(host, ".gov") || strings.HasSuffix(host, ".gov.uk") ||
		strings.HasSuffix(host, ".gouv.fr") || strings.HasSuffix(host, ".go.jp") {
		return 0.18
	}

	// EU公式サイト
	if strings.Contains(host, "europa.eu") {
		return 0.16
	}

	// 米国政府機関
	if strings.Contains(host, "sec.gov") || strings.Contains(host, "epa.gov") ||
		strings.Contains(host, "energy.gov") || strings.Contains(host, "ec.europa.eu") {
		return 0.18
	}

	// 企業IR（投資家向け情報）
	if strings.Contains(path, "/investor") || strings.Contains(path, "/investors") ||
		strings.Contains(path, "/ir") || strings.Contains(host, "investor") {
		return 0.12
	}

	// NGO / 政策機関
	ngos := []string{
		"carbonmarketwatch.org",
		"forest-trends.org",
		"ecosystemmarketplace.com",
		"icvcm.org",
		"unfccc.int",
		"iea.org",
	}
	for _, d := range ngos {
		if strings.HasSuffix(host, d) {
			return 0.12
		}
	}

	// プレスリリースワイヤー
	pr := []string{"prnewswire.com", "businesswire.com", "globenewswire.com"}
	for _, d := range pr {
		if strings.HasSuffix(host, d) {
			return 0.08
		}
	}

	return 0
}

// =============================================================================
// マッチング
// =============================================================================

// scored はスコア付き候補を表す内部構造体
type scored struct {
	item   FreeArticle // 候補記事
	score  float64     // 最終スコア
	reason string      // スコアの内訳
}

// scoreHeadlineCandidate は見出しと候補記事のスコアを計算する
//
// 【処理の流れ】
//  1. トークン化
//  2. シグナル抽出
//  3. 類似度計算（リコール、Jaccard）
//  4. フィルタリング（市場・地域マッチ必須等）
//  5. 最終スコア計算
//
// 【フィルタリングで除外される場合】
//   - strictMarket時に市場シグナルが一致しない
//   - 特定地域がある場合に地域が一致しない
//   - 共有トークンが少なく類似度も低い
//   - 広い地域のみでマッチし語彙的重複が少ない
func scoreHeadlineCandidate(h Headline, cand FreeArticle, idf map[string]float64, now time.Time, daysBack int, strictMarket bool, minScore float64) (scored, bool) {
	// トークン化
	hTok := tokenize(h.Title)
	cTok := tokenize(cand.Title)

	// シグナル抽出
	hs := extractSignals(h.Title)
	cs := extractSignals(cand.Title)

	// 類似度計算
	overlap, sharedTokens := idfWeightedRecallOverlap(hTok, cTok, idf)
	titleSim := idfWeightedJaccard(hTok, cTok, idf)
	rec := recencyScoreRFC3339(cand.PublishedAt, now, daysBack)

	// シグナルマッチ計算
	marketMatch := intersectScore(hs.Markets, cs.Markets)
	topicMatch := intersectScore(hs.Topics, cs.Topics)
	geoMatch := intersectScore(hs.Geos, cs.Geos)

	// =========================================================================
	// フィルタリング
	// =========================================================================

	// strictMarketモードで市場シグナルがある場合、候補も同じ市場を持つ必要
	if strictMarket && len(hs.Markets) > 0 && marketMatch == 0 {
		return scored{}, false
	}

	// 特定の地域が見出しにある場合、候補も同じ地域を持つ必要
	if hasSpecificGeo(hs) && geoMatch == 0 {
		return scored{}, false
	}

	// 広い地域のみでマッチし、語彙的重複が少ない場合は除外
	if marketMatch == 0 && topicMatch == 0 && geoMatch > 0 && overlap < 0.50 && titleSim < 0.84 {
		return scored{}, false
	}

	// 共有トークンが2未満かつタイトル類似度0.90未満の場合は除外
	if sharedTokens < 2 && titleSim < 0.90 {
		return scored{}, false
	}

	// 品質ブースト
	qBoost := sourceQualityBoost(cand.URL)

	// =========================================================================
	// 最終スコア計算
	// =========================================================================
	// 類似度重視、シグナル + 品質は補助的
	score := 0.56*overlap + 0.28*titleSim + 0.06*marketMatch + 0.04*topicMatch + 0.02*geoMatch + 0.04*rec + qBoost

	if score < minScore {
		return scored{}, false
	}

	// スコアの内訳を文字列化
	reason := fmt.Sprintf("overlap=%.2f titleSim=%.2f recency=%.2f market=%.2f topic=%.2f geo=%.2f quality=%.2f sharedTokens=%d",
		overlap, titleSim, rec, marketMatch, topicMatch, geoMatch, qBoost, sharedTokens)

	return scored{item: cand, score: score, reason: reason}, true
}

// topKRelated は見出しに対して上位K件の関連記事を返す
//
// 【処理の流れ】
//  1. 全候補に対してスコアを計算
//  2. スコアでソート（降順）
//  3. 上位K件を返す
func topKRelated(h Headline, candidates []FreeArticle, idf map[string]float64, now time.Time, daysBack int, strictMarket bool, topK int, minScore float64) []RelatedFree {
	best := make([]scored, 0, topK)
	seenURL := map[string]bool{}

	for _, cand := range candidates {
		// 空のURLやタイトルはスキップ
		if cand.URL == "" || cand.Title == "" {
			continue
		}
		// 重複URLはスキップ
		if seenURL[cand.URL] {
			continue
		}
		seenURL[cand.URL] = true

		// スコア計算（フィルタリング込み）
		if s, ok := scoreHeadlineCandidate(h, cand, idf, now, daysBack, strictMarket, minScore); ok {
			best = append(best, s)
		}
	}

	// スコアでソート（降順）、同点の場合はタイトルでソート
	sort.Slice(best, func(i, j int) bool {
		if best[i].score == best[j].score {
			return best[i].item.Title < best[j].item.Title
		}
		return best[i].score > best[j].score
	})

	// 上位K件に絞る
	if len(best) > topK {
		best = best[:topK]
	}

	// RelatedFree形式に変換
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

// clamp01 は値を0〜1の範囲に制限する
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
