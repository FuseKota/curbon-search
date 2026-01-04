# carbon-relay

**カーボンニュース収集・分析・配信の自動化システム**

> 🧪 **全機能テスト完了**: [テストレポート](TEST_REPORT.md) | [サマリー](TEST_SUMMARY.md)
> ✅ 成功率: 92% (11/12機能) - 本番環境使用可能

## プロジェクトの目的

本システムは、カーボン関連ニュースの収集・分析・配信を自動化する**2つの運用モード**をサポートします：

### 🟢 モード1: 無料記事収集モード

**目的**: Carbon関連の無料記事を幅広く収集し、要約してメール配信

**使用例**:
```bash
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -sendEmail
```

**特徴**:
- 16の無料ソースから直接記事を収集
- OpenAI API不要（コスト効率が高い）
- 高速実行（5-15秒程度）
- 日次レビューに最適

### 🔵 モード2: 有料記事マッチングモード

**目的**: 有料記事のヘッドラインから関連する無料の一次情報源を発見

Carbon Pulse / Quantum Commodity Intelligence (QCI) の**無料版で閲覧できる有料記事の見出し**から、その見出しの**元となる一次情報・現地情報・無料公開資料**をWeb上から探索し、ユーザーが有料課金をしなくても記事の背景・根拠・周辺情報を追えるようにします。

**使用例**:
```bash
./pipeline -sources=carbonpulse,qci -perSource=5 -queriesPerHeadline=3 -notionClip
```

**特徴**:
- OpenAI検索で関連無料記事を発見
- IDF（逆文書頻度）ベースの高精度マッチング
- Notion Databaseで体系的に整理
- Weekly レビューに最適

**処理フロー**:
- 見出しタイトルを「検索クエリの種」として使用
- 検索エンジン（OpenAI API）でWeb探索
- 無料・一次情報候補を収集（政府サイト、PDF、IR、NGOレポート等）
- 類似度 + market/geo/topic シグナルで関連付け
- 結果を `relatedFree` として出力

### やらないこと ❌
- **有料記事本文の取得**
- **free.json を事前に人手で用意する設計**
- **Carbon Pulse / QCI の本文コピー**

---

## 現在の実装状態（2026-01-03）

### ✅ 実装済み機能

#### 1. ヘッドライン収集 (`cmd/pipeline/headlines.go`)

**有料ソース（見出しのみ）：**
- Carbon Pulse の無料ページ（timeline/newsletters）からスクレイピング
- QCI のホームページからスクレイピング
- 無意味なリンクテキスト（"Read more"等）を自動除外

**無料ソース（全文取得）：** 🆕 **9サイト実装完了**

**日本市場：**
- **CarbonCredits.jp** - 日本のカーボンクレジット市場ニュース（WordPress REST API）

**欧州・国際：**
- **Sandbag** - EU排出権取引システム（EU ETS）分析（WordPress REST API）
- **Ecosystem Marketplace** - 自然ベースソリューション（NbS）市場（WordPress REST API）
- **Carbon Brief** - 気候科学・政策ニュース（WordPress REST API）
- **Climate Home News** - 国際交渉・政策情報（WordPress REST API）
- **ICAP** - 国際炭素行動パートナーシップ（HTML Scraping）
- **IETA** - 国際排出量取引協会（HTML Scraping）

**グローバルメディア：**
- **Carbon Herald** - CDR技術・スタートアップ情報（WordPress REST API）
- **CarbonCredits.com** - 初心者向け解説記事（WordPress REST API）
- **Energy Monitor** - エネルギー移行ニュース（HTML Scraping）

**技術スタック：**
- WordPress REST API（7サイト）- 標準化されたJSON endpoint
- HTML Scraping + goquery（3サイト）- カスタムHTML構造解析

#### 2. OpenAI検索統合 (`cmd/pipeline/search_openai.go`)
**重要な技術的発見：**
- OpenAI Responses API は `web_search_call.results` を返さない
- `action.sources` も空
- → **message.content からURLを正規表現で抽出**する実装に変更
- → **URLから疑似タイトルを自動生成**（例：`carbon-pulse.com/timeline/...` → `"Carbon Pulse Timeline"`）

#### 3. 検索クエリ戦略 (`cmd/pipeline/search_queries.go`)
- 見出しの完全一致検索（引用符付き）
- カーボン市場キーワード補助（VCM, ETS, CORSIA, CCER等）
- **地域別site:演算子**：
  - 韓国：`site:go.kr`
  - EU：`site:europa.eu`
  - 日本：`site:go.jp`
  - 英国：`site:gov.uk`
  - 中国：`site:gov.cn`
  - 豪州：`site:gov.au`
