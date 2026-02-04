# Claude Code - プロジェクト固有指示

このファイルはClaude Codeがこのプロジェクトで作業する際の重要な指示を含みます。

---

## 🎯 プロジェクトの理解

**Carbon Relay**は、カーボン関連ニュースの収集・分析・配信を自動化するGo製システムです。

### 運用モード

このシステムは**無料記事収集モード**で運用します：

- 36の無料ソースから記事を直接収集
- OpenAI API不要（コスト効率が高い）
- メール配信・Notion統合が主な用途

---

## 📚 必読ドキュメント

作業開始前に以下を参照：

1. **docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md**
   - セクション8: 使用方法と実行例
   - 全36ソースの実装詳細

2. **.claude/PROJECT_CONTEXT.md**
   - プロジェクト概要とコンテキスト
   - よく使うコマンド
   - トラブルシューティング

3. **.claude/COMMANDS.md**
   - コマンドクイックリファレンス

4. **docs/README.md**
   - ドキュメント目次とナビゲーション
   - 目的別ガイドへのリンク

---

## 🔧 コード変更時の重要事項

### ソース実装（headlines.go）を変更する場合

1. **テストを先に実行**
   ```bash
   ./pipeline -sources={source-name} -perSource=5 -queriesPerHeadline=0 -out=/tmp/test.json
   ```

2. **デバッグフラグを活用**
   ```bash
   DEBUG_SCRAPING=1 ./pipeline -sources={source-name} -perSource=1 -queriesPerHeadline=0
   ```

3. **PwC Japanの特殊性に注意**
   - 3重エスケープされたJSON解析
   - 正規表現パターンが複雑
   - 変更時は必ず動作確認

4. **日本語ソースのキーワードフィルタリング**
   - JRI、環境省、METI、Mizuho R&Tはキーワードフィルタあり
   - `carbonKeywords`配列を参照

### スコアリング（matcher.go）を変更する場合

1. **IDF計算ロジックを理解する**
   - セクション5参照（COMPLETE_IMPLEMENTATION_GUIDE_2026.md）

2. **重みの調整は慎重に**
   - 現在の重み: IDF加重リコール56% + Jaccard28% + Market6% + Topic4% + Geo2% + Recency4%

3. **品質ブーストの変更**
   - `.gov`ドメイン: +0.18
   - `.pdf`: +0.18
   - NGO: +0.12

### Notion統合（notion.go）を変更する場合

1. **Database ID自動保存機能を維持**
   - `.env`への自動保存は重要な機能

2. **リッチテキスト分割（2000文字制限）**
   - Notion APIの制限に注意

---

## 🚫 やってはいけないこと

1. **過度なリクエストを送らない**
   - スクレイピング時は適切な間隔を保つ
   - テスト時は`-perSource`を小さく設定

2. **環境変数をコミットしない**
   - `.env`は`.gitignore`に含まれている

---

## 🐛 デバッグ時の手順

### ステップ1: 問題を特定
```bash
# エラーログを確認
./pipeline ... 2>&1 | grep ERROR
```

### ステップ2: 該当ソースを単独テスト
```bash
./pipeline -sources={問題のソース} -perSource=1 -queriesPerHeadline=0
```

### ステップ3: デバッグフラグを有効化
```bash
DEBUG_SCRAPING=1 ./pipeline -sources={問題のソース} -perSource=1 -queriesPerHeadline=0
```

### ステップ4: コードを確認
- `cmd/pipeline/headlines.go`の該当関数を確認
- セレクタやURLパターンを検証

---

## 📝 コミットメッセージ規約

このプロジェクトでは以下の規約を使用：

- `feat:` - 新機能追加
- `fix:` - バグ修正
- `docs:` - ドキュメント変更
- `refactor:` - リファクタリング
- `test:` - テスト追加・修正

**例**:
```
feat: Add PwC Japan source with JSON parsing

- Implemented 3-level unescaping for embedded JSON
- Added date parsing for YYYY-MM-DD format
- Tested with 5 articles successfully
```

