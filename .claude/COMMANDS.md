# Carbon Relay - コマンドクイックリファレンス

## 🚀 クイックスタート

### ビルド
```bash
go build -o pipeline ./cmd/pipeline
```

---

## 🟢 モード1: 無料記事収集モード

### 基本的な収集
```bash
# 全無料ソースから10記事ずつ収集
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -out=free_articles.json
```

### メール配信
```bash
# 無料記事を収集してメール送信
./pipeline -sources=all-free -perSource=15 -queriesPerHeadline=0 -sendEmail
```

### 日本市場のみ
```bash
./pipeline -sources=carboncredits-jp,jri,env-ministry,jpx,meti,mizuho-rt,pwc-japan -perSource=10 -queriesPerHeadline=0
```

---

## 🔵 モード2: 有料記事マッチングモード

### 基本的なマッチング
```bash
# 有料記事から無料記事を検索
./pipeline -sources=carbonpulse,qci -perSource=5 -queriesPerHeadline=3 -out=matched.json
```

### Notionクリッピング（初回）
```bash
# データベースを新規作成
./pipeline \
  -sources=carbonpulse,qci \
  -perSource=10 \
  -queriesPerHeadline=3 \
  -notionClip \
  -notionPageID=YOUR_PAGE_ID
```

### Notionクリッピング（2回目以降）
```bash
# 既存データベースに追加
./pipeline -sources=carbonpulse,qci -perSource=10 -queriesPerHeadline=3 -notionClip
```

### メール送信（Notionから）
```bash
# Notionにクリップした記事をメール送信
./pipeline -sendEmail -emailDaysBack=1
```

---

## 🧪 テストコマンド

### 単一ソーステスト
```bash
# Carbon Pulse
./pipeline -sources=carbonpulse -perSource=5 -queriesPerHeadline=0 -out=/tmp/test_carbonpulse.json

# PwC Japan（複雑な解析）
./pipeline -sources=pwc-japan -perSource=5 -queriesPerHeadline=0 -out=/tmp/test_pwc.json

# Carbon Knowledge Hub
./pipeline -sources=carbon-knowledge-hub -perSource=5 -queriesPerHeadline=0 -out=/tmp/test_ckh.json
```

### 全ソーステスト（ループ）
```bash
for source in carbonpulse qci sandbag carbon-brief climate-home carbon-herald carboncredits-com carbon-knowledge-hub; do
  echo "Testing: $source"
  ./pipeline -sources=$source -perSource=3 -queriesPerHeadline=0 -out=/tmp/test_${source}.json
done
```

---

## 🐛 デバッグコマンド

### OpenAI検索のデバッグ
```bash
DEBUG_OPENAI=1 ./pipeline -sources=carbonpulse -perSource=2 -queriesPerHeadline=1
```

### スクレイピングのデバッグ
```bash
DEBUG_SCRAPING=1 ./pipeline -sources=pwc-japan -perSource=5 -queriesPerHeadline=0
```

### HTML出力のデバッグ
```bash
DEBUG_HTML=1 ./pipeline -sources=carbon-knowledge-hub -perSource=1 -queriesPerHeadline=0
```

### 完全デバッグ
```bash
DEBUG_OPENAI_FULL=1 DEBUG_SCRAPING=1 DEBUG_HTML=1 ./pipeline -sources=carbonpulse -perSource=1 -queriesPerHeadline=1
```

---

## 📊 JSON出力の確認

### 記事数カウント
```bash
cat free_articles.json | jq 'length'
```

### ソース別カウント
```bash
cat free_articles.json | jq 'group_by(.source) | map({source: .[0].source, count: length})'
```

### 関連記事ありの件数
```bash
cat matched.json | jq 'map(select(.relatedFree | length > 0)) | length'
```

### 平均マッチングスコア
```bash
cat matched.json | jq '[.[].relatedFree[]?.score] | add / length'
```

### タイトル一覧表示
```bash
cat free_articles.json | jq '.[] | .title'
```

---

## 🔧 環境設定コマンド

### .envファイル作成
```bash
cat > .env << 'EOF'
OPENAI_API_KEY=sk-your-key-here
NOTION_API_KEY=secret_your-key-here
NOTION_PAGE_ID=your-page-id-here
EMAIL_FROM=your-email@gmail.com
EMAIL_PASSWORD=your-app-password
EMAIL_TO=recipient@example.com
EOF
```

### .env確認
```bash
cat .env | grep -v PASSWORD | grep -v API_KEY
```

---

## 📦 パッケージ管理

### 依存関係の更新
```bash
go get -u ./...
go mod tidy
```

### ビルド（各OS用）
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o pipeline-linux ./cmd/pipeline

# macOS
GOOS=darwin GOARCH=amd64 go build -o pipeline-macos ./cmd/pipeline

# Windows
GOOS=windows GOARCH=amd64 go build -o pipeline.exe ./cmd/pipeline
```

---

## 🔄 Git操作

### 状態確認
```bash
git status
git log --oneline -10
```

### コミット＆プッシュ
```bash
git add .
git commit -m "your commit message"
git push
```

---

## 📝 ログ確認

### エラーのみ表示
```bash
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 2>&1 | grep ERROR
```

### タイミング計測
```bash
time ./pipeline -sources=carbonpulse -perSource=10 -queriesPerHeadline=3
```

---

## 🎯 実用的な組み合わせ例

### 毎日の無料記事レビュー
```bash
#!/bin/bash
# daily_free_review.sh
./pipeline \
  -sources=all-free \
  -perSource=15 \
  -queriesPerHeadline=0 \
  -sendEmail
```

### 週次の有料記事マッチング
```bash
#!/bin/bash
# weekly_paid_matching.sh
./pipeline \
  -sources=carbonpulse,qci \
  -perSource=50 \
  -queriesPerHeadline=3 \
  -notionClip
```

### 日本市場の深堀り
```bash
#!/bin/bash
# japan_deep_dive.sh
./pipeline \
  -sources=carboncredits-jp,jri,env-ministry,jpx,meti,mizuho-rt,pwc-japan \
  -perSource=20 \
  -queriesPerHeadline=0 \
  -out=japan_articles_$(date +%Y%m%d).json
```

---

## 🆘 トラブルシューティングコマンド

### Notion Database ID リセット
```bash
# .envからDATABASE_IDを削除
sed -i '' '/NOTION_DATABASE_ID/d' .env

# 再度初回セットアップを実行
./pipeline -sources=carbonpulse -perSource=1 -queriesPerHeadline=0 -notionClip -notionPageID=YOUR_PAGE_ID
```

### OpenAI APIキーテスト
```bash
# 最小限のリクエストでテスト
./pipeline -sources=carbonpulse -perSource=1 -queriesPerHeadline=1 -out=/tmp/openai_test.json
```

### スクレイピング成功率チェック
```bash
# 各ソースを1記事ずつテスト
for source in carbonpulse qci sandbag carbon-brief pwc-japan; do
  echo "Testing $source..."
  ./pipeline -sources=$source -perSource=1 -queriesPerHeadline=0 2>&1 | grep -E "ERROR|Collected"
done
```
