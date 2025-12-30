#!/bin/bash
# ヘッドライン確認ツール

if [ -z "$1" ]; then
    echo "使い方: $0 <headlines.json>"
    echo ""
    echo "例:"
    echo "  $0 headlines.json"
    echo "  $0 latest_headlines.json"
    exit 1
fi

FILE="$1"

if [ ! -f "$FILE" ]; then
    echo "❌ エラー: $FILE が見つかりません"
    exit 1
fi

echo "========================================="
echo "📰 ヘッドライン確認ツール"
echo "========================================="
echo ""
echo "📄 ファイル: $FILE"
echo ""

# 総数
TOTAL=$(cat "$FILE" | jq '. | length')
echo "📊 総件数: $TOTAL 件"
echo ""

# ソース別集計
echo "========================================="
echo "📂 ソース別内訳"
echo "========================================="
cat "$FILE" | jq -r '.[].source' | sort | uniq -c | awk '{printf "  %-20s: %s 件\n", $2, $1}'
echo ""

# 最新5件のタイトル表示
echo "========================================="
echo "🆕 最新5件"
echo "========================================="
cat "$FILE" | jq -r 'limit(5;.[]) | "[\(.source)] \(.title)"' | nl
echo ""

# 全タイトル一覧（番号付き）
echo "========================================="
echo "📋 全タイトル一覧"
echo "========================================="
cat "$FILE" | jq -r '.[] | "[\(.source)] \(.title)"' | nl
echo ""

# URL一覧（コピー用）
echo "========================================="
echo "🔗 URL一覧"
echo "========================================="
cat "$FILE" | jq -r '.[].url'
echo ""

# 詳細表示オプション
read -p "詳細情報を表示しますか？ (y/N): " DETAIL
if [[ "$DETAIL" =~ ^[Yy]$ ]]; then
    echo ""
    echo "========================================="
    echo "📝 詳細情報"
    echo "========================================="
    cat "$FILE" | jq -r '.[] | "
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📰 タイトル: \(.title)
🏢 ソース  : \(.source)
🔗 URL     : \(.url)
"'
fi

echo ""
echo "========================================="
echo "✅ 確認完了"
echo "========================================="
echo ""
echo "💡 ヒント:"
echo "  - タイトルのみ: cat $FILE | jq -r '.[].title'"
echo "  - URLのみ: cat $FILE | jq -r '.[].url'"
echo "  - JSONフォーマット: cat $FILE | jq"
echo "  - 特定ワード検索: cat $FILE | jq '.[] | select(.title | contains(\"climate\"))'"
echo ""
