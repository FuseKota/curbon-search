#!/bin/bash

# Load .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
fi

echo "Clipping articles from all 4 free sources to Notion..."
if [ -n "$NOTION_DATABASE_ID" ]; then
    echo "Using existing database ID: $NOTION_DATABASE_ID"
else
    echo "Will create new database (NOTION_DATABASE_ID not set in .env)"
fi
echo ""

if [ -n "$NOTION_DATABASE_ID" ]; then
    # Use existing database
    ./carbon-relay \
      -sources=carboncredits.jp,carbonherald,climatehomenews,carboncredits.com \
      -perSource=5 \
      -queriesPerHeadline=0 \
      -notionClip \
      -notionDatabaseID=$NOTION_DATABASE_ID
else
    # Create new database
    if [ -z "$NOTION_PAGE_ID" ]; then
        echo "ERROR: NOTION_PAGE_ID is required to create new database"
        exit 1
    fi
    ./carbon-relay \
      -sources=carboncredits.jp,carbonherald,climatehomenews,carboncredits.com \
      -perSource=5 \
      -queriesPerHeadline=0 \
      -notionClip \
      -notionPageID=$NOTION_PAGE_ID
fi
