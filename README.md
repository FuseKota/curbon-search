# carbon-relay

**カーボンニュース収集・分析・配信の自動化システム**

> 🧪 **全機能テスト完了**: [テストレポート](docs/reports/TEST_REPORT.md) | [サマリー](docs/reports/TEST_SUMMARY.md)
> ✅ 成功率: 92% (11/12機能) - 本番環境使用可能

## プロジェクトの目的

本システムは、カーボン関連ニュースの収集・分析・配信を自動化します。

### 🟢 無料記事収集モード

**目的**: Carbon関連の無料記事を幅広く収集し、要約してメール配信

**使用例**:
```bash
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -sendEmail
```

**特徴**:
- 40の無料ソースから直接記事を収集
- 高速実行（5-15秒程度）
- 日次レビューに最適
- Notion統合・メール配信に対応

---

## 現在の実装状態（2026-02-04）

### ✅ 実装済み機能

#### 1. ヘッドライン収集 (`cmd/pipeline/headlines.go`)

**無料ソース（全文取得）：** **40サイト実装完了**

**日本市場（7ソース）：**
- CarbonCredits.jp、JRI、環境省、METI、PwC Japan、Mizuho R&T、JPX

**WordPress REST API（7ソース）：**
- Carbon Herald、Climate Home News、CarbonCredits.com、Sandbag、Ecosystem Marketplace、Carbon Brief、RMI

**HTMLスクレイピング（6ソース）：**
- ICAP、IETA、Energy Monitor、World Bank、NewClimate、Carbon Knowledge Hub

**VCM認証団体（4ソース）：**
- Verra、Gold Standard、ACR、CAR

**国際機関（2ソース）：**
- IISD ENB、Climate Focus

**地域ETS（5ソース）：**
- EU ETS、UK ETS、CARB、RGGI、Australia CER

**RSSフィード（2ソース）：**
- Politico EU、Euractiv

**学術・研究（5ソース）：**
- arXiv、OIES、IOP Science (ERL)、Nature Eco&Evo、ScienceDirect

**CDR関連（2ソース）：**
- Puro.earth、Isometric

**技術スタック：**
- WordPress REST API（8サイト）- 標準化されたJSON endpoint
- HTML Scraping + goquery（多数サイト）- カスタムHTML構造解析
- RSSフィード解析（5サイト）

#### 2. Notion統合 (`internal/pipeline/notion.go`)
- Notion Databaseへの自動クリッピング
- データベース自動作成・再利用
- リッチテキスト分割（2000文字/ブロック）

#### 3. メール送信 (`internal/pipeline/email.go`)
- Gmail SMTP経由でのメール配信
- 収集記事の要約をメール形式で送信

---

## 実行例

### ビルド
```bash
go build -o pipeline ./cmd/pipeline
```

### ヘッドライン＋記事要約の収集
```bash
# 全無料ソースからヘッドラインと記事を収集
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -queriesPerHeadline=0 \
  -out=headlines.json

# または専用スクリプトを使用
./scripts/collect_headlines_only.sh
```

### デバッグモード
```bash
# スクレイピングのデバッグ
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=2 -queriesPerHeadline=0
```

---

## コマンドラインオプション

| オプション | デフォルト | 説明 |
|----------|----------|------|
| `-headlines` | - | 既存のheadlines.jsonを読み込む（指定しない場合はスクレイピング） |
| `-sources` | `all-free` | スクレイピング対象（カンマ区切り）<br>carboncredits.jp, carbonherald, climatehomenews, carboncredits.com, sandbag, ecosystem-marketplace, carbon-brief, icap, ieta, energy-monitor, world-bank, newclimate, carbon-knowledge-hub, jri, env-ministry, meti, pwc-japan, mizuho-rt, jpx, politico-eu |
| `-perSource` | `30` | 各ソースから収集する最大件数 |
| `-queriesPerHeadline` | `0` | 検索クエリ数（現在は0のみ使用） |
| `-hoursBack` | `0` | 指定時間以内に公開された記事のみ収集（0で無効） |
| `-out` | - | 出力先（指定しない場合はstdout） |
| `-notionClip` | `false` | Notionにクリップ |
| `-sendEmail` | `false` | メール送信 |

---

## 環境変数

推奨：`.env`ファイルを作成して管理

