// =============================================================================
// handlers.go - コマンドハンドラ
// =============================================================================
//
// このファイルはCLIコマンドの各ハンドラ関数を提供します。
//
// 【このファイルで提供する機能】
//   - handleEmailSend:          フルメールサマリー送信
//   - handleShortEmailSend:     50文字ヘッドラインダイジェスト送信
//   - handleListShortHeadlines: ShortHeadline診断表示
//
// 【共通ヘルパー関数】
//   - validateNotionEnv:    Notion環境変数の検証
//   - validateEmailEnv:     Email環境変数の検証
//   - createNotionClipper:  Notionクライアント作成
//   - fetchNotionHeadlines: Notionから記事取得
//
// =============================================================================
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// =============================================================================
// 環境変数バリデーション
// =============================================================================

// validateNotionEnv はNotion関連の環境変数を検証し、値を返す
//
// 【必要な環境変数】
//   - NOTION_TOKEN:       Notion API トークン
//   - NOTION_DATABASE_ID: NotionデータベースID
//
// エラー時はfatalf()で終了する
func validateNotionEnv() (token, dbID string) {
	token = os.Getenv("NOTION_TOKEN")
	dbID = os.Getenv("NOTION_DATABASE_ID")

	if token == "" {
		fatalf("ERROR: NOTION_TOKEN environment variable is required")
	}
	if dbID == "" {
		fatalf("ERROR: NOTION_DATABASE_ID environment variable is required (run with -notionClip first to create database)")
	}
	return token, dbID
}

// validateEmailEnv はEmail関連の環境変数を検証し、値を返す
//
// 【必要な環境変数】
//   - EMAIL_FROM:     送信元メールアドレス
//   - EMAIL_PASSWORD: Gmailアプリパスワード
//   - EMAIL_TO:       送信先メールアドレス
//
// エラー時はfatalf()で終了する
func validateEmailEnv() (from, password, to string) {
	from = os.Getenv("EMAIL_FROM")
	password = os.Getenv("EMAIL_PASSWORD")
	to = os.Getenv("EMAIL_TO")

	if from == "" {
		fatalf("ERROR: EMAIL_FROM environment variable is required for email sending")
	}
	if password == "" {
		fatalf("ERROR: EMAIL_PASSWORD environment variable is required (use Gmail App Password)")
	}
	if to == "" {
		fatalf("ERROR: EMAIL_TO environment variable is required")
	}
	return from, password, to
}

// =============================================================================
// 共通ヘルパー関数
// =============================================================================

// createNotionClipper はNotion環境変数を使用してNotionClipperを作成する
//
// 環境変数のバリデーションも行う
func createNotionClipper() *NotionClipper {
	token, dbID := validateNotionEnv()
	clipper, err := NewNotionClipper(token, dbID)
	if err != nil {
		fatalf("ERROR creating Notion clipper: %v", err)
	}
	return clipper
}

