#!/bin/bash
# ヘッドライン収集専用スクリプト

set -e

echo "========================================="
echo "ヘッドライン収集"
echo "========================================="
echo ""

# ビルド
if [ ! -f "pipeline" ]; then
    echo "🔨 ビルド中..."
    go build -o pipeline ./cmd/pipeline
    echo "✅ ビルド完了"
    echo ""
fi

# 出力ディレクトリ作成
mkdir -p headlines_output
echo "📁 出力ディレクトリ作成: headlines_output/"
echo ""

# ========================================
# 全無料ソースから収集
# ========================================
echo "========================================="
echo "全無料ソースからヘッドライン収集"
echo "========================================="
./pipeline \
  -sources=all-free \
  -perSource=10 \
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
