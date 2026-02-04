# ヘッドライン確認ガイド

## 🎯 目的

収集したヘッドライン情報を**見やすく確認**するためのツールとコマンド集です。

---

## 🚀 クイックスタート

### 方法1: 収集 & 即確認（最も簡単）

```bash
# 全無料ソースから10件ずつ収集して即確認
./scripts/collect_and_view.sh all-free 10

# 日本ソースのみから各20件収集
./scripts/collect_and_view.sh jri,env-ministry,meti 20

# 国際ソースから各30件収集
./scripts/collect_and_view.sh carbonherald,carbon-brief,sandbag 30
```

### 方法2: 既存ファイルを確認

```bash
# 詳細確認ツール
./scripts/view_headlines.sh headlines.json

# または別のファイル
./scripts/view_headlines.sh latest_headlines.json
```

---

## 📋 確認ツールの機能

### view_headlines.sh で表示される情報

```bash
./scripts/view_headlines.sh <ファイル名>
```

**表示内容：**
1. 📊 総件数
2. 📂 ソース別内訳
3. 🆕 最新5件のタイトル
4. 📋 全タイトル一覧（番号付き）
5. 🔗 URL一覧
6. 📝 詳細情報（オプション）

---

## 💡 便利なコマンド集

### タイトルのみ表示

```bash
cat headlines.json | jq -r '.[].title'
```

**出力例：**
```
Climate litigation marks 'turning point' in 2025
US DOE expands technologies eligible for 45V clean hydrogen tax credits
Hawaii court declines to block cruise ship climate levy
```

### URL一覧を取得

```bash
cat headlines.json | jq -r '.[].url'
```

**用途：** コピー＆ペースト、スクリプト処理

### 特定キーワードで検索

```bash
# "climate" を含む記事のみ
cat headlines.json | jq '.[] | select(.title | contains("climate"))'

# "carbon" を含む記事のタイトルのみ
cat headlines.json | jq -r '.[] | select(.title | contains("carbon")) | .title'

# "US" または "USA" を含む記事
cat headlines.json | jq '.[] | select(.title | test("US|USA"; "i"))'
```

### 件数カウント

```bash
# 総件数
cat headlines.json | jq '. | length'

# 特定ソースの件数
cat headlines.json | jq '[.[] | select(.source == "Carbon Herald")] | length'

# 特定キーワードを含む記事数
cat headlines.json | jq '[.[] | select(.title | contains("climate"))] | length'
```

### ソース別に分ける

```bash
# Carbon Herald のみ
cat headlines.json | jq '[.[] | select(.source == "Carbon Herald")]'

# JRI のみ
cat headlines.json | jq '[.[] | select(.source == "JRI")]'

# 日本ソースのみ
cat headlines.json | jq '[.[] | select(.source | test("JRI|環境省|METI|Mizuho"))]'
```

### CSV形式で出力

```bash
cat headlines.json | jq -r '.[] | [.source, .title, .url] | @csv' > headlines.csv
```

### Markdown形式で出力

```bash
cat headlines.json | jq -r '.[] | "- [\(.title)](\(.url))"' > headlines.md
```

### 最初/最後のN件を表示

```bash
# 最初の5件
cat headlines.json | jq '.[0:5]'

# 最後の5件
cat headlines.json | jq '.[-5:]'

# 6件目から10件目
cat headlines.json | jq '.[5:10]'
```

---

## 🌐 ブラウザで開く

### macOSの場合

```bash
# 最初の記事をブラウザで開く
cat headlines.json | jq -r '.[0].url' | xargs open

# すべての記事を開く（注意：大量のタブが開きます）
cat headlines.json | jq -r '.[].url' | xargs -n1 open
```

### Linuxの場合

```bash
# 最初の記事をブラウザで開く
cat headlines.json | jq -r '.[0].url' | xargs xdg-open

# または
cat headlines.json | jq -r '.[0].url' | xargs firefox
```

---

## 📊 統計情報の取得

### タイトルの文字数分布

```bash
cat headlines.json | jq -r '.[].title | length' | sort -n | uniq -c
```

### 最も長いタイトル

```bash
cat headlines.json | jq -r '.[] | "\(.title | length) \(.title)"' | sort -rn | head -1
```

### URLパターンの分析

```bash
# ドメイン別集計
cat headlines.json | jq -r '.[].url' | sed 's|https://||' | cut -d'/' -f1 | sort | uniq -c
```

---

## 🔍 高度な検索

### 複数条件で検索（AND）

