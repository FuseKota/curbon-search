package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Pipeline overview:
// 1. Collect paywalled headlines (Carbon Pulse / QCI free listing pages)
// 2. For each headline, perform OpenAI web search to find related free/primary sources
// 3. Score + attach relatedFree links
func main() {
	// Load .env file if it exists (silently ignore if not found)
	_ = godotenv.Load()

	var (
		headlinesFile = flag.String("headlines", "", "optional: path to headlines.json; if empty, scrape from sources")
		outFile       = flag.String("out", "", "optional: write matched output JSON to this path (default: stdout)")
		sources       = flag.String("sources", "carbonpulse,qci,carboncredits.jp,carbonherald,climatehomenews,carboncredits.com,sandbag,ecosystem-marketplace,carbon-brief,icap,ieta,energy-monitor,jri,env-ministry", "sources to scrape when --headlines is empty")
		perSource     = flag.Int("perSource", 30, "max headlines to collect per source")

		searchPerHeadline = flag.Int("searchPerHeadline", 25, "max candidate results kept per headline")
		queriesPerHead    = flag.Int("queriesPerHeadline", 3, "max queries to issue per headline")
		resultsPerQuery   = flag.Int("resultsPerQuery", 10, "results per query")

		daysBack     = flag.Int("daysBack", 60, "recency window in days (0 disables)")
		topK         = flag.Int("topK", 3, "max relatedFree per headline")
		minScore     = flag.Float64("minScore", 0.32, "minimum score threshold")
		strictMarket = flag.Bool("strictMarket", true, "require market match if headline has market signal")

		saveFree = flag.String("saveFree", "", "optional: write pooled free candidates to file")
		// --- new flags for OpenAI ---
		searchProvider = flag.String("searchProvider", "openai", "search provider: openai|brave")
		openaiModel    = flag.String("openaiModel", "gpt-4o-mini", "OpenAI model to use")
		// openaiModel = flag.String("openaiModel", "gpt-5.1", "OpenAI model to use")
		openaiTool = flag.String("openaiTool", "web_search", "OpenAI tool type: web_search|web_search_preview")

		// --- Notion integration ---
		notionClip       = flag.Bool("notionClip", false, "clip articles to Notion database")
		notionPageID     = flag.String("notionPageID", "", "parent page ID for creating new Notion database (required for new DB)")
		notionDatabaseID = flag.String("notionDatabaseID", "", "existing Notion database ID (optional, will create new if empty)")

		// --- Email integration ---
		sendEmail      = flag.Bool("sendEmail", false, "send headlines summary via email")
		emailDaysBack  = flag.Int("emailDaysBack", 1, "fetch headlines from last N days for email")
	)
	flag.Parse()

	// --- Early exit for email-only mode ---
	if *sendEmail {
		handleEmailSend(*emailDaysBack)
		return
	}

	// OpenAI API key check (only if search is enabled)
	if *queriesPerHead > 0 && os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "ERROR: set OPENAI_API_KEY (OpenAI API key) in your environment")
		fmt.Fprintln(os.Stderr, "NOTE: To skip search and only collect headlines, use -queriesPerHeadline=0")
		os.Exit(1)
	}

	// --- 1) Collect or read headlines ---
	var headlines []Headline
	if *headlinesFile != "" {
		if err := readJSONFile(*headlinesFile, &headlines); err != nil {
			fatalf("ERROR reading headlines: %v", err)
		}
	} else {
		cfg := defaultHeadlineConfig()
		want := map[string]bool{}
		for _, s := range strings.Split(*sources, ",") {
			s = strings.TrimSpace(strings.ToLower(s))
			if s != "" {
				want[s] = true
			}
		}
		if want["carbonpulse"] {
			hs, err := collectHeadlinesCarbonPulse(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Carbon Pulse headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["qci"] {
			hs, err := collectHeadlinesQCI(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting QCI headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["carboncredits.jp"] {
			hs, err := collectHeadlinesCarbonCreditsJP(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting CarbonCredits.jp headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["carbonherald"] {
			hs, err := collectHeadlinesCarbonHerald(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Carbon Herald headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["climatehomenews"] {
			hs, err := collectHeadlinesClimateHomeNews(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Climate Home News headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["carboncredits.com"] {
			hs, err := collectHeadlinesCarbonCreditscom(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting CarbonCredits.com headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["sandbag"] {
			hs, err := collectHeadlinesSandbag(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Sandbag headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["ecosystem-marketplace"] {
			hs, err := collectHeadlinesEcosystemMarketplace(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Ecosystem Marketplace headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["carbon-brief"] {
			hs, err := collectHeadlinesCarbonBrief(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Carbon Brief headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["icap"] {
			hs, err := collectHeadlinesICAP(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting ICAP headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["ieta"] {
			hs, err := collectHeadlinesIETA(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting IETA headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["energy-monitor"] {
			hs, err := collectHeadlinesEnergyMonitor(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Energy Monitor headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["jri"] {
			hs, err := collectHeadlinesJRI(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting JRI headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		if want["env-ministry"] {
			hs, err := collectHeadlinesEnvMinistry(*perSource, cfg)
			if err != nil {
				fatalf("ERROR collecting Environment Ministry headlines: %v", err)
			}
			headlines = append(headlines, hs...)
		}
		headlines = uniqueHeadlinesByURL(headlines)
	}

	if len(headlines) == 0 {
		fatalf("no headlines collected")
	}

	// --- 2) For each headline, perform web search ---
	now := time.Now()
	candsByIdx := make([][]FreeArticle, len(headlines))
	globalSeen := map[string]bool{}
	globalPool := make([]FreeArticle, 0, len(headlines)*(*searchPerHeadline))

	if *queriesPerHead == 0 {
		fmt.Fprintln(os.Stderr, "INFO: Search disabled (queriesPerHeadline=0), skipping web search phase")
	}

	for i, h := range headlines {
		queries := h.SearchQueries
		if len(queries) == 0 {
			queries = buildSearchQueries(h.Title, h.Excerpt)
		}
		if len(queries) > *queriesPerHead {
			queries = queries[:*queriesPerHead]
		}

		merged := map[string]FreeArticle{}
		for _, q := range queries {
			var res []FreeArticle
			var err error

			switch *searchProvider {
			case "openai":
				res, err = openaiWebSearch(q, *resultsPerQuery, *openaiModel, *openaiTool)
			default:
				err = fmt.Errorf("unsupported searchProvider: %s", *searchProvider)
			}

			if err != nil {
				fmt.Fprintln(os.Stderr, "WARN search:", err)
				continue
			}
			for _, a := range res {
				if a.URL == "" || a.Title == "" {
					continue
				}
				merged[a.URL] = a
				if len(merged) >= *searchPerHeadline {
					break
				}
			}
			if len(merged) >= *searchPerHeadline {
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
			*daysBack,
			*strictMarket,
			*topK,
			*minScore,
		)
	}

	// --- 5) Save results ---
	if *saveFree != "" {
		if err := writeJSONFile(*saveFree, globalPool); err != nil {
			fatalf("ERROR writing free pool: %v", err)
		}
	}

	if *outFile != "" {
		if err := writeJSONFile(*outFile, headlines); err != nil {
			fatalf("ERROR writing output: %v", err)
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(headlines)
	}

	// --- 6) Clip to Notion (if enabled) ---
	if *notionClip {
		fmt.Fprintln(os.Stderr, "\n========================================")
		fmt.Fprintln(os.Stderr, "📎 Clipping to Notion Database")
		fmt.Fprintln(os.Stderr, "========================================")

		notionToken := os.Getenv("NOTION_TOKEN")
		if notionToken == "" {
			fatalf("ERROR: NOTION_TOKEN environment variable is required for Notion integration")
		}

		clipper, err := NewNotionClipper(notionToken, *notionDatabaseID)
		if err != nil {
			fatalf("ERROR creating Notion clipper: %v", err)
		}

		ctx := context.Background()

		// Create database if needed
		if *notionDatabaseID == "" {
			if *notionPageID == "" {
				fatalf("ERROR: -notionPageID is required when creating a new Notion database")
			}
			fmt.Fprintln(os.Stderr, "Creating new Notion database...")
			dbID, err := clipper.CreateDatabase(ctx, *notionPageID)
			if err != nil {
				fatalf("ERROR creating Notion database: %v", err)
			}

			// Save database ID to .env file for future use
			if err := appendToEnvFile(".env", "NOTION_DATABASE_ID", dbID); err != nil {
				fmt.Fprintf(os.Stderr, "WARN: Failed to save database ID to .env: %v\n", err)
				fmt.Fprintf(os.Stderr, "Please manually add to .env:\nNOTION_DATABASE_ID=%s\n", dbID)
			} else {
				fmt.Fprintf(os.Stderr, "✅ Database ID saved to .env file\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Using existing Notion database: %s\n", *notionDatabaseID)
		}

		// Clip all headlines and their related articles
		fmt.Fprintln(os.Stderr, "\nClipping articles...")
		clippedCount := 0
		for _, h := range headlines {
			if err := clipper.ClipHeadlineWithRelated(ctx, h); err != nil {
				fmt.Fprintf(os.Stderr, "WARN: failed to clip headline '%s': %v\n", h.Title, err)
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

// handleEmailSend handles email sending flow
func handleEmailSend(emailDaysBack int) {
	fmt.Fprintln(os.Stderr, "\n========================================")
	fmt.Fprintln(os.Stderr, "📧 Sending Email Summary")
	fmt.Fprintln(os.Stderr, "========================================")

	// Validate environment variables
	emailFrom := os.Getenv("EMAIL_FROM")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	emailTo := os.Getenv("EMAIL_TO")
	notionToken := os.Getenv("NOTION_TOKEN")
	notionDatabaseID := os.Getenv("NOTION_DATABASE_ID")

	if emailFrom == "" {
		fatalf("ERROR: EMAIL_FROM environment variable is required for email sending")
	}
	if emailPassword == "" {
		fatalf("ERROR: EMAIL_PASSWORD environment variable is required (use Gmail App Password)")
	}
	if emailTo == "" {
		fatalf("ERROR: EMAIL_TO environment variable is required")
	}
	if notionToken == "" {
		fatalf("ERROR: NOTION_TOKEN environment variable is required to fetch headlines")
	}
	if notionDatabaseID == "" {
		fatalf("ERROR: NOTION_DATABASE_ID environment variable is required (run with -notionClip first to create database)")
	}

	// Create Notion clipper
	clipper, err := NewNotionClipper(notionToken, notionDatabaseID)
	if err != nil {
		fatalf("ERROR creating Notion clipper: %v", err)
	}

	// Fetch headlines from Notion DB
	ctx := context.Background()
	notionHeadlines, err := clipper.FetchRecentHeadlines(ctx, emailDaysBack)
	if err != nil {
		fatalf("ERROR fetching headlines from Notion: %v", err)
	}

	if len(notionHeadlines) == 0 {
		fmt.Fprintf(os.Stderr, "⚠️  No headlines found in the last %d days\n", emailDaysBack)
		fmt.Fprintln(os.Stderr, "========================================")
		return
	}

	fmt.Fprintf(os.Stderr, "Fetched %d headlines from Notion (last %d days)\n", len(notionHeadlines), emailDaysBack)

	// Create email sender
	sender, err := NewEmailSender(emailFrom, emailPassword, emailTo)
	if err != nil {
		fatalf("ERROR creating email sender: %v", err)
	}

	// Send email
	if err := sender.SendHeadlinesSummary(ctx, notionHeadlines); err != nil {
		fatalf("ERROR sending email: %v", err)
	}

	fmt.Fprintln(os.Stderr, "✅ Email sent successfully")
	fmt.Fprintf(os.Stderr, "   From: %s\n", emailFrom)
	fmt.Fprintf(os.Stderr, "   To: %s\n", emailTo)
	fmt.Fprintln(os.Stderr, "========================================")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// appendToEnvFile appends or updates a key=value pair in an .env file
func appendToEnvFile(filename, key, value string) error {
	// Read existing .env file if it exists
	content := ""
	data, err := os.ReadFile(filename)
	if err == nil {
		content = string(data)
	}

	// Check if key already exists
	lines := strings.Split(content, "\n")
	keyExists := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") || strings.HasPrefix(line, "#"+key+"=") {
			lines[i] = key + "=" + value
			keyExists = true
			break
		}
	}

	// If key doesn't exist, append it
	if !keyExists {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		lines = append(lines, key+"="+value)
	}

	// Write back to file
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}
