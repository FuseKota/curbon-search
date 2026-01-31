#!/bin/bash
# ヘッドライン収集 & 即座に確認

set -e

echo "========================================="
echo "📰 ヘッドライン収集 & 確認ツール"
echo "========================================="
echo ""

# デフォルト設定
SOURCE="${1:-carbonpulse}"
COUNT="${2:-30}"
OUTPUT="collected_headlines.json"

echo "⚙️  設定:"
echo "  - ソース: $SOURCE"
echo "  - 件数  : $COUNT"
echo "  - 出力  : $OUTPUT"
echo ""

# ビルド確認
if [ ! -f "carbon-relay" ]; then
    echo "🔨 ビルド中..."
    go build -o carbon-relay ./cmd/pipeline
    echo "✅ ビルド完了"
    echo ""
fi

# ヘッドライン収集
echo "========================================="
echo "🔄 ヘッドライン収集中..."
echo "========================================="
DEBUG_SCRAPING=1 ./carbon-relay \
  -sources="$SOURCE" \
  -perSource="$COUNT" \
  -queriesPerHeadline=0 \
  -out="$OUTPUT" 2>&1 | grep -E "\[DEBUG\]|INFO:"

echo ""
echo "✅ 収集完了: $OUTPUT"
echo ""

# 即座に確認
echo "========================================="
echo "📊 収集結果サマリー"
echo "========================================="
echo ""

TOTAL=$(cat "$OUTPUT" | jq '. | length')
echo "📈 総件数: $TOTAL 件"
echo ""

# ソース別
echo "📂 ソース別:"
cat "$OUTPUT" | jq -r '.[].source' | sort | uniq -c | awk '{printf "  - %-20s: %s 件\n", $2, $1}'
echo ""

# 最新3件
echo "🆕 最新3件:"
cat "$OUTPUT" | jq -r 'limit(3;.[]) | "  [\(.source)] \(.title)"'
echo ""

# 詳細確認の提案
echo "========================================="
echo "💡 次のステップ"
echo "========================================="
echo ""
echo "詳細確認:"
echo "  ./view_headlines.sh $OUTPUT"
echo ""
echo "タイトル一覧:"
echo "  cat $OUTPUT | jq -r '.[].title'"
echo ""
echo "URL一覧:"
echo "  cat $OUTPUT | jq -r '.[].url'"
echo ""
echo "特定ワード検索（例：climate）:"
echo "  cat $OUTPUT | jq '.[] | select(.title | contains(\"climate\"))'"
echo ""
echo "ブラウザで開く（macOS）:"
echo "  cat $OUTPUT | jq -r '.[0].url' | xargs open"
echo ""
