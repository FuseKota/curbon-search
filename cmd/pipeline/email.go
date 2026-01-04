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
//   EMAIL_FROM     - 送信元メールアドレス（Gmail）
//   EMAIL_PASSWORD - Gmailアプリパスワード（通常のパスワードではない！）
//   EMAIL_TO       - 送信先メールアドレス（カンマ区切りで複数可）
//
// =============================================================================
// 【Gmailアプリパスワードについて】
// =============================================================================
//
// Googleアカウントの2段階認証を有効にした上で、
// 「アプリパスワード」を生成する必要があります。
//
// 生成方法:
//   1. https://myaccount.google.com/security にアクセス
//   2. 「2段階認証プロセス」を有効化
//   3. 「アプリパスワード」を選択
//   4. 「メール」と「その他（カスタム名）」を選択
//   5. 生成された16文字のパスワードをEMAIL_PASSWORDに設定
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
// メール送信
// =============================================================================

// SendHeadlinesSummary は見出しサマリーメールを送信する
//
// 【処理の流れ】
//  1. メール本文を生成
//  2. 件名を生成（日付と記事数を含む）
//  3. RFC 5322準拠のメッセージを構築
//  4. リトライ付きで送信
func (es *EmailSender) SendHeadlinesSummary(ctx context.Context, headlines []NotionHeadline) error {
	if len(headlines) == 0 {
		return fmt.Errorf("no headlines to send")
	}

	// メール本文を生成
	body := es.generateEmailBody(headlines)

	// 件名を生成
	// 例: "Carbon News Headlines - 2026-01-05 (15 articles)"
	subject := fmt.Sprintf("Carbon News Headlines - %s (%d articles)",
		time.Now().Format("2006-01-02"),
		len(headlines))

	// RFC 5322準拠のメッセージを構築
	msg := es.buildEmailMessage(subject, body)

	// リトライ付きで送信
	return es.sendWithRetry(msg)
}

// =============================================================================
// メール本文生成
// =============================================================================

// generateEmailBody はプレーンテキストのメール本文を生成する
//
// 【出力フォーマット】
//
//	Carbon News Headlines Summary
//	Generated: 2026-01-05 12:00:00
//
//	========================================
//	Total Headlines: 15
//	========================================
//
//	[1] Title: "記事タイトル"
//	    Source: Carbon Pulse
//	    URL: https://...
//
//	    Summary:
//	    記事の要約テキスト...
//
//	----------------------------------------
func (es *EmailSender) generateEmailBody(headlines []NotionHeadline) string {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString("Carbon News Headlines Summary\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("Total Headlines: %d\n", len(headlines)))
	sb.WriteString("========================================\n\n")

	// 各記事
	for i, h := range headlines {
		sb.WriteString(fmt.Sprintf("[%d] Title: \"%s\"\n", i+1, h.Title))
		sb.WriteString(fmt.Sprintf("    Source: %s\n", h.Source))
		sb.WriteString(fmt.Sprintf("    URL: %s\n", h.URL))
		sb.WriteString("\n")

		// AI要約がある場合は表示
		if h.AISummary != "" {
			sb.WriteString("    Summary:\n")
			// 要約テキストをインデント
			summaryLines := strings.Split(h.AISummary, "\n")
			for _, line := range summaryLines {
				if strings.TrimSpace(line) != "" {
					sb.WriteString(fmt.Sprintf("    %s\n", line))
				}
			}
		} else {
			sb.WriteString("    Summary: (No AI summary available)\n")
		}

		sb.WriteString("\n")
		sb.WriteString("----------------------------------------\n\n")
	}

	// フッター
	sb.WriteString("\n")
	sb.WriteString("Generated by carbon-relay\n")
	sb.WriteString("https://github.com/FuseKota/curbon-search\n")

	return sb.String()
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
		fmt.Fprintf(os.Stderr, "WARN: Email send failed (attempt %d/%d): %v\n", i+1, maxRetries, err)
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
