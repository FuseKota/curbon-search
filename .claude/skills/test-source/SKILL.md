---
name: test-source
description: Test a single carbon news source with debugging
invocation: /test-source <source-name>
model: claude-opus-4-5
allowed-tools:
  - Bash(./pipeline:*)
  - Bash(DEBUG_*:*)
  - Read
---

# Test Carbon News Source

Test the carbon news source: **{{ARGUMENTS}}**

## Steps

1. **Run basic test** (3 articles, no search):
   ```bash
   ./pipeline -sources={{ARGUMENTS}} -perSource=3 -queriesPerHeadline=0 -out=/tmp/test_{{ARGUMENTS}}.json
   ```

2. **Check for errors**:
   - If errors appear, enable debug mode

3. **Debug mode** (if needed):
   ```bash
   DEBUG_SCRAPING=1 ./pipeline -sources={{ARGUMENTS}} -perSource=1 -queriesPerHeadline=0
   ```

4. **Verify output**:
   ```bash
   cat /tmp/test_{{ARGUMENTS}}.json | jq '.[0]'
   ```

5. **Check implementation**:
   - Read `cmd/pipeline/headlines.go`
   - Find `collectHeadlines{{ARGUMENTS}}()` function
   - Verify selectors and URL patterns

## Common Issues

- **No headlines found**: Selector changed, check website structure
- **Date parsing error**: Date format changed, update regex
- **Japanese encoding**: Use `\xE2\x80\xA6` for ellipsis
