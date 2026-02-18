// =============================================================================
// config.go - パイプライン設定
// =============================================================================
//
// このファイルはCLIフラグの解析と設定管理を行います。
//
// 【設定グループ】
//   - InputConfig:    入力ソース設定
//   - SearchConfig:   OpenAI検索設定
//   - MatchingConfig: スコアリング設定
//   - OutputConfig:   出力設定
//   - EmailConfig:    メール設定
//
// =============================================================================
package pipeline

import (
	"flag"
	"strings"
)

// =============================================================================
// 設定構造体
// =============================================================================

// PipelineConfig はパイプラインの全設定を保持する
type PipelineConfig struct {
	Input    InputConfig
	Search   SearchConfig
	Matching MatchingConfig
	Output   OutputConfig
	Email    EmailModeConfig
}

// InputConfig は入力ソースに関する設定
type InputConfig struct {
	// HeadlinesFile が指定された場合、スクレイピングせずにファイルから読み込む
	HeadlinesFile string

	// SourcesRaw はカンマ区切りのソース文字列（-sources フラグの値）
	SourcesRaw string

	// PerSource はソースあたりの最大記事数
	PerSource int
}

// Sources はSourcesRawをパースしてスライスで返す
func (c *InputConfig) Sources() []string {
	var result []string
	for _, s := range strings.Split(c.SourcesRaw, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// SearchConfig はOpenAI検索に関する設定
type SearchConfig struct {
	// QueriesPerHeadline は見出しあたりのクエリ数（0で検索無効）
	QueriesPerHeadline int

	// SearchPerHeadline は見出しあたりの候補上限
	SearchPerHeadline int

	// ResultsPerQuery はクエリあたりの結果数
	ResultsPerQuery int

	// Provider は検索プロバイダ（"openai" | "brave"）
	Provider string

	// OpenAIModel は使用するOpenAIモデル
	OpenAIModel string

	// OpenAITool は使用するOpenAIツール（"web_search" | "web_search_preview"）
	OpenAITool string
}

// IsEnabled は検索が有効かどうかを返す
func (c *SearchConfig) IsEnabled() bool {
	return c.QueriesPerHeadline > 0
}

// MatchingConfig はスコアリングに関する設定
type MatchingConfig struct {
	// DaysBack は新しさの考慮期間（0で無効）
	DaysBack int

	// TopK は見出しあたりの関連記事上限
	TopK int

	// MinScore は最小スコア閾値
	MinScore float64

	// StrictMarket は市場シグナル一致を必須にするか
	StrictMarket bool
}

// OutputConfig は出力に関する設定
type OutputConfig struct {
	// OutFile が指定された場合、ファイルに出力（空の場合はstdout）
	OutFile string

	// SaveFree が指定された場合、候補プールをファイルに保存
	SaveFree string

	// NotionClip がtrueの場合、Notionに保存
	NotionClip bool

	// NotionPageID は新規データベース作成時の親ページID
	NotionPageID string

	// NotionDatabaseID は既存のデータベースID
	NotionDatabaseID string
}

// EmailModeConfig はメール送信モードに関する設定
//
// 【注意】email.goのEmailConfig（SMTP設定）とは別物
type EmailModeConfig struct {
	// SendEmail がtrueの場合、フルメールサマリーを送信
	SendEmail bool

	// SendShortEmail がtrueの場合、50文字ダイジェストを送信
	SendShortEmail bool

	// ListShortHeadlines がtrueの場合、Article Summary 300診断を表示
	ListShortHeadlines bool

	// DaysBack はメール用の取得期間（日数）
	DaysBack int
}

// =============================================================================
// フラグ解析
// =============================================================================

// DefaultSources はデフォルトソースリスト（全38アクティブソース）
// 2026-02-17更新: nature-ecoevo を停止（有料記事のため）
// 2026-02-18更新: env-ministry, meti を停止
const DefaultSources = "carboncredits.jp,carbonherald,climatehomenews,carboncredits.com,sandbag,ecosystem-marketplace,carbon-brief,rmi,icap,ieta,energy-monitor,world-bank,newclimate,carbon-knowledge-hub,carbon-market-watch,jri,pwc-japan,mizuho-rt,jpx,politico-eu,euractiv,arxiv,oies,iopscience,sciencedirect,verra,gold-standard,acr,car,iisd,climate-focus,eu-ets,uk-ets,carb,rggi,australia-cer,puro-earth,isometric"

// ParseFlags はCLIフラグを解析してPipelineConfigを返す
func ParseFlags() *PipelineConfig {
	cfg := &PipelineConfig{}

	// Input flags
	flag.StringVar(&cfg.Input.HeadlinesFile, "headlines", "", "optional: path to headlines.json; if empty, scrape from sources")
	flag.StringVar(&cfg.Input.SourcesRaw, "sources", DefaultSources, "sources to scrape when --headlines is empty")
	flag.IntVar(&cfg.Input.PerSource, "perSource", 30, "max headlines to collect per source")

	// Search flags
	flag.IntVar(&cfg.Search.SearchPerHeadline, "searchPerHeadline", 25, "max candidate results kept per headline")
	flag.IntVar(&cfg.Search.QueriesPerHeadline, "queriesPerHeadline", 3, "max queries to issue per headline")
	flag.IntVar(&cfg.Search.ResultsPerQuery, "resultsPerQuery", 10, "results per query")
	flag.StringVar(&cfg.Search.Provider, "searchProvider", "openai", "search provider: openai|brave")
	flag.StringVar(&cfg.Search.OpenAIModel, "openaiModel", "gpt-4o-mini", "OpenAI model to use")
	flag.StringVar(&cfg.Search.OpenAITool, "openaiTool", "web_search", "OpenAI tool type: web_search|web_search_preview")

	// Matching flags
	flag.IntVar(&cfg.Matching.DaysBack, "daysBack", 60, "recency window in days (0 disables)")
	flag.IntVar(&cfg.Matching.TopK, "topK", 3, "max relatedFree per headline")
	flag.Float64Var(&cfg.Matching.MinScore, "minScore", 0.32, "minimum score threshold")
	flag.BoolVar(&cfg.Matching.StrictMarket, "strictMarket", true, "require market match if headline has market signal")

	// Output flags
	flag.StringVar(&cfg.Output.OutFile, "out", "", "optional: write matched output JSON to this path (default: stdout)")
	flag.StringVar(&cfg.Output.SaveFree, "saveFree", "", "optional: write pooled free candidates to file")
	flag.BoolVar(&cfg.Output.NotionClip, "notionClip", false, "clip articles to Notion database")
	flag.StringVar(&cfg.Output.NotionPageID, "notionPageID", "", "parent page ID for creating new Notion database (required for new DB)")
	flag.StringVar(&cfg.Output.NotionDatabaseID, "notionDatabaseID", "", "existing Notion database ID (optional, will create new if empty)")

	// Email flags
	flag.BoolVar(&cfg.Email.SendEmail, "sendEmail", false, "send headlines summary via email")
	flag.BoolVar(&cfg.Email.SendShortEmail, "sendShortEmail", false, "send 50-char short headlines digest via email")
	flag.BoolVar(&cfg.Email.ListShortHeadlines, "listShortHeadlines", false, "list Article Summary 300 values from NotionDB (diagnostic)")
	flag.IntVar(&cfg.Email.DaysBack, "emailDaysBack", 1, "fetch headlines from last N days for email")

	flag.Parse()
	return cfg
}

// IsEmailMode はメール関連モードかどうかを返す
func (c *PipelineConfig) IsEmailMode() bool {
	return c.Email.SendEmail || c.Email.SendShortEmail || c.Email.ListShortHeadlines
}
