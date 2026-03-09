// =============================================================================
// main.go - Carbon Relay パイプラインのエントリーポイント
// =============================================================================
//
// このプログラムは、カーボンニュース収集・配信を自動化するCLIツールです。
// ロジックは internal/pipeline パッケージに集約されており、
// このファイルは .env 読み込みとフラグ解析のみを行う薄いエントリーポイントです。
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
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"carbon-relay/internal/pipeline"

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
		fmt.Fprintf(os.Stderr, "WARN: .env file not loaded: %v (using environment variables only)\n", err)
	}

	// CLIフラグを解析
	cfg := pipeline.ParseFlags()

	// --- メール専用モードの早期終了 ---
	if cfg.Email.SendShortEmail {
		pipeline.HandleShortEmailSend(cfg.Email.DaysBack)
		return
	}
	if cfg.Email.ListShortHeadlines {
		pipeline.HandleListShortHeadlines(cfg.Email.DaysBack)
		return
	}

	// --- 1) ヘッドラインの収集または読み込み ---
	var headlines []pipeline.Headline
	var collectResult *pipeline.CollectResult
	if cfg.Input.HeadlinesFile != "" {
		if err := readJSONFile(cfg.Input.HeadlinesFile, &headlines); err != nil {
			fatalf("reading headlines: %v", err)
		}
	} else {
		headlineCfg := pipeline.DefaultHeadlineConfig()
		result, err := pipeline.CollectFromSources(cfg.Input.Sources(), cfg.Input.PerSource, headlineCfg)
		if err != nil {
			fatalf("collecting headlines: %v", err)
		}
		headlines = result.Headlines
		collectResult = result
	}

	if len(headlines) == 0 {
		// fatalf前にエラー通知を送る
		pipeline.SendErrorNotification(collectResult, nil)
		fatalf("no headlines collected")
	}

	// --- 1.5) 時間指定フィルタリング ---
	if cfg.Input.HoursBack > 0 {
		headlines = pipeline.FilterHeadlinesByHours(headlines, cfg.Input.HoursBack)
		if len(headlines) == 0 {
			fatalf("no headlines after filtering by %d hours", cfg.Input.HoursBack)
		}
	}

	// --- 2) 結果の出力 ---
	pipeline.HandleJSONOutput(headlines, &cfg.Output)

	// --- 3) Notionへのクリップ（有効な場合） ---
	var notionResult *pipeline.NotionClipResult
	if cfg.Output.NotionClip {
		notionResult = pipeline.HandleNotionClip(headlines, &cfg.Output)
	}

	// --- 4) エラー通知（全処理完了後） ---
	pipeline.SendErrorNotification(collectResult, notionResult)
}

// =============================================================================
// CLI専用ヘルパー関数
// =============================================================================

// readJSONFile はJSONファイルを読み込んで指定した型に変換する
func readJSONFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// fatalf はエラーメッセージを出力してプログラムを終了する
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
