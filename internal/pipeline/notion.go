// =============================================================================
// notion.go - Notion統合モジュール
// =============================================================================
//
// このファイルはNotionデータベースへの記事保存機能を提供します。
// 無料記事収集モードで使用されます。
//
// =============================================================================
// 【主要な機能】
// =============================================================================
//
// 1. データベース作成
//    - 新規Notionデータベースの自動作成
//    - 作成したデータベースIDを.envに自動保存
//
// 2. 記事のクリッピング
//    - 記事の見出しをデータベースに保存
//
// 3. 記事の取得
//    - Notionデータベースから最近の記事を取得
//    - メール送信機能で使用
//
// =============================================================================
// 【データベーススキーマ】
// =============================================================================
//
// 以下のプロパティを持つデータベースを作成/使用:
//
//   ┌────────────────┬──────────────┬────────────────────────────────┐
//   │ プロパティ名   │ 型           │ 説明                           │
//   ├────────────────┼──────────────┼────────────────────────────────┤
//   │ Title          │ Title        │ 記事タイトル                   │
//   │ URL            │ URL          │ 記事URL                        │
//   │ Source         │ Select       │ ソース名（22種類のオプション） │
//   │ Type           │ Select       │ Headline / Related Free        │
//   │ Score          │ Number       │ マッチングスコア（0-1）        │
//   │ Published Date │ Date         │ 記事の公開日                   │
//   └────────────────┴──────────────┴────────────────────────────────┘
//
// =============================================================================
// 【Notion API制限への対応】
// =============================================================================
//
// - RichTextプロパティ: 最大2000文字
//   → splitIntoRichTextBlocks() で分割して対応
//
// - ブロックコンテンツ: 最大2000文字/ブロック
//   → createContentBlocks() で分割して対応
//
// =============================================================================
// 【必要な環境変数】
// =============================================================================
//
//   NOTION_TOKEN     - Notion Integration Token（必須）
//   NOTION_PAGE_ID   - 新規DB作成時の親ページID
//   NOTION_DATABASE_ID - 既存DBのID（作成済みの場合）
//
// =============================================================================
// 【初心者向けポイント】
// =============================================================================
//
// - Notion APIはOAuth認証ではなくIntegration Tokenを使用
// - データベースIDは32文字のハイフン区切り文字列
// - ページIDはデータベース内の個々のレコードを指す
// - github.com/jomei/notionapi ライブラリを使用
//
// =============================================================================
package pipeline

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jomei/notionapi" // Notion API クライアントライブラリ
)

// notionRetry はNotion API呼び出しをリトライ付きで実行する
// レート制限（429）やサーバーエラー（5xx）でHTMLが返された場合、
// JSONパースエラー（"invalid character '<'"）として検出しリトライする
func notionRetry(operation string, fn func() error) error {
	const maxRetries = 3
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			wait := time.Duration(1<<uint(i)) * time.Second // 2s, 4s, 8s
			fmt.Fprintf(os.Stderr, "⏳ Notion API retry %d/%d for %s (waiting %v)...\n", i, maxRetries, operation, wait)
			time.Sleep(wait)
		}

		err := fn()
		if err == nil {
			if i > 0 {
				fmt.Fprintf(os.Stderr, "✅ Notion API %s succeeded after %d retries\n", operation, i)
			}
			return nil
		}

		lastErr = err

		// HTMLレスポンス（レート制限/サーバーエラー）の場合のみリトライ
		if !strings.Contains(err.Error(), "invalid character '<'") {
			return err
		}
	}

	return lastErr
}

// academicSources は査読付き学術論文・プレプリントのソース
var academicSources = map[string]bool{
	"arXiv":              true,
	"Nature Communications": true,
	"IOP Science (ERL)":  true,
	"ScienceDirect":      true,
}

// =============================================================================
// 設定・構造体
// =============================================================================

// NotionClipperConfig はNotion統合の設定を保持する
type NotionClipperConfig struct {
	Token      string // Notion Integration Token
	PageID     string // 新規DB作成時の親ページID（オプション）
	DatabaseID string // 既存DBのID（オプション）
}

// NotionClipper はNotionへの記事保存を担当する
//
// 【使用方法】
//
//	clipper, err := NewNotionClipper(token, dbID)
//	err = clipper.ClipHeadline(ctx, headline)
type NotionClipper struct {
	client                     *notionapi.Client     // Notion APIクライアント
	dbID                       notionapi.DatabaseID  // 操作対象のデータベースID
	shortHeadlinePropertyEnsured bool                // Article Summary 300プロパティ確認済みフラグ
}

