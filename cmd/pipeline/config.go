// =============================================================================
// config.go - パイプライン設定
// =============================================================================
//
// このファイルはCLIフラグの解析と設定管理を行います。
//
// 【設定グループ】
//   - InputConfig:    入力ソース設定
//   - OutputConfig:   出力設定
//   - EmailConfig:    メール設定
//
// =============================================================================
package main

import (
	"flag"
	"os"
	"strings"
)

// =============================================================================
// 設定構造体
// =============================================================================

// PipelineConfig はパイプラインの全設定を保持する
type PipelineConfig struct {
	Input  InputConfig
	Output OutputConfig
	Email  EmailModeConfig
}

// InputConfig は入力ソースに関する設定
type InputConfig struct {
	// HeadlinesFile が指定された場合、スクレイピングせずにファイルから読み込む
	HeadlinesFile string

	// SourcesRaw はカンマ区切りのソース文字列（-sources フラグの値）
	SourcesRaw string

	// PerSource はソースあたりの最大記事数
	PerSource int

	// HoursBack は収集後に過去N時間以内の記事のみにフィルタ（0=フィルタなし）
	HoursBack int
}

// Sources はSourcesRawをパースしてスライスで返す
// "all-free" を指定すると全ソースに展開される
func (c *InputConfig) Sources() []string {
	var result []string
	for _, s := range strings.Split(c.SourcesRaw, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		// "all-free" は全ソースに展開
		if s == "all-free" {
			return strings.Split(defaultSources, ",")
		}
		result = append(result, s)
	}
	return result
}

// OutputConfig は出力に関する設定
type OutputConfig struct {
	// OutFile が指定された場合、ファイルに出力（空の場合はstdout）
	OutFile string

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
	// SendShortEmail がtrueの場合、50文字ダイジェストを送信
	SendShortEmail bool

	// ListShortHeadlines がtrueの場合、ShortHeadline診断を表示
	ListShortHeadlines bool

	// DaysBack はメール用の取得期間（日数）
	DaysBack int
}

// =============================================================================
// フラグ解析
// =============================================================================

// デフォルトソースリスト（全41アクティブソース）
// 2026-02-11更新: METI審議会を復帰（excerpt・日付抽出改善済み）
const defaultSources = "carboncredits.jp,carbonherald,climatehomenews,carboncredits.com,sandbag,ecosystem-marketplace,carbon-brief,rmi,icap,ieta,energy-monitor,world-bank,newclimate,carbon-knowledge-hub,carbon-market-watch,jri,env-ministry,meti,pwc-japan,mizuho-rt,jpx,politico-eu,euractiv,arxiv,oies,iopscience,nature-ecoevo,sciencedirect,verra,gold-standard,acr,car,iisd,climate-focus,eu-ets,uk-ets,carb,rggi,australia-cer,puro-earth,isometric"

// ParseFlags はCLIフラグを解析してPipelineConfigを返す
func ParseFlags() *PipelineConfig {
	cfg := &PipelineConfig{}

	// Input flags
	flag.StringVar(&cfg.Input.HeadlinesFile, "headlines", "", "optional: path to headlines.json; if empty, scrape from sources")
	flag.StringVar(&cfg.Input.SourcesRaw, "sources", defaultSources, "sources to scrape when --headlines is empty")
	flag.IntVar(&cfg.Input.PerSource, "perSource", 30, "max headlines to collect per source")
	flag.IntVar(&cfg.Input.HoursBack, "hoursBack", 0, "filter headlines to last N hours (0=no filter)")

	// Output flags
	flag.StringVar(&cfg.Output.OutFile, "out", "", "optional: write output JSON to this path (default: stdout)")
	flag.BoolVar(&cfg.Output.NotionClip, "notionClip", false, "clip articles to Notion database")
	flag.StringVar(&cfg.Output.NotionPageID, "notionPageID", os.Getenv("NOTION_PAGE_ID"), "parent page ID for creating new Notion database")
	flag.StringVar(&cfg.Output.NotionDatabaseID, "notionDatabaseID", os.Getenv("NOTION_DATABASE_ID"), "existing Notion database ID")

	// Email flags
	flag.BoolVar(&cfg.Email.SendShortEmail, "sendShortEmail", false, "send 50-char short headlines digest via email")
	flag.BoolVar(&cfg.Email.ListShortHeadlines, "listShortHeadlines", false, "list ShortHeadline values from NotionDB (diagnostic)")
	flag.IntVar(&cfg.Email.DaysBack, "emailDaysBack", 1, "fetch headlines from last N days for email")

	flag.Parse()
	return cfg
}

// IsEmailMode はメール関連モードかどうかを返す
func (c *PipelineConfig) IsEmailMode() bool {
	return c.Email.SendShortEmail || c.Email.ListShortHeadlines
}
