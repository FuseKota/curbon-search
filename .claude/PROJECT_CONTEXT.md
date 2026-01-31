# Carbon Relay - プロジェクトコンテキスト

このファイルは新しいClaude Codeセッションへの引き継ぎ用です。

## プロジェクト概要

**Carbon Relay**は、カーボン関連ニュースの収集・分析・配信を自動化するGo製システムです。

**プロジェクトパス**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/`

## 🔑 2つの運用モード（重要）

### 🟢 モード1: 無料記事収集モード
- **目的**: Carbon関連の無料記事を幅広く収集してメール配信
- **コマンド**: `./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -sendEmail`
- **特徴**: OpenAI API不要、コスト効率が高い、高速（5-15秒）
- **用途**: 日次の無料記事レビュー

### 🔵 モード2: 有料記事マッチングモード
- **目的**: 有料記事のヘッドラインから関連する無料記事を発見
- **コマンド**: `./pipeline -sources=carbonpulse,qci -queriesPerHeadline=3 -notionClip`
- **特徴**: OpenAI検索、IDFマッチング、Notion統合
- **用途**: 有料記事の裏付け情報収集、Weekly整理

## 📁 主要ファイル構成

```
cmd/pipeline/
├── main.go              (515行) - エントリーポイント、CLI制御
├── headlines.go         (2,354行) - 18ソース実装
├── matcher.go           (506行) - IDFスコアリング
├── search_openai.go     (295行) - OpenAI検索統合
├── search_queries.go    (232行) - クエリ生成
├── notion.go            (554行) - Notion統合
├── email.go             (175行) - メール送信
├── types.go             (42行) - データ構造
└── utils.go             (78行) - ユーティリティ
```

## 🗂️ データソース（18ソース）

### 有料ソース（見出しのみ）
1. Carbon Pulse
2. QCI (Quantum Commodity Intelligence)

### 無料ソース（全文取得）

**日本市場（7ソース）**:
3. CarbonCredits.jp
4. JRI (日本総研)
5. Environment Ministry (環境省)
6. JPX (日本取引所グループ)
7. METI (経済産業省)
8. Mizuho R&T (みずほリサーチ＆テクノロジーズ)
9. PwC Japan

**欧州・国際（6ソース）**:
10. Sandbag
11. Carbon Brief
12. Climate Home News
13. ICAP
14. IETA
15. Carbon Market Watch

**グローバル（3ソース）**:
16. Carbon Herald
17. CarbonCredits.com
18. Energy Monitor

**その他（3ソース）**:
19. Carbon Knowledge Hub
20. Ecosystem Marketplace
21. New Climate Institute

## 🛠️ よく使うコマンド

### ビルド
```bash
cd /Users/kotafuse/Yasui/Prog/Test/carbon-relay
go build -o pipeline ./cmd/pipeline
```

### モード1: 無料記事収集
```bash
# 全無料ソースから収集
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -out=free_articles.json

# メール送信付き
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -sendEmail
```

### モード2: 有料記事マッチング
```bash
# 基本的なマッチング
./pipeline -sources=carbonpulse,qci -perSource=5 -queriesPerHeadline=3 -out=matched.json

# Notionクリッピング（推奨）
./pipeline -sources=carbonpulse,qci -perSource=10 -queriesPerHeadline=3 -notionClip
```

### 特定ソースのテスト
```bash
# 日本市場のみ
./pipeline -sources=jri,env-ministry,jpx,pwc-japan -perSource=5 -queriesPerHeadline=0

# PwC Japanのテスト（複雑なJSON解析）
./pipeline -sources=pwc-japan -perSource=5 -queriesPerHeadline=0 -out=/tmp/pwc_test.json
```

### デバッグ
```bash
# OpenAI検索のデバッグ
DEBUG_OPENAI=1 ./pipeline -sources=carbonpulse -perSource=2 -queriesPerHeadline=1

# スクレイピングのデバッグ
DEBUG_SCRAPING=1 ./pipeline -sources=pwc-japan -perSource=5 -queriesPerHeadline=0

# 完全デバッグ
DEBUG_OPENAI_FULL=1 DEBUG_SCRAPING=1 ./pipeline -sources=carbonpulse -perSource=1 -queriesPerHeadline=1
```

## 🔧 環境変数（.env）

必須の環境変数:
```bash
# OpenAI（モード2で必要）
OPENAI_API_KEY=sk-...