- **PDF優先**：`filetype:pdf`
- **NGO/国際機関優先**：`site:unfccc.int OR site:icvcm.org OR site:iea.org`

#### 4. マッチングエンジン (`cmd/pipeline/matcher.go`)
- IDF（逆文書頻度）ベースの類似度計算
- Market/Topic/Geo シグナル抽出
- ドメイン品質スコア（.gov = +0.18, .pdf = +0.18, NGO = +0.12等）
- 厳格な市場マッチング（`strictMarket`）
- トップK件の関連記事選定

---

## 実行例

### ビルド
```bash
go build -o carbon-relay ./cmd/pipeline
```

### ヘッドライン＋記事要約の収集（OpenAI API不要）🆕
```bash
# OpenAI APIキーなしでヘッドラインと記事の要約を収集
./carbon-relay \
  -sources=carbonpulse \
  -perSource=30 \
  -queriesPerHeadline=0 \
  -out=headlines.json

# または専用スクリプトを使用
./collect_headlines_only.sh
```

**✨ 新機能:** トップページから記事の要約（無料で見れる部分）を自動的に取得します

**詳細は [HEADLINES_ONLY.md](HEADLINES_ONLY.md) を参照**

### ヘッドライン確認ツール🆕
```bash
# 収集と同時に確認
./collect_and_view.sh carbonpulse 10

# 既存ファイルを確認
./view_headlines.sh headlines.json
```

**詳細は [VIEWING_GUIDE.md](VIEWING_GUIDE.md) を参照**

### 基本実行（検索あり）
```bash
# Carbon Pulseから5件、QCIから5件を処理（関連記事検索込み）
./carbon-relay \
  -sources=carbonpulse,qci \
  -perSource=5 \
  -queriesPerHeadline=3 \
  -resultsPerQuery=12 \
  -topK=3 \
  -minScore=0.25 \
  -out=output.json
```

### デバッグモード
```bash
# OpenAI API レスポンスの詳細を表示
DEBUG_OPENAI=1 ./carbon-relay -sources=carbonpulse -perSource=2

# OpenAI API レスポンス全体を表示
DEBUG_OPENAI_FULL=1 ./carbon-relay -sources=carbonpulse -perSource=1
```

### 候補プールの保存
```bash
./carbon-relay -saveFree=candidates.json -out=output.json
```

---

## コマンドラインオプション

| オプション | デフォルト | 説明 |
|----------|----------|------|
| `-headlines` | - | 既存のheadlines.jsonを読み込む（指定しない場合はスクレイピング） |
| `-sources` | `carbonpulse,qci,carboncredits.jp,carbonherald,climatehomenews,carboncredits.com,sandbag,ecosystem-marketplace,carbon-brief,icap,ieta,energy-monitor` | スクレイピング対象（カンマ区切り）<br>**有料:** carbonpulse, qci<br>**無料:** carboncredits.jp, sandbag, ecosystem-marketplace, carbon-brief, climatehomenews, carboncredits.com, icap, ieta, energy-monitor |
| `-perSource` | `30` | 各ソースから収集する最大件数 |
| `-queriesPerHeadline` | `3` | 見出しごとに発行する検索クエリ数 |
| `-resultsPerQuery` | `10` | クエリごとの最大結果数 |
| `-searchPerHeadline` | `25` | 見出しごとに保持する候補数 |
| `-topK` | `3` | 見出しごとの最大relatedFree数 |
| `-minScore` | `0.32` | 最小スコア閾値 |
| `-daysBack` | `60` | 新しさフィルタ（日数、0で無効） |
| `-strictMarket` | `true` | 見出しにmarket信号がある場合、候補もmarketマッチ必須 |
| `-saveFree` | - | 候補プール全体を保存するパス |
| `-out` | - | 出力先（指定しない場合はstdout） |
| `-searchProvider` | `openai` | 検索プロバイダ（現在はopenaiのみ） |
| `-openaiModel` | `gpt-4o-mini` | OpenAIモデル |
| `-openaiTool` | `web_search` | OpenAIツールタイプ |

---

## 環境変数

推奨：`.env`ファイルを作成して管理

```bash
# OpenAI API Key（検索機能使用時に必須）
OPENAI_API_KEY=sk-...

# Notion統合（オプション）
NOTION_TOKEN=ntn_...              # Notion Integration Token
NOTION_PAGE_ID=xxx...             # 新規DB作成時の親ページID
NOTION_DATABASE_ID=xxx...         # 既存DB使用時（自動保存される）

# デバッグ用（オプション）
DEBUG_OPENAI=1                    # 検索結果のサマリー表示
DEBUG_OPENAI_FULL=1               # APIレスポンス全体を表示
DEBUG_SCRAPING=1                  # スクレイピング詳細表示
```

