#!/bin/bash
# carbon-relay 実行サンプルスクリプト

set -e  # エラー時に停止

echo "========================================="
echo "carbon-relay 実行サンプル"
echo "========================================="
echo ""

# OPENAI_API_KEY チェック
if [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ ERROR: OPENAI_API_KEY が設定されていません"
    echo ""
    echo "以下のコマンドでAPIキーを設定してください："
    echo "  export OPENAI_API_KEY=\"sk-...\""
    echo ""
    exit 1
fi

echo "✅ OPENAI_API_KEY が設定されています"
echo ""

# ビルド
echo "🔨 ビルド中..."
go build -o carbon-relay ./cmd/pipeline
echo "✅ ビルド完了"
echo ""

# 出力ディレクトリ作成
mkdir -p outputs
echo "📁 出力ディレクトリ作成: outputs/"
echo ""

# ========================================
# 例1: クイックテスト（2件のみ）
# ========================================
echo "========================================="
echo "例1: クイックテスト（Carbon Pulse 2件）"
echo "========================================="
./carbon-relay \
  -sources=carbonpulse \
  -perSource=2 \
  -queriesPerHeadline=2 \
  -resultsPerQuery=8 \
  -topK=3 \
  -minScore=0.25 \
  -out=outputs/quick_test.json

echo "✅ 完了: outputs/quick_test.json"
echo ""
sleep 2

# ========================================
# 例2: 標準実行（両ソース10件ずつ）
# ========================================
echo "========================================="
echo "例2: 標準実行（Carbon Pulse + QCI 各10件）"
echo "========================================="
./carbon-relay \
  -sources=carbonpulse,qci \
  -perSource=10 \
  -queriesPerHeadline=3 \
  -resultsPerQuery=12 \
  -topK=3 \
  -minScore=0.25 \
  -saveFree=outputs/candidates_pool.json \
  -out=outputs/standard_output.json

echo "✅ 完了:"
echo "  - outputs/standard_output.json"
echo "  - outputs/candidates_pool.json"
echo ""
sleep 2

# ========================================
# 例3: 高品質モード（厳格なフィルタ）
# ========================================
echo "========================================="
echo "例3: 高品質モード（厳格なフィルタ）"
echo "========================================="
./carbon-relay \
  -sources=carbonpulse \
  -perSource=5 \
  -queriesPerHeadline=4 \
  -resultsPerQuery=15 \
  -topK=5 \
  -minScore=0.35 \
  -strictMarket=true \
  -out=outputs/high_quality.json

echo "✅ 完了: outputs/high_quality.json"
echo ""
sleep 2

# ========================================
# 例4: 探索的モード（低スコア閾値）
# ========================================
echo "========================================="
echo "例4: 探索的モード（低スコア閾値、多数の候補）"
echo "========================================="
./carbon-relay \
  -sources=carbonpulse \
  -perSource=3 \
  -queriesPerHeadline=5 \
  -resultsPerQuery=20 \
  -topK=10 \
  -minScore=0.15 \
  -strictMarket=false \
  -out=outputs/exploratory.json

echo "✅ 完了: outputs/exploratory.json"
echo ""
sleep 2

# ========================================
# 例5: デバッグモード
# ========================================
echo "========================================="
echo "例5: デバッグモード（詳細ログ出力）"
echo "========================================="
DEBUG_OPENAI=1 ./carbon-relay \
  -sources=carbonpulse \
  -perSource=1 \
  -queriesPerHeadline=2 \
  -resultsPerQuery=8 \
  -topK=3 \
  -out=outputs/debug_output.json \
  2>&1 | tee outputs/debug.log

echo "✅ 完了:"
echo "  - outputs/debug_output.json"
echo "  - outputs/debug.log（デバッグログ）"
echo ""

# ========================================
# 結果サマリー表示
# ========================================
echo "========================================="
echo "📊 結果サマリー"
echo "========================================="
echo ""

for file in outputs/*.json; do
    if [ -f "$file" ]; then
        headline_count=$(cat "$file" | grep -c '"isHeadline": true' || echo "0")
        related_count=$(cat "$file" | grep -c '"relatedFree":' || echo "0")
        echo "📄 $(basename $file)"
        echo "   見出し数: $headline_count"
        echo "   relatedFree付き: $related_count"
        echo ""
    fi
done

echo "========================================="
echo "✅ すべてのサンプル実行完了"
echo "========================================="
echo ""
echo "結果ファイル:"
ls -lh outputs/
echo ""
echo "💡 ヒント："
echo "  - JSON結果を確認: cat outputs/standard_output.json | jq"
echo "  - デバッグログ確認: less outputs/debug.log"
echo "  - 候補プール確認: cat outputs/candidates_pool.json | jq '.[] | .url'"
echo ""
