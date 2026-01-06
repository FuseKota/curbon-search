# Carbon Relay システム全体フロー図

このドキュメントはCarbon Relayシステムの全処理フローを図解しています。

---

## システム概要

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Carbon Relay パイプライン                           │
│                                                                             │
│   ファイル構成:                                                              │
│     main.go          - エントリーポイント、CLI処理                           │
│     headlines.go     - 22ソースからのスクレイピング                          │
│     search_openai.go - OpenAI Web検索                                       │
│     matcher.go       - IDFスコアリング                                      │
│     notion.go        - Notion API連携                                       │
│     email.go         - Gmail SMTP送信                                       │
│     types.go         - データ構造定義                                        │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## メイン処理フロー

```
                              ┌──────────────┐
                              │   CLI起動    │
                              │  ./pipeline  │
                              └──────┬───────┘
                                     │
                              フラグ解析
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         │                           │                           │
         ▼                           ▼                           ▼
  ┌─────────────┐           ┌─────────────┐           ┌─────────────────┐
  │ -sendEmail  │           │-sendShortEmail│          │   それ以外      │
  │   フラグ    │           │   フラグ    │           │  (記事収集)     │
  └──────┬──────┘           └──────┬──────┘           └────────┬────────┘
         │                         │                          │
         ▼                         ▼                          ▼
  ┌─────────────┐           ┌─────────────┐          ┌────────────────┐
  │handleEmail  │           │handleShort  │          │   記事収集     │
  │   Send()    │           │ EmailSend() │          │    モード      │
  └──────┬──────┘           └──────┬──────┘          └────────┬───────┘
         │                         │                          │
         ▼                         ▼                          │
┌─────────────────────────────────────────────┐               │
│           メール送信フロー                   │               │
│                                             │               │
│  1. 環境変数チェック                        │               │
│     EMAIL_FROM, EMAIL_PASSWORD, EMAIL_TO    │               │
│     NOTION_TOKEN, NOTION_DATABASE_ID        │               │
│                                             │               │
│  2. NotionDBから記事取得                    │               │
│     FetchRecentHeadlines(daysBack)          │               │
│                                             │               │
│  3. メール送信                              │               │
│     ┌─────────────────┬─────────────────┐   │               │
│     │  -sendEmail     │ -sendShortEmail │   │               │
│     ├─────────────────┼─────────────────┤   │               │
│     │ フル要約メール  │ 50文字ダイジェスト│   │               │
│     │ タイトル+要約   │ タイトル+URL    │   │               │
│     │ +URL+ソース    │ カーボンキーワード │   │               │
│     │                │ フィルタリング   │   │               │
│     └─────────────────┴─────────────────┘   │               │
│                                             │               │
│  4. Gmail SMTP送信（リトライ付き）          │               │
└──────────────────────┬──────────────────────┘               │
                       │                                      │
                       ▼                                      │
                   [終了]                                     │
                                                              │
                          ┌───────────────────────────────────┘
                          │
                          ▼
           【記事収集フローへ続く】
```

---

