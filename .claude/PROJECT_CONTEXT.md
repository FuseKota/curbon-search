# Carbon Relay - プロジェクトコンテキスト

このファイルは新しいClaude Codeセッションへの引き継ぎ用です。

## プロジェクト概要

**Carbon Relay**は、カーボン関連ニュースの収集・分析・配信を自動化するGo製システムです。

**プロジェクトパス**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/`

## 🔑 運用モード

### 🟢 無料記事収集モード
- **目的**: Carbon関連の無料記事を幅広く収集してメール配信/Notion統合
- **コマンド**: `./pipeline -sources=all-free -perSource=10 -sendEmail`
- **特徴**: 高速実行（5-15秒）、コスト効率が高い
- **用途**: 日次のカーボンニュースレビュー

## 📁 主要ファイル構成

```
cmd/pipeline/
├── main.go              - エントリーポイント、CLI制御
internal/pipeline/
├── headlines.go         - 共通ロジック
├── sources_wordpress.go - WordPress REST APIソース
├── sources_html.go      - HTMLスクレイピングソース
├── sources_japan.go     - 日本語ソース
├── sources_rss.go       - RSSフィードソース
├── notion.go            - Notion統合
├── email.go             - メール送信
├── types.go             - データ構造
└── utils.go             - ユーティリティ
```

## 🗂️ データソース（36ソース）

### 無料ソース

**日本市場（7ソース）**:
1. CarbonCredits.jp
2. JRI (日本総研)
3. Environment Ministry (環境省)
4. JPX (日本取引所グループ)
5. METI (経済産業省)
6. Mizuho R&T (みずほリサーチ＆テクノロジーズ)
7. PwC Japan

**WordPress API（6ソース）**:
8. Carbon Herald
9. Climate Home News
10. CarbonCredits.com
11. Sandbag
12. Ecosystem Marketplace
13. Carbon Brief

**HTMLスクレイピング（6ソース）**:
14. ICAP
15. IETA
16. Energy Monitor
17. World Bank
18. NewClimate Institute
19. Carbon Knowledge Hub

**VCM認証団体（4ソース）**:
20. Verra
21. Gold Standard
22. ACR
23. CAR

**国際機関（2ソース）**:
24. IISD ENB
25. Climate Focus

**地域ETS（5ソース）**:
26. EU ETS
27. UK ETS
28. CARB
29. RGGI
30. Australia CER

**RSSフィード（3ソース）**:
31. Politico EU
32. Euractiv（RSS + 記事ページスクレイピングで全文取得）
33. Carbon Market Watch

**学術・研究（6ソース）**:
34. arXiv
35. Nature Communications（curl方式でTLSフィンガープリント回避）
36. OIES
37. IOP Science (ERL)
38. Nature Eco&Evo
39. ScienceDirect

**CDR関連（2ソース）**:
40. Puro.earth
41. Isometric

## 🛠️ よく使うコマンド

### ビルド
```bash
cd /Users/kotafuse/Yasui/Prog/Test/carbon-relay
go build -o pipeline ./cmd/pipeline
```

### 無料記事収集
```bash
# 全無料ソースから収集（all-freeで36ソース全て指定）
./pipeline -sources=all-free -perSource=10 -out=free_articles.json

# メール送信付き
./pipeline -sources=all-free -perSource=10 -sendEmail

# Notion挿入付き
./pipeline -sources=all-free -perSource=10 -notionClip
```

### 特定ソースのテスト
```bash
# 日本市場のみ
./pipeline -sources=jri,env-ministry,jpx,pwc-japan -perSource=5

# PwC Japanのテスト（複雑なJSON解析）
./pipeline -sources=pwc-japan -perSource=5 -out=/tmp/pwc_test.json
```

### デバッグ
```bash
# スクレイピングのデバッグ
DEBUG_SCRAPING=1 ./pipeline -sources=pwc-japan -perSource=5

# 詳細デバッグ
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=1
```

## 🔧 環境変数（.env）

必須の環境変数:
```bash
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
-sources              # ソース指定（CSV形式、"all-free"で全36ソース）
-perSource            # ソースあたりの記事数（デフォルト: 30）
-hoursBack            # 指定時間以内の記事のみ（デフォルト: 0、日付なし記事は保持）
-out                  # 出力ファイル（省略時はstdout）
-notionClip           # Notionにクリップ
-notionPageID         # Notion親ページID（初回のみ）
-sendEmail            # メール送信
```

## 🐛 トラブルシューティング

### スクレイピングエラー
- **確認**: `DEBUG_SCRAPING=1 ./pipeline -sources=問題のソース -perSource=1`
- **対処**: sources_*.go の該当関数のセレクタを確認

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

## 🔄 最近の重要な変更（2026年2月4日）

### インフラ改善
1. **HTTPクライアント共有（コネクションプーリング）**
   - 全ソースで共有クライアントを使用
   - MaxIdleConns: 100, MaxIdleConnsPerHost: 10
   - タイムアウト: 30秒（20秒から増加）

2. **WordPress API日付処理改善**
   - `date`から`date_gmt`フィールドに変更
   - UTC形式（Z suffix）で統一

3. **日付フィルタリング改善**
   - `FilterHeadlinesByHours`: 日付なし記事を保持
   - `time.Now()`フォールバックを廃止（空文字列に変更）
   - 全ソースでUTC形式に統一（JST→UTC）

### ソース修正
4. **Mizuho R&T**: 年を動的取得（`time.Now().Year()`）
5. **リソースリーク修正**: JRI、環境省のdeferループ問題を修正
6. **正規表現最適化**: パッケージレベルで事前コンパイル

### CLI改善
7. **`all-free`サポート**: `-sources=all-free`で全36ソース指定可能

## 💡 開発のヒント

- **新しいソース追加**: `headlines.go`に`collectHeadlines{SourceName}()`関数を追加
- **スクレイピング手法**: WordPress API > HTML Scraping > RSS Feed の順に検討
- **日本語ソース**: キーワードフィルタリングを実装（`carbonKeywords`配列）
- **テスト**: `/tmp/test_*.json`に出力してから実装
- **デバッグ**: 環境変数`DEBUG_*`を活用

## ⚠️ 注意事項

1. **スクレイピングマナー** - 過度なリクエストを避ける
2. **環境変数の保護** - `.env`は`.gitignore`に含まれている

## 📞 参考情報

- **リポジトリ**: https://github.com/FuseKota/curbon-search.git
- **ブランチ**: main
- **Go バージョン**: 1.23+
- **ステータス**: 本番環境対応済み ✅
