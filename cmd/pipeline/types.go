// =============================================================================
// types.go - データ構造定義
// =============================================================================
//
// このファイルはCarbon Relayシステム全体で使用するデータ構造（型）を定義します。
//
// 【このファイルで定義している型】
//   - Headline:       記事の見出し情報
//   - FreeArticle:    無料記事の情報
//   - NotionHeadline: Notionから取得した見出し
//
// 【初心者向けポイント】
//   - Go言語では`type 型名 struct { ... }`で構造体（複数のデータをまとめた型）を定義
//   - `json:"フィールド名"`はJSONに変換する際のキー名を指定するタグ
//   - `omitempty`は値が空の場合、JSONに出力しないことを意味
//
// =============================================================================
package main

// -----------------------------------------------------------------------------
// Headline - 記事の見出し情報
// -----------------------------------------------------------------------------
//
// ニュースソースから取得した記事の見出しを表します。
//
// 【フィールドの説明】
//   Source:        記事のソース名（例: "Carbon Herald", "Carbon Brief"）
//   Title:         記事のタイトル
//   URL:           記事のURL
//   PublishedAt:   公開日時（RFC3339形式、例: "2026-01-05T12:00:00Z"）
//   Excerpt:       記事の要約・プレビューテキスト
//   IsHeadline:    これが有料記事の見出しかどうかを示すフラグ
//
type Headline struct {
	Source      string `json:"source"`                // ソース名
	Title       string `json:"title"`                 // 記事タイトル
	URL         string `json:"url"`                   // 記事URL
	PublishedAt string `json:"publishedAt,omitempty"` // 公開日時（RFC3339形式）
	Excerpt     string `json:"excerpt,omitempty"`     // 要約テキスト
	IsHeadline  bool   `json:"isHeadline,omitempty"`  // 有料記事フラグ
}

// -----------------------------------------------------------------------------
// FreeArticle - 無料記事の情報
// -----------------------------------------------------------------------------
//
// RSS、WordPress API、またはHTMLスクレイピングから取得した無料記事を表します。
//
// 【使用場面】
//   - 無料ソースから直接収集した記事
//
type FreeArticle struct {
	Source      string `json:"source"`                // ソース名
	Title       string `json:"title"`                 // 記事タイトル
	URL         string `json:"url"`                   // 記事URL
	PublishedAt string `json:"publishedAt,omitempty"` // 公開日時
	Excerpt     string `json:"excerpt,omitempty"`     // 要約テキスト
}

// -----------------------------------------------------------------------------
// NotionHeadline - Notionデータベースから取得した見出し
// -----------------------------------------------------------------------------
//
// Notionデータベースに保存された記事情報を表します。
// メール送信機能でNotionから記事を取得する際に使用されます。
//
// 【使用場面】
//   - email.goでNotionから最近の記事を取得してメール本文を生成
//   - SendShortHeadlinesDigest()で50文字ヘッドラインメールを送信
//
type NotionHeadline struct {
	Title         string // 記事タイトル
	URL           string // 記事URL
	Source        string // ソース名
	AISummary     string // AIによる要約
	ShortHeadline string // 50文字ヘッドライン（Notion AIで生成）
	CreatedAt     string // 作成日時（RFC3339形式）
}

// -----------------------------------------------------------------------------
// WPPost - WordPress REST API レスポンス用構造体
// -----------------------------------------------------------------------------
//
// WordPress REST API（/wp-json/wp/v2/posts）から取得した記事データを表します。
// 複数のWordPressベースのニュースサイトで共通して使用されます。
//
// 【使用しているソース】
//   - CarbonCredits.jp
//   - Carbon Herald
//   - Climate Home News
//   - CarbonCredits.com
//   - Sandbag
//   - Ecosystem Marketplace
//   - Carbon Brief
//
// 【WordPress REST API について】
//   WordPressサイトには標準でREST APIが用意されており、
//   /wp-json/wp/v2/posts エンドポイントで記事一覧を取得できる
//
type WPPost struct {
	Title   struct{ Rendered string `json:"rendered"` } `json:"title"`    // 記事タイトル（HTMLエンコード済み）
	Link    string                                      `json:"link"`     // 記事URL
	Date    string                                      `json:"date"`     // 公開日時（ローカルタイムゾーン、非推奨）
	DateGMT string                                      `json:"date_gmt"` // 公開日時（UTC）
	Content struct{ Rendered string `json:"rendered"` } `json:"content"`  // 記事本文（HTML形式）
}
