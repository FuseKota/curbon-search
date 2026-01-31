# .claude ディレクトリ

このディレクトリには、Claude Codeの新しいセッションへの引き継ぎに必要な情報が含まれています。

## 📁 ファイル構成

### 1. PROJECT_CONTEXT.md
**プロジェクト全体のコンテキスト情報**

内容:
- プロジェクト概要と2つの運用モード
- 主要ファイル構成とソースリスト
- よく使うコマンド
- 環境変数設定
- トラブルシューティング

**こんな時に参照**:
- 新しいセッションを開始した時
- プロジェクトの全体像を思い出したい時
- どのコマンドを使うべきか迷った時

---

### 2. COMMANDS.md
**コマンドのクイックリファレンス**

内容:
- モード別のコマンド例
- テストコマンド
- デバッグコマンド
- JSON出力の確認方法
- 実用的なスクリプト例

**こんな時に参照**:
- 具体的なコマンドを忘れた時
- デバッグ方法を知りたい時
- JSONの確認方法を調べたい時

---

### 3. settings.json
**プロジェクト全体のHooks設定**

内容:
- .env保護フック（誤編集防止）
- Go自動フォーマットフック
- コンテキスト再注入フック
- モード認識リマインダー

**こんな時に役立つ**:
- 自動的に動作（手動操作不要）
- セキュリティとコード品質を自動保証

---

### 4. config.json
**プロジェクト設定とメタデータ**

内容:
- 2つの運用モードの詳細
- 環境変数の説明とセットアップガイド
- 全18ソースのリスト
- テストコマンドパターン
- コーディング規約
- ワークフロー定義

**こんな時に参照**:
- プロジェクト全体の構成を理解したい時
- 環境変数のセットアップ方法を知りたい時
- ベストプラクティスを確認したい時

---

### 5. settings.local.json
**Claude Code のローカル設定**

自動生成されるファイル。手動編集は通常不要。

---

## 🚀 クイックスタート

新しいClaude Codeセッションを開始したら：

1. **PROJECT_CONTEXT.md**を読む
   - プロジェクトの2つのモードを理解
   - 主要ファイルの場所を確認

2. **CLAUDE.md**（プロジェクトルート）を確認
   - プロジェクト固有の重要な指示
   - やってはいけないこと

3. **COMMANDS.md**を参照
   - 必要なコマンドをコピー＆実行

---

## 📚 その他の重要ドキュメント

プロジェクトルートにある以下のファイルも参照：

- **CLAUDE.md** - Claude Code向けプロジェクト固有指示
- **docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md** - 完全実装ガイド（2,756行）
- **README.md** - プロジェクト概要
- **docs/architecture/PROJECT_COMPREHENSIVE_REPORT.md** - プロジェクト分析レポート
- **docs/README.md** - ドキュメント目次とナビゲーション
- **scripts/README.md** - スクリプト一覧と使い方

---

## 💡 使い方のヒント

### パターン1: 特定のソースでエラーが出た
```bash
# 1. COMMANDS.mdの「デバッグコマンド」セクションを参照
# 2. 該当ソースを単独でテスト
DEBUG_SCRAPING=1 ./pipeline -sources={問題のソース} -perSource=1 -queriesPerHeadline=0

# 3. PROJECT_CONTEXT.mdの「トラブルシューティング」を確認
```

### パターン2: 新しい機能を追加したい
```bash
# 1. CLAUDE.mdの「よくあるタスクのフロー」を確認
# 2. docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.mdの該当セクションを詳細に読む
# 3. 既存の実装パターンを参考にする
```

### パターン3: コマンドを忘れた
```bash
# COMMANDS.mdをざっと眺めて該当コマンドを探す
# または、PROJECT_CONTEXT.mdの「よく使うコマンド」セクション
```

### パターン4: 新しいニュースソースを追加したい（推奨ワークフロー）
```bash
# 1. source-researcherエージェントでソースを調査
#    Claude Codeで: "Launch source-researcher agent to analyze https://example.com"
#
# 2. エージェントが提供するコードテンプレートを実装
#
# 3. /test-sourceスキルでテスト
#    Claude Codeで: /test-source newsource
#
# 4. code-reviewerエージェントでレビュー
#    Claude Codeで: "Launch code-reviewer agent to review my changes"
#
# 5. /commit-patternスキルでコミット
#    Claude Codeで: /commit-pattern "Add new source: Example News"
```

### パターン5: コード変更をレビューしてほしい
```bash
# code-reviewerエージェントを起動
# Claude Codeで:
# "Launch code-reviewer agent to review my recent changes"
#
# エージェントが以下をチェック:
# - セキュリティ問題
# - ベストプラクティス準拠
# - パフォーマンス問題
# - Carbon Relay固有パターン
```

### パターン6: 環境変数の設定方法がわからない
```bash
# config.jsonを参照
# すべての環境変数の説明とセットアップガイドが記載されている
#
# または /check-env コマンドで現在の設定を確認
```

---

## 🔄 このディレクトリの更新

このディレクトリのファイルは、プロジェクトの大きな変更があった時に更新してください：

- 新しいソースを追加した時
- 新しい運用モードを追加した時
- よく使うコマンドが変わった時
- 重要なトラブルシューティング情報が見つかった時

---

**最終更新**: 2026年1月31日（ディレクトリ構造改善に伴う更新）

---

## 🎯 自動化機能（2026年1月31日追加）

このプロジェクトには、Claude Codeが自動的にサポートする機能が組み込まれています：

### Hooks（自動保護）
- `.env`ファイルの誤編集を防止
- Go自動フォーマット（編集後）
- セッション開始時のコンテキスト再注入
- 初回プロンプト時の2モードリマインダー

### Skills（タスクパターン）
- `/test-source <name>` - ソースのテスト
- `/verify-mode <task>` - モード確認
- `/commit-pattern <desc>` - コミットメッセージ生成
- `/add-source <url>` - 新しいソースの追加ガイド

### Commands（クイックアクセス）
- `/build` - パイプラインのビルド
- `/test-all` - 全ソースのクイックテスト
- `/check-env` - 環境変数の確認

### Agents（専門タスク）
- `source-researcher` - 新しいニュースソースの調査・分析
  - WordPress API vs HTML scraping判定
  - セレクタ抽出と実装推奨
  - Goコードテンプレート生成
- `code-reviewer` - コード変更のレビュー
  - Carbon Relayベストプラクティスチェック
  - セキュリティ脆弱性検出
  - パフォーマンス問題の指摘
  - アーキテクチャ準拠確認

### Config（プロジェクト設定）
- `config.json` - プロジェクト全体の設定とメタデータ
  - 運用モードの定義
  - 環境変数ガイド
  - コーディング規約
  - 推奨ワークフロー

これらの機能は、GitHubからcloneした直後から自動的に利用可能です。

---

## 📁 ディレクトリ構造の改善（2026年1月31日）

プロジェクトのディレクトリ構造が整理されました：

- **docs/** - すべてのMarkdownドキュメント
  - guides/ - 使い方ガイド
  - reports/ - テストレポート
  - architecture/ - 実装詳細
- **scripts/** - すべてのシェルスクリプト

詳細は `docs/README.md` と `scripts/README.md` を参照してください。