## 記事収集フロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        記事収集フロー                                        │
│                                                                             │
│  ステップ1: ソースから見出し収集                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    22のニュースソース                               │    │
│  │                                                                     │    │
│  │  【有料ソース（ヘッドラインのみ取得）】 2ソース                      │    │
│  │    ・Carbon Pulse        ・QCI                                      │    │
│  │                                                                     │    │
│  │  【海外無料ソース】 12ソース                                        │    │
│  │    ・CarbonCredits.jp    ・Carbon Herald     ・Climate Home News    │    │
│  │    ・CarbonCredits.com   ・Sandbag           ・Ecosystem Marketplace│    │
│  │    ・Carbon Brief        ・ICAP              ・IETA                 │    │
│  │    ・Energy Monitor      ・Carbon Market Watch ・NewClimate         │    │
│  │    ・Carbon Knowledge Hub ・World Bank                              │    │
│  │                                                                     │    │
│  │  【日本語ソース】 6ソース                                           │    │
│  │    ・JRI（日本総研）     ・環境省（env-ministry）                   │    │
│  │    ・PwC Japan           ・Mizuho R&T                               │    │
│  │    ・JPX（東京証券取引所）・METI（経産省中小企業庁）                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                   4つの取得方式                                     │    │
│  │                                                                     │    │
│  │  ① WordPress REST API (7ソース)                                    │    │
│  │     CarbonCredits.jp, Carbon Herald, Climate Home News,             │    │
│  │     CarbonCredits.com, Sandbag, Ecosystem Marketplace, Carbon Brief │    │
│  │                                                                     │    │
│  │  ② HTML スクレイピング (11ソース)                                  │    │
│  │     Carbon Pulse, QCI, ICAP, IETA, Energy Monitor, 環境省,          │    │
│  │     World Bank, Carbon Market Watch, NewClimate,                    │    │
│  │     Carbon Knowledge Hub, Mizuho R&T                                │    │
│  │                                                                     │    │
│  │  ③ RSS フィード (3ソース)                                          │    │
│  │     JRI（日本総研）, JPX, METI（経産省）                            │    │
│  │                                                                     │    │
│  │  ④ HTML + 埋込JSON (1ソース)                                       │    │
│  │     PwC Japan（3重エスケープJSON解析）                              │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└──────────────────────────────────────────┬──────────────────────────────────┘
                                           │
                                           ▼
                           ┌───────────────────────────────┐
                           │  -queriesPerHeadline の値は？ │
                           └───────────────┬───────────────┘
                                           │
                    ┌──────────────────────┴──────────────────────┐
                    │                                             │
                    ▼                                             ▼
         ┌─────────────────┐                           ┌─────────────────┐
         │      = 0        │                           │      > 0        │
         │  (モード1)      │                           │  (モード2)      │
         └────────┬────────┘                           └────────┬────────┘
                  │                                             │
                  ▼                                             ▼
┌─────────────────────────────────┐           ┌─────────────────────────────────┐
│  モード1: 無料記事収集          │           │  モード2: 有料記事マッチング     │
│                                 │           │                                 │
│  ・OpenAI API不要（無料）       │           │  ・OpenAI API使用（有料）       │
│  ・検索スキップ                 │           │  ・Web検索実行                   │
│  ・収集した記事をそのまま出力   │           │  ・IDFスコアリング              │
│                                 │           │  ・関連記事マッチング            │
└────────────────┬────────────────┘           └────────────────┬────────────────┘
                 │                                              │
                 │                             ┌────────────────┘
                 │                             │
                 │                             ▼
                 │           ┌─────────────────────────────────────────────────┐
                 │           │  ステップ2: OpenAI Web検索                      │
                 │           │  ───────────────────────────────────────────── │
                 │           │                                                 │
                 │           │  各見出しに対して:                              │
                 │           │    1. 検索クエリ生成（buildSearchQueries）      │
                 │           │    2. OpenAI API呼び出し（gpt-4o-mini）         │
                 │           │    3. web_search ツールで候補記事取得           │
                 │           │    4. 候補をマージ・重複排除                    │
                 │           └─────────────────────────────────────────────────┘
                 │                             │
                 │                             ▼
                 │           ┌─────────────────────────────────────────────────┐
                 │           │  ステップ3: IDFスコアリング                     │
                 │           │  ───────────────────────────────────────────── │
                 │           │                                                 │
                 │           │  buildIDF() → 全文書からIDF辞書構築            │
                 │           │                                                 │
                 │           │  topKRelated() スコア計算:                      │
                 │           │    ・IDF加重リコール類似度:  56%                │
                 │           │    ・IDF加重Jaccard類似度:   28%                │
                 │           │    ・マーケットマッチ:        6%                │
                 │           │    ・トピックマッチ:          4%                │
                 │           │    ・地理的マッチ:            2%                │
                 │           │    ・新しさ（Recency）:       4%                │
                 │           │                                                 │
                 │           │  品質ブースト:                                  │
                 │           │    ・.gov ドメイン:  +0.18                      │
                 │           │    ・.pdf ファイル:  +0.18                      │
                 │           │    ・EU公式サイト:   +0.16                      │
                 │           │    ・NGO/政策機関:   +0.12                      │
                 │           │    ・プレスリリース: +0.08                      │
                 │           └─────────────────────────────────────────────────┘
                 │                             │
                 └─────────────────────────────┤
                                               │
                                               ▼
                 ┌─────────────────────────────────────────────────────────────┐
                 │  ステップ4: 出力                                            │
                 │  ─────────────────────────────────────────────────────────  │
                 │                                                             │
                 │  ┌──────────────────────────────────────────────────────┐   │
                 │  │  JSON出力                                            │   │
                 │  │    -out=file.json → ファイルに保存                   │   │
                 │  │    -out 省略      → stdout に出力                    │   │
                 │  └──────────────────────────────────────────────────────┘   │
                 │                                                             │
                 └──────────────────────────────┬──────────────────────────────┘
                                                │
                                    ┌───────────┴───────────┐
                                    │  -notionClip フラグ?  │
                                    └───────────┬───────────┘
                                                │
                              ┌─────────────────┴─────────────────┐
                              │                                   │
                              ▼                                   ▼
                       ┌────────────┐                      ┌────────────┐
                       │    あり    │                      │    なし    │
                       └─────┬──────┘                      └─────┬──────┘
                             │                                   │
                             ▼                                   ▼