**注意：** `NOTION_DATABASE_ID`は初回データベース作成時に自動的に`.env`に追加されます。

---

## 出力フォーマット

```json
[
  {
    "source": "Carbon Pulse",
    "title": "Climate litigation marks 'turning point' in 2025 but expanded scope on horizon -report",
    "url": "https://carbon-pulse.com/470719/",
    "excerpt": "Global climate litigation grew and diversified in 2025, marking a turning point especially at the international court level, according to a year-end review by a New York-based legal center.",
    "isHeadline": true,
    "relatedFree": [
      {
        "source": "OpenAI(text_extract)",
        "title": "Sendeco2 Noticias Climate Litigation Marks Turning Point In 2025",
        "url": "https://www.sendeco2.com/es/noticias/2025/12/25/climate-litigation...",
        "score": 0.7875447027505618,
        "reason": "overlap=1.00 titleSim=0.81 recency=0.00 market=0.00 topic=0.00 geo=0.00 quality=0.00 sharedTokens=11"
      },
      {
        "source": "OpenAI(text_extract)",
        "title": "Lse Granthaminstitute Global Trends In Climate Change Litigation 2025 Snapshot.pdf",
        "url": "https://www.lse.ac.uk/granthaminstitute/wp-content/uploads/.../Global-Trends-in-Climate-Change-Litigation-2025-Snapshot.pdf",
        "score": 0.3126258263080772,
        "reason": "overlap=0.19 titleSim=0.10 recency=0.00 market=0.00 topic=0.00 geo=0.00 quality=0.18 sharedTokens=3"
      }
    ]
  }
]
```

---

## ファイル構成

```
carbon-relay/
├── cmd/pipeline/
│   ├── main.go              # パイプライン司令塔 + データベースID永続化
│   ├── headlines.go         # 全ソーススクレイピング
│   │                        # - 有料: Carbon Pulse, QCI
│   │                        # - 無料: CarbonCredits.jp, Carbon Herald,
│   │                        #        Climate Home News, CarbonCredits.com
│   ├── notion.go            # Notion統合
│   │                        # - データベース作成・管理
│   │                        # - 全文保存（ページブロック）
│   │                        # - AI Summaryフィールド自動入力
│   ├── search_openai.go     # OpenAI検索 + URL抽出 + 疑似タイトル生成
│   ├── search_queries.go    # 検索クエリ生成戦略
│   ├── matcher.go           # IDF + 類似度 + シグナルベースマッチング
│   ├── types.go             # データ型定義
│   └── utils.go             # ユーティリティ
├── test-notion.sh           # Notion統合テストスクリプト
├── clip-all-sources.sh      # 全4無料ソース一括クリッピング
├── .env                     # 環境変数（自動生成）
├── go.mod
├── go.sum
└── README.md
```

---

## 既知の制約・課題

### 🚨 OpenAI Responses API の限界

**問題：**
- `web_search_call.results` が常に空
- 構造化されたデータ（title, url, snippet）が取得できない
- message.contentにテキスト形式の解説が返される

**現在の対策：**
- ✅ テキストから正規表現でURL抽出
- ✅ URLから疑似タイトル自動生成
- ✅ MVPとして動作可能

**長期的な推奨解決策：**
- 

  - 理由：構造化データが確実に取得できる、検索品質が安定
  - 実装予定：`cmd/pipeline/search_brave.go`

---

## 実際の成果例

### 見出し：「Climate litigation marks 'turning point' in 2025」
**発見した一次情報：**
- ✅ Sendeco2（カーボン市場専門サイト）- スコア0.79
- ✅ LSE Grantham Institute PDF（学術機関）- スコア0.38
- ✅ rinnovabili.it PDF（環境メディア）- スコア0.38

### 見出し：「US DOE expands technologies eligible for 45V clean hydrogen tax credits」
**発見した一次情報：**
- ✅ energy.gov PDF（米国エネルギー省公式）- スコア0.45
- ✅ Sendeco2 - スコア0.81

### 見出し：「Hawaii court declines to block cruise ship climate levy」
**発見した一次情報：**
- ✅ civilbeat.org（ハワイ現地メディア）- スコア0.80
- ✅ hawaiinewsnow.com（現地ニュース）- スコア0.80
- ✅ hawaiitribune-herald.com（現地新聞）- スコア0.80

---

## Notion統合 🆕

収集した記事をNotion Databaseに自動的にクリッピングできます。

### 🚀 クイックスタート

#### 初回実行（新規データベース作成）

