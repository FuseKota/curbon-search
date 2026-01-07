// =============================================================================
// main.go - Carbon Relay パイプラインのエントリーポイント
// =============================================================================
//
// このプログラムは、カーボンニュース収集・分析・配信を自動化するCLIツールです。
//
// =============================================================================
// 【2つの運用モード】
// =============================================================================
//
// 🟢 モード1: 無料記事収集モード（-queriesPerHeadline=0）
//    ┌─────────────────────────────────────────────────────────────────┐
//    │ 目的:     16の無料ソースから記事を直接収集                       │
//    │ コスト:   OpenAI API不要（無料）                                 │
//    │ 速度:     5-15秒                                                 │
//    │ 出力:     JSON、メール送信                                       │
//    │ コマンド: ./pipeline -sources=all-free -perSource=10            │
//    │           -queriesPerHeadline=0 -sendEmail                       │
//    └─────────────────────────────────────────────────────────────────┘
//
// 🔵 モード2: 有料記事マッチングモード（-queriesPerHeadline>0）
//    ┌─────────────────────────────────────────────────────────────────┐
//    │ 目的:     有料記事のヘッドラインから関連無料記事を検索           │
//    │ コスト:   OpenAI API使用（有料）                                 │
//    │ 速度:     1-5分                                                  │
//    │ 出力:     JSON、Notionデータベース                               │
//    │ コマンド: ./pipeline -sources=carbonpulse,qci -perSource=5      │
//    │           -queriesPerHeadline=3 -notionClip                      │
//    └─────────────────────────────────────────────────────────────────┘
//
// =============================================================================
// 【処理フロー】
// =============================================================================
//
//   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
//   │  1. 設定    │ -> │  2. 収集    │ -> │  3. 検索    │
//   │  読み込み   │    │  スクレイピ │    │  OpenAI API │
//   └─────────────┘    └─────────────┘    └─────────────┘
//          │                  │                  │
//          v                  v                  v
//   .env読み込み        18ソースから      各見出しに対して
//   CLIフラグ解析       見出し収集         Web検索実行
//
//   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
//   │  4. マッチ  │ -> │  5. 出力    │ -> │  6. 配信    │
//   │  スコアリング│    │  JSON生成   │    │  Notion/Mail│
//   └─────────────┘    └─────────────┘    └─────────────┘
//          │                  │                  │
//          v                  v                  v
//   IDF重み計算         結果をJSON化       Notion保存 or
//   候補をランキング    ファイル/stdout    メール送信
//
// =============================================================================
// 【CLIフラグ一覧】
// =============================================================================
//
// ▼ 基本設定
//   -headlines       既存のJSONファイルから見出しを読み込む
//   -out             出力JSONファイルパス（省略時: stdout）
//   -sources         収集するソース（カンマ区切り）
//   -perSource       ソースあたりの最大記事数（デフォルト: 30）
//
// ▼ 検索設定
//   -queriesPerHeadline  見出しあたりのクエリ数（デフォルト: 3、0で無効）
//   -searchPerHeadline   見出しあたりの候補上限（デフォルト: 25）
//   -resultsPerQuery     クエリあたりの結果数（デフォルト: 10）
//
// ▼ マッチング設定
//   -daysBack        新しさの考慮期間（デフォルト: 60日）
//   -topK            見出しあたりの関連記事上限（デフォルト: 3）
//   -minScore        最小スコア閾値（デフォルト: 0.32）
//   -strictMarket    市場シグナル一致を必須にする（デフォルト: true）
//
// ▼ 出力設定
//   -notionClip      Notionデータベースに保存
//   -sendEmail       メール送信モード
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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv" // .env ファイル読み込み
)