┌─────────────────────────────────────────────────┐          [終了]
│  ステップ5: Notion保存                          │
│  ─────────────────────────────────────────────  │
│                                                 │
│  1. 環境変数チェック                            │
│     NOTION_TOKEN                                │
│                                                 │
│  2. データベース作成/選択                       │
│     -notionDatabaseID 指定 → 既存DB使用         │
│     -notionPageID 指定     → 新規DB作成         │
│                                                 │
│  3. 記事をクリップ                              │
│     ClipHeadlineWithRelated()                   │
│       ├─ 見出し情報（Title, URL, Source）       │
│       ├─ AI Summary（記事本文）                 │
│       ├─ ShortHeadline（50文字要約用）          │
│       └─ 関連記事（RelatedFree）               │
│                                                 │
│  4. ページブロックとして本文追加                │
│     createContentBlocks()                       │
│                                                 │
└────────────────────────┬────────────────────────┘
                         │
                         ▼
                     [終了]
```

---

## 処理モード一覧

| モード | フラグ | 処理内容 | OpenAI API | 用途 |
|--------|--------|----------|------------|------|
| **モード1** | `-queriesPerHeadline=0` | 無料記事収集のみ | 不要 | ニュースレター配信 |
| **モード2** | `-queriesPerHeadline=3` | 有料記事→無料記事マッチング | 必要 | リサーチ・分析 |
| **メール送信A** | `-sendEmail` | フル要約メール送信 | 不要 | 詳細レポート |
| **メール送信B** | `-sendShortEmail` | 50文字ダイジェストメール | 不要 | 速報配信 |

---

## 22ソース一覧

### 有料ソース（2ソース）
ヘッドラインのみ取得し、関連する無料記事を検索

| ソース名 | ソースID | 取得方式 |
|----------|----------|----------|
| Carbon Pulse | `carbonpulse` | HTML Scraping |
| QCI | `qci` | HTML Scraping |

### 海外無料ソース（14ソース）

| ソース名 | ソースID | 取得方式 |
|----------|----------|----------|
| CarbonCredits.jp | `carboncredits.jp` | WordPress API |
| Carbon Herald | `carbonherald` | WordPress API |
| Climate Home News | `climatehomenews` | WordPress API |
| CarbonCredits.com | `carboncredits.com` | WordPress API |
| Sandbag | `sandbag` | WordPress API |
| Ecosystem Marketplace | `ecosystem-marketplace` | WordPress API |
| Carbon Brief | `carbon-brief` | WordPress API |
| ICAP | `icap` | HTML Scraping |
| IETA | `ieta` | HTML Scraping |
| Energy Monitor | `energy-monitor` | HTML Scraping |
| Carbon Market Watch | `carbon-market-watch` | HTML Scraping |
| NewClimate | `newclimate` | HTML Scraping |
| Carbon Knowledge Hub | `carbon-knowledge-hub` | HTML Scraping |
| World Bank | `world-bank` | HTML Scraping |

### 日本語ソース（6ソース）

| ソース名 | ソースID | 取得方式 | 備考 |
|----------|----------|----------|------|
| JRI（日本総研） | `jri` | RSS Feed | キーワードフィルタあり |
| 環境省 | `env-ministry` | HTML Scraping | キーワードフィルタあり |
| PwC Japan | `pwc-japan` | HTML + 埋込JSON | 3重エスケープJSON |
| Mizuho R&T | `mizuho-rt` | HTML Scraping | キーワードフィルタあり |
| JPX（東京証券取引所） | `jpx` | RSS Feed | |
| METI（経産省） | `meti` | RSS Feed | 中小企業庁RSS |

---

## コマンド例

### モード1: 無料記事収集 → Notion保存
```bash
./pipeline -sources=carbonherald,carboncredits.jp \
           -perSource=10 \
           -queriesPerHeadline=0 \
           -notionClip \
           -notionDatabaseID=xxx
