#!/bin/bash
# フルパイプライン：ヘッドライン収集 → メール送信

set -e

echo "========================================="
echo "🚀 carbon-relay フルパイプライン"
echo "========================================="
echo ""
echo "このスクリプトは："
echo "  1. ヘッドライン収集"
echo "  2. メール送信（設定されている場合）"
echo "を一気に実行します。"
echo ""

# パラメータ
SOURCE="${1:-all-free}"
COUNT="${2:-10}"
OUTPUT="full_pipeline_output.json"

echo "⚙️  設定:"
echo "  - ソース        : $SOURCE"
echo "  - 見出し数      : $COUNT"
echo "  - 出力ファイル  : $OUTPUT"
echo ""

# ビルド確認
if [ ! -f "pipeline" ]; then
    echo "🔨 ビルド中..."
    go build -o pipeline ./cmd/pipeline
    echo "✅ ビルド完了"
    echo ""
fi

# Step 1: ヘッドライン収集
echo "========================================="
echo "📰 Step 1: ヘッドライン収集"
echo "========================================="
echo ""

./pipeline \
  -sources="$SOURCE" \
  -perSource="$COUNT" \
  -queriesPerHeadline=0 \
  -out="$OUTPUT" 2>&1 | grep -E "INFO:|WARN:" || true

echo ""
echo "✅ 収集完了: $OUTPUT"
echo ""

# Step 2: サマリー表示
echo "========================================="
echo "📊 Step 2: サマリー"
echo "========================================="
echo ""

TOTAL=$(cat "$OUTPUT" | jq '. | length')
echo "📈 見出し総数: $TOTAL 件"
echo ""

# ソース別
echo "📂 ソース別:"
cat "$OUTPUT" | jq -r '.[].source' | sort | uniq -c | awk '{printf "  - %-20s: %s 件\n", $2, $1}'
echo ""

# 最新3件
echo "🆕 最新3件:"
cat "$OUTPUT" | jq -r 'limit(3;.[]) | "  [\(.source)] \(.title)"'
echo ""

echo "========================================="
echo "✅ パイプライン完了"
echo "========================================="
echo ""
echo "📄 結果ファイル: $OUTPUT"
echo ""
echo "💡 次のステップ:"
echo ""
echo "  # 詳細確認"
echo "  cat $OUTPUT | jq"
echo ""
echo "  # Notionにクリップ"
echo "  ./pipeline -sources=$SOURCE -perSource=$COUNT -queriesPerHeadline=0 -notionClip"
echo ""
echo "  # メール送信"
echo "  ./pipeline -sources=$SOURCE -perSource=$COUNT -queriesPerHeadline=0 -sendEmail"
echo ""
