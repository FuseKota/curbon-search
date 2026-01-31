#!/bin/bash
# relatedFree（一次情報・無料資料）確認ツール

if [ -z "$1" ]; then
    echo "使い方: $0 <output.json>"
    echo ""
    echo "例:"
    echo "  $0 with_related.json"
    echo "  $0 output.json"
    exit 1
fi

FILE="$1"

if [ ! -f "$FILE" ]; then
    echo "❌ エラー: $FILE が見つかりません"
    exit 1
fi

echo "========================================="
echo "🔍 relatedFree（一次情報）確認レポート"
echo "========================================="
echo ""
echo "📄 ファイル: $FILE"
echo ""

# 基本統計
TOTAL_HEADLINES=$(cat "$FILE" | jq '. | length')
WITH_RELATED=$(cat "$FILE" | jq '[.[] | select(.relatedFree != null and (.relatedFree | length) > 0)] | length')
WITHOUT_RELATED=$(cat "$FILE" | jq '[.[] | select(.relatedFree == null or (.relatedFree | length) == 0)] | length')
TOTAL_RELATED=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[]] | length')

echo "📊 基本統計"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  見出し総数          : $TOTAL_HEADLINES 件"
if [ $TOTAL_HEADLINES -gt 0 ]; then
    PERCENT=$(awk "BEGIN {printf \"%.1f\", ($WITH_RELATED/$TOTAL_HEADLINES)*100}")
    echo "  関連記事あり        : $WITH_RELATED 件 ($PERCENT%)"
else
    echo "  関連記事あり        : $WITH_RELATED 件"
fi
echo "  関連記事なし        : $WITHOUT_RELATED 件"
echo "  関連記事総数        : $TOTAL_RELATED 件"
if [ $WITH_RELATED -gt 0 ]; then
    AVG=$(awk "BEGIN {printf \"%.1f\", ($TOTAL_RELATED/$WITH_RELATED)}")
    echo "  平均件数/見出し     : $AVG 件"
fi
echo ""

# 一次情報の判定
echo "📂 一次情報の種類"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# PDFファイル
PDF_COUNT=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.url | test("\\.pdf$"; "i"))] | length')
echo "  📄 PDF              : $PDF_COUNT 件"

# 政府サイト (.gov, .go.kr, europa.eu等)
GOV_COUNT=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.url | test("\\.(gov|go\\.kr|gov\\.uk|gov\\.au|gov\\.cn|europa\\.eu)"; "i"))] | length')
echo "  🏛️  政府サイト       : $GOV_COUNT 件"

# NGO/国際機関
NGO_COUNT=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.url | test("unfccc\\.int|icvcm\\.org|iea\\.org|carbonmarketwatch|forest-trends"; "i"))] | length')
echo "  🌍 NGO/国際機関     : $NGO_COUNT 件"

# 企業IR
IR_COUNT=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.url | test("/investor|/ir/"; "i"))] | length')
echo "  💼 企業IR           : $IR_COUNT 件"

# その他
OTHER_COUNT=$(awk "BEGIN {print ($TOTAL_RELATED - $PDF_COUNT - $GOV_COUNT - $NGO_COUNT - $IR_COUNT)}")
echo "  📰 その他           : $OTHER_COUNT 件"

PRIMARY_COUNT=$(awk "BEGIN {print ($PDF_COUNT + $GOV_COUNT + $NGO_COUNT + $IR_COUNT)}")
if [ $TOTAL_RELATED -gt 0 ]; then
    PRIMARY_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($PRIMARY_COUNT/$TOTAL_RELATED)*100}")
    echo ""
    echo "  ✅ 一次情報率       : $PRIMARY_PERCENT%"
fi
echo ""

# スコア分布
echo "📈 スコア分布"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
HIGH_SCORE=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.score >= 0.7)] | length')
MID_SCORE=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.score >= 0.5 and .score < 0.7)] | length')
LOW_SCORE=$(cat "$FILE" | jq '[.[].relatedFree // [] | .[] | select(.score < 0.5)] | length')

echo "  🟢 高スコア (≥0.7)  : $HIGH_SCORE 件"
echo "  🟡 中スコア (0.5-0.7): $MID_SCORE 件"
echo "  🔴 低スコア (<0.5)  : $LOW_SCORE 件"
echo ""

# 詳細表示
echo "========================================="
echo "📋 詳細レポート"
echo "========================================="
echo ""

cat "$FILE" | jq -r '.[] |
if .relatedFree != null and (.relatedFree | length) > 0 then
  "【見出し】\(.title)\n" +
  "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
  "🔗 \(.url)\n" +
  "📊 関連記事: \(.relatedFree | length) 件\n\n" +
  (.relatedFree | to_entries | map(
    "  [\(.key + 1)] \(.value.title)\n" +
    "      🔗 \(.value.url)\n" +
    "      📊 スコア: \(.value.score | tostring | .[0:4])\n" +
    "      📝 理由: \(.value.reason)\n"
  ) | join("\n")) +
  "\n"
else
  "【見出し】\(.title)\n" +
  "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
  "🔗 \(.url)\n" +
  "❌ 関連記事なし\n\n"
end'

echo ""
echo "========================================="
echo "✅ レポート完了"
echo "========================================="
echo ""
echo "💡 便利コマンド:"
echo ""
echo "  # 一次情報のみ抽出（PDF）"
echo "  cat $FILE | jq '[.[].relatedFree // [] | .[] | select(.url | test(\"\\.pdf$\"))]'"
echo ""
echo "  # 政府サイトのみ抽出"
echo "  cat $FILE | jq '[.[].relatedFree // [] | .[] | select(.url | test(\"\\.gov\"))]'"
echo ""
echo "  # 高スコアのみ抽出"
echo "  cat $FILE | jq '[.[].relatedFree // [] | .[] | select(.score >= 0.7)]'"
echo ""
echo "  # 関連記事なしの見出し一覧"
echo "  cat $FILE | jq -r '.[] | select(.relatedFree == null or (.relatedFree | length) == 0) | .title'"
echo ""