末尾に以下を追加：
```
🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

---

## 🔄 よくあるタスクのフロー

### 新しいソースを追加する場合

1. **調査フェーズ**
   - Webサイトの構造を確認
   - WordPress API、HTML構造、RSSフィードを調査

2. **実装フェーズ**
   - `headlines.go`に`collectHeadlines{SourceName}()`関数を追加
   - `main.go`の`sourceMap`に追加

3. **テストフェーズ**
   ```bash
   ./pipeline -sources={new-source} -perSource=5 -queriesPerHeadline=0 -out=/tmp/test_new.json
   cat /tmp/test_new.json | jq '.'
   ```

4. **ドキュメント更新**
   - docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.mdのセクション3に追加
   - README.mdのソースリストに追加

### スコアリングアルゴリズムを調整する場合

1. **現在のスコアを記録**
   ```bash
   ./pipeline ... -out=before.json
   cat before.json | jq '[.[].relatedFree[]?.score] | add / length'
   ```

2. **変更を実装**
   - `matcher.go`を編集

3. **効果を測定**
   ```bash
   ./pipeline ... -out=after.json
   cat after.json | jq '[.[].relatedFree[]?.score] | add / length'
   ```

4. **A/Bテスト**
   - 同じ見出しで変更前後を比較

---

## 🎓 学習リソース

- **Notion API制限**: セクション6.3
- **全36ソースの実装パターン**: セクション3

---

## 🔑 環境変数チェックリスト

作業開始前に`.env`が正しく設定されているか確認：

- [ ] `NOTION_API_KEY` - Notionクリップで必要
- [ ] `NOTION_PAGE_ID` - 初回クリップで必要
- [ ] `EMAIL_FROM` - メール送信で必要
- [ ] `EMAIL_PASSWORD` - Gmailアプリパスワード
- [ ] `EMAIL_TO` - メール送信先

---

## 💡 ベストプラクティス

1. **小さく始める**
   - `-perSource=1`でテストしてから増やす

2. **ログを確認する**
   - エラーが出ても慌てない
   - デバッグフラグで詳細を確認

3. **既存のパターンに従う**
   - 新機能は既存のコードスタイルに合わせる

4. **ドキュメントを更新する**
   - コード変更時は必ずドキュメントも更新

5. **テストを忘れない**
   - 変更後は必ず動作確認

---

## 📞 緊急時の対応

### システムが動作しない場合

1. **ビルドを確認**
   ```bash
   go build -o pipeline ./cmd/pipeline
   ```

2. **環境変数を確認**
   ```bash
   cat .env | grep -v PASSWORD
   ```

3. **最小構成でテスト**
   ```bash
   ./pipeline -sources=carbonherald -perSource=1 -queriesPerHeadline=0
   ```

4. **ドキュメントを参照**
   - トラブルシューティング: docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md セクション10

---

## 📊 プロジェクト統計

- **総コード行数**: 約6,000行（Go）
- **実装ソース数**: 36（無料ソースのみ）
- **ステータス**: 本番環境対応済み ✅

---

**最終更新**: 2026年2月4日（全36ソース対応）
**プロジェクトパス**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/`
**リポジトリ**: https://github.com/FuseKota/curbon-search.git

---

## 📁 ディレクトリ構造（2026年1月31日改善）

プロジェクトディレクトリは以下のように整理されています：

```
carbon-relay/
├── docs/                    # 📚 ドキュメント
│   ├── README.md           # ドキュメント目次
│   ├── guides/             # 使い方ガイド
│   ├── reports/            # テストレポート
│   └── architecture/       # 実装詳細
├── scripts/                 # 🛠️ スクリプト
│   ├── README.md           # スクリプト一覧
│   └── *.sh                # 各種便利スクリプト
├── cmd/pipeline/           # Go実装
├── CLAUDE.md               # このファイル
└── README.md               # メインREADME
```

**重要な変更点**:
- すべてのMarkdownドキュメントは`docs/`配下に移動
- すべてのシェルスクリプトは`scripts/`配下に移動
- ルートディレクトリがすっきりして見やすくなった