// fetchNotionHeadlines はNotionDBから最近の記事を取得する
//
// 記事が0件の場合は警告を表示してnilを返す
func fetchNotionHeadlines(clipper *NotionClipper, daysBack int) []NotionHeadline {
	ctx := context.Background()
	headlines, err := clipper.FetchRecentHeadlines(ctx, daysBack)
	if err != nil {
		fatalf("ERROR fetching headlines from Notion: %v", err)
	}

	if len(headlines) == 0 {
		fmt.Fprintf(os.Stderr, "⚠️  No headlines found in the last %d days\n", daysBack)
		fmt.Fprintln(os.Stderr, "========================================")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Fetched %d headlines from Notion (last %d days)\n", len(headlines), daysBack)
	return headlines
}

// createEmailSender はEmail環境変数を使用してEmailSenderを作成する
//
// 環境変数のバリデーションも行い、値も返す（表示用）
func createEmailSender() (*EmailSender, string, string) {
	from, password, to := validateEmailEnv()
	sender, err := NewEmailSender(from, password, to)
	if err != nil {
		fatalf("ERROR creating email sender: %v", err)
	}
	return sender, from, to
}

// =============================================================================
// メールハンドラ
// =============================================================================

// handleEmailSend はフルメールサマリーを送信する
//
// 【処理の流れ】
//  1. 環境変数をチェック（Notion + Email）
//  2. NotionDBから記事を取得
//  3. AI要約付きのフルメールを送信
func handleEmailSend(emailDaysBack int) {
	fmt.Fprintln(os.Stderr, "\n========================================")
	fmt.Fprintln(os.Stderr, "📧 Sending Email Summary")
	fmt.Fprintln(os.Stderr, "========================================")

	// Create Notion clipper and fetch headlines
	clipper := createNotionClipper()
	headlines := fetchNotionHeadlines(clipper, emailDaysBack)
	if headlines == nil {
		return
	}

	// Create email sender and send
	sender, from, to := createEmailSender()
	ctx := context.Background()
	if err := sender.SendHeadlinesSummary(ctx, headlines); err != nil {
		fatalf("ERROR sending email: %v", err)
	}

	fmt.Fprintln(os.Stderr, "✅ Email sent successfully")
	fmt.Fprintf(os.Stderr, "   From: %s\n", from)
	fmt.Fprintf(os.Stderr, "   To: %s\n", to)
	fmt.Fprintln(os.Stderr, "========================================")
}

// handleShortEmailSend は50文字ヘッドラインダイジェストメールを送信する
//
// 【処理の流れ】
//  1. 環境変数をチェック（Notion + Email）
//  2. NotionDBから記事を取得
//  3. カーボンキーワードでフィルタリング（email.go内で実行）
//  4. 50文字ヘッドライン + URLのメールを送信
func handleShortEmailSend(emailDaysBack int) {
	fmt.Fprintln(os.Stderr, "\n========================================")
	fmt.Fprintln(os.Stderr, "📧 Sending Short Headlines Digest")
	fmt.Fprintln(os.Stderr, "========================================")

	// Create Notion clipper and fetch headlines
	clipper := createNotionClipper()
	headlines := fetchNotionHeadlines(clipper, emailDaysBack)
	if headlines == nil {
		return
	}

	// Create email sender and send
	sender, from, to := createEmailSender()
	ctx := context.Background()
	if err := sender.SendShortHeadlinesDigest(ctx, headlines); err != nil {
		fatalf("ERROR sending email: %v", err)
	}

	fmt.Fprintln(os.Stderr, "✅ Short headlines digest email sent successfully")
	fmt.Fprintf(os.Stderr, "   From: %s\n", from)
	fmt.Fprintf(os.Stderr, "   To: %s\n", to)
	fmt.Fprintln(os.Stderr, "========================================")
}

// =============================================================================
// 診断ハンドラ
// =============================================================================

// handleListShortHeadlines はNotionDBのShortHeadline値を一覧表示する
//
// Notion AIによるフィルタリング結果を確認するための診断機能。
// ShortHeadlineの状態（要約あり、"-"、空）でグループ化して表示する。
func handleListShortHeadlines(emailDaysBack int) {
	fmt.Fprintln(os.Stderr, "\n========================================")
	fmt.Fprintln(os.Stderr, "📋 Listing ShortHeadline Values from NotionDB")
	fmt.Fprintln(os.Stderr, "========================================")

	// Create Notion clipper and fetch headlines
	clipper := createNotionClipper()
	headlines := fetchNotionHeadlines(clipper, emailDaysBack)
	if headlines == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "Found %d headlines (last %d days)\n\n", len(headlines), emailDaysBack)

	// Group by ShortHeadline status
	var withSummary, withDash, empty []NotionHeadline
	for _, h := range headlines {
		switch {
		case h.ShortHeadline == "":
			empty = append(empty, h)
		case h.ShortHeadline == "-" || h.ShortHeadline == "−" || h.ShortHeadline == "—":
			withDash = append(withDash, h)
		default:
			withSummary = append(withSummary, h)
		}
	}

	// Display statistics
	fmt.Fprintf(os.Stderr, "📊 Statistics:\n")
	fmt.Fprintf(os.Stderr, "   ✅ With Summary: %d\n", len(withSummary))
	fmt.Fprintf(os.Stderr, "   ❌ Filtered (-): %d\n", len(withDash))
	fmt.Fprintf(os.Stderr, "   ⏳ Empty:        %d\n", len(empty))
	fmt.Fprintln(os.Stderr, "")

	// Display headlines with summary
	if len(withSummary) > 0 {
		fmt.Fprintln(os.Stderr, "✅ Headlines with Summary:")
		fmt.Fprintln(os.Stderr, "----------------------------------------")
		for i, h := range withSummary {
			fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, h.Source)
			fmt.Fprintf(os.Stderr, "    Title: %s\n", truncateString(h.Title, 60))
			fmt.Fprintf(os.Stderr, "    ShortHeadline: %s\n", h.ShortHeadline)
			fmt.Fprintln(os.Stderr, "")
		}
	}

	// Display filtered headlines
	if len(withDash) > 0 {
		fmt.Fprintln(os.Stderr, "❌ Filtered Headlines (-):")
		fmt.Fprintln(os.Stderr, "----------------------------------------")
		for i, h := range withDash {
			fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, h.Source)
			fmt.Fprintf(os.Stderr, "    Title: %s\n", truncateString(h.Title, 60))
			fmt.Fprintln(os.Stderr, "")
		}
	}

	// Display empty headlines
	if len(empty) > 0 {
		fmt.Fprintln(os.Stderr, "⏳ Headlines without ShortHeadline (need Notion AI processing):")
		fmt.Fprintln(os.Stderr, "----------------------------------------")
		for i, h := range empty {
			fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, h.Source)
			fmt.Fprintf(os.Stderr, "    Title: %s\n", truncateString(h.Title, 60))
			fmt.Fprintln(os.Stderr, "")
		}
	}

	fmt.Fprintln(os.Stderr, "========================================")
}

