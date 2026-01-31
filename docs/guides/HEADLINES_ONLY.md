# ヘッドライン＋記事要約の収集（OpenAI API不要）

## 🎯 概要

このモードでは、**OpenAI APIを使わずに**Carbon Pulse / QCI からヘッドラインと記事要約（無料で見れる部分）を収集できます。

- ✅ OpenAI API不要（OPENAI_API_KEY不要）
- ✅ スクレイピングのみ
- ✅ 記事の要約も自動取得（Carbon Pulseトップページから）
- ✅ 高速（検索なし）
- ❌ relatedFree は付かない（検索しないため）

---

## 🚀 クイックスタート

### 方法1: コマンドライン（最もシンプル）

```bash
# ビルド（初回のみ）
go build -o carbon-relay ./cmd/pipeline

# ヘッドライン収集（OpenAI API不要）
./carbon-relay \
  -sources=carbonpulse \
  -perSource=30 \
  -queriesPerHeadline=0 \
  -out=headlines.json
```

**重要：** `-queriesPerHeadline=0` を指定することで検索をスキップします。

### 方法2: 専用スクリプト（推奨）

```bash
# すべてのヘッドライン収集パターンを一度に実行
./collect_headlines_only.sh
```

実行後、`headlines_output/`に以下のファイルが生成されます：
- `carbonpulse_headlines.json` - Carbon Pulseのみ
- `qci_headlines.json` - QCIのみ
- `all_headlines.json` - 両方

---

## 📋 出力フォーマット

```json
[
  {
    "source": "Carbon Pulse",
    "title": "Climate litigation marks 'turning point' in 2025 but expanded scope on horizon -report",
    "url": "https://carbon-pulse.com/470719/",
    "excerpt": "Global climate litigation grew and diversified in 2025, marking a turning point especially at the international court level, according to a year-end review by a New York-based legal center.",
    "isHeadline": true
  },
  {
    "source": "QCI",
    "title": "EU carbon price hits record high amid supply concerns",
    "url": "https://www.qcintel.com/carbon/article/...",
    "isHeadline": true
  }
]
```

**新機能：** Carbon Pulseのトップページからは記事の要約（無料で見れる部分）も自動的に取得されます。

**注意：** このモードでは`relatedFree`フィールドは含まれません（空配列になります）。

---

## 🔧 オプション

| オプション | デフォルト | 説明 |
|----------|----------|------|
| `-sources` | `carbonpulse,qci` | 収集元（カンマ区切り） |
| `-perSource` | `30` | 各ソースから収集する最大件数 |
| `-queriesPerHeadline` | `3` | **0に設定して検索をスキップ** |
| `-out` | - | 出力ファイルパス（未指定で標準出力） |

---

## 📊 実行例

### Carbon Pulseから10件収集
```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=cp_headlines.json
```

### QCIから50件収集
```bash
./carbon-relay \
  -sources=qci \
  -perSource=50 \
  -queriesPerHeadline=0 \
  -out=qci_headlines.json
```

### 両ソースから各100件収集
```bash
./carbon-relay \
  -sources=carbonpulse,qci \
  -perSource=100 \
  -queriesPerHeadline=0 \
  -out=all_headlines.json
```

### 標準出力（パイプで利用）
```bash
./carbon-relay \
  -sources=carbonpulse \
  -perSource=5 \
  -queriesPerHeadline=0 | jq -r '.[].title'
```

---

## 💡 次のステップ：関連記事の取得

ヘッドラインを収集した後、別途無料記事を集めてマッチングすることができます：

### ステップ1: ヘッドライン収集（このモード）
```bash
./carbon-relay -sources=carbonpulse -perSource=30 -queriesPerHeadline=0 -out=headlines.json
```

### ステップ2: 無料記事を別途収集
（RSSフィード、別のスクレイピング、またはOpenAI検索を使用）

### ステップ3: マッチング
ユーザーが提供した`match_headlines.go`を使用：
```bash
go run match_headlines.go \
  --headlines headlines.json \
  --free free.json \
  --topK 3 \
  --minScore 0.32 \
  > matched.json
```

---

## 🆚 モード比較

| モード | OpenAI API | 速度 | relatedFree |
|-------|-----------|------|------------|
| **ヘッドラインのみ** | ❌ 不要 | 🚀 高速 | ❌ なし |
| **標準モード** | ✅ 必要 | 🐢 遅い | ✅ あり |

---

## 🔍 収集されるサイト

### Carbon Pulse
- `https://carbon-pulse.com/daily-timeline/`
- `https://carbon-pulse.com/category/newsletters/`

### QCI
- `https://www.qcintel.com/carbon/`

---

## ⚠️ 注意事項

1. **スクレイピング制約**
   - サイトのレイアウト変更で動作しなくなる可能性があります
   - 過度なアクセスは避けてください

2. **無意味なリンクは自動除外**
   - "Read more", "Click here"等は除外されます
   - 10文字未満のタイトルも除外されます

3. **relatedFreeについて**
   - このモードでは空配列になります
   - 関連記事が必要な場合は標準モード（`run_examples.sh`）を使用してください

---

## 🆘 トラブルシューティング

### エラー: "no headlines collected"
```bash
# サイトがブロックしている可能性
# → User-Agentを確認（headlines.go:32）
# → 手動でサイトにアクセスできるか確認
```

### エラー: "no Carbon Pulse headlines found"
```bash
# サイトレイアウトが変更された可能性
# → headlines.go の正規表現を確認
# → 対象ページを手動で確認
```

### ヘッドライン数が少ない
```bash
# perSource を増やす
./carbon-relay -perSource=100 -queriesPerHeadline=0
```

---

## 📚 関連ドキュメント

- **README.md** - プロジェクト全体の説明
- **QUICKSTART.md** - 5分で始めるガイド
- **DEVELOPMENT.md** - アーキテクチャ詳細

---

**Have fun collecting! 📰**
