---
name: check-env
description: Check environment variables
---

Verify environment setup:

```bash
echo "Checking .env file..."
cat .env | grep -v "PASSWORD\|API_KEY" || echo "No .env file found"

echo ""
echo "Required for Mode 2:"
echo ""
echo "Required for Notion:"
echo "- NOTION_API_KEY: ${NOTION_API_KEY:+SET}"
echo "- NOTION_PAGE_ID: ${NOTION_PAGE_ID:+SET}"
```
