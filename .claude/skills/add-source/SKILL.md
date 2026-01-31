---
name: add-source
description: Guide for adding a new carbon news source
invocation: /add-source <source-url>
model: claude-opus-4-5
allowed-tools:
  - WebFetch
  - Read
  - Grep
---

# Add New Carbon News Source

Adding source: **{{ARGUMENTS}}**

## Step 1: Research Website Structure

1. **Visit the URL**: {{ARGUMENTS}}
2. **Check for WordPress API**:
   ```bash
   curl {{ARGUMENTS}}/wp-json/wp/v2/posts?per_page=1
   ```
   - If returns JSON → Use WordPress API approach
   - If 404 → Use HTML scraping

3. **Inspect HTML structure** (if not WordPress):
   - Article container selector
   - Title selector
   - URL selector
   - Date selector
   - Excerpt selector

## Step 2: Determine Implementation Approach

### Option A: WordPress REST API
**Best for**: Sites with `/wp-json/` endpoint

**Template**:
```go
func collectHeadlinesSourceName(maxArticles int) ([]Article, error) {
    url := "https://example.com/wp-json/wp/v2/posts?per_page=" + strconv.Itoa(maxArticles)
    // ... WordPress API logic
}
```

### Option B: HTML Scraping
**Best for**: Custom HTML structure

**Template**:
```go
func collectHeadlinesSourceName(maxArticles int) ([]Article, error) {
    doc, err := fetchAndParse("https://example.com/news")
    // ... goquery selectors
}
```

## Step 3: Implement in `cmd/pipeline/headlines.go`

1. Add function `collectHeadlinesSourceName()`
2. Add to `sourceMap` in `main.go`
3. Handle date parsing (common formats below)

### Common Date Formats

- ISO 8601: `2006-01-02T15:04:05Z07:00`
- Japanese: `2006年01月02日`
- Slash: `01/02/2006`
- Dash: `2006-01-02`

## Step 4: Test

```bash
./pipeline -sources=sourcename -perSource=5 -queriesPerHeadline=0 -out=/tmp/test_new.json
cat /tmp/test_new.json | jq '.'
```

## Step 5: Document

Update:
- `docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md` (Section 3)
- `README.md` (source list)
- `CLAUDE.md` (if special handling needed)

## Checklist

- [ ] Source function implemented
- [ ] Added to sourceMap
- [ ] Date parsing works
- [ ] Excerpt extracted correctly
- [ ] Tested with 5+ articles
- [ ] Documentation updated
- [ ] Commit created with proper format
