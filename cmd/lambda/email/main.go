// =============================================================================
// Lambda: send-email
// =============================================================================
//
// Notion DBから記事を取得し、メール送信するLambda関数
//
// 環境変数:
//   - NOTION_TOKEN:       Notion API Token (必須)
//   - NOTION_DATABASE_ID: NotionデータベースID (必須)
//   - EMAIL_FROM:         送信元メールアドレス (必須)
//   - EMAIL_PASSWORD:     Gmailアプリパスワード (必須)
//   - EMAIL_TO:           送信先メールアドレス (必須)
//   - DAYS_BACK:          取得期間（日数、デフォルト: 1）
//   - EMAIL_TYPE:         メールタイプ（full/short、デフォルト: full）
//
// =============================================================================
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"

	"carbon-relay/internal/pipeline"
)

// LambdaConfig は環境変数から読み込む設定
type LambdaConfig struct {
	NotionToken      string
	NotionDatabaseID string
	EmailFrom        string
	EmailPassword    string
	EmailTo          string
	DaysBack         int
	EmailType        string // "full" or "short"
}

// Response はLambdaレスポンス
type Response struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Fetched    int    `json:"fetched"`
	Sent       bool   `json:"sent"`
}

// Handler はLambdaのメインハンドラー
func Handler(ctx context.Context, event interface{}) (Response, error) {
	log.Println("Starting send-email Lambda...")

	// 1. 環境変数から設定を読み込む
	cfg := loadConfig()

	// 環境変数の検証
	if err := validateConfig(cfg); err != nil {
		return Response{StatusCode: 400, Message: err.Error()}, err
	}

	log.Printf("Config: daysBack=%d, emailType=%s", cfg.DaysBack, cfg.EmailType)

	// 2. Notionから記事を取得
	clipper, err := pipeline.NewNotionClipper(cfg.NotionToken, cfg.NotionDatabaseID)
	if err != nil {
		log.Printf("Error creating Notion clipper: %v", err)
		return Response{StatusCode: 500, Message: err.Error()}, err
	}

	headlines, err := clipper.FetchRecentHeadlines(ctx, cfg.DaysBack)
	if err != nil {
		log.Printf("Error fetching headlines from Notion: %v", err)
		return Response{StatusCode: 500, Message: err.Error()}, err
	}

	log.Printf("Fetched %d headlines from Notion (last %d days)", len(headlines), cfg.DaysBack)

	// 3. メール送信（0件でも送信する）
	sender, err := pipeline.NewEmailSender(cfg.EmailFrom, cfg.EmailPassword, cfg.EmailTo)
	if err != nil {
		log.Printf("Error creating email sender: %v", err)
		return Response{StatusCode: 500, Message: err.Error(), Fetched: len(headlines)}, err
	}

	var sendErr error
	if cfg.EmailType == "short" {
		sendErr = sender.SendShortHeadlinesDigest(ctx, headlines)
	} else {
		sendErr = sender.SendHeadlinesSummary(ctx, headlines)
	}

	if sendErr != nil {
		log.Printf("Error sending email: %v", sendErr)
		return Response{StatusCode: 500, Message: sendErr.Error(), Fetched: len(headlines)}, sendErr
	}

	log.Printf("Email sent successfully to %s", cfg.EmailTo)

	return Response{
		StatusCode: 200,
		Message:    fmt.Sprintf("Successfully sent %d headlines via email to %s", len(headlines), cfg.EmailTo),
		Fetched:    len(headlines),
		Sent:       true,
	}, nil
}

// loadConfig は環境変数から設定を読み込む
func loadConfig() LambdaConfig {
	daysBack := 1
	if db := os.Getenv("DAYS_BACK"); db != "" {
		if val, err := strconv.Atoi(db); err == nil && val > 0 {
			daysBack = val
		}
	}

	emailType := os.Getenv("EMAIL_TYPE")
	if emailType == "" {
		emailType = "full"
	}

	return LambdaConfig{
		NotionToken:      os.Getenv("NOTION_TOKEN"),
		NotionDatabaseID: os.Getenv("NOTION_DATABASE_ID"),
		EmailFrom:        os.Getenv("EMAIL_FROM"),
		EmailPassword:    os.Getenv("EMAIL_PASSWORD"),
		EmailTo:          os.Getenv("EMAIL_TO"),
		DaysBack:         daysBack,
		EmailType:        emailType,
	}
}

// validateConfig は設定の妥当性を検証する
func validateConfig(cfg LambdaConfig) error {
	if cfg.NotionToken == "" {
		return fmt.Errorf("NOTION_TOKEN is required")
	}
	if cfg.NotionDatabaseID == "" {
		return fmt.Errorf("NOTION_DATABASE_ID is required")
	}
	if cfg.EmailFrom == "" {
		return fmt.Errorf("EMAIL_FROM is required")
	}
	if cfg.EmailPassword == "" {
		return fmt.Errorf("EMAIL_PASSWORD is required")
	}
	if cfg.EmailTo == "" {
		return fmt.Errorf("EMAIL_TO is required")
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
