// =============================================================================
// Lambda: 例外ヘッドライン収集
// =============================================================================
//
// タイミング問題・レート制限の影響を受けるソース専用のLambda関数
//
// 対象ソース:
//   - RGGI:      UTC正午ごろ公開 → 朝UTC実行では「未来記事」として除外される
//   - JRI:       UTC 15:00ごろ公開 → 同上
//   - arXiv:     IPベースのレート制限（429）が厳しい
//   - IISD ENB:  403レスポンスが頻発（リトライあり）
//
// 環境変数:
//   - NOTION_TOKEN:       Notion API Token (必須)
//   - NOTION_DATABASE_ID: NotionデータベースID (必須)
//   - SOURCES:            収集するソース (デフォルト: rggi,jri,arxiv,iisd)
//   - PER_SOURCE:         ソースあたりの記事数 (デフォルト: 100)
//   - HOURS_BACK:         何時間以内の記事を取得するか (デフォルト: 48、0=フィルタなし)
//   - EMAIL_FROM:         エラー通知メール送信元 (任意)
//   - EMAIL_PASSWORD:     Gmailアプリパスワード (任意)
//   - EMAIL_TO:           エラー通知メール送信先 (任意)
//
// =============================================================================
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"carbon-relay/internal/pipeline"
)

// LambdaConfig は環境変数から読み込む設定
type LambdaConfig struct {
	Sources          string
	PerSource        int
	HoursBack        int // 何時間以内の記事を取得するか（0=フィルタなし）
	NotionToken      string
	NotionDatabaseID string
	EmailFrom        string // エラー通知用（任意）
	EmailPassword    string // エラー通知用（任意）
	EmailTo          string // エラー通知用（任意）
}

// Response はLambdaレスポンス
type Response struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Collected  int    `json:"collected"`
	Clipped    int    `json:"clipped"`
}

// Handler はLambdaのメインハンドラー
func Handler(ctx context.Context, event interface{}) (Response, error) {
	log.Println("Starting collect-exception Lambda...")

	// 1. 環境変数から設定を読み込む
	cfg := loadConfig()

	// 環境変数の検証
	if cfg.NotionToken == "" {
		return Response{StatusCode: 400, Message: "NOTION_TOKEN is required"}, fmt.Errorf("NOTION_TOKEN is required")
	}
	if cfg.NotionDatabaseID == "" {
		return Response{StatusCode: 400, Message: "NOTION_DATABASE_ID is required"}, fmt.Errorf("NOTION_DATABASE_ID is required")
	}

	log.Printf("Config: sources=%s, perSource=%d, hoursBack=%d", cfg.Sources, cfg.PerSource, cfg.HoursBack)

	// 2. 記事を収集
	sources := parseSources(cfg.Sources)
	headlineCfg := pipeline.DefaultHeadlineConfig()

	result, err := pipeline.CollectFromSources(sources, cfg.PerSource, headlineCfg)
	if err != nil {
		log.Printf("Error collecting headlines: %v", err)
		return Response{StatusCode: 500, Message: err.Error()}, err
	}
	headlines := result.Headlines

	// エラーがあればログに記録し、メールで通知
	if len(result.Errors) > 0 {
		log.Printf("WARNING: %d source(s) failed:", len(result.Errors))
		for _, e := range result.Errors {
			log.Printf("  %s", e)
		}
		sendErrorNotification(cfg, result.Errors, len(headlines))
	}

	log.Printf("Collected %d headlines (before time filter)", len(headlines))

	// 3. 時間フィルタリング（HOURS_BACK > 0 の場合のみ）
	if cfg.HoursBack > 0 {
		headlines = pipeline.FilterHeadlinesByHours(headlines, cfg.HoursBack)
		log.Printf("After time filter: %d headlines (last %d hours)", len(headlines), cfg.HoursBack)
	}

	if len(headlines) == 0 {
		return Response{
			StatusCode: 200,
			Message:    "No headlines collected",
			Collected:  0,
			Clipped:    0,
		}, nil
	}

	// 4. Notionに保存
	clipper, err := pipeline.NewNotionClipper(cfg.NotionToken, cfg.NotionDatabaseID)
	if err != nil {
		log.Printf("Error creating Notion clipper: %v", err)
		return Response{StatusCode: 500, Message: err.Error(), Collected: len(headlines)}, err
	}

	clipped := 0
	for _, h := range headlines {
		if err := clipper.ClipHeadlineWithRelated(ctx, h); err != nil {
			log.Printf("Warning: failed to clip headline '%s': %v", h.Title, err)
			continue
		}
		clipped++
	}

	log.Printf("Clipped %d headlines to Notion", clipped)

	return Response{
		StatusCode: 200,
		Message:    fmt.Sprintf("Successfully collected %d headlines, clipped %d to Notion", len(headlines), clipped),
		Collected:  len(headlines),
		Clipped:    clipped,
	}, nil
}

