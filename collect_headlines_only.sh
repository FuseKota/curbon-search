#!/bin/bash
# ヘッドライン収集専用スクリプト（OpenAI API不要）

set -e

echo "========================================="
echo "ヘッドライン収集（OpenAI API不要）"
echo "========================================="
echo ""

# ビルド
if [ ! -f "carbon-relay" ]; then
    echo "🔨 ビルド中..."
    go build -o carbon-relay ./cmd/pipeline
    echo "✅ ビルド完了"
    echo ""
fi

# 出力ディレクトリ作成
mkdir -p headlines_output
echo "📁 出力ディレクトリ作成: headlines_output/"
echo ""

# ========================================
# Carbon Pulse のみ収集
# ========================================
echo "========================================="
echo "1. Carbon Pulse ヘッドライン収集"
echo "========================================="
./carbon-relay \
  -sources=carbonpulse \
  -perSource=30 \
  -queriesPerHeadline=0 \
  -out=headlines_output/carbonpulse_headlines.json

echo "✅ 完了: headlines_output/carbonpulse_headlines.json"
echo ""

# ========================================
# QCI のみ収集
# ========================================
echo "========================================="
echo "2. QCI ヘッドライン収集"
echo "========================================="
./carbon-relay \
  -sources=qci \
  -perSource=30 \
  -queriesPerHeadline=0 \
  -out=headlines_output/qci_headlines.json

echo "✅ 完了: headlines_output/qci_headlines.json"
echo ""

# ========================================
# 両方収集
# ========================================
echo "========================================="
echo "3. Carbon Pulse + QCI ヘッドライン収集"
echo "========================================="
./carbon-relay \
  -sources=carbonpulse,qci \
  -perSource=30 \
  -queriesPerHeadline=0 \
  -out=headlines_output/all_headlines.json

echo "✅ 完了: headlines_output/all_headlines.json"
echo ""

# ========================================
# 結果サマリー
# ========================================
echo "========================================="
echo "📊 結果サマリー"
echo "========================================="
echo ""

for file in headlines_output/*.json; do
    if [ -f "$file" ]; then
        count=$(cat "$file" | grep -c '"isHeadline": true' || echo "0")
        echo "📄 $(basename $file): $count 件"
    fi
done

echo ""
echo "========================================="
echo "✅ ヘッドライン収集完了"
echo "========================================="
echo ""
echo "結果ファイル:"
ls -lh headlines_output/
echo ""
echo "💡 ヒント："
echo "  - JSON確認: cat headlines_output/all_headlines.json | jq"
echo "  - タイトル一覧: cat headlines_output/all_headlines.json | jq -r '.[].title'"
echo "  - URL一覧: cat headlines_output/all_headlines.json | jq -r '.[].url'"
echo ""
echo "📝 注意："
echo "  - このモードではrelatedFreeは付きません（検索なし）"
echo "  - 関連記事を取得したい場合は run_examples.sh を使用してください"
echo ""
