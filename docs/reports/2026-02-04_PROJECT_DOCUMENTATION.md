# Carbon Relay プロジェクト詳細ドキュメント

**作成日**: 2026年2月4日
**バージョン**: 1.0
**ステータス**: 本番運用中

---

## 目次

1. [プロジェクト概要](#1-プロジェクト概要)
2. [システムアーキテクチャ](#2-システムアーキテクチャ)
3. [運用モード](#3-運用モード)
4. [ソース一覧](#4-ソース一覧)
5. [ファイル構成](#5-ファイル構成)
6. [主要機能](#6-主要機能)
7. [環境変数](#7-環境変数)
8. [コマンドラインオプション](#8-コマンドラインオプション)
9. [スクリプト一覧](#9-スクリプト一覧)
10. [ドキュメント一覧](#10-ドキュメント一覧)
11. [技術的詳細](#11-技術的詳細)
12. [既知の制約・課題](#12-既知の制約課題)
13. [開発履歴](#13-開発履歴)

---

## 1. プロジェクト概要

### 1.1 プロジェクト名
**Carbon Relay** - カーボンニュース収集・分析・配信自動化システム

### 1.2 目的
カーボン関連ニュースの収集・分析・配信を自動化し、以下を実現する：
- 複数のカーボン関連ニュースソースからの記事自動収集
- 有料記事のヘッドラインから関連する無料の一次情報源を発見
- 収集した記事のNotion Databaseへの自動クリッピング
- メールによる記事要約の自動配信

### 1.3 技術スタック
- **言語**: Go 1.21+
- **主要ライブラリ**:
  - `github.com/PuerkitoBio/goquery` - HTMLスクレイピング
  - `github.com/mmcdole/gofeed` - RSS/Atomフィード解析
  - `github.com/jomei/notionapi` - Notion API統合
- **外部API**:
  - OpenAI API (Web検索機能)
  - Notion API (データベース管理)
  - Gmail SMTP (メール配信)

### 1.4 リポジトリ
- **GitHub**: https://github.com/FuseKota/curbon-search.git
- **ローカルパス**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/`

---

## 2. システムアーキテクチャ

### 2.1 全体構成

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Carbon Relay Pipeline                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │   Sources    │───>│  Headlines   │───>│   Output     │          │
│  │  Collection  │    │  Processing  │    │  Delivery    │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│         │                   │                   │                   │
│         ▼                   ▼                   ▼                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │ - WordPress  │    │ - OpenAI     │    │ - JSON出力   │          │
│  │ - HTML       │    │   Search     │    │ - Notion     │          │
│  │ - RSS/Atom   │    │ - IDF        │    │ - Email      │          │
│  │ - 有料サイト │    │   Matching   │    │              │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 データフロー

1. **収集フェーズ**: 各ソースから記事のヘッドライン・本文を取得
2. **処理フェーズ**: OpenAI検索で関連記事を発見、IDFマッチングでスコアリング
3. **出力フェーズ**: JSON/Notion/Emailへの出力

---

## 3. 運用モード

### 3.1 モード1: 無料記事収集モード 🟢

**目的**: Carbon関連の無料記事を幅広く収集し、要約してメール配信

**特徴**:
- 37の無料ソースから直接記事を収集
- OpenAI API不要（コスト効率が高い）
- 高速実行（数十秒〜数分）
- 日次レビューに最適

**使用例**:
```bash
./pipeline -perSource=10 -queriesPerHeadline=0 -sendEmail
```

**フラグのポイント**:
- `-queriesPerHeadline=0`: 検索なし（コスト削減）

### 3.2 モード2: 有料記事マッチングモード 🔵

**目的**: 有料記事のヘッドラインから関連する無料の一次情報源を発見

**特徴**:
- OpenAI検索で関連無料記事を発見
- IDF（逆文書頻度）ベースの高精度マッチング
- Notion Databaseで体系的に整理
- Weeklyレビューに最適

**使用例**:
```bash
./pipeline -sources=carbonpulse,qci -perSource=5 -queriesPerHeadline=3 -notionClip
```

**フラグのポイント**:
- `-queriesPerHeadline=3`: 各見出しに対して3つの検索クエリを発行

---

## 4. ソース一覧

### 4.1 有効なソース（37ソース）

#### 4.1.1 有料ソース（見出しのみ）- 2ソース

| ソース名 | 識別子 | 説明 | 実装ファイル |
|---------|--------|------|-------------|
| Carbon Pulse | `carbonpulse` | 世界最大のカーボン市場ニュース | sources_paid.go |
| QCI | `qci` | Quantum Commodity Intelligence | sources_paid.go |

#### 4.1.2 WordPress REST APIソース - 7ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| CarbonCredits.jp | `carboncredits.jp` | 日本のカーボンクレジット市場ニュース | ✅ |
| Carbon Herald | `carbonherald` | CDR技術・スタートアップ情報 | ✅ |
| Climate Home News | `climatehomenews` | 国際交渉・政策情報 | ✅ |
| CarbonCredits.com | `carboncredits.com` | 初心者向け解説記事 | ✅ |
| Sandbag | `sandbag` | EU排出権取引システム分析 | ✅ |
| Ecosystem Marketplace | `ecosystem-marketplace` | 自然ベースソリューション市場 | ✅ |
| Carbon Brief | `carbon-brief` | 気候科学・政策ニュース | ✅ |

#### 4.1.3 HTMLスクレイピングソース - 12ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| ICAP | `icap` | 国際炭素行動パートナーシップ | ✅ |
| IETA | `ieta` | 国際排出量取引協会 | ⚠️ 要改善 |
| Energy Monitor | `energy-monitor` | エネルギー移行ニュース | ✅ |
| World Bank | `world-bank` | 世界銀行気候変動 | ✅ |
| NewClimate | `newclimate` | 気候研究機関 | ✅ |
| Carbon Knowledge Hub | `carbon-knowledge-hub` | 教育プラットフォーム | ✅ |
| Verra | `verra` | VCS規格運営団体 | ✅ |
| Gold Standard | `gold-standard` | 高品質カーボンクレジット規格 | ✅ |
| ACR | `acr` | American Carbon Registry | ✅ |
| CAR | `car` | Climate Action Reserve | ✅ |
| Puro.earth | `puro-earth` | 炭素除去認証プラットフォーム | ✅ |
| Isometric | `isometric` | 炭素除去検証 | ✅ |

#### 4.1.4 日本語ソース - 6ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| JRI | `jri` | 日本総合研究所 | ⚠️ 部分的 |
| 環境省 | `env-ministry` | 日本環境省 | ✅ |
| METI | `meti` | 経済産業省 | ✅ |
| PwC Japan | `pwc-japan` | PwC Japan | ⚠️ 要改善 |
| Mizuho R&T | `mizuho-rt` | みずほリサーチ＆テクノロジーズ | ⚠️ 要改善 |
| JPX | `jpx` | 日本取引所グループ | ✅ |

#### 4.1.5 国際機関ソース - 2ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| IISD ENB | `iisd` | 環境交渉速報（国際持続可能開発研究所） | ✅ |
| Climate Focus | `climate-focus` | 気候政策コンサルティング | ✅ |

#### 4.1.6 地域ETSソース - 5ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| EU-ETS | `eu-ets` | EU排出量取引制度 | ✅ |
| UK-ETS | `uk-ets` | 英国排出量取引制度 | ✅ |
| CARB | `carb` | カリフォルニア大気資源局 | ✅ |
| RGGI | `rggi` | 地域温室効果ガスイニシアティブ | ✅ |
| Australia CER | `australia-cer` | オーストラリアClean Energy Regulator | ✅ |

#### 4.1.7 RSSフィードソース - 2ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| Politico EU | `politico-eu` | EU政策・エネルギー・気候変動ニュース | ✅ |
| Euractiv | `euractiv` | EU政策ニュース | ✅ |

#### 4.1.8 学術・研究機関ソース - 3ソース

| ソース名 | 識別子 | 説明 | 本文取得 |
|---------|--------|------|---------|
| arXiv | `arxiv` | プレプリントサーバー | ✅ |
| Nature Communications | `nature-comms` | 学術誌 | ✅ |
| OIES | `oies` | オックスフォードエネルギー研究所 | ✅ |

### 4.2 無効化中のソース（2ソース）

| ソース名 | 識別子 | 理由 | 状態 |
|---------|--------|------|------|
| UNFCCC | `unfccc` | Incapsula保護（Bot対策） | ❌ 無効 |
| UN News | `un-news` | 本文取得の改善待ち | ⏸️ ペンディング |
| Carbon Market Watch | `carbon-market-watch` | 403 Forbiddenエラー | ❌ 無効 |

---

## 5. ファイル構成

### 5.1 ディレクトリ構造

```
carbon-relay/
├── cmd/pipeline/              # メインアプリケーション
│   ├── main.go               # エントリーポイント、CLI処理
│   ├── config.go             # 設定管理
│   ├── types.go              # データ型定義
│   ├── utils.go              # ユーティリティ関数
│   ├── handlers.go           # リクエストハンドラー
│   │
│   ├── headlines.go          # ソースレジストリ、共通収集関数
│   ├── sources_paid.go       # 有料ソース（Carbon Pulse, QCI）
│   ├── sources_wordpress.go  # WordPress REST APIソース
│   ├── sources_html.go       # HTMLスクレイピングソース
│   ├── sources_japan.go      # 日本語ソース
│   ├── sources_rss.go        # RSSフィードソース
│   ├── sources_academic.go   # 学術・研究機関ソース
│   ├── sources_regional_ets.go # 地域ETSソース
│   │
│   ├── search_openai.go      # OpenAI検索統合
│   ├── search_queries.go     # 検索クエリ生成戦略
│   ├── matcher.go            # IDFマッチングエンジン
│   │
│   ├── notion.go             # Notion API統合
│   └── email.go              # メール送信機能
│
├── docs/                      # ドキュメント
│   ├── README.md             # ドキュメント目次
│   ├── guides/               # 使い方ガイド
│   │   ├── QUICKSTART.md
│   │   ├── DEVELOPMENT.md
│   │   ├── HEADLINES_ONLY.md
│   │   ├── NOTION_INTEGRATION.md
│   │   └── VIEWING_GUIDE.md
│   ├── reports/              # レポート・テスト結果
│   │   ├── STATUS.md
│   │   ├── PROJECT_STATE.md
│   │   ├── TEST_REPORT.md
│   │   ├── TEST_SUMMARY.md
│   │   └── INTEGRATION_TEST_REPORT.md
│   └── architecture/         # アーキテクチャドキュメント
│       ├── COMPLETE_IMPLEMENTATION_GUIDE.md
│       ├── PROJECT_COMPREHENSIVE_REPORT.md
│       └── SYSTEM_FLOW.md
│
├── scripts/                   # 便利スクリプト
│   ├── collect_headlines_only.sh
│   ├── collect_and_view.sh
│   ├── view_headlines.sh
│   ├── clip_to_notion.sh
│   ├── clip_all_sources.sh
│   ├── test_notion.sh
│   ├── full_pipeline.sh
│   ├── build_lambda.sh
│   ├── check_related.sh
│   └── run_examples.sh
│
├── .env                       # 環境変数（gitignore）
├── .env.example              # 環境変数テンプレート
├── go.mod                    # Goモジュール定義
├── go.sum                    # 依存関係ロック
├── CLAUDE.md                 # Claude Code向け指示書
└── README.md                 # メインREADME
```

### 5.2 ソースコードファイル詳細

| ファイル | 行数（概算） | 役割 |
|---------|-------------|------|
| main.go | ~300 | CLI処理、パイプライン実行 |
| headlines.go | ~350 | ソースレジストリ、共通関数 |
| sources_paid.go | ~200 | Carbon Pulse, QCI |
| sources_wordpress.go | ~500 | WordPress REST API 7ソース |
| sources_html.go | ~1800 | HTMLスクレイピング 12ソース |
| sources_japan.go | ~600 | 日本語ソース 6つ |
| sources_rss.go | ~400 | RSSフィード 2ソース |
| sources_academic.go | ~300 | 学術ソース 3つ |
| sources_regional_ets.go | ~500 | 地域ETS 5ソース |
| search_openai.go | ~250 | OpenAI検索統合 |
| search_queries.go | ~200 | 検索クエリ生成 |
| matcher.go | ~400 | IDFマッチング |
| notion.go | ~500 | Notion統合 |
| email.go | ~150 | メール送信 |
| types.go | ~100 | データ型定義 |
| **合計** | **~6000** | |

---

## 6. 主要機能

### 6.1 ヘッドライン収集

**機能**: 各ソースから記事のタイトル、URL、公開日、本文（excerpt）を取得

**技術**:
- WordPress REST API（7ソース）
- HTMLスクレイピング + goquery（多数ソース）
- RSSフィード解析（2ソース）
- Cookie jarによるセッション維持（IISD等）

**特殊処理**:
- PwC Japan: 3重エスケープされたJSON解析
- 日本語ソース: キーワードフィルタリング（`carbonKeywords`配列）
- IISD ENB: Cookie jarによる個別記事ページアクセス

### 6.2 OpenAI検索統合

**機能**: 見出しから関連する無料記事をWeb検索で発見

**実装詳細**:
- OpenAI Responses API使用
- `web_search_call.results`が空のため、message.contentからURL正規表現抽出
- URLから疑似タイトル自動生成

**検索クエリ戦略**:
- 見出しの完全一致検索（引用符付き）
- カーボン市場キーワード補助（VCM, ETS, CORSIA, CCER等）
- 地域別`site:`演算子（韓国、EU、日本、英国、中国、豪州）
- PDF優先: `filetype:pdf`
- NGO/国際機関優先

### 6.3 IDFマッチングエンジン

**機能**: 収集した記事と検索結果の関連度をスコアリング

**スコアリング要素**:
| 要素 | 重み | 説明 |
|------|------|------|
| IDF加重リコール | 56% | 逆文書頻度ベースの類似度 |
| Jaccard係数 | 28% | 単語集合の類似度 |
| Market信号 | 6% | 市場名の一致 |
| Topic信号 | 4% | トピックの一致 |
| Geo信号 | 2% | 地域の一致 |
| Recency | 4% | 新しさ |

**品質ブースト**:
- `.gov`ドメイン: +0.18
- `.pdf`: +0.18
- NGO: +0.12

### 6.4 Notion統合

**機能**: 収集した記事をNotion Databaseに自動クリッピング

**主要機能**:
- 新規データベース自動作成
- データベースID自動永続化（.envファイル）
- 既存データベース自動再利用
- 全文保存（ページブロック、2000文字/ブロック制限対応）
- AI Summaryフィールド自動入力

**データベーススキーマ**:
| プロパティ | タイプ | 説明 |
|----------|--------|------|
| Title | Title | 記事タイトル |
| URL | URL | 記事URL |
| Source | Select | ソース名 |
| Type | Select | Headline / Related Free |
| Score | Number | 関連度スコア |
| Excerpt | Rich Text | 本文（最初2000文字） |
| AI Summary | Rich Text | 要約用 |
| ShortHeadline | Rich Text | 短縮見出し |

### 6.5 メール配信

**機能**: 収集した記事の要約をメールで配信

**対応**:
- Gmail SMTP
- HTML形式
- 日次ダイジェスト

---

## 7. 環境変数

### 7.1 必須（モード依存）

| 変数名 | 用途 | 必須条件 |
|--------|------|---------|
| `OPENAI_API_KEY` | OpenAI API | モード2（検索あり） |
| `NOTION_TOKEN` | Notion API | Notionクリップ時 |
| `NOTION_PAGE_ID` | 新規DB作成時の親ページ | 初回のみ |
| `EMAIL_FROM` | 送信元メールアドレス | メール送信時 |
| `EMAIL_PASSWORD` | Gmailアプリパスワード | メール送信時 |
| `EMAIL_TO` | 送信先（カンマ区切り） | メール送信時 |

### 7.2 オプション

| 変数名 | 用途 | デフォルト |
|--------|------|-----------|
| `NOTION_DATABASE_ID` | 既存DB使用時 | 自動保存される |
| `DEBUG_OPENAI` | 検索結果サマリー表示 | 無効 |
| `DEBUG_OPENAI_FULL` | APIレスポンス全体表示 | 無効 |
| `DEBUG_SCRAPING` | スクレイピング詳細表示 | 無効 |

---

## 8. コマンドラインオプション

| オプション | デフォルト | 説明 |
|----------|----------|------|
| `-sources` | 全無料ソース | スクレイピング対象（カンマ区切り） |
| `-perSource` | `30` | 各ソースから収集する最大件数 |
| `-queriesPerHeadline` | `3` | 見出しごとの検索クエリ数（0で無効） |
| `-resultsPerQuery` | `10` | クエリごとの最大結果数 |
| `-searchPerHeadline` | `25` | 見出しごとに保持する候補数 |
| `-topK` | `3` | 見出しごとの最大relatedFree数 |
| `-minScore` | `0.32` | 最小スコア閾値 |
| `-daysBack` | `60` | 新しさフィルタ（日数、0で無効） |
| `-strictMarket` | `true` | 市場マッチ必須 |
| `-out` | stdout | 出力先パス |
| `-saveFree` | - | 候補プール保存パス |
| `-notionClip` | `false` | Notionにクリップ |
| `-sendEmail` | `false` | メール送信 |
| `-sendShortEmail` | `false` | 短縮ダイジェスト送信 |
| `-headlines` | - | 既存headlines.jsonを読み込む |

---

## 9. スクリプト一覧

| スクリプト | 用途 |
|-----------|------|
| `collect_headlines_only.sh` | ヘッドラインのみ収集（検索なし） |
| `collect_and_view.sh` | 収集と同時に確認 |
| `view_headlines.sh` | 既存ファイルを確認 |
| `clip_to_notion.sh` | Notionにクリッピング |
| `clip_all_sources.sh` | 全ソースからNotionへ |
| `test_notion.sh` | Notion統合テスト |
| `full_pipeline.sh` | フルパイプライン実行 |
| `build_lambda.sh` | AWS Lambda用ビルド |
| `check_related.sh` | 関連記事確認 |
| `run_examples.sh` | サンプル実行 |

---

## 10. ドキュメント一覧

### 10.1 ガイド（docs/guides/）

| ファイル | 内容 |
|---------|------|
| QUICKSTART.md | クイックスタートガイド |
| DEVELOPMENT.md | 開発ガイド |
| HEADLINES_ONLY.md | ヘッドライン収集ガイド |
| NOTION_INTEGRATION.md | Notion統合ガイド |
| VIEWING_GUIDE.md | 結果確認ガイド |

### 10.2 レポート（docs/reports/）

| ファイル | 内容 |
|---------|------|
| STATUS.md | 現在のステータス |
| PROJECT_STATE.md | プロジェクト状態 |
| TEST_REPORT.md | テストレポート |
| TEST_SUMMARY.md | テストサマリー |
| INTEGRATION_TEST_REPORT.md | 統合テストレポート |
| 2026-01-31_new_sources_implementation.md | 新ソース実装レポート |
| 2026-02-02_source_fixes_and_additions.md | ソース修正・追加レポート |

### 10.3 アーキテクチャ（docs/architecture/）

| ファイル | 内容 |
|---------|------|
| COMPLETE_IMPLEMENTATION_GUIDE.md | 完全実装ガイド（2,756行） |
| PROJECT_COMPREHENSIVE_REPORT.md | プロジェクト総合レポート |
| SYSTEM_FLOW.md | システムフロー図 |

---

## 11. 技術的詳細

### 11.1 IISD ENB本文取得の実装

**課題**: 個別記事ページがIncapsula保護でアクセス拒否

**解決策**:
1. Cookie jarを使用してセッション維持
2. ホームページにアクセスしてCookie取得
3. 個別記事ページにアクセスして本文取得

**取得内容**:
- `og:description`から【About】セクション
- `.c-wysiwyg__content`から【Content】セクション
- 画像で区切られた複数セクションにも対応
- リスト項目（`<li>`）も「• 」付きで取得

### 11.2 日本語ソースのキーワードフィルタリング

**対象**: JRI、環境省、METI、Mizuho R&T

**キーワード例**:
```go
carbonKeywords = []string{
    "カーボン", "炭素", "排出", "CO2", "温室効果",
    "気候", "脱炭素", "グリーン", "サステナ", "ESG",
    "再エネ", "再生可能", "水素", "アンモニア",
    // ...
}
```

### 11.3 OpenAI検索の制約と対策

**問題**:
- `web_search_call.results`が常に空
- 構造化データ（title, url, snippet）が取得できない

**対策**:
- message.contentからURL正規表現抽出
- URLから疑似タイトル自動生成

---

## 12. 既知の制約・課題

### 12.1 アクセス不可ソース

| ソース | 理由 | 代替 |
|--------|------|------|
| UNFCCC | Incapsula保護 | IISD ENB |
| Carbon Market Watch | 403 Forbidden | なし |

### 12.2 本文取得要改善

| ソース | 状態 | 備考 |
|--------|------|------|
| IETA | 0/2 | セレクタ調整必要 |
| PwC Japan | 0/2 | JSON解析改善必要 |
| Mizuho R&T | 0/2 | セレクタ調整必要 |
| JRI | 1/2 | 部分的に取得 |

### 12.3 OpenAI API制限

- Responses APIの構造化データ問題
- 長期的にはBrave Search API / SerpAPIへの移行推奨

---

## 13. 開発履歴

### 2026-02-04
- IISD ENB本文取得の完全実装
  - Cookie jarによるセッション維持
  - 複数セクションからの本文取得
- UN Newsソースをペンディングに変更
- 全37ソースのテスト完了（105記事をNotionにクリップ）

### 2026-02-02
- 新ソース追加（OIES, Australia CER等）
- 既存ソースの修正（Puro.earth, Isometric等）

### 2026-01-31
- ディレクトリ構造の改善
- ドキュメント整理

### 2026-01-03
- 9つの無料ソース実装完了
- Notion統合機能実装
- データベースID自動永続化

### 2025-12-31
- 4つの無料ソース追加
- WordPress REST API統合
- Notion全文保存機能

### 2025-12-30
- 記事要約自動抽出機能
- Notion Database統合

### 2025-12-29
- OpenAI Responses API統合
- MVP完成

---

## 付録

### A. テスト実行例

```bash
# 全ソーステスト
./pipeline -perSource=3 -queriesPerHeadline=0 -out=/tmp/test.json

# Notionクリップ
source .env && ./pipeline -perSource=5 -queriesPerHeadline=0 -notionClip

# 特定ソースのみ
./pipeline -sources=iisd,verra -perSource=5 -queriesPerHeadline=0
```

### B. デバッグ方法

```bash
# スクレイピングデバッグ
DEBUG_SCRAPING=1 ./pipeline -sources=iisd -perSource=1 -queriesPerHeadline=0

# OpenAIデバッグ
DEBUG_OPENAI=1 ./pipeline -sources=carbonpulse -perSource=2 -queriesPerHeadline=3
```

---

**作成者**: Claude Code
**最終更新**: 2026年2月4日
