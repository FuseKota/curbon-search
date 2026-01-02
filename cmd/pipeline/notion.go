package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

// NotionClipperConfig holds configuration for Notion integration
type NotionClipperConfig struct {
	Token      string // Notion Integration Token
	PageID     string // Parent page ID where DB will be created (optional)
	DatabaseID string // Existing database ID (optional)
}

// NotionClipper handles clipping articles to Notion
type NotionClipper struct {
	client *notionapi.Client
	dbID   notionapi.DatabaseID
}

// NewNotionClipper creates a new Notion clipper
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

// CreateDatabase creates a new Notion database for article clipping
// Returns the database ID and error
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
						{Name: "Carbon Pulse", Color: notionapi.ColorBlue},
						{Name: "QCI", Color: notionapi.ColorGreen},
						{Name: "CarbonCredits.jp", Color: notionapi.ColorOrange},
						{Name: "Carbon Herald", Color: notionapi.ColorPink},
						{Name: "Climate Home News", Color: notionapi.ColorPurple},
						{Name: "CarbonCredits.com", Color: notionapi.ColorYellow},
						{Name: "Sandbag", Color: notionapi.ColorBlue},
						{Name: "Ecosystem Marketplace", Color: notionapi.ColorGreen},
						{Name: "Carbon Brief", Color: notionapi.ColorPurple},
						{Name: "OpenAI(text_extract)", Color: notionapi.ColorGray},
						{Name: "Free Article", Color: notionapi.ColorDefault},
					},
				},
			},
			"AI Summary": notionapi.RichTextPropertyConfig{
				Type: notionapi.PropertyConfigTypeRichText,
			},
			"Type": notionapi.SelectPropertyConfig{
				Type: notionapi.PropertyConfigTypeSelect,
				Select: notionapi.Select{
					Options: []notionapi.Option{
						{Name: "Headline", Color: notionapi.ColorRed},
						{Name: "Related Free", Color: notionapi.ColorGreen},
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

// ClipHeadline clips a headline to Notion
func (nc *NotionClipper) ClipHeadline(ctx context.Context, h Headline) error {
	if nc.dbID == "" {
		return fmt.Errorf("database ID not set")
	}

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
		"Type": notionapi.SelectProperty{
			Type: notionapi.PropertyTypeSelect,
			Select: notionapi.Option{
				Name: "Headline",
			},
		},
	}

	// Add Published Date if available
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

	// Add full content to AI Summary field (split into multiple RichText blocks if needed)
	if h.Excerpt != "" {
		properties["AI Summary"] = notionapi.RichTextProperty{
			Type:     notionapi.PropertyTypeRichText,
			RichText: splitIntoRichTextBlocks(h.Excerpt),
		}
	}

	// Create page request (without content blocks - will add separately)
	pageRequest := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: nc.dbID,
		},
		Properties: properties,
	}

	page, err := nc.client.Page.Create(ctx, pageRequest)
	if err != nil {
		return fmt.Errorf("failed to clip headline: %w", err)
	}

	// Add full content as page blocks if available
	if h.Excerpt != "" {
		blocks := createContentBlocks(h.Excerpt)
		if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Adding %d content blocks to page (total chars: %d)\n", len(blocks), len(h.Excerpt))
		}

		// Append blocks to the page
		_, err = nc.client.Block.AppendChildren(ctx, notionapi.BlockID(page.ID), &notionapi.AppendBlockChildrenRequest{
			Children: blocks,
		})
		if err != nil {
			return fmt.Errorf("failed to add content blocks: %w", err)
		}
	}

	return nil
}

// ClipRelatedFree clips a related free article to Notion
func (nc *NotionClipper) ClipRelatedFree(ctx context.Context, rf RelatedFree) error {
	if nc.dbID == "" {
		return fmt.Errorf("database ID not set")
	}

	properties := notionapi.Properties{
		"Title": notionapi.TitleProperty{
			Type: notionapi.PropertyTypeTitle,
			Title: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: rf.Title,
					},
				},
			},
		},
		"URL": notionapi.URLProperty{
			Type: notionapi.PropertyTypeURL,
			URL:  rf.URL,
		},
		"Source": notionapi.SelectProperty{
			Type: notionapi.PropertyTypeSelect,
			Select: notionapi.Option{
				Name: rf.Source,
			},
		},
		"Type": notionapi.SelectProperty{
			Type: notionapi.PropertyTypeSelect,
			Select: notionapi.Option{
				Name: "Related Free",
			},
		},
		"Score": notionapi.NumberProperty{
			Type:   notionapi.PropertyTypeNumber,
			Number: rf.Score,
		},
	}

	// Add Published Date if available
	if rf.PublishedAt != "" {
		publishedTime, err := parsePublishedDate(rf.PublishedAt)
		if err == nil {
			properties["Published Date"] = notionapi.DateProperty{
				Type: notionapi.PropertyTypeDate,
				Date: &notionapi.DateObject{
					Start: (*notionapi.Date)(&publishedTime),
				},
			}
		} else if os.Getenv("DEBUG_SCRAPING") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to parse PublishedAt '%s': %v\n", rf.PublishedAt, err)
		}
	}

	// Add full content to AI Summary field (split into multiple RichText blocks if needed)
	if rf.Excerpt != "" {
		properties["AI Summary"] = notionapi.RichTextProperty{
			Type:     notionapi.PropertyTypeRichText,
			RichText: splitIntoRichTextBlocks(rf.Excerpt),
		}
	}

	// Create page request (without content blocks - will add separately)
	pageRequest := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: nc.dbID,
		},
		Properties: properties,
	}

	page, err := nc.client.Page.Create(ctx, pageRequest)
	if err != nil {
		return fmt.Errorf("failed to clip related free article: %w", err)
	}

	// Add full content as page blocks if available
	if rf.Excerpt != "" {
		blocks := createContentBlocks(rf.Excerpt)
		_, err = nc.client.Block.AppendChildren(ctx, notionapi.BlockID(page.ID), &notionapi.AppendBlockChildrenRequest{
			Children: blocks,
		})
		if err != nil {
			return fmt.Errorf("failed to add content blocks: %w", err)
		}
	}

	return nil
}