// NewNotionClipper は新しいNotionクリッパーを作成する
func NewNotionClipper(token string, databaseID string) (*NotionClipper, error) {
	if token == "" {
		return nil, fmt.Errorf("NOTION_TOKEN is required")
	}

	client := notionapi.NewClient(notionapi.Token(token))

	nc := &NotionClipper{
		client: client,
	}

	if databaseID != "" {
		nc.dbID = notionapi.DatabaseID(databaseID)
	}

	return nc, nil
}

// CreateDatabase は記事クリッピング用の新しいNotionデータベースを作成する
// データベースIDとエラーを返す
func (nc *NotionClipper) CreateDatabase(ctx context.Context, pageID string) (string, error) {
	if pageID == "" {
		return "", fmt.Errorf("NOTION_PAGE_ID is required to create a new database")
	}

	dbRequest := &notionapi.DatabaseCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(pageID),
		},
		Title: []notionapi.RichText{
			{
				Text: &notionapi.Text{
					Content: "Carbon News Clippings",
				},
			},
		},
		Properties: notionapi.PropertyConfigs{
			"Title": notionapi.TitlePropertyConfig{
				Type: notionapi.PropertyConfigTypeTitle,
			},
			"URL": notionapi.URLPropertyConfig{
				Type: notionapi.PropertyConfigTypeURL,
			},
			"Source": notionapi.SelectPropertyConfig{
				Type: notionapi.PropertyConfigTypeSelect,
				Select: notionapi.Select{
					Options: []notionapi.Option{
						{Name: "CarbonCredits.jp", Color: notionapi.ColorOrange},
						{Name: "Carbon Herald", Color: notionapi.ColorPink},
						{Name: "Climate Home News", Color: notionapi.ColorPurple},
						{Name: "CarbonCredits.com", Color: notionapi.ColorYellow},
						{Name: "Sandbag", Color: notionapi.ColorBlue},
						{Name: "Ecosystem Marketplace", Color: notionapi.ColorGreen},
						{Name: "Carbon Brief", Color: notionapi.ColorPurple},
						{Name: "ICAP", Color: notionapi.ColorRed},
						{Name: "IETA", Color: notionapi.ColorBrown},
						{Name: "Energy Monitor", Color: notionapi.ColorPink},
						{Name: "Japan Research Institute", Color: notionapi.ColorGreen},
						{Name: "Japan Environment Ministry", Color: notionapi.ColorBlue},
						{Name: "Japan Exchange Group (JPX)", Color: notionapi.ColorRed},
						{Name: "Japan Ministry of Economy (METI)", Color: notionapi.ColorRed},
						{Name: "World Bank", Color: notionapi.ColorBrown},
						{Name: "Carbon Market Watch", Color: notionapi.ColorPurple},
						{Name: "NewClimate Institute", Color: notionapi.ColorGreen},
						{Name: "Carbon Knowledge Hub", Color: notionapi.ColorOrange},
						{Name: "PwC Japan", Color: notionapi.ColorPink},
						{Name: "Mizuho Research & Technologies", Color: notionapi.ColorBlue},
						{Name: "Free Article", Color: notionapi.ColorDefault},
					},
				},
			},
			"Article Summary 300": notionapi.RichTextPropertyConfig{
				Type: notionapi.PropertyConfigTypeRichText,
			},
			"Type": notionapi.SelectPropertyConfig{
				Type: notionapi.PropertyConfigTypeSelect,
				Select: notionapi.Select{
					Options: []notionapi.Option{
						{Name: "News", Color: notionapi.ColorBlue},
						{Name: "Academic", Color: notionapi.ColorGreen},
					},
				},
			},
			"Score": notionapi.NumberPropertyConfig{
				Type: notionapi.PropertyConfigTypeNumber,
				Number: notionapi.NumberFormat{
					Format: notionapi.FormatNumber,
				},
			},
			"Published Date": notionapi.DatePropertyConfig{
				Type: notionapi.PropertyConfigTypeDate,
			},
		},
	}

	db, err := nc.client.Database.Create(ctx, dbRequest)
	if err != nil {
		return "", fmt.Errorf("failed to create Notion database: %w", err)
	}

	nc.dbID = notionapi.DatabaseID(db.ID)
	fmt.Fprintf(os.Stderr, "✅ Notion database created: %s\n", db.ID)
	fmt.Fprintf(os.Stderr, "   Database URL: https://notion.so/%s\n", db.ID)

	return string(db.ID), nil
}