```bash
# "climate" AND "litigation" を含む記事
cat headlines.json | jq '.[] | select(.title | contains("climate") and contains("litigation"))'
```

### 複数条件で検索（OR）

```bash
# "climate" OR "carbon" を含む記事
cat headlines.json | jq '.[] | select(.title | contains("climate") or contains("carbon"))'
```

### 正規表現で検索

```bash
# "EU" または "US" を含む記事（大文字小文字無視）
cat headlines.json | jq '.[] | select(.title | test("EU|US"; "i"))'

# 数字を含む記事
cat headlines.json | jq '.[] | select(.title | test("[0-9]+"))'
```

### タイトルの単語頻度分析

```bash
# 最も頻出する単語トップ10
cat headlines.json | jq -r '.[].title' | tr ' ' '\n' | tr '[:upper:]' '[:lower:]' | sort | uniq -c | sort -rn | head -10
```

---

## 📁 ファイル操作

### 複数のファイルをマージ

```bash
# 2つのファイルを結合
jq -s 'add' file1.json file2.json > merged.json

# 3つ以上
jq -s 'add' file1.json file2.json file3.json > merged.json
```

### 重複削除

```bash
# URLで重複削除
cat headlines.json | jq 'unique_by(.url)'
```

### ソート

```bash
# タイトルでソート（アルファベット順）
cat headlines.json | jq 'sort_by(.title)'

# URLでソート
cat headlines.json | jq 'sort_by(.url)'

# 逆順
cat headlines.json | jq 'sort_by(.title) | reverse'
```

---

## 🎨 カスタム表示フォーマット

### 見やすい一覧表示

```bash
cat headlines.json | jq -r '.[] | "
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📰 \(.title)
🏢 \(.source)
🔗 \(.url)
"'
```

### 番号付き一覧

```bash
cat headlines.json | jq -r 'to_entries | .[] | "\(.key + 1). [\(.value.source)] \(.value.title)"'
```

### HTML形式で出力

```bash
echo "<ul>" > headlines.html
cat headlines.json | jq -r '.[] | "<li><a href=\"\(.url)\">\(.title)</a> <em>(\(.source))</em></li>"' >> headlines.html
echo "</ul>" >> headlines.html
```

---

## 🔧 トラブルシューティング

### jqコマンドがない場合

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# CentOS/RHEL
sudo yum install jq
```

### ファイルが見つからない

```bash
# 現在のディレクトリのJSONファイル一覧
ls -lh *.json

# carbon-relay実行後の出力ファイル
ls -lh *headlines*.json
```

### JSON形式が壊れている場合

```bash
# JSON検証
cat headlines.json | jq empty

# エラー箇所を特定
cat headlines.json | jq . > /dev/null
```

---

## 📚 実用例

### 毎日の確認ルーティン

```bash
#!/bin/bash
# daily_check.sh

# 最新データ収集
./scripts/collect_and_view.sh all-free 30

# climate関連のみ抽出
cat collected_headlines.json | jq '.[] | select(.title | contains("climate"))' > climate_news.json

# 確認
./scripts/view_headlines.sh climate_news.json
```

### 週次レポート作成

```bash
#!/bin/bash
# weekly_report.sh

DATE=$(date +%Y-%m-%d)
OUTPUT="weekly_report_${DATE}.md"

echo "# Carbon Market Weekly Report - $DATE" > $OUTPUT
echo "" >> $OUTPUT
echo "## Headlines" >> $OUTPUT
cat headlines.json | jq -r '.[] | "- [\(.title)](\(.url))"' >> $OUTPUT

echo "Report created: $OUTPUT"
```

---

## 💻 ワンライナー集

```bash
# タイトル数
cat headlines.json | jq '. | length'

# 最新記事のタイトル
cat headlines.json | jq -r '.[0].title'

# 最新記事のURL
cat headlines.json | jq -r '.[0].url'

# 特定ソースの件数
cat headlines.json | jq '[.[] | select(.source=="Carbon Herald")] | length'

# climateを含む記事数
cat headlines.json | jq '[.[] | select(.title|contains("climate"))] | length'

# タイトルを番号付きで表示
cat headlines.json | jq -r 'to_entries | .[] | "\(.key+1). \(.value.title)"'
```

---

## 🎓 さらに学ぶ

- **jq公式ドキュメント**: https://stedolan.github.io/jq/manual/
- **jqチュートリアル**: https://stedolan.github.io/jq/tutorial/

---

**Happy Viewing! 👀**
