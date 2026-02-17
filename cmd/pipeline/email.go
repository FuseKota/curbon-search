// =============================================================================
// email.go - メール送信モジュール
// =============================================================================
//
// このファイルはGmail SMTPを使用したメール送信機能を提供します。
// Carbon Relayのモード1（無料記事収集）で、ニュースレター配信に使用されます。
//
// =============================================================================
// 【処理の流れ】
// =============================================================================
//
// 1. Notionデータベースから最近の記事を取得
// 2. プレーンテキスト形式のメール本文を生成
// 3. RFC 5322準拠のメールメッセージを構築
// 4. Gmail SMTP経由で送信（リトライ付き）
//
// =============================================================================
// 【必要な環境変数】
// =============================================================================
//
//	EMAIL_FROM     - 送信元メールアドレス（Gmail）
//	EMAIL_PASSWORD - Gmailアプリパスワード（通常のパスワードではない！）
//	EMAIL_TO       - 送信先メールアドレス（カンマ区切りで複数可）
//
// =============================================================================
// 【Gmailアプリパスワードについて】
// =============================================================================
//
// Googleアカウントの2段階認証を有効にした上で、
// 「アプリパスワード」を生成する必要があります。
//
// 生成方法:
//  1. https://myaccount.google.com/security にアクセス
//  2. 「2段階認証プロセス」を有効化
//  3. 「アプリパスワード」を選択
//  4. 「メール」と「その他（カスタム名）」を選択
//  5. 生成された16文字のパスワードをEMAIL_PASSWORDに設定
//
// =============================================================================
// 【初心者向けポイント】
// =============================================================================
//
// - SMTPはメール送信のための標準プロトコル
// - Gmail SMTPはポート587（TLS）を使用
// - 指数バックオフ: 失敗時に2秒→4秒→8秒と待機時間を増やしてリトライ
// - RFC 5322: メールフォーマットの標準規格
//
// =============================================================================
package main

import (
	"context"
	"fmt"
	"math"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// =============================================================================
// 設定・構造体
// =============================================================================

// EmailConfig はメール送信の設定を保持する
type EmailConfig struct {
	From     string   // 送信元メールアドレス
	Password string   // Gmailアプリパスワード
	To       []string // 送信先メールアドレス（複数可）
	SMTPHost string   // SMTPサーバーホスト（"smtp.gmail.com"）
	SMTPPort string   // SMTPポート（"587"）
}

// EmailSender はメール送信を担当する
type EmailSender struct {
	config EmailConfig
}

// =============================================================================
// 初期化
// =============================================================================

// NewEmailSender は新しいメール送信者を作成する
//
// 引数:
//
//	from:     送信元メールアドレス
//	password: Gmailアプリパスワード
//	to:       送信先メールアドレス（カンマ区切りで複数可）
//
// 【注意】通常のGmailパスワードは使用できません。
// 必ずアプリパスワードを使用してください。
func NewEmailSender(from, password, to string) (*EmailSender, error) {
	// 必須パラメータのチェック
	if from == "" {
		return nil, fmt.Errorf("EMAIL_FROM is required")
	}
	if password == "" {
		return nil, fmt.Errorf("EMAIL_PASSWORD is required (use Gmail App Password)")
	}
	if to == "" {
		return nil, fmt.Errorf("EMAIL_TO is required")
	}

	// カンマ区切りのメールアドレスを分割
	toList := strings.Split(to, ",")
	for i, addr := range toList {
		toList[i] = strings.TrimSpace(addr)
	}

	return &EmailSender{
		config: EmailConfig{
			From:     from,
			Password: password,
			To:       toList,
			SMTPHost: "smtp.gmail.com",
			SMTPPort: "587", // TLSポート
		},
	}, nil
}


// =============================================================================
// メールメッセージ構築
// =============================================================================

// buildEmailMessage はRFC 5322準拠のメールメッセージを構築する
//
// 【RFC 5322フォーマット】
//
//	From: sender@example.com\r\n
//	To: recipient@example.com\r\n
//	Subject: メール件名\r\n
//	Content-Type: text/plain; charset=UTF-8\r\n
//	\r\n
//	メール本文...
//
// 注意: ヘッダーと本文は空行（\r\n）で区切る
func (es *EmailSender) buildEmailMessage(subject, body string) []byte {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", es.config.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(es.config.To, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n") // ヘッダーと本文の区切り
	msg.WriteString(body)

	return []byte(msg.String())
}

// =============================================================================
// 送信（リトライ付き）
// =============================================================================

// sendWithRetry は指数バックオフでリトライしながらメールを送信する
//
// 【指数バックオフとは】
//
//	失敗するたびに待機時間を2倍にしていく方式
//	1回目失敗: 2秒待機
//	2回目失敗: 4秒待機
//	3回目失敗: 8秒待機
//
// これにより、一時的なネットワーク障害やサーバー過負荷に対応できる
func (es *EmailSender) sendWithRetry(msg []byte) error {
	maxRetries := 3 // 最大リトライ回数
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// 指数バックオフ: 2^i 秒待機
			wait := time.Duration(math.Pow(2, float64(i))) * time.Second
			fmt.Fprintf(os.Stderr, "Retrying email send in %v...\n", wait)
			time.Sleep(wait)
		}

		// 送信を試行
		err := es.send(msg)
		if err == nil {
			return nil // 成功
		}

		lastErr = err
		warnf("Email send failed (attempt %d/%d): %v", i+1, maxRetries, err)
	}

	return fmt.Errorf("failed to send email after %d retries: %w", maxRetries, lastErr)
}

