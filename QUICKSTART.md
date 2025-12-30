# carbon-relay クイックスタート

## 🚀 5分で始める

### 1. 環境変数設定
```bash
export OPENAI_API_KEY="sk-..."
```

### 2. ビルド
```bash
go build -o carbon-relay ./cmd/pipeline
```

### 3. 実行 & 確認
```bash
# 方法1: 収集と確認を同時に（最も簡単）
./collect_and_view.sh carbonpulse 10

# 方法2: 個別実行
./carbon-relay -sources=carbonpulse -perSource=5 -out=result.json

# 結果確認
./view_headlines.sh result.json
```

---

## 📋 サンプル実行スクリプト

すべてのサンプルを一度に実行：
```bash
./run_examples.sh
```

実行後、`outputs/`ディレクトリに以下のファイルが生成されます：
- `quick_test.json` - クイックテスト結果
- `standard_output.json` - 標準実行結果
- `high_quality.json` - 高品質モード結果
- `exploratory.json` - 探索的モード結果
- `candidates_pool.json` - 候補プール全体
- `debug.log` - デバッグログ

---

## 🎯 このプロジェクトは何をするのか？

Carbon Pulse / QCI の**有料記事の見出し**（無料で見れる部分）から、その記事の**元ネタとなる一次情報**（政府サイト、PDF、企業IR、NGOレポート等）を自動的に見つけ出します。

### 入力（例）
```
"Climate litigation marks 'turning point' in 2025 but expanded scope on horizon -report"
```

### 出力（例）
```json
{
  "title": "Climate litigation marks 'turning point' in 2025...",
  "url": "https://carbon-pulse.com/470719/",
  "relatedFree": [
    {
      "title": "Sendeco2 Noticias Climate Litigation...",
      "url": "https://www.sendeco2.com/es/noticias/2025/12/25/...",
      "score": 0.79
    },
    {
      "title": "LSE Grantham Institute Global Trends...pdf",
      "url": "https://www.lse.ac.uk/.../Climate-Change-Litigation-2025.pdf",
      "score": 0.38
    }
  ]
}
```

---

## 🔧 よく使うオプション

```bash
# 処理する見出し数を増やす
./carbon-relay -perSource=20

# より多くの関連記事を取得
./carbon-relay -topK=5

# スコア閾値を下げて候補を増やす
./carbon-relay -minScore=0.2

# 両ソース（Carbon Pulse + QCI）から取得
./carbon-relay -sources=carbonpulse,qci

# デバッグモード
DEBUG_OPENAI=1 ./carbon-relay ...
```

---

## 📚 詳しく知りたい場合

- **README.md** - プロジェクト全体の説明・実行方法
- **DEVELOPMENT.md** - アーキテクチャ・アルゴリズム詳細
- **STATUS.md** - 現状・課題・次のステップ

---

## 🆘 トラブルシューティング

### relatedFreeが空になる
```bash
# スコア閾値を下げる
./carbon-relay -minScore=0.15

# 検索結果数を増やす
./carbon-relay -queriesPerHeadline=5 -resultsPerQuery=20
```

### OPENAI_API_KEYエラー
```bash
# 環境変数を確認
echo $OPENAI_API_KEY

# 未設定の場合
export OPENAI_API_KEY="sk-..."
```

### ビルドエラー
```bash
# 依存関係を更新
go mod tidy
go build -o carbon-relay ./cmd/pipeline
```

---

## 💡 次のステップ

1. **Brave Search API導入**（推奨）
   - より精度の高い検索結果が得られます
   - `DEVELOPMENT.md` の「Q1: Brave Search APIに移行したい」を参照

2. **検索クエリのカスタマイズ**
   - `cmd/pipeline/search_queries.go` の `buildSearchQueries` を編集
   - 特定の市場・地域に特化したクエリを追加

3. **マッチングスコアの調整**
   - `cmd/pipeline/matcher.go` の `scoreHeadlineCandidate` を編集
   - market/topic/geoの重みを調整

---

**Have fun exploring! 🌍**
