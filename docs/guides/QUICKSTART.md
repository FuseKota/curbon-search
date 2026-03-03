# carbon-relay クイックスタート

## 🚀 5分で始める

### 1. ビルド
```bash
go build -o pipeline ./cmd/pipeline
```

### 2. 実行 & 確認
```bash
# 方法1: 収集と確認を同時に（最も簡単）
./scripts/collect_and_view.sh all-free 10

# 方法2: 個別実行
./pipeline -sources=all-free -perSource=5 -out=result.json

# 結果確認
./scripts/view_headlines.sh result.json
```

---

## 🎯 このプロジェクトは何をするのか？

**39の無料ソース**からカーボン関連ニュースのヘッドラインと要約を自動収集します。

### 入力（コマンド実行）
```bash
./pipeline -sources=all-free -perSource=10
```

### 出力（例）
```json
{
  "source": "Carbon Herald",
  "title": "EU carbon price hits record high amid supply concerns",
  "url": "https://carbonherald.com/article/...",
  "excerpt": "EU carbon prices reached a new record...",
  "isHeadline": true
}
```

---

## 📰 利用可能なソース（39ソース）

### 日本ソース（5つ）
- `jri` - 日本総研
- `pwc-japan` - PwC Japan
- `mizuho-rt` - みずほリサーチ＆テクノロジーズ
- `jpx` - 日本取引所グループ
- `carboncredits.jp` - カーボンクレジット.jp

### WordPress APIソース（6つ）
- `carbonherald` - Carbon Herald
- `climatehomenews` - Climate Home News
- `carboncredits.com` - CarbonCredits.com
- `sandbag` - Sandbag
- `ecosystem-marketplace` - Ecosystem Marketplace
- `carbon-brief` - Carbon Brief

### HTMLスクレイピングソース（6つ）
- `icap` - ICAP
- `ieta` - IETA
- `energy-monitor` - Energy Monitor
- `world-bank` - World Bank
- `newclimate` - NewClimate Institute
- `carbon-knowledge-hub` - Carbon Knowledge Hub

### VCM認証団体（4つ）
- `verra` - Verra
- `gold-standard` - Gold Standard
- `acr` - American Carbon Registry
- `car` - Climate Action Reserve

### 国際機関（2つ）
- `iisd` - IISD ENB
- `climate-focus` - Climate Focus

### 地域ETS（5つ）
- `eu-ets` - EU ETS
- `uk-ets` - UK ETS
- `carb` - カリフォルニア大気資源局
- `rggi` - RGGI
- `australia-cer` - オーストラリアCER

### RSSフィード（3つ）
- `politico-eu` - Politico EU
- `euractiv` - Euractiv（RSS + 記事ページスクレイピングで全文取得）
- `carbon-market-watch` - Carbon Market Watch

### 学術・研究（2つ）
- `arxiv` - arXiv
- `oies` - オックスフォードエネルギー研究所

### CDR関連（2つ）
- `puro-earth` - Puro.earth
- `isometric` - Isometric

---

## 🔧 よく使うオプション

```bash
# 処理するソースを指定
./pipeline -sources=carbonherald,carbon-brief

# 各ソースからの収集数を増やす
./pipeline -sources=all-free -perSource=20

# 過去24時間の記事のみ
./pipeline -sources=all-free -perSource=30 -hoursBack=24

# デバッグモード
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=2
```

---

## 📚 詳しく知りたい場合

- **README.md** - プロジェクト全体の説明・実行方法
- **HEADLINES_ONLY.md** - ヘッドライン収集の詳細
- **VIEWING_GUIDE.md** - 収集結果の確認方法
- **NOTION_INTEGRATION.md** - Notion連携の設定

---

## 🆘 トラブルシューティング

### ヘッドラインが収集されない
```bash
# デバッグモードで詳細確認
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=1
```

### ビルドエラー
```bash
# 依存関係を更新
go mod tidy
go build -o pipeline ./cmd/pipeline
```

### 特定ソースがエラー
```bash
# そのソースのみテスト
./pipeline -sources=jri -perSource=3
```

---

## 💡 次のステップ

1. **Notion連携を設定**
   - `NOTION_INTEGRATION.md` を参照
   - 収集した記事をNotionデータベースに自動クリップ

2. **メール配信を設定**
   - `.env`にメール設定を追加
   - `-sendShortEmail` フラグでダイジェストメール送信

3. **定期実行の設定**
   - cronやAWS Lambdaで定期実行
   - `scripts/build_lambda.sh` でLambdaパッケージを作成

---

**Have fun exploring! 🌍**