// ensureShortHeadlineProperty は既存のデータベースにArticle Summary 300プロパティを追加する
//
// 【背景】
//   - 既存のデータベースにはArticle Summary 300プロパティが存在しない場合がある
//   - この関数はプロパティが存在しない場合のみ追加する
//   - 既存プロパティのAI機能設定を上書きしないよう、存在確認してから追加
func (nc *NotionClipper) ensureShortHeadlineProperty(ctx context.Context) error {
	// 既に確認済みの場合はスキップ
	if nc.shortHeadlinePropertyEnsured {
		return nil
	}

	if nc.dbID == "" {
		return nil
	}

	// データベースのスキーマを取得してArticle Summary 300プロパティの存在を確認
	db, err := nc.client.Database.Get(ctx, nc.dbID)
	if err != nil {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to get database schema: %v\n", err)
		}
		nc.shortHeadlinePropertyEnsured = true
		return nil
	}

	// Article Summary 300プロパティが既に存在する場合はスキップ（AI機能設定を保持）
	if _, exists := db.Properties["Article Summary 300"]; exists {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Article Summary 300 property already exists, skipping update\n")
		}
		nc.shortHeadlinePropertyEnsured = true
		return nil
	}

	// Article Summary 300プロパティが存在しない場合のみ追加
	_, err = nc.client.Database.Update(ctx, nc.dbID, &notionapi.DatabaseUpdateRequest{
		Properties: notionapi.PropertyConfigs{
			"Article Summary 300": notionapi.RichTextPropertyConfig{
				Type: notionapi.PropertyConfigTypeRichText,
			},
		},
	})
	if err != nil {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to add Article Summary 300 property: %v\n", err)
		}
	} else {
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Article Summary 300 property added to database\n")
		}
	}

	nc.shortHeadlinePropertyEnsured = true
	return nil
}

// ClipHeadline はヘッドラインをNotionにクリップする
func (nc *NotionClipper) ClipHeadline(ctx context.Context, h Headline) error {
	if nc.dbID == "" {
		return fmt.Errorf("database ID not set")
	}

	// 既存DBにArticle Summary 300プロパティがない場合に追加
	nc.ensureShortHeadlineProperty(ctx)

	properties := notionapi.Properties{
		"Title": notionapi.TitleProperty{
			Type: notionapi.PropertyTypeTitle,
			Title: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: h.Title,
					},
				},
			},
		},
		"URL": notionapi.URLProperty{
			Type: notionapi.PropertyTypeURL,
			URL:  h.URL,
		},
		"Source": notionapi.SelectProperty{
			Type: notionapi.PropertyTypeSelect,
			Select: notionapi.Option{
				Name: h.Source,
			},
		},
	}

	// Typeプロパティを設定: 学術ソースはAcademic、それ以外はNews
	typeName := "News"
	if academicSources[h.Source] {
		typeName = "Academic"
	}
	properties["Type"] = notionapi.SelectProperty{
		Type:   notionapi.PropertyTypeSelect,
		Select: notionapi.Option{Name: typeName},
	}

	// Published Dateがあれば追加
	if h.PublishedAt != "" {
		publishedTime, err := parsePublishedDate(h.PublishedAt)
		if err == nil {
			properties["Published Date"] = notionapi.DateProperty{
				Type: notionapi.PropertyTypeDate,
				Date: &notionapi.DateObject{
					Start: (*notionapi.Date)(&publishedTime),
				},
			}
		} else if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to parse PublishedAt '%s': %v\n", h.PublishedAt, err)
		}
	}

	// Article Summary 300フィールドに全文を追加
	// （2000文字制限のため、必要に応じて複数のRichTextブロックに分割）
	if h.Excerpt != "" {
		richTextBlocks := splitIntoRichTextBlocks(h.Excerpt)
		properties["Article Summary 300"] = notionapi.RichTextProperty{
			Type:     notionapi.PropertyTypeRichText,
			RichText: richTextBlocks,
		}
	}

	// ページ作成
	pageRequest := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: nc.dbID,
		},
		Properties: properties,
	}

	var page *notionapi.Page
	err := notionRetry("Page.Create", func() error {
		var createErr error
		page, createErr = nc.client.Page.Create(ctx, pageRequest)
		return createErr
	})
	if err != nil {
		return fmt.Errorf("failed to clip headline: %w", err)
	}

	// 全文をページブロックとして追加（Excerptがある場合）
	if h.Excerpt != "" {
		blocks := createContentBlocks(h.Excerpt)
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Adding %d content blocks to page (total chars: %d)\n", len(blocks), len(h.Excerpt))
		}

		// ページにブロックを追加
		err = notionRetry("Block.AppendChildren", func() error {
			_, appendErr := nc.client.Block.AppendChildren(ctx, notionapi.BlockID(page.ID), &notionapi.AppendBlockChildrenRequest{
				Children: blocks,
			})
			return appendErr
		})
		if err != nil {
			return fmt.Errorf("failed to add content blocks: %w", err)
		}
	}

	return nil
}

