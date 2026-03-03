// =============================================================================
// main.go - Carbon Relay パイプラインのエントリーポイント
// =============================================================================
//
// このプログラムは、カーボンニュース収集・配信を自動化するCLIツールです。
//
// =============================================================================
// 【主な機能】
// =============================================================================
//
// 🟢 無料記事収集モード
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│ 目的:     複数のソースから記事を直接収集                         │
//	│ コスト:   無料                                                   │
//	│ 速度:     5-15秒                                                 │
//	│ 出力:     JSON、メール送信                                       │
//	│ コマンド: ./pipeline -sources=carbonherald -perSource=10        │
//	└─────────────────────────────────────────────────────────────────┘
//
// =============================================================================
// 【処理フロー】
// =============================================================================
//
//	┌─────────────┐    ┌─────────────┐    ┌─────────────┐
//	│  1. 設定    │ -> │  2. 収集    │ -> │  3. 出力    │
//	│  読み込み   │    │  スクレイピ │    │  JSON/Mail  │
//	└─────────────┘    └─────────────┘    └─────────────┘
//	       │                  │                  │
//	       v                  v                  v
//	.env読み込み        各ソースから      JSON出力 or
//	CLIフラグ解析       見出し収集        メール送信
//
// =============================================================================
// 【CLIフラグ一覧】
// =============================================================================
//
// ▼ 基本設定
//
//	-headlines       既存のJSONファイルから見出しを読み込む
//	-out             出力JSONファイルパス（省略時: stdout）
//	-sources         収集するソース（カンマ区切り）
//	-perSource       ソースあたりの最大記事数（デフォルト: 30）
//
// ▼ メール設定
//
//	-sendShortEmail  50文字ヘッドラインダイジェスト送信
//	-notionClip      Notionデータベースに保存
//
// =============================================================================
// 【初心者向けポイント】
// =============================================================================
//
// - flag パッケージでCLI引数を解析
// - godotenv パッケージで.envファイルを読み込み
// - エラーは標準エラー出力（os.Stderr）に出力
// - 処理の進捗も標準エラー出力に出力（stdoutはJSONのみ）
//
// =============================================================================
package main

import (
	"github.com/joho/godotenv" // .env ファイル読み込み
)

// main はパイプライン全体の制御フロー
//
// パイプライン処理の概要:
//  1. 各ソースから見出し収集
//  2. 結果をJSON出力またはメール送信
func main() {
	// .env ファイルから環境変数を読み込み
	// ファイルが存在しない場合はログを出力するが、処理は続行する
	if err := godotenv.Load(); err != nil {
		warnf(".env file not loaded: %v (using environment variables only)", err)
	}

	// CLIフラグを解析（config.goのParseFlags）
	cfg := ParseFlags()

	// --- メール専用モードの早期終了 ---
	if cfg.Email.SendShortEmail {
		handleShortEmailSend(cfg.Email.DaysBack)
		return
	}
	if cfg.Email.ListShortHeadlines {
		handleListShortHeadlines(cfg.Email.DaysBack)
		return
	}

	// --- 1) ヘッドラインの収集または読み込み ---
	var headlines []Headline
	var collectResult *CollectResult
	if cfg.Input.HeadlinesFile != "" {
		if err := readJSONFile(cfg.Input.HeadlinesFile, &headlines); err != nil {
			fatalf("reading headlines: %v", err)
		}
	} else {
		headlineCfg := defaultHeadlineConfig()
		result, err := CollectFromSources(cfg.Input.Sources(), cfg.Input.PerSource, headlineCfg)
		if err != nil {
			fatalf("collecting headlines: %v", err)
		}
		headlines = result.Headlines
		collectResult = result
	}

	if len(headlines) == 0 {
		// fatalf前にエラー通知を送る
		sendErrorNotification(collectResult, nil)
		fatalf("no headlines collected")
	}

	// --- 1.5) 時間指定フィルタリング ---
	if cfg.Input.HoursBack > 0 {
		headlines = FilterHeadlinesByHours(headlines, cfg.Input.HoursBack)
		if len(headlines) == 0 {
			fatalf("no headlines after filtering by %d hours", cfg.Input.HoursBack)
		}
	}

	// --- 2) 結果の出力 ---
	handleJSONOutput(headlines, &cfg.Output)

	// --- 3) Notionへのクリップ（有効な場合） ---
	var notionResult *NotionClipResult
	if cfg.Output.NotionClip {
		notionResult = handleNotionClip(headlines, &cfg.Output)
	}

	// --- 4) エラー通知（全処理完了後） ---
	sendErrorNotification(collectResult, notionResult)
}

// ハンドラは handlers.go で定義:
// - handleShortEmailSend（ショートメール送信）
// - handleListShortHeadlines（ショートヘッドライン一覧表示）
// - handleJSONOutput（JSON出力）
// - handleNotionClip（Notionクリップ）
