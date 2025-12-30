package main

import (
	"context"
	"fmt"
	"os"

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
func (nc *NotionClipper) CreateDatabase(ctx context.Context, pageID string) error {
	if pageID == "" {
		return fmt.Errorf("NOTION_PAGE_ID is required to create a new database")
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
						{Name: "OpenAI(text_extract)", Color: notionapi.ColorPurple},
						{Name: "Free Article", Color: notionapi.ColorYellow},
					},
				},
			},
			"Excerpt": notionapi.RichTextPropertyConfig{
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
		},
	}

	db, err := nc.client.Database.Create(ctx, dbRequest)
	if err != nil {
		return fmt.Errorf("failed to create Notion database: %w", err)
	}

	nc.dbID = notionapi.DatabaseID(db.ID)
	fmt.Fprintf(os.Stderr, "✅ Notion database created: %s\n", db.ID)
	fmt.Fprintf(os.Stderr, "   Database URL: https://notion.so/%s\n", db.ID)

	return nil
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

	// Add excerpt if available
	if h.Excerpt != "" {
		properties["Excerpt"] = notionapi.RichTextProperty{
			Type: notionapi.PropertyTypeRichText,
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: truncateText(h.Excerpt, 2000), // Notion limit
					},
				},
			},
		}
	}

	pageRequest := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: nc.dbID,
		},
		Properties: properties,
	}

	_, err := nc.client.Page.Create(ctx, pageRequest)
	if err != nil {
		return fmt.Errorf("failed to clip headline: %w", err)
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

	pageRequest := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: nc.dbID,
		},
		Properties: properties,
	}

	_, err := nc.client.Page.Create(ctx, pageRequest)
	if err != nil {
		return fmt.Errorf("failed to clip related free article: %w", err)
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

// truncateText truncates text to maxLen characters
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