// ClipHeadlineWithRelated はClipHeadlineのエイリアス（Lambda互換用）
func (nc *NotionClipper) ClipHeadlineWithRelated(ctx context.Context, h Headline) error {
	return nc.ClipHeadline(ctx, h)
}

// splitIntoRichTextBlocks は長いテキストを複数のRichTextブロックに分割する
// NotionプロパティのRichTextブロックには2000文字の制限がある
func splitIntoRichTextBlocks(text string) []notionapi.RichText {
	const maxChars = 2000
	var richTexts []notionapi.RichText

	if len(text) == 0 {
		return richTexts
	}

	// テキストをmaxChars文字ごとのチャンクに分割
	for i := 0; i < len(text); i += maxChars {
		end := i + maxChars
		if end > len(text) {
			end = len(text)
		}
		richTexts = append(richTexts, notionapi.RichText{
			Text: &notionapi.Text{
				Content: text[i:end],
			},
		})
	}

	return richTexts
}

// createContentBlocks は長いテキストをNotionの段落ブロックに分割する
// Notionはブロックあたり2000文字の制限があるため、長文を分割する
func createContentBlocks(content string) notionapi.Blocks {
	const maxBlockSize  = 2000
	const maxBlockCount = 100 // Notion APIの上限
	blocks := notionapi.Blocks{}

	// まず段落（空行）で分割
	paragraphs := []string{}
	currentPara := ""
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			if currentPara != "" {
				paragraphs = append(paragraphs, strings.TrimSpace(currentPara))
				currentPara = ""
			}
		} else {
			if currentPara != "" {
				currentPara += "\n"
			}
			currentPara += line
		}
	}
	if currentPara != "" {
		paragraphs = append(paragraphs, strings.TrimSpace(currentPara))
	}

	// ブロックを作成（段落が最大サイズを超える場合は分割）
	for _, para := range paragraphs {
		if len(blocks) >= maxBlockCount {
			break
		}
		if len(para) <= maxBlockSize {
			blocks = append(blocks, notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Type:   notionapi.BlockTypeParagraph,
					Object: notionapi.ObjectTypeBlock,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{
						{
							Text: &notionapi.Text{
								Content: para,
							},
						},
					},
				},
			})
		} else {
			// 長い段落をチャンクに分割
			for i := 0; i < len(para); i += maxBlockSize {
				if len(blocks) >= maxBlockCount {
					break
				}
				end := i + maxBlockSize
				if end > len(para) {
					end = len(para)
				}
				blocks = append(blocks, notionapi.ParagraphBlock{
					BasicBlock: notionapi.BasicBlock{
						Type:   notionapi.BlockTypeParagraph,
						Object: notionapi.ObjectTypeBlock,
					},
					Paragraph: notionapi.Paragraph{
						RichText: []notionapi.RichText{
							{
								Text: &notionapi.Text{
									Content: para[i:end],
								},
							},
						},
					},
				})
			}
		}
	}

	return blocks
}

// parsePublishedDate は各種フォーマットの公開日をパースする
// WordPress APIがタイムゾーンなしの日付を返す場合があるため、複数フォーマットを試行する
func parsePublishedDate(dateStr string) (time.Time, error) {
	// まずRFC3339形式（タイムゾーン付き）を試行
	t, err := time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return t, nil
	}

	// コロンなしタイムゾーンオフセット付きISO 8601を試す
	t, err = time.Parse("2006-01-02T15:04:05-0700", dateStr)
	if err == nil {
		return t, nil
	}

	// タイムゾーンなし形式を試行（UTCとみなす）
	// WordPressは "2025-12-26T14:42:50" のような形式を返すことがある
	t, err = time.Parse("2006-01-02T15:04:05", dateStr)
	if err == nil {
		// タイムゾーン情報がないためUTCとして扱う
		return t.UTC(), nil
	}

	// ISO 8601の日付のみ形式を試行
	t, err = time.Parse("2006-01-02", dateStr)
	if err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// FetchRecentHeadlines はNotionデータベースからヘッドラインを取得する
