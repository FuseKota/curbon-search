package main

// Headline: paywalled/free-visible headline entry (e.g., Carbon Pulse / QCI listing).
// We use its Title as the query seed and attach RelatedFree results.
type Headline struct {
	Source        string        `json:"source"`
	Title         string        `json:"title"`
	URL           string        `json:"url"`
	PublishedAt   string        `json:"publishedAt,omitempty"` // RFC3339 format
	Excerpt       string        `json:"excerpt,omitempty"`     // Free preview text visible without subscription
	IsHeadline    bool          `json:"isHeadline,omitempty"`
	SearchQueries []string      `json:"searchQueries,omitempty"`
	RelatedFree   []RelatedFree `json:"relatedFree,omitempty"`
}

// FreeArticle: candidate “free / primary source” URL found via RSS or search.
type FreeArticle struct {
	Source      string `json:"source"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	PublishedAt string `json:"publishedAt,omitempty"` // RFC3339 if known
	Excerpt     string `json:"excerpt,omitempty"`
}

type RelatedFree struct {
	Source      string  `json:"source"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	PublishedAt string  `json:"publishedAt,omitempty"`
	Excerpt     string  `json:"excerpt,omitempty"`
	Score       float64 `json:"score"`
	Reason      string  `json:"reason"`
}

// NotionHeadline: headline fetched from Notion Database for email sending.
type NotionHeadline struct {
	Title      string
	URL        string
	Source     string
	AISummary  string
	CreatedAt  string // RFC3339 format
}