```bash
# .envファイルに環境変数を設定
cat > .env << 'EOF'
OPENAI_API_KEY=sk-...
NOTION_TOKEN=ntn_...
NOTION_PAGE_ID=xxx...
EOF

# 無料ソースから記事を収集してNotionにクリッピング
./test-notion.sh

# または全4ソースから5記事ずつ収集
./clip-all-sources.sh
```

#### 2回目以降（既存データベースに追加）

データベースIDは自動的に`.env`に保存されるため、次回からは同じデータベースに追加されます：

```bash
# 同じコマンドを実行するだけ
./clip-all-sources.sh
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

**対応ソース：**
- Carbon Pulse（有料見出し）
- QCI（有料見出し）
- CarbonCredits.jp（全文）
- Sandbag（全文）
- Ecosystem Marketplace（全文）
- Carbon Brief（全文）
- Climate Home News（全文）
- CarbonCredits.com（全文）
- ICAP（全文）
- IETA（全文）
- Energy Monitor（全文）
- OpenAI検索結果（関連記事）

### 🗂️ Notionデータベーススキーマ

| プロパティ | タイプ | 説明 |
|----------|--------|------|
| Title | Title | 記事タイトル |
| URL | URL | 記事URL |
| Source | Select | ソース名（カラー分け） |
| Type | Select | Headline / Related Free |
| Score | Number | 関連度スコア（Related Freeのみ） |
| Excerpt | Rich Text | 全文の最初2000文字 |
| AI Summary | Rich Text | 要約用フィールド（初期値はExcerptと同じ） |
| Content | Rich Text | 将来の拡張用 |
| ページ本文 | Blocks | 記事全文（段落ブロック） |

### 📚 詳細ドキュメント

Notion統合の詳しい使い方は **[NOTION_INTEGRATION.md](NOTION_INTEGRATION.md)** を参照してください。

---

## 次のステップ（優先度順）

### 優先度：高
1. **Brave Search API / SerpAPI の統合**
   - 構造化データ取得による精度向上
   - OpenAI API コスト削減

### 優先度：中
2. **検索クエリのさらなる改善**
   - 企業名・制度名の自動抽出
   - 時間範囲の絞り込み（`after:2025-01-01`）

3. **マッチングスコアの最適化**
   - market/topic/geo signalsの重み調整
   - ドメイン品質スコアのさらなる改善

### 優先度：低
4. **UI/定期実行**
   - Webインターフェース
   - cron/定期実行スクリプト

---

## トラブルシューティング

### relatedFreeが空になる場合

1. **minScoreが高すぎる**
   ```bash
   # スコア閾値を下げる
   ./carbon-relay -minScore=0.15
   ```

2. **検索クエリが少なすぎる**
   ```bash
   # クエリ数と結果数を増やす
   ./carbon-relay -queriesPerHeadline=5 -resultsPerQuery=15
   ```

3. **OpenAI APIキーが未設定**
   ```bash
   export OPENAI_API_KEY="sk-..."
   ```

### スクレイピングエラー

```
ERROR: no Carbon Pulse headlines found
```
→ サイトのレイアウト変更の可能性。`headlines.go` の正規表現を確認。

---

## 開発履歴

### 2026-01-03 🆕
- ✅ **9つの無料ソース実装完了**（18サイト中）
  - バッチ1（WordPress REST API）: Sandbag, Ecosystem Marketplace, Carbon Brief
  - バッチ2（HTML Scraping）: ICAP, IETA
  - バッチ3（HTML Scraping）: Energy Monitor
- ✅ **実装困難サイトの詳細分析**
  - 10サイトの技術的課題を文書化
  - 将来の実装に向けた解決策を整理
  - プランファイルに完全な分析レポート追加
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
  - ページ本文にparagraphブロックとして保存
  - 2000文字/ブロック制限に対応した自動分割
- ✅ **データベースID自動永続化**
  - `.env`ファイルに自動保存
  - 既存データベース自動再利用
  - 重複データベース作成を防止
- ✅ **AI Summaryフィールド自動入力**
  - 初期値として全文を自動挿入
  - 後からNotion AIで要約可能

### 2025-12-30
- ✅ 記事要約（excerpt）の自動抽出機能を実装
- ✅ excerptを活用した検索クエリ生成（固有名詞・数値抽出）
- ✅ Notion Database統合機能を実装
- ✅ 自動クリッピング機能（ヘッドライン + 関連記事）

### 2025-12-29
- ✅ OpenAI Responses API 統合
- ✅ URL抽出 + 疑似タイトル生成実装
- ✅ 検索クエリ戦略強化（site:, filetype:）
- ✅ 無意味リンクテキストフィルタ
- ✅ MVP完成

---

## ライセンス

（プロジェクトのライセンスをここに記載）

---

## 作成者

carbon-relay development team
