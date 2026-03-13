# Carbon Relay - コマンドクイックリファレンス

## 🚀 クイックスタート

### ビルド
```bash
go build -o pipeline ./cmd/pipeline
```

---

## 🟢 無料記事収集モード

### 基本的な収集
```bash
# 全ソースから10記事ずつ収集
./pipeline -sources=all-free -perSource=10 -out=free_articles.json

# デフォルト（-sourcesを省略すると全ソース）
./pipeline -perSource=10 -out=free_articles.json
```

### メール配信
```bash
# 無料記事のダイジェストメール送信
./pipeline -sendShortEmail
```

### Notion挿入
```bash
# 初回（データベース新規作成）
./pipeline -sources=all-free -perSource=10 -notionClip -notionPageID=YOUR_PAGE_ID

# 2回目以降（既存データベースに追加）
./pipeline -sources=all-free -perSource=10 -notionClip
```

### 日本市場のみ
```bash
./pipeline -sources=carboncredits.jp,jri,jpx,mizuho-rt,pwc-japan -perSource=10
```

### 時間フィルタ
```bash
# 過去24時間の記事のみ（日付なし記事は保持）
./pipeline -sources=all-free -perSource=10 -hoursBack=24
```

---

## 🧪 テストコマンド

### 単一ソーステスト
```bash
# PwC Japan（複雑な解析）
./pipeline -sources=pwc-japan -perSource=5 -out=/tmp/test_pwc.json

# Carbon Knowledge Hub
./pipeline -sources=carbon-knowledge-hub -perSource=5 -out=/tmp/test_ckh.json

# METI
./pipeline -sources=meti -perSource=5 -out=/tmp/test_meti.json
```

### 全ソーステスト
```bash
# 全ソースを一度にテスト
./pipeline -sources=all-free -perSource=2 -out=/tmp/all_sources_test.json

# ソース別件数を確認
cat /tmp/all_sources_test.json | jq 'group_by(.source) | map({source: .[0].source, count: length})'
```

---

## 🐛 デバッグコマンド

### スクレイピングのデバッグ
```bash
DEBUG_SCRAPING=1 ./pipeline -sources=pwc-japan -perSource=5
```

### HTML出力のデバッグ
```bash
DEBUG_HTML=1 ./pipeline -sources=carbon-knowledge-hub -perSource=1
```

### 完全デバッグ
```bash
DEBUG_SCRAPING=1 DEBUG_HTML=1 ./pipeline -sources=meti -perSource=1
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

### タイトル一覧表示
```bash
cat free_articles.json | jq '.[] | .title'
```

### 日付確認
```bash
cat free_articles.json | jq '[.[] | {source: .source, publishedAt: .publishedAt}]'
```

---

## 🔧 環境設定コマンド

### .envファイル作成
```bash
cat > .env << 'EOF'
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

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o pipeline-macos ./cmd/pipeline

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o pipeline-macos-arm64 ./cmd/pipeline

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
./pipeline -sources=all-free -perSource=10 2>&1 | grep ERROR
```

### タイミング計測
```bash
time ./pipeline -sources=all-free -perSource=10 -out=/tmp/timing_test.json
```

---

## 🎯 実用的な組み合わせ例

### 毎日の無料記事レビュー
```bash
#!/bin/bash
# daily_free_review.sh
./pipeline -sendShortEmail
```

### 毎日のNotion保存
```bash
#!/bin/bash
# daily_notion_save.sh
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -notionClip
```

### 日本市場の深堀り
```bash
#!/bin/bash
# japan_deep_dive.sh
./pipeline \
  -sources=carboncredits.jp,jri,jpx,mizuho-rt,pwc-japan \
  -perSource=20 \
  -out=japan_articles_$(date +%Y%m%d).json
```

---

## 🆘 トラブルシューティングコマンド

### Notion Database ID リセット
```bash
# .envからDATABASE_IDを削除
sed -i '' '/NOTION_DATABASE_ID/d' .env

# 再度初回セットアップを実行
./pipeline -sources=carbonherald -perSource=1 -notionClip -notionPageID=YOUR_PAGE_ID
```

### スクレイピング成功率チェック
```bash
# 各ソースを1記事ずつテスト
for source in carbonherald sandbag carbon-brief pwc-japan meti; do
  echo "Testing $source..."
  ./pipeline -sources=$source -perSource=1 2>&1 | grep -E "ERROR|collected"
done
```

### タイムアウト問題の確認
```bash
# 遅いソースのテスト（タイムアウト30秒）
time ./pipeline -sources=climatehomenews -perSource=1
```

---

## 📋 利用可能なソース一覧

### 日本市場
`carboncredits.jp`, `jri`, `pwc-japan`, `mizuho-rt`, `jpx`

### WordPress REST API
`carboncredits.jp`, `carbonherald`, `climatehomenews`, `carboncredits.com`, `sandbag`, `ecosystem-marketplace`, `carbon-brief`, `rmi`

### HTMLスクレイピング
`icap`, `ieta`, `energy-monitor`, `world-bank`, `newclimate`, `carbon-knowledge-hub`

### VCM認証団体
`verra`, `gold-standard`, `acr`, `car`

### 国際機関
`iisd`, `climate-focus`

### 地域ETS
`eu-ets`, `uk-ets`, `carb`, `rggi`, `australia-cer`

### RSSフィード
`politico-eu`, `euractiv`, `carbon-market-watch`, `un-news`

### 学術・研究
`arxiv`, `nature-comms`, `oies`, `iopscience`, `sciencedirect`

### CDR関連
`puro-earth`, `isometric`