// ClipHeadlineWithRelated clips a headline and all its related articles
func (nc *NotionClipper) ClipHeadlineWithRelated(ctx context.Context, h Headline) error {
	// Clip the headline
	if err := nc.ClipHeadline(ctx, h); err != nil {
		return fmt.Errorf("failed to clip headline: %w", err)
	}

	// Clip all related free articles
	for _, rf := range h.RelatedFree {
		if err := nc.ClipRelatedFree(ctx, rf); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: failed to clip related article %s: %v\n", rf.URL, err)
			// Continue with other articles even if one fails
		}
	}

	return nil
}

// splitIntoRichTextBlocks splits long text into multiple RichText blocks
// Each RichText block in Notion property has a 2000 character limit
func splitIntoRichTextBlocks(text string) []notionapi.RichText {
	const maxChars = 2000
	var richTexts []notionapi.RichText

	if len(text) == 0 {
		return richTexts
	}

	// Split text into chunks of maxChars
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

// createContentBlocks splits long text into Notion paragraph blocks
// Notion has a 2000 character limit per block, so we split long text
func createContentBlocks(content string) notionapi.Blocks {
	const maxBlockSize = 2000
	blocks := notionapi.Blocks{}

	// Split by paragraphs first (double newlines)
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

	// Create blocks, splitting if any paragraph exceeds max size
	for _, para := range paragraphs {
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
			// Split long paragraph into chunks
			for i := 0; i < len(para); i += maxBlockSize {
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

// parsePublishedDate parses published date from various formats
// WordPress API may return dates without timezone, so we try multiple formats
func parsePublishedDate(dateStr string) (time.Time, error) {
	// Try RFC3339 format first (with timezone)
	t, err := time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return t, nil
	}

	// Try format without timezone (assume UTC)
	// WordPress often returns: "2025-12-26T14:42:50"
	t, err = time.Parse("2006-01-02T15:04:05", dateStr)
	if err == nil {
		// Treat as UTC since no timezone info
		return t.UTC(), nil
	}

	// Try ISO 8601 date-only format
	t, err = time.Parse("2006-01-02", dateStr)
	if err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// FetchRecentHeadlines fetches headlines from Notion database
// Returns headlines created within the last 'daysBack' days with Type="Headline"
func (nc *NotionClipper) FetchRecentHeadlines(ctx context.Context, daysBack int) ([]NotionHeadline, error) {
	if nc.dbID == "" {
		return nil, fmt.Errorf("database ID not set")
	}

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -daysBack)

	// Query database with pagination (no filter - will filter in code)
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

		// Process results
		for _, page := range resp.Results {
			// Extract Type and filter for "Headline"
			pageType := ""
			if typeProp, ok := page.Properties["Type"].(*notionapi.SelectProperty); ok && typeProp.Select.Name != "" {
				pageType = typeProp.Select.Name
			}

			// Skip if not a Headline
			if pageType != "Headline" {
				continue
			}

			// Filter by creation date
			if !page.CreatedTime.After(cutoffDate) {
				continue
			}

			// Extract Title
			title := ""
			if titleProp, ok := page.Properties["Title"].(*notionapi.TitleProperty); ok && len(titleProp.Title) > 0 {
				title = titleProp.Title[0].PlainText
			}

			// Extract URL
			url := ""
			if urlProp, ok := page.Properties["URL"].(*notionapi.URLProperty); ok {
				url = string(urlProp.URL)
			}

			// Extract Source
			source := ""
			if sourceProp, ok := page.Properties["Source"].(*notionapi.SelectProperty); ok && sourceProp.Select.Name != "" {
				source = sourceProp.Select.Name
			}

			// Extract AI Summary
			aiSummary := ""
			if summaryProp, ok := page.Properties["AI Summary"].(*notionapi.RichTextProperty); ok && len(summaryProp.RichText) > 0 {
				// Concatenate all rich text segments
				for _, rt := range summaryProp.RichText {
					aiSummary += rt.PlainText
				}
			}

			// Extract Created time
			createdAt := page.CreatedTime.Format(time.RFC3339)

			allHeadlines = append(allHeadlines, NotionHeadline{
				Title:      title,
				URL:        url,
				Source:     source,
				AISummary:  aiSummary,
				CreatedAt:  createdAt,
			})
		}

		// Check if there are more pages
		if !resp.HasMore {
			break
		}
		cursor = &resp.NextCursor
	}

	return allHeadlines, nil
}