// =============================================================================
// Notionハンドラ
// =============================================================================

// handleNotionClip は見出しと関連記事をNotionデータベースに保存する
//
// 【処理の流れ】
//  1. Notion環境変数を確認
//  2. 必要に応じて新規データベースを作成
//  3. 各見出しと関連記事をクリップ
func handleNotionClip(headlines []Headline, cfg *OutputConfig) {
	fmt.Fprintln(os.Stderr, "\n========================================")
	fmt.Fprintln(os.Stderr, "📎 Clipping to Notion Database")
	fmt.Fprintln(os.Stderr, "========================================")

	notionToken := os.Getenv("NOTION_TOKEN")
	if notionToken == "" {
		fatalf("NOTION_TOKEN environment variable is required for Notion integration")
	}

	clipper, err := NewNotionClipper(notionToken, cfg.NotionDatabaseID)
	if err != nil {
		fatalf("creating Notion clipper: %v", err)
	}

	ctx := context.Background()

	// 必要に応じてデータベースを作成
	if cfg.NotionDatabaseID == "" {
		if cfg.NotionPageID == "" {
			fatalf("-notionPageID is required when creating a new Notion database")
		}
		fmt.Fprintln(os.Stderr, "Creating new Notion database...")
		dbID, err := clipper.CreateDatabase(ctx, cfg.NotionPageID)
		if err != nil {
			fatalf("creating Notion database: %v", err)
		}

		// データベースIDを.envに保存
		if err := appendToEnvFile(".env", "NOTION_DATABASE_ID", dbID); err != nil {
			warnf("Failed to save database ID to .env: %v", err)
			fmt.Fprintf(os.Stderr, "Please manually add to .env:\nNOTION_DATABASE_ID=%s\n", dbID)
		} else {
			fmt.Fprintf(os.Stderr, "✅ Database ID saved to .env file\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "Using existing Notion database: %s\n", cfg.NotionDatabaseID)
	}

	// 各見出しをクリップ
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

// =============================================================================
// JSON出力ハンドラ
// =============================================================================

// handleJSONOutput は見出しをJSON形式で出力する
//
// cfg.OutFileが指定されている場合はファイルに、
// 指定されていない場合はstdoutに出力する
func handleJSONOutput(headlines []Headline, cfg *OutputConfig) {
	if cfg.OutFile != "" {
		if err := writeJSONFile(cfg.OutFile, headlines); err != nil {
			fatalf("writing output: %v", err)
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(headlines)
	}
}

// handleSaveFreePool は候補プールをファイルに保存する
func handleSaveFreePool(globalPool []FreeArticle, cfg *OutputConfig) {
	if cfg.SaveFree != "" {
		if err := writeJSONFile(cfg.SaveFree, globalPool); err != nil {
			fatalf("writing free pool: %v", err)
		}
	}
}