```bash
# Notion統合（オプション）
NOTION_TOKEN=ntn_...              # Notion Integration Token
NOTION_PAGE_ID=xxx...             # 新規DB作成時の親ページID
NOTION_DATABASE_ID=xxx...         # 既存DB使用時（自動保存される）

# メール送信（オプション）
EMAIL_FROM=your-email@gmail.com
EMAIL_PASSWORD=...                # Gmailアプリパスワード
EMAIL_TO=recipient@example.com

# デバッグ用（オプション）
DEBUG_SCRAPING=1                  # スクレイピング詳細表示
```

**注意：** `NOTION_DATABASE_ID`は初回データベース作成時に自動的に`.env`に追加されます。

---

## 出力フォーマット

```json
[
  {
    "source": "Carbon Herald",
    "title": "New Carbon Capture Project Launches in Europe",
    "url": "https://carbonherald.com/new-carbon-capture-project/",
    "excerpt": "A new carbon capture and storage project has been announced...",
    "publishedAt": "2026-02-04T10:00:00Z",
    "isHeadline": true
  }
]
```

---

## 📚 ドキュメント

詳細なドキュメントは [docs/](./docs/) ディレクトリに整理されています：

- **クイックスタート**: [docs/guides/QUICKSTART.md](./docs/guides/QUICKSTART.md)
- **開発ガイド**: [docs/guides/DEVELOPMENT.md](./docs/guides/DEVELOPMENT.md)
- **完全実装ガイド**: [docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md](./docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md)
- **Notion統合**: [docs/guides/NOTION_INTEGRATION.md](./docs/guides/NOTION_INTEGRATION.md)

すべてのドキュメントは [docs/README.md](./docs/README.md) から参照できます。

## 🛠️ スクリプト

便利なスクリプトは [scripts/](./scripts/) ディレクトリにあります：

```bash
# ヘッドライン収集
./scripts/collect_headlines_only.sh

# Notionクリッピング
./scripts/clip_to_notion.sh

# Lambda用ビルド
./scripts/build_lambda.sh

# フルパイプライン実行
./scripts/full_pipeline.sh
```

詳細は [scripts/README.md](./scripts/README.md) を参照してください。

## ファイル構成

```
carbon-relay/
├── cmd/pipeline/
│   └── main.go              # パイプライン司令塔
├── internal/pipeline/
│   ├── headlines.go         # 共通ロジック
│   ├── sources_wordpress.go # WordPress REST APIソース
│   ├── sources_html.go      # HTMLスクレイピングソース
│   ├── sources_japan.go     # 日本語ソース
│   ├── sources_rss.go       # RSSフィードソース
│   ├── notion.go            # Notion統合
│   ├── email.go             # メール送信
│   ├── types.go             # データ型定義
│   └── utils.go             # ユーティリティ
├── docs/                    # ドキュメント
│   ├── guides/              # 使い方ガイド
│   ├── reports/             # テストレポート・ステータス
│   └── architecture/        # アーキテクチャドキュメント
├── scripts/                 # スクリプト
│   ├── collect_headlines_only.sh
│   ├── clip_to_notion.sh
│   └── build_lambda.sh
├── .env                     # 環境変数
├── go.mod
├── go.sum
├── CLAUDE.md                # Claude Code向け指示書
└── README.md                # このファイル
```

---

## Notion統合

収集した記事をNotion Databaseに自動的にクリッピングできます。

### 🚀 クイックスタート

#### 初回実行（新規データベース作成）

```bash
# .envファイルに環境変数を設定
cat > .env << 'EOF'
NOTION_TOKEN=ntn_...
NOTION_PAGE_ID=xxx...
EOF

# 無料ソースから記事を収集してNotionにクリッピング
./pipeline -sources=all-free -perSource=5 -queriesPerHeadline=0 -notionClip
```

#### 2回目以降（既存データベースに追加）

データベースIDは自動的に`.env`に保存されるため、次回からは同じデータベースに追加されます：

```bash
# 同じコマンドを実行するだけ
./scripts/clip_all_sources.sh
# → 既存データベースに自動追加
```

### 📋 主要機能

**データベース管理：**
- ✅ **新規データベース自動作成**
- ✅ **データベースID自動永続化** - `.env`ファイルに保存
- ✅ **既存データベース自動再利用** - 毎回新規作成されない

