#!/bin/bash

# Load .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
fi

# Check if NOTION_TOKEN is set
if [ -z "$NOTION_TOKEN" ]; then
    echo "ERROR: NOTION_TOKEN is not set in .env file"
    exit 1
fi

if [ -z "$NOTION_PAGE_ID" ]; then
    echo "ERROR: NOTION_PAGE_ID is not set in .env file"
    exit 1
fi

echo "Environment variables loaded:"
echo "NOTION_TOKEN: ${NOTION_TOKEN:0:20}..."
echo "NOTION_PAGE_ID: $NOTION_PAGE_ID"
if [ -n "$NOTION_DATABASE_ID" ]; then
    echo "NOTION_DATABASE_ID: $NOTION_DATABASE_ID"
fi
echo ""

# Run the test
if [ -n "$NOTION_DATABASE_ID" ]; then
    # Use existing database if ID is set
    ./carbon-relay \
      -sources=carboncredits.jp \
      -perSource=1 \
      -queriesPerHeadline=0 \
      -notionClip \
      -notionDatabaseID=$NOTION_DATABASE_ID
else
    # Create new database
    ./carbon-relay \
      -sources=carboncredits.jp \
      -perSource=1 \
      -queriesPerHeadline=0 \
      -notionClip \
      -notionPageID=$NOTION_PAGE_ID
fi
