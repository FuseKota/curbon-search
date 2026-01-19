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
package pipeline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
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

// errorf はエラーメッセージを標準エラー出力に書き出す
//
// フォーマット: "ERROR: メッセージ\n"
//
// 【注意】この関数はログ出力のみでプログラムは終了しない
// プログラムを終了させる場合は fatalf() を使用する
func errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
}

// -----------------------------------------------------------------------------
// HTTP操作関数
// -----------------------------------------------------------------------------

// httpGet はHTTP GETリクエストを実行する
//
// User-Agentヘッダーを設定し、タイムアウト付きでリクエストを送信する。
// 呼び出し元でresp.Body.Close()を行う必要がある。
//
// 引数:
//
//	url:       リクエスト先URL
//	userAgent: User-Agentヘッダーの値
//	timeout:   リクエストタイムアウト
//
// 使用例:
//
//	resp, err := httpGet("https://example.com/api", "MyBot/1.0", 20*time.Second)
//	if err != nil { return err }
//	defer resp.Body.Close()
func httpGet(url, userAgent string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

// httpGetWithConfig はHeadlineSourceConfigを使用してHTTP GETリクエストを実行する
//
// HeadlineSourceConfigのUserAgentとTimeoutを使用する便利関数。
//
// 使用例:
//
//	resp, err := httpGetWithConfig("https://example.com/api", cfg)
//	if err != nil { return err }
//	defer resp.Body.Close()
func httpGetWithConfig(url string, cfg HeadlineSourceConfig) (*http.Response, error) {
	return httpGet(url, cfg.UserAgent, cfg.Timeout)
}

// httpGetJSON はHTTP GETリクエストを実行し、JSONレスポンスをデコードする
//
// レスポンスボディを自動的にクローズし、指定した型にJSONをデコードする。
//
// 引数:
//
//	url:       リクエスト先URL
//	cfg:       HeadlineSourceConfig（UserAgentとTimeoutを使用）
//	v:         デコード先の変数（ポインタで渡す）
//
// 使用例:
//
//	var posts []WPPost
//	err := httpGetJSON("https://example.com/wp-json/wp/v2/posts", cfg, &posts)
func httpGetJSON(url string, cfg HeadlineSourceConfig, v interface{}) error {
	resp, err := httpGetWithConfig(url, cfg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// fatalf はエラーメッセージを出力してプログラムを終了する
//
// フォーマット: "メッセージ\n" の後にos.Exit(1)で終了
//
// 【使用場面】
//
//	致命的なエラーが発生し、処理を継続できない場合
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// truncateString は文字列を指定した長さに切り詰める
//
// maxLen文字を超える場合、末尾に"..."を付けて切り詰める
// 日本語などのマルチバイト文字も正しく処理する（runeを使用）
//
// 使用例:
//
//	truncateString("Hello World", 8)  // "Hello..."
//	truncateString("短い", 10)        // "短い"（そのまま）
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