// 過去daysBack日以内に作成されたヘッドラインを返す
func (nc *NotionClipper) FetchRecentHeadlines(ctx context.Context, daysBack int) ([]NotionHeadline, error) {
	if nc.dbID == "" {
		return nil, fmt.Errorf("database ID not set")
	}

	// 取得対象の基準日を算出
	cutoffDate := time.Now().AddDate(0, 0, -daysBack)

	// ページネーション付きでデータベースを検索（フィルタはコード側で適用）
	var allHeadlines []NotionHeadline
	var cursor *notionapi.Cursor

	for {
		query := &notionapi.DatabaseQueryRequest{
			PageSize: 100,
		}
		if cursor != nil {
			query.StartCursor = *cursor
		}

		resp, err := nc.client.Database.Query(ctx, nc.dbID, query)
		if err != nil {
			return nil, fmt.Errorf("failed to query database: %w", err)
		}

		// 結果を処理
		for _, page := range resp.Results {
			// 作成日でフィルタ
			if !page.CreatedTime.After(cutoffDate) {
				continue
			}

			// Titleを抽出
			title := ""
			if titleProp, ok := page.Properties["Title"].(*notionapi.TitleProperty); ok && len(titleProp.Title) > 0 {
				title = titleProp.Title[0].PlainText
			}

			// URLを抽出
			url := ""
			if urlProp, ok := page.Properties["URL"].(*notionapi.URLProperty); ok {
				url = string(urlProp.URL)
			}

			// Sourceを抽出
			source := ""
			if sourceProp, ok := page.Properties["Source"].(*notionapi.SelectProperty); ok && sourceProp.Select.Name != "" {
				source = sourceProp.Select.Name
			}

			// Type（Academic/News）を抽出
			articleType := ""
			if typeProp, ok := page.Properties["Type"].(*notionapi.SelectProperty); ok && typeProp.Select.Name != "" {
				articleType = typeProp.Select.Name
			}

			// Article Summary 300を抽出
			shortHeadline := ""
			if shortProp, ok := page.Properties["Article Summary 300"].(*notionapi.RichTextProperty); ok && len(shortProp.RichText) > 0 {
				for _, rt := range shortProp.RichText {
					shortHeadline += rt.PlainText
				}
			}

			// Published Dateを抽出
			publishedDate := ""
			if dateProp, ok := page.Properties["Published Date"].(*notionapi.DateProperty); ok && dateProp.Date != nil && dateProp.Date.Start != nil {
				publishedDate = time.Time(*dateProp.Date.Start).Format(time.RFC3339)
			}

			// 作成日時を抽出
			createdAt := page.CreatedTime.Format(time.RFC3339)

			allHeadlines = append(allHeadlines, NotionHeadline{
				Title:         title,
				URL:           url,
				Source:        source,
				Type:          articleType,
				ShortHeadline: shortHeadline,
				PublishedDate: publishedDate,
				CreatedAt:     createdAt,
			})
		}

		// 次のページがあるか確認
		if !resp.HasMore {
			break
		}
		cursor = &resp.NextCursor
	}

	return allHeadlines, nil
}

// =============================================================================
// 環境変数ファイル操作
// =============================================================================

// appendToEnvFile は.envファイルにキーと値のペアを追加または更新する
//
// キーが既に存在する場合は値を更新、存在しない場合は末尾に追加する。
// コメントアウトされたキー（#KEY=value）も検出して上書きする。
//
// 【使用場面】
//
//	新しいNotionデータベースを作成した際に、NOTION_DATABASE_IDを
//	.envファイルに自動保存する
//
// 引数:
//
//	filename: .envファイルのパス
//	key:      環境変数名（例: "NOTION_DATABASE_ID"）
//	value:    設定する値
func appendToEnvFile(filename, key, value string) error {
	// 既存の.envファイルを読み込む（存在しない場合は空文字）
	content := ""
	data, err := os.ReadFile(filename)
	if err == nil {
		content = string(data)
	}

	// キーが既に存在するかチェック
	lines := strings.Split(content, "\n")
	keyExists := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") || strings.HasPrefix(line, "#"+key+"=") {
			lines[i] = key + "=" + value
			keyExists = true
			break
		}
	}

	// キーが存在しない場合は末尾に追加
	if !keyExists {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		lines = append(lines, key+"="+value)
	}

	// ファイルに書き戻す
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}
