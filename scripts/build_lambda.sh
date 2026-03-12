#!/bin/bash
# =============================================================================
# Lambda関数ビルドスクリプト
# =============================================================================
#
# 使用方法:
#   ./scripts/build-lambda.sh
#
# 出力:
#   - dist/collect-headlines.zip
#   - dist/send-email.zip
#
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==================================="
echo "Carbon Relay Lambda Build Script"
echo "==================================="
echo ""

# ビルドディレクトリの作成
mkdir -p "$PROJECT_ROOT/dist"

# Lambda 1: collect-headlines のビルド
echo "[1/3] Building collect-headlines Lambda..."
cd "$PROJECT_ROOT"
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc \
    -o dist/bootstrap \
    ./cmd/lambda/collect/
cd dist
zip -j collect-headlines.zip bootstrap
rm bootstrap
echo "      -> dist/collect-headlines.zip"

# Lambda 2: send-email のビルド
echo "[2/3] Building send-email Lambda..."
cd "$PROJECT_ROOT"
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc \
    -o dist/bootstrap \
    ./cmd/lambda/email/
cd dist
zip -j send-email.zip bootstrap
rm bootstrap
echo "      -> dist/send-email.zip"

# Lambda 3: collect-exception のビルド
echo "[3/3] Building collect-exception Lambda..."
cd "$PROJECT_ROOT"
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc \
    -o dist/bootstrap \
    ./cmd/lambda/collect-exception/
cd dist
zip -j collect-exception.zip bootstrap
rm bootstrap
echo "      -> dist/collect-exception.zip"

echo ""
echo "==================================="
echo "Build complete!"
echo "==================================="
echo ""
echo "Output files:"
ls -lh "$PROJECT_ROOT/dist/"*.zip
echo ""
echo "Next steps:"
echo "  1. AWS Console > Lambda > 関数の作成"
echo "  2. ランタイム: Amazon Linux 2023 (provided.al2023)"
echo "  3. アーキテクチャ: arm64"
echo "  4. ZIPファイルをアップロード"
echo "  5. 環境変数を設定"
echo "  6. EventBridgeでスケジュール設定"
echo ""
