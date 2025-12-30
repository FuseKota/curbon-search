#!/bin/bash
# フルパイプライン：ヘッドライン収集 → 一次情報検索 → 確認

set -e

echo "========================================="
echo "🚀 carbon-relay フルパイプライン"
echo "========================================="
echo ""
echo "このスクリプトは："
echo "  1. ヘッドライン収集"
echo "  2. 一次情報・無料資料の検索"
echo "  3. 結果の確認レポート"
echo "を一気に実行します。"
echo ""

# パラメータ
SOURCE="${1:-carbonpulse}"
COUNT="${2:-10}"
QUERIES="${3:-3}"
OUTPUT="full_pipeline_output.json"

echo "⚙️  設定:"
echo "  - ソース        : $SOURCE"
echo "  - 見出し数      : $COUNT"
echo "  - 検索クエリ数  : $QUERIES"
echo "  - 出力ファイル  : $OUTPUT"
echo ""

# ビルド確認
if [ ! -f "carbon-relay" ]; then
    echo "🔨 ビルド中..."
    go build -o carbon-relay ./cmd/pipeline
    echo "✅ ビルド完了"
    echo ""
fi

# Step 1: ヘッドライン収集 & 一次情報検索
echo "========================================="
echo "📰 Step 1: ヘッドライン収集 & 一次情報検索"
echo "========================================="
echo ""
echo "⏳ 処理中... (OpenAI API使用)"
echo ""

./carbon-relay \
  -sources="$SOURCE" \
  -perSource="$COUNT" \
  -queriesPerHeadline="$QUERIES" \
  -resultsPerQuery=12 \
  -topK=3 \
  -minScore=0.25 \
  -out="$OUTPUT" 2>&1 | grep -E "INFO:|WARN:" || true

echo ""
echo "✅ 収集完了: $OUTPUT"
echo ""

# Step 2: サマリー表示
echo "========================================="
echo "📊 Step 2: クイックサマリー"
echo "========================================="
echo ""

TOTAL=$(cat "$OUTPUT" | jq '. | length')
WITH_RELATED=$(cat "$OUTPUT" | jq '[.[] | select(.relatedFree != null and (.relatedFree | length) > 0)] | length')

echo "📈 見出し総数: $TOTAL 件"
echo "🔗 関連記事あり: $WITH_RELATED 件"
echo ""

if [ $WITH_RELATED -gt 0 ]; then
    echo "🆕 最新の成果（上位3件）:"
    cat "$OUTPUT" | jq -r 'limit(3; .[] | select(.relatedFree != null and (.relatedFree | length) > 0)) |
    "  【\(.title)】\n    関連記事: \(.relatedFree | length) 件"' | head -20
    echo ""
fi

# Step 3: 詳細レポート
echo "========================================="
echo "📋 Step 3: 詳細レポート生成中..."
echo "========================================="
echo ""

./check_related.sh "$OUTPUT"

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
echo "  ./view_headlines.sh $OUTPUT"
echo ""
echo "  # 一次情報のみ抽出（PDF）"
echo "  cat $OUTPUT | jq '[.[].relatedFree // [] | .[] | select(.url | test(\"\\.pdf$\"))]' > primary_pdfs.json"
echo ""
echo "  # 政府サイトのみ抽出"
echo "  cat $OUTPUT | jq '[.[].relatedFree // [] | .[] | select(.url | test(\"\\.gov\"))]' > government_sources.json"
echo ""
echo "  # 高スコアのみ抽出"
echo "  cat $OUTPUT | jq '[.[].relatedFree // [] | .[] | select(.score >= 0.7)]' > high_quality.json"
echo ""
