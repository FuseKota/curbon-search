#!/bin/bash
# carbon-relay 実行サンプルスクリプト

set -e  # エラー時に停止

echo "========================================="
echo "carbon-relay 実行サンプル"
echo "========================================="
echo ""

# ビルド
echo "🔨 ビルド中..."
go build -o pipeline ./cmd/pipeline
echo "✅ ビルド完了"
echo ""

# 出力ディレクトリ作成
mkdir -p outputs
echo "📁 出力ディレクトリ作成: outputs/"
echo ""

# ========================================
# 例1: クイックテスト（全ソース、少数）
# ========================================
echo "========================================="
echo "例1: クイックテスト（全ソース 各2件）"
echo "========================================="
./pipeline \
  -sources=all-free \
  -perSource=2 \
  -queriesPerHeadline=0 \
  -out=outputs/quick_test.json

echo "✅ 完了: outputs/quick_test.json"
echo ""
sleep 1

# ========================================
# 例2: 標準実行（全ソース10件ずつ）
# ========================================
echo "========================================="
echo "例2: 標準実行（全ソース 各10件）"
echo "========================================="
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=outputs/standard_output.json

echo "✅ 完了: outputs/standard_output.json"
echo ""
sleep 1

# ========================================
# 例3: 日本ソースのみ
# ========================================
echo "========================================="
echo "例3: 日本ソースのみ"
echo "========================================="
./pipeline \
  -sources=jri,env-ministry,meti,pwc-japan,mizuho-rt,jpx,carboncredits.jp \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=outputs/japan_sources.json

echo "✅ 完了: outputs/japan_sources.json"
echo ""
sleep 1

# ========================================
# 例4: 欧州ソースのみ
# ========================================
echo "========================================="
echo "例4: 欧州・国際ソースのみ"
echo "========================================="
./pipeline \
  -sources=sandbag,carbon-brief,icap,ieta,politico-eu \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=outputs/europe_sources.json

echo "✅ 完了: outputs/europe_sources.json"
echo ""
sleep 1

# ========================================
# 例5: デバッグモード
# ========================================
echo "========================================="
echo "例5: デバッグモード（詳細ログ出力）"
echo "========================================="
DEBUG_SCRAPING=1 ./pipeline \
  -sources=carbonherald \
  -perSource=2 \
  -queriesPerHeadline=0 \
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
        echo "📄 $(basename $file): $headline_count 件"
    fi
done

echo ""
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
echo ""
