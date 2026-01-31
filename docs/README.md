# Carbon Relay Documentation

このディレクトリには、Carbon Relayプロジェクトの全ドキュメントが整理されています。

## 📖 クイックナビゲーション

### 🚀 ガイド (guides/)

実際の使い方や開発方法に関するガイドです。

- **[QUICKSTART.md](./guides/QUICKSTART.md)** - 最速で動かすためのクイックスタートガイド
- **[DEVELOPMENT.md](./guides/DEVELOPMENT.md)** - 開発環境のセットアップと開発ワークフロー
- **[HEADLINES_ONLY.md](./guides/HEADLINES_ONLY.md)** - ヘッドライン収集のみのモード（OpenAI不要）
- **[NOTION_INTEGRATION.md](./guides/NOTION_INTEGRATION.md)** - Notion統合の設定と使い方
- **[VIEWING_GUIDE.md](./guides/VIEWING_GUIDE.md)** - 収集結果の閲覧方法

### 📊 レポート (reports/)

プロジェクトの状態やテスト結果に関するレポートです。

- **[STATUS.md](./reports/STATUS.md)** - プロジェクトの現在の状態
- **[PROJECT_STATE.md](./reports/PROJECT_STATE.md)** - プロジェクト状態の詳細レポート
- **[TEST_REPORT.md](./reports/TEST_REPORT.md)** - テスト結果レポート
- **[TEST_SUMMARY.md](./reports/TEST_SUMMARY.md)** - テスト結果サマリー
- **[INTEGRATION_TEST_REPORT.md](./reports/INTEGRATION_TEST_REPORT.md)** - 統合テスト結果レポート

### 🏗️ アーキテクチャ (architecture/)

システムの設計と実装の詳細ドキュメントです。

- **[COMPLETE_IMPLEMENTATION_GUIDE.md](./architecture/COMPLETE_IMPLEMENTATION_GUIDE.md)** - 完全実装ガイド（2,756行）
  - 全18ソースの実装詳細
  - スコアリングアルゴリズム解説
  - 2つの運用モードの詳細説明
- **[PROJECT_COMPREHENSIVE_REPORT.md](./architecture/PROJECT_COMPREHENSIVE_REPORT.md)** - プロジェクト包括レポート
- **[SYSTEM_FLOW.md](./architecture/SYSTEM_FLOW.md)** - システムフロー図と説明

## 🎯 目的別ガイド

### 初めての方

1. [QUICKSTART.md](./guides/QUICKSTART.md) - まずはこちらから
2. [HEADLINES_ONLY.md](./guides/HEADLINES_ONLY.md) - OpenAI不要でヘッドライン収集を試す
3. [VIEWING_GUIDE.md](./guides/VIEWING_GUIDE.md) - 結果の見方を学ぶ

### 開発者の方

1. [DEVELOPMENT.md](./guides/DEVELOPMENT.md) - 開発環境のセットアップ
2. [COMPLETE_IMPLEMENTATION_GUIDE.md](./architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) - 実装の全貌を理解
3. [SYSTEM_FLOW.md](./architecture/SYSTEM_FLOW.md) - システムアーキテクチャの理解

### Notion統合したい方

1. [NOTION_INTEGRATION.md](./guides/NOTION_INTEGRATION.md) - Notionの設定方法
2. [COMPLETE_IMPLEMENTATION_GUIDE.md](./architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) セクション6 - API詳細

### テスト結果を確認したい方

1. [TEST_SUMMARY.md](./reports/TEST_SUMMARY.md) - サマリーから確認
2. [TEST_REPORT.md](./reports/TEST_REPORT.md) - 詳細なテスト結果
3. [INTEGRATION_TEST_REPORT.md](./reports/INTEGRATION_TEST_REPORT.md) - 統合テスト

## 📚 その他の重要ドキュメント

プロジェクトルートにも重要なドキュメントがあります：

- **[../README.md](../README.md)** - プロジェクトのメインREADME
- **[../CLAUDE.md](../CLAUDE.md)** - Claude Code向け開発指示書
- **[../.claude/](../.claude/)** - Claude Code用コンテキストファイル
  - [PROJECT_CONTEXT.md](../.claude/PROJECT_CONTEXT.md) - プロジェクトコンテキスト
  - [COMMANDS.md](../.claude/COMMANDS.md) - コマンドクイックリファレンス

## 💡 ドキュメント更新ポリシー

- **ガイド**: ユーザー向けの手順書は簡潔に保つ
- **レポート**: テスト後は必ず更新する
- **アーキテクチャ**: コード変更時は該当セクションを更新

## 🔍 検索のヒント

特定の情報を探す場合：

- **ソース実装の詳細**: [COMPLETE_IMPLEMENTATION_GUIDE.md](./architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) セクション3
- **スコアリングアルゴリズム**: [COMPLETE_IMPLEMENTATION_GUIDE.md](./architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) セクション5
- **トラブルシューティング**: [COMPLETE_IMPLEMENTATION_GUIDE.md](./architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) セクション10
- **コマンド例**: [../.claude/COMMANDS.md](../.claude/COMMANDS.md)

---

**最終更新**: 2026-01-31
**ディレクトリ整理**: プロジェクト構造改善の一環として実施