// send はGmail SMTPを使用してメールを送信する
//
// 【SMTP認証】
//
//	PLAIN認証を使用（ユーザー名とパスワードを送信）
//	TLS（ポート587）で暗号化されるため安全
func (es *EmailSender) send(msg []byte) error {
	// PLAIN認証を設定
	auth := smtp.PlainAuth("", es.config.From, es.config.Password, es.config.SMTPHost)

	// SMTPサーバーアドレス
	addr := es.config.SMTPHost + ":" + es.config.SMTPPort

	// メール送信
	err := smtp.SendMail(addr, auth, es.config.From, es.config.To, msg)
	if err != nil {
		return fmt.Errorf("SMTP send failed: %w (check EMAIL_PASSWORD is a Gmail App Password)", err)
	}

	return nil
}

// =============================================================================
// 50文字ヘッドラインメール送信
// =============================================================================

// carbonKeywordsForFilter はカーボン関連記事のフィルタリング用キーワード
//
// タイトルまたはAISummaryにこれらのキーワードが含まれる記事のみを
// メール送信対象とする。
var carbonKeywordsForFilter = []string{
	// 日本語キーワード
	"カーボン", "炭素", "脱炭素", "CO2", "温室効果ガス", "GHG",
	"気候変動", "クライメート", "排出量取引", "ETS", "カーボンプライシング",
	"カーボンクレジット", "クレジット市場", "JCM", "二国間クレジット",
	"カーボンニュートラル", "地球温暖化", "パリ協定", "COP",
	// 英語キーワード
	"carbon", "climate", "emission", "offset", "credit",
	"EUA", "VCM", "CDR", "CORSIA", "CBAM",
}

// containsCarbonKeyword はテキストにカーボン関連キーワードが含まれるかチェック
func containsCarbonKeyword(text string) bool {
	textLower := strings.ToLower(text)
	for _, kw := range carbonKeywordsForFilter {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// SendShortHeadlinesDigest は50文字ヘッドラインのダイジェストメールを送信する
//
// 【処理の流れ】
//  1. カーボンキーワードでフィルタリング
//  2. 番号付きリスト + URL形式で本文生成
//  3. リトライ付きで送信
//
// 【メール形式】
//
//	Carbon Headlines Digest - 2026-01-06
//	Total: 25 articles
//
//	1. EU carbon prices hit record high...
//	   https://carbonherald.com/...
//
//	2. Japan launches new GX initiative...
//	   https://carboncredits.jp/...
func (es *EmailSender) SendShortHeadlinesDigest(ctx context.Context, headlines []NotionHeadline) error {
	// カーボンキーワードでフィルタリング + ShortHeadlineが"-"のものを除外
	filtered := make([]NotionHeadline, 0, len(headlines))
	for _, h := range headlines {
		// ShortHeadlineが"-"系の場合は除外（Notion AIが要約できなかった記事）
		if h.ShortHeadline == "-" || h.ShortHeadline == "−" || h.ShortHeadline == "—" {
			continue
		}
		if containsCarbonKeyword(h.Title) || containsCarbonKeyword(h.AISummary) {
			filtered = append(filtered, h)
		}
	}

	var body, subject string
	if len(filtered) == 0 {
		subject = fmt.Sprintf("炭素関連記事一覧 - %s (0 記事)",
			time.Now().Format("2006-01-02"))
		body = fmt.Sprintf("炭素関連記事一覧 - %s\n合計: 0 記事\n\nこの期間にカーボン関連の記事は見つかりませんでした。\n",
			time.Now().Format("2006-01-02"))
	} else {
		body = es.generateShortHeadlinesBody(filtered)
		subject = fmt.Sprintf("炭素関連記事一覧 - %s (%d 記事)",
			time.Now().Format("2006-01-02"),
			len(filtered))
	}

	// RFC 5322準拠のメッセージを構築
	msg := es.buildEmailMessage(subject, body)

	// リトライ付きで送信
	return es.sendWithRetry(msg)
}

// generateShortHeadlinesBody は50文字ヘッドラインのメール本文を生成する
//
// 【出力フォーマット】
//
//	Carbon Headlines Digest - 2026-01-06
//	Total: 25 articles
//
//	1. EU carbon prices hit record high...
//	   https://carbonherald.com/...
func (es *EmailSender) generateShortHeadlinesBody(headlines []NotionHeadline) string {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString(fmt.Sprintf("炭素関連記事一覧 - %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("合計: %d 記事s\n\n", len(headlines)))

	// 各記事
	for i, h := range headlines {
		// ShortHeadlineがあればそれを使用、なければTitleを50文字に切り詰め
		displayText := h.ShortHeadline
		if displayText == "" {
			// フォールバック: タイトルを50文字に切り詰め
			runes := []rune(h.Title)
			if len(runes) > 50 {
				displayText = string(runes[:47]) + "..."
			} else {
				displayText = h.Title
			}
		}

		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, displayText))
		sb.WriteString(fmt.Sprintf("   %s\n\n", h.URL))
	}

	// フッター
	sb.WriteString("---\n")
	sb.WriteString("Generated by carbon-relay\n")
	sb.WriteString("https://github.com/FuseKota/curbon-search\n")

	return sb.String()
}