// loadConfig は環境変数から設定を読み込む
func loadConfig() LambdaConfig {
	perSource := 100 // デフォルト: 100件（WordPress APIの上限）
	if ps := os.Getenv("PER_SOURCE"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 {
			perSource = val
		}
	}

	hoursBack := 48 // デフォルト: 過去48時間（タイミングずれを吸収するため余裕を持たせる）
	if hb := os.Getenv("HOURS_BACK"); hb != "" {
		if val, err := strconv.Atoi(hb); err == nil && val >= 0 {
			hoursBack = val
		}
	}

	sources := os.Getenv("SOURCES")
	if sources == "" {
		sources = pipeline.ExceptionSources
	}

	return LambdaConfig{
		Sources:          sources,
		PerSource:        perSource,
		HoursBack:        hoursBack,
		NotionToken:      os.Getenv("NOTION_TOKEN"),
		NotionDatabaseID: os.Getenv("NOTION_DATABASE_ID"),
		EmailFrom:        os.Getenv("EMAIL_FROM"),
		EmailPassword:    os.Getenv("EMAIL_PASSWORD"),
		EmailTo:          os.Getenv("EMAIL_TO"),
	}
}

// parseSources はソース文字列をパースしてスライスで返す
func parseSources(sourcesRaw string) []string {
	var result []string
	for _, s := range strings.Split(sourcesRaw, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		result = append(result, s)
	}
	return result
}

// sendErrorNotification はエラー通知メールを送信する
// EMAIL_FROM, EMAIL_PASSWORD, EMAIL_TO が設定されている場合のみ送信
func sendErrorNotification(cfg LambdaConfig, errors []string, headlineCount int) {
	if cfg.EmailFrom == "" || cfg.EmailPassword == "" || cfg.EmailTo == "" {
		log.Println("Email env vars not set, skipping error notification email")
		return
	}

	sender, err := pipeline.NewEmailSender(cfg.EmailFrom, cfg.EmailPassword, cfg.EmailTo)
	if err != nil {
		log.Printf("Failed to create email sender: %v", err)
		return
	}

	subject := fmt.Sprintf("[Carbon Relay] collect-exception: %d source(s) failed - %s",
		len(errors), time.Now().Format("2006-01-02 15:04"))

	var body strings.Builder
	body.WriteString("Carbon Relay collect-exception source collection errors:\n\n")
	for _, e := range errors {
		body.WriteString("  " + e + "\n")
	}
	body.WriteString(fmt.Sprintf("\nSuccessfully collected: %d headlines\n", headlineCount))
	body.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))

	msg := sender.BuildEmailMessage(subject, body.String())
	if err := sender.SendWithRetry(msg); err != nil {
		log.Printf("Failed to send error notification email: %v", err)
	} else {
		log.Println("Error notification email sent")
	}
}

func main() {
	lambda.Start(Handler)
}