**記事クリッピング：**
- ✅ **全文保存** - Notionページ本文に段落ブロックとして保存
- ✅ **Excerptフィールド** - 全文の最初2000文字（プロパティ制限）
- ✅ **AI Summaryフィールド** - 全文の最初2000文字（後から手動要約可能）
- ✅ **メタデータ** - Title, URL, Source, Type, Score

**対応ソース（40ソース）：**
- **日本（7）**: CarbonCredits.jp、JRI、環境省、METI、PwC Japan、Mizuho R&T、JPX
- **WordPress API（7）**: Carbon Herald、Climate Home News、CarbonCredits.com、Sandbag、Ecosystem Marketplace、Carbon Brief、RMI
- **HTML（6）**: ICAP、IETA、Energy Monitor、World Bank、NewClimate、Carbon Knowledge Hub
- **VCM認証（4）**: Verra、Gold Standard、ACR、CAR
- **国際機関（2）**: IISD ENB、Climate Focus
- **地域ETS（5）**: EU ETS、UK ETS、CARB、RGGI、Australia CER
- **RSS（2）**: Politico EU、Euractiv
- **学術（5）**: arXiv、OIES、IOP Science (ERL)、Nature Eco&Evo、ScienceDirect
- **CDR（2）**: Puro.earth、Isometric

### 🗂️ Notionデータベーススキーマ

| プロパティ | タイプ | 説明 |
|----------|--------|------|
| Title | Title | 記事タイトル |
| URL | URL | 記事URL |
| Source | Select | ソース名（カラー分け） |
| Excerpt | Rich Text | 全文の最初2000文字 |
| AI Summary | Rich Text | 要約用フィールド（初期値はExcerptと同じ） |
| ページ本文 | Blocks | 記事全文（段落ブロック） |

### 📚 詳細ドキュメント

Notion統合の詳しい使い方は **[docs/guides/NOTION_INTEGRATION.md](docs/guides/NOTION_INTEGRATION.md)** を参照してください。

---

## 次のステップ（優先度順）

### 優先度：高
1. **新規ソースの追加**
   - 追加可能なカーボン関連情報源の調査
   - RSSフィード対応サイトの追加

### 優先度：中
2. **UI/定期実行**
   - Webインターフェース
   - cron/定期実行スクリプト

---

## トラブルシューティング

### スクレイピングエラー

サイトのレイアウト変更が原因の可能性があります。
```bash
# デバッグモードで詳細を確認
DEBUG_SCRAPING=1 ./pipeline -sources=問題のソース -perSource=1 -queriesPerHeadline=0
```

### Notionクリップでエラー

1. `.env`の`NOTION_DATABASE_ID`を削除
2. `-notionPageID`フラグで再実行
3. 新しいDATABASE_IDが自動保存される

---

## 開発履歴

### 2026-02-04
- ✅ **有料ソース（Carbon Pulse, QCI）を削除**
  - 無料ソースのみの運用に変更
  - ドキュメント・コメントを更新

### 2026-01-03
- ✅ **複数の無料ソース実装完了**
  - WordPress REST API: Sandbag, Ecosystem Marketplace, Carbon Brief
  - HTML Scraping: ICAP, IETA, Energy Monitor
- ✅ **カバレッジ達成**
  - EU ETS分析（Sandbag）
  - 自然ベースソリューション市場（Ecosystem Marketplace）
  - 気候科学（Carbon Brief）
  - 国際機関（ICAP, IETA）
  - エネルギー移行（Energy Monitor）

### 2025-12-31
- ✅ **4つの無料ソース追加**
  - CarbonCredits.jp（日本語）
  - Carbon Herald（CDR技術）
  - Climate Home News（国際交渉）
  - CarbonCredits.com（初心者向け）
- ✅ **WordPress REST API統合** - 全文コンテンツ取得
- ✅ **Notion全文保存機能**
- ✅ **データベースID自動永続化**

### 2025-12-30
- ✅ Notion Database統合機能を実装
- ✅ 自動クリッピング機能

### 2025-12-29
- ✅ MVP完成

---

## ライセンス

（プロジェクトのライセンスをここに記載）

---

## 作成者

carbon-relay development team