```

### モード2: 有料記事マッチング → JSON出力
```bash
./pipeline -sources=carbonpulse,qci \
           -perSource=5 \
           -queriesPerHeadline=3 \
           -out=matched.json
```

### メール送信: フル要約
```bash
./pipeline -sendEmail -emailDaysBack=7
```

### メール送信: 50文字ダイジェスト
```bash
./pipeline -sendShortEmail -emailDaysBack=7
```

---

## 必要な環境変数

| 変数名 | 用途 | 必要なモード |
|--------|------|-------------|
| `OPENAI_API_KEY` | OpenAI Web検索 | モード2 |
| `NOTION_TOKEN` | Notion API認証 | Notion保存/メール送信 |
| `NOTION_DATABASE_ID` | NotionデータベースID | Notion保存/メール送信 |
| `NOTION_PAGE_ID` | 新規DB作成時の親ページ | 新規Notion DB作成時 |
| `EMAIL_FROM` | 送信元Gmail | メール送信 |
| `EMAIL_PASSWORD` | Gmailアプリパスワード | メール送信 |
| `EMAIL_TO` | 送信先メール | メール送信 |

---

## CLIフラグ一覧

### 基本設定
| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-headlines` | (なし) | 既存JSONファイルから見出し読み込み |
| `-out` | stdout | 出力JSONファイルパス |
| `-sources` | 全22ソース | 収集ソース（カンマ区切り） |
| `-perSource` | 30 | ソースあたり最大記事数 |

### 検索設定
| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-queriesPerHeadline` | 3 | 見出しあたりクエリ数（0で無効） |
| `-searchPerHeadline` | 25 | 見出しあたり候補上限 |
| `-resultsPerQuery` | 10 | クエリあたり結果数 |

### マッチング設定
| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-daysBack` | 60 | 新しさ考慮期間（日） |
| `-topK` | 3 | 見出しあたり関連記事上限 |
| `-minScore` | 0.32 | 最小スコア閾値 |
| `-strictMarket` | true | 市場シグナル一致必須 |

### 出力設定
| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-notionClip` | false | Notion保存を有効化 |
| `-notionPageID` | (なし) | 新規DB作成時の親ページID |
| `-notionDatabaseID` | (なし) | 既存NotionデータベースID |
| `-sendEmail` | false | フル要約メール送信 |
| `-sendShortEmail` | false | 50文字ダイジェストメール送信 |
| `-emailDaysBack` | 1 | メール送信時の取得日数 |

---

**最終更新**: 2026年1月6日