// main はパイプライン全体の制御フロー
//
// パイプライン処理の概要:
//   1. 有料ソース（Carbon Pulse / QCI）の無料ページから見出し収集
//   2. 各見出しに対してOpenAI Web検索を実行し、関連する無料/一次情報源を発見
//   3. IDF（逆文書頻度）ベースでスコアリングし、relatedFree リンクを付与
//   4. 結果をJSON出力、Notionクリップ、またはメール送信
func main() {
	// .env ファイルから環境変数を読み込み
	// ファイルが存在しない場合はログを出力するが、処理は続行する
	if err := godotenv.Load(); err != nil {
		warnf(".env file not loaded: %v (using environment variables only)", err)
	}

	// CLIフラグを解析（config.goのParseFlags）
	cfg := ParseFlags()

	// --- Early exit for email-only modes ---
	if cfg.Email.SendEmail {
		handleEmailSend(cfg.Email.DaysBack)
		return
	}
	if cfg.Email.SendShortEmail {
		handleShortEmailSend(cfg.Email.DaysBack)
		return
	}
	if cfg.Email.ListShortHeadlines {
		handleListShortHeadlines(cfg.Email.DaysBack)
		return
	}

	// OpenAI API key check (only if search is enabled)
	if cfg.Search.IsEnabled() && os.Getenv("OPENAI_API_KEY") == "" {
		errorf("set OPENAI_API_KEY (OpenAI API key) in your environment")
		infof("To skip search and only collect headlines, use -queriesPerHeadline=0")
		os.Exit(1)
	}

	// --- 1) Collect or read headlines ---
	var headlines []Headline
	if cfg.Input.HeadlinesFile != "" {
		if err := readJSONFile(cfg.Input.HeadlinesFile, &headlines); err != nil {
			fatalf("reading headlines: %v", err)
		}
	} else {
		headlineCfg := defaultHeadlineConfig()

		// ソースレジストリを使用して収集（headlines.goのCollectFromSourcesを呼び出し）
		var err error
		headlines, err = CollectFromSources(cfg.Input.Sources(), cfg.Input.PerSource, headlineCfg)
		if err != nil {
			fatalf("collecting headlines: %v", err)
		}
	}

	if len(headlines) == 0 {
		fatalf("no headlines collected")
	}

	// --- 2) For each headline, perform web search ---
	now := time.Now()
	candsByIdx := make([][]FreeArticle, len(headlines))
	globalSeen := map[string]bool{}
	globalPool := make([]FreeArticle, 0, len(headlines)*cfg.Search.SearchPerHeadline)

	if !cfg.Search.IsEnabled() {
		infof("Search disabled (queriesPerHeadline=0), skipping web search phase")
	}

	for i, h := range headlines {
		queries := h.SearchQueries
		if len(queries) == 0 {
			queries = buildSearchQueries(h.Title, h.Excerpt)
		}
		if len(queries) > cfg.Search.QueriesPerHeadline {
			queries = queries[:cfg.Search.QueriesPerHeadline]
		}

		merged := map[string]FreeArticle{}
		for _, q := range queries {
			var res []FreeArticle
			var err error

			switch cfg.Search.Provider {
			case "openai":
				res, err = openaiWebSearch(q, cfg.Search.ResultsPerQuery, cfg.Search.OpenAIModel, cfg.Search.OpenAITool)
			default:
				err = fmt.Errorf("unsupported searchProvider: %s", cfg.Search.Provider)
			}

			if err != nil {
				warnf("search: %v", err)
				continue
			}
			for _, a := range res {
				if a.URL == "" || a.Title == "" {
					continue
				}
				merged[a.URL] = a
				if len(merged) >= cfg.Search.SearchPerHeadline {
					break
				}
			}
			if len(merged) >= cfg.Search.SearchPerHeadline {
				break
			}
		}

		// flatten and dedupe
		cands := make([]FreeArticle, 0, len(merged))
		for _, a := range merged {
			cands = append(cands, a)
			if !globalSeen[a.URL] {
				globalSeen[a.URL] = true
				globalPool = append(globalPool, a)
			}
		}
		candsByIdx[i] = cands
	}

	// --- 3) Build IDF corpus (headlines + all candidates) ---
	docs := make([][]string, 0, len(headlines)+len(globalPool))
	for _, h := range headlines {
		docs = append(docs, tokenize(h.Title))
	}
	for _, a := range globalPool {
		docs = append(docs, tokenize(a.Title))
	}
	idf := buildIDF(docs)

	// --- 4) Match / score ---
	for i := range headlines {
		headlines[i].IsHeadline = true
		headlines[i].SearchQueries = nil // compact output
		headlines[i].RelatedFree = topKRelated(
			headlines[i],
			candsByIdx[i],
			idf,
			now,
			cfg.Matching.DaysBack,
			cfg.Matching.StrictMarket,
			cfg.Matching.TopK,
			cfg.Matching.MinScore,
		)
	}

	// --- 5) Save results ---
	if cfg.Output.SaveFree != "" {
		if err := writeJSONFile(cfg.Output.SaveFree, globalPool); err != nil {
			fatalf("writing free pool: %v", err)
		}
	}

	if cfg.Output.OutFile != "" {
		if err := writeJSONFile(cfg.Output.OutFile, headlines); err != nil {
			fatalf("writing output: %v", err)
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(headlines)
	}

	// --- 6) Clip to Notion (if enabled) ---
	if cfg.Output.NotionClip {
		fmt.Fprintln(os.Stderr, "\n========================================")
		fmt.Fprintln(os.Stderr, "📎 Clipping to Notion Database")
		fmt.Fprintln(os.Stderr, "========================================")

		notionToken := os.Getenv("NOTION_TOKEN")
		if notionToken == "" {
			fatalf("NOTION_TOKEN environment variable is required for Notion integration")
		}

		clipper, err := NewNotionClipper(notionToken, cfg.Output.NotionDatabaseID)
		if err != nil {
			fatalf("creating Notion clipper: %v", err)
		}

		ctx := context.Background()

		// Create database if needed
		if cfg.Output.NotionDatabaseID == "" {
			if cfg.Output.NotionPageID == "" {
				fatalf("-notionPageID is required when creating a new Notion database")
			}
			fmt.Fprintln(os.Stderr, "Creating new Notion database...")
			dbID, err := clipper.CreateDatabase(ctx, cfg.Output.NotionPageID)
			if err != nil {
				fatalf("creating Notion database: %v", err)
			}

			// Save database ID to .env file for future use
			if err := appendToEnvFile(".env", "NOTION_DATABASE_ID", dbID); err != nil {
				warnf("Failed to save database ID to .env: %v", err)
				fmt.Fprintf(os.Stderr, "Please manually add to .env:\nNOTION_DATABASE_ID=%s\n", dbID)
			} else {
				fmt.Fprintf(os.Stderr, "✅ Database ID saved to .env file\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Using existing Notion database: %s\n", cfg.Output.NotionDatabaseID)
		}

		// Clip all headlines and their related articles
		fmt.Fprintln(os.Stderr, "\nClipping articles...")
		clippedCount := 0
		for _, h := range headlines {
			if err := clipper.ClipHeadlineWithRelated(ctx, h); err != nil {
				warnf("failed to clip headline '%s': %v", h.Title, err)
				continue
			}
			clippedCount++
			fmt.Fprintf(os.Stderr, "  ✅ Clipped: %s (%d related articles)\n", h.Title, len(h.RelatedFree))
		}

		fmt.Fprintln(os.Stderr, "========================================")
		fmt.Fprintf(os.Stderr, "✅ Clipped %d headlines to Notion\n", clippedCount)
		fmt.Fprintln(os.Stderr, "========================================")
	}
}

// Handlers are defined in handlers.go:
// - handleEmailSend
// - handleShortEmailSend
// - handleListShortHeadlines