# Notion統合（Notionクリップ時に必要）
NOTION_API_KEY=secret_...
NOTION_PAGE_ID=...           # 初回のみ必要（自動保存される）
NOTION_DATABASE_ID=...       # 自動保存される

# メール送信（メール配信時に必要）
EMAIL_FROM=your-email@gmail.com
EMAIL_PASSWORD=...           # Gmailアプリパスワード
EMAIL_TO=recipient@example.com
```

## 📊 主要なフラグ

```bash
-sources              # ソース指定（CSV形式）
-perSource            # ソースあたりの記事数（デフォルト: 30）
-queriesPerHeadline   # 記事あたりのクエリ数（0=検索なし、デフォルト: 3）
-topK                 # 上位K件のマッチング結果（デフォルト: 3）
-minScore             # 最小マッチングスコア（デフォルト: 0.32）
-out                  # 出力ファイル（省略時はstdout）
-notionClip           # Notionにクリップ
-notionPageID         # Notion親ページID（初回のみ）
-sendEmail            # メール送信
-emailDaysBack        # メール対象期間（日数、デフォルト: 1）
```

## 🐛 トラブルシューティング

### PwC Japanのスクレイピングエラー
- **原因**: 3重エスケープされたJSON解析の失敗
- **確認**: `DEBUG_SCRAPING=1 ./pipeline -sources=pwc-japan -perSource=1 -queriesPerHeadline=0`
- **対処**: headlines.go の `collectHeadlinesPwCJapan()` の正規表現を確認

### OpenAI検索で結果が取得できない
- **原因**: `web_search_call.results` が空（仕様）
- **対処**: 正規表現によるURL抽出を使用（実装済み）

### Notionクリップでエラー
- **原因**: DATABASE_IDが保存されていない、または無効
- **対処**:
  1. `.env`の`NOTION_DATABASE_ID`を削除
  2. `-notionPageID`フラグで再実行
  3. 新しいDATABASE_IDが自動保存される

### メール送信エラー
- **原因**: Gmailアプリパスワード未設定または2段階認証未有効
- **対処**:
  1. Googleアカウント設定で2段階認証を有効化
  2. アプリパスワードを生成
  3. `.env`の`EMAIL_PASSWORD`に設定

## 📚 詳細ドキュメント

- **完全ガイド**: `docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md` (2,756行)
- **README**: `README.md`
- **プロジェクト分析**: `docs/architecture/PROJECT_COMPREHENSIVE_REPORT.md`
- **ドキュメント目次**: `docs/README.md`
- **スクリプト一覧**: `scripts/README.md`

## 🔄 最近の重要な変更（2026年1月4日）

1. **2つの運用モードを明確化**
   - モード1（無料記事収集）とモード2（有料記事マッチング）を区別
   - ドキュメント全体を更新

2. **PwC Japan実装修正**
   - 3重エスケープJSON解析を改善
   - 動作確認済み

3. **Carbon Knowledge Hub改善**
   - URL重複排除機能追加
   - CSSセレクタ改善

4. **統合テスト完了**
   - 全18ソース動作確認
   - 100%テスト合格（15/15）

## 💡 開発のヒント

- **新しいソース追加**: `headlines.go`に`collectHeadlines{SourceName}()`関数を追加
- **スクレイピング手法**: WordPress API > HTML Scraping > RSS Feed の順に検討
- **日本語ソース**: キーワードフィルタリングを実装（`carbonKeywords`配列）
- **テスト**: `/tmp/test_*.json`に出力してから実装
- **デバッグ**: 環境変数`DEBUG_*`を活用

## ⚠️ 注意事項

1. **有料記事の本文取得は実装しない** - ヘッドラインのみ使用
2. **OpenAI APIコスト** - モード2では1見出し×3クエリ = 3回のAPI呼び出し
3. **スクレイピングマナー** - 過度なリクエストを避ける
4. **環境変数の保護** - `.env`は`.gitignore`に含まれている

## 📞 参考情報

- **リポジトリ**: https://github.com/FuseKota/curbon-search.git
- **ブランチ**: main
- **Go バージョン**: 1.23+
- **ステータス**: 本番環境対応済み ✅
