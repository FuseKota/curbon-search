// =============================================================================
// utils.go - ユーティリティ関数
// =============================================================================
//
// このファイルはシステム全体で使用する汎用的なヘルパー関数を提供します。
//
// 【このファイルで提供する機能】
//   - 文字列操作: ソート、重複削除、空白正規化
//   - JSON操作: ファイル読み書き、標準出力への出力
//   - ログ出力: 警告・情報メッセージの出力
//   - データ重複排除: URLベースのHeadline重複削除
//
// 【初心者向けポイント】
//   - Goでは小文字始まりの関数はパッケージ内でのみ使用可能（プライベート）
//   - `any`は任意の型を受け取れる特殊な型（Go 1.18以降）
//   - `...any`は可変長引数（任意の数の引数を受け取れる）
//
// =============================================================================
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// -----------------------------------------------------------------------------
// 文字列操作関数
// -----------------------------------------------------------------------------

// sortStrings は文字列スライスをアルファベット順にソートして返す
//
// 【重要】元のスライスは変更せず、新しいスライスを返す（非破壊的操作）
//
// 使用例:
//
//	input := []string{"banana", "apple", "cherry"}
//	sorted := sortStrings(input)  // ["apple", "banana", "cherry"]
func sortStrings(in []string) []string {
	// append([]string{}, in...) で元のスライスをコピー
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}

// normalizeWhitespace は文字列内の連続する空白を単一スペースに正規化する
//
// 使用例:
//
//	normalizeWhitespace("  hello   world  ")  // "hello world"
//
// 【処理の流れ】
//  1. strings.Fields: 空白で分割してスライスに（連続空白は無視される）
//  2. strings.Join: スペースで再結合
func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// uniqStrings は文字列スライスから重複と空文字列を除去する
//
// 使用例:
//
//	input := []string{"a", "b", "a", "", "c", "b"}
//	unique := uniqStrings(input)  // ["a", "b", "c"]
//
// 【アルゴリズム】
//
//	mapを使って既出の文字列を記録し、O(n)の計算量で重複を検出
func uniqStrings(in []string) []string {
	seen := map[string]bool{} // 既出の文字列を記録するマップ
	out := make([]string, 0, len(in))
	for _, s := range in {
		// 空文字列または既出の場合はスキップ
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// -----------------------------------------------------------------------------
// データ重複排除関数
// -----------------------------------------------------------------------------

// uniqueHeadlinesByURL はURLに基づいてHeadlineの重複を除去する
//
// 同じURLの記事が複数回収集された場合、最初に出現したものだけを残す。
// URLが空の記事は除外される。
//
// 【使用場面】
//
//	複数のソースから同じ記事が収集された場合の重複排除
func uniqueHeadlinesByURL(in []Headline) []Headline {
	seen := map[string]bool{}
	out := make([]Headline, 0, len(in))
	for _, h := range in {
		// URLが空の場合はスキップ
		if h.URL == "" {
			continue
		}
		// 既に同じURLが出現していたらスキップ
		if seen[h.URL] {
			continue
		}
		seen[h.URL] = true
		out = append(out, h)
	}
	return out
}

// -----------------------------------------------------------------------------
// JSON操作関数
// -----------------------------------------------------------------------------

// writeJSONToStdout は任意のデータをJSON形式で標準出力に書き出す
//
// 出力は2スペースでインデントされた読みやすい形式になる。
//
// 【使用場面】
//
//	パイプライン処理でJSONを次のコマンドに渡す場合
//	./pipeline ... | jq '.'
func writeJSONToStdout(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ") // 2スペースインデント
	return enc.Encode(v)
}

// writeJSONFile は任意のデータをJSON形式でファイルに保存する
//
// 引数:
//
//	path: 保存先のファイルパス
//	v:    JSON化するデータ（構造体、マップ、スライスなど）
//
// 【ファイル権限】0o644 = 所有者は読み書き可、他は読み取りのみ
func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// readJSONFile はJSONファイルを読み込んで指定した型に変換する
//
// 引数:
//
//	path: 読み込むファイルパス
//	out:  変換先の変数（ポインタで渡す必要がある）
//
// 使用例:
//
//	var headlines []Headline
//	err := readJSONFile("data.json", &headlines)
func readJSONFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// -----------------------------------------------------------------------------
// ログ出力関数
// -----------------------------------------------------------------------------

// warnf は警告メッセージを標準エラー出力に書き出す
//
// フォーマット: "WARN: メッセージ\n"
//
// 【なぜ標準エラー出力を使うか】
//
//	標準出力（stdout）はパイプラインでデータを渡すために使用するため、
//	ログメッセージは標準エラー出力（stderr）に出力する
func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", args...)
}

// infof は情報メッセージを標準エラー出力に書き出す
//
// フォーマット: "INFO: メッセージ\n"
func infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", args...)
}
