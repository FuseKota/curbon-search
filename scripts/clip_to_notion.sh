#!/bin/bash
# Notion クリッピング実行スクリプト

set -e

# .envファイルが存在するか確認
if [ ! -f .env ]; then
    echo "ERROR: .env file not found"
    echo ""
    echo "Create .env file with the following content:"
    echo ""
    cat << 'EOF'
# OpenAI API
OPENAI_API_KEY=sk-your-key-here

# Notion Integration
NOTION_TOKEN=secret_your-token-here

# Notion Page ID (optional for new DB creation)
NOTION_PAGE_ID=your-page-id-here

# Notion Database ID (optional for existing DB)
# NOTION_DATABASE_ID=your-database-id-here
EOF
    exit 1
fi

# .envファイルを読み込む
echo "Loading .env file..."
set -a
source .env
set +a

# 環境変数チェック
if [ -z "$NOTION_TOKEN" ]; then
    echo "ERROR: NOTION_TOKEN is not set in .env"
    exit 1
fi

# ビルド確認
if [ ! -f "carbon-relay" ]; then
    echo "Building carbon-relay..."
    go build -o carbon-relay ./cmd/pipeline
fi

# デフォルト設定
HEADLINES_FILE="${1:-search_results.json}"
QUERIES_PER_HEADLINE="${2:-0}"

echo "========================================"
echo "📎 Notion Clipping Tool"
echo "========================================"
echo "Headlines file: $HEADLINES_FILE"
echo "Queries per headline: $QUERIES_PER_HEADLINE"
echo ""

# Notion Database IDが設定されているかチェック
if [ -n "$NOTION_DATABASE_ID" ]; then
    echo "Using existing Notion Database: $NOTION_DATABASE_ID"
    echo ""

    ./carbon-relay \
        -headlines="$HEADLINES_FILE" \
        -queriesPerHeadline="$QUERIES_PER_HEADLINE" \
        -notionClip \
        -notionDatabaseID="$NOTION_DATABASE_ID"
else
    if [ -z "$NOTION_PAGE_ID" ]; then
        echo "ERROR: Either NOTION_DATABASE_ID or NOTION_PAGE_ID must be set in .env"
        exit 1
    fi

    echo "Creating new Notion Database under page: $NOTION_PAGE_ID"
    echo ""

    ./carbon-relay \
        -headlines="$HEADLINES_FILE" \
        -queriesPerHeadline="$QUERIES_PER_HEADLINE" \
        -notionClip \
        -notionPageID="$NOTION_PAGE_ID"
fi

echo ""
echo "✅ Clipping completed!"
