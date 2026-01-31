# Carbon Relay Scripts

このディレクトリには、Carbon Relayプロジェクトの便利なスクリプトが含まれています。

## 📜 スクリプト一覧

### 🔨 ビルド・デプロイ

#### `build_lambda.sh`
AWS Lambda用のデプロイパッケージをビルドします。

```bash
./scripts/build_lambda.sh
```

**出力**: `carbon-relay-lambda.zip` (Lambda関数としてアップロード可能)

---

### 📰 ヘッドライン収集

#### `collect_headlines_only.sh`
有料記事と無料記事のヘッドラインのみを収集します（OpenAI不要）。

```bash
./scripts/collect_headlines_only.sh
```

**モード**: 無料記事収集モード（モード1）
**出力**: `headlines_YYYYMMDD_HHMMSS.json`

#### `view_headlines.sh`
収集済みのヘッドラインJSONファイルを見やすく表示します。

```bash
./scripts/view_headlines.sh headlines_20260131_120000.json
```

**用途**: 収集結果の確認

#### `collect_and_view.sh`
ヘッドライン収集と表示を一度に実行します。

```bash
./scripts/collect_and_view.sh
```

**便利機能**: 収集→即座に表示

---

### 🔍 関連記事検索・マッチング

#### `check_related.sh`
有料記事に対する関連無料記事を検索・スコアリングします（OpenAI使用）。

```bash
./scripts/check_related.sh
```

**モード**: 有料記事マッチングモード（モード2）
**要件**: `OPENAI_API_KEY`必須
**出力**: `related_articles_YYYYMMDD_HHMMSS.json`

#### `full_pipeline.sh`
フルパイプライン（ヘッドライン収集 + 関連記事検索 + メール送信）を実行します。

```bash
./scripts/full_pipeline.sh
```

**モード**: モード2（完全版）
**要件**:
- `OPENAI_API_KEY`
- `EMAIL_FROM`, `EMAIL_PASSWORD`, `EMAIL_TO`（メール送信時）

**出力**:
- JSON結果ファイル
- メール送信（設定されている場合）

---

### 📋 Notion統合

#### `clip_to_notion.sh`
指定したソースの記事をNotionデータベースにクリップします。

```bash
./scripts/clip_to_notion.sh carbonpulse
```

**要件**:
- `NOTION_API_KEY`
- `NOTION_PAGE_ID`（初回のみ）

**動作**:
- Database IDを`.env`に自動保存
- 既存記事の重複チェック
- リッチテキスト形式でクリップ

#### `clip_all_sources.sh`
全ソースの記事を一括でNotionにクリップします。

```bash
./scripts/clip_all_sources.sh
```

**注意**: 大量のAPI呼び出しが発生するため、Notion APIレート制限に注意

#### `test_notion.sh`
Notion統合のテスト用スクリプト。

```bash
./scripts/test_notion.sh
```

**用途**: Notion API接続のデバッグ・検証

---

### 🎓 実行例・学習

#### `run_examples.sh`
Carbon Relayの様々な使い方の実行例を表示します。

```bash
./scripts/run_examples.sh
```

**内容**:
- モード1（ヘッドライン収集のみ）の例
- モード2（関連記事検索）の例
- Notion統合の例
- メール送信の例

---

## 🎯 使用シナリオ別ガイド

### シナリオ1: 初めて使う

```bash
# 1. ヘッドライン収集を試す（OpenAI不要）
./scripts/collect_headlines_only.sh

# 2. 結果を確認
./scripts/view_headlines.sh headlines_*.json
```

### シナリオ2: 関連記事を検索したい

```bash
# OpenAI APIキーが設定されていることを確認
echo $OPENAI_API_KEY

# 関連記事検索を実行
./scripts/check_related.sh
```

### シナリオ3: Notionに記事をクリップしたい

```bash
# 1. 環境変数を設定（.envファイルに記載）
# NOTION_API_KEY=secret_xxxxx
# NOTION_PAGE_ID=xxxxx-xxxxx-xxxxx

# 2. 特定ソースをクリップ
./scripts/clip_to_notion.sh carbonpulse

# 3. 全ソースをクリップ（時間がかかります）
./scripts/clip_all_sources.sh
```

### シナリオ4: AWS Lambdaにデプロイしたい

```bash
# 1. Lambda用パッケージをビルド
./scripts/build_lambda.sh

# 2. 生成されたzipファイルをAWSにアップロード
# carbon-relay-lambda.zip を Lambda コンソールでアップロード
```

### シナリオ5: フルパイプラインを定期実行したい

```bash
# cronで定期実行する例
# 毎日午前9時に実行
# 0 9 * * * cd /path/to/carbon-relay && ./scripts/full_pipeline.sh
```

---

## ⚙️ スクリプトのカスタマイズ

### パラメータの調整

各スクリプト内で以下のパラメータを調整できます：

**収集記事数**:
```bash
# 各ソースから5記事取得
./pipeline -sources=all -perSource=5 -queriesPerHeadline=0
```

**検索クエリ数**:
```bash
# 各見出しから3つの検索クエリを生成
./pipeline -sources=carbonpulse -perSource=3 -queriesPerHeadline=3
```

**特定ソースのみ**:
```bash
# Carbon PulseとQCIのみ
./pipeline -sources=carbonpulse,qci -perSource=5 -queriesPerHeadline=3
```

### 出力先の変更

```bash
# 出力ファイル名を指定
./pipeline -sources=all -perSource=5 -out=/tmp/custom_output.json
```

---

## 🐛 トラブルシューティング

### スクリプトが実行できない

```bash
# 実行権限を付与
chmod +x ./scripts/*.sh
```

### パスの問題

全てのスクリプトは**プロジェクトルートから実行**することを想定しています：

```bash
# 正しい実行方法
cd /path/to/carbon-relay
./scripts/collect_headlines_only.sh

# 間違った実行方法（エラーになる可能性）
cd /path/to/carbon-relay/scripts
./collect_headlines_only.sh
```

### 環境変数が読み込まれない

```bash
# .envファイルが存在するか確認
ls -la .env

# 環境変数を手動で読み込み
source .env
```

---

## 📝 スクリプト作成時の規約

新しいスクリプトを追加する場合：

1. **命名規則**: アンダースコア区切り（例: `new_script.sh`）
2. **シバン**: `#!/bin/bash`を先頭に記載
3. **実行権限**: `chmod +x scripts/new_script.sh`
4. **エラーハンドリング**: `set -e`でエラー時に停止
5. **ドキュメント**: このREADMEに説明を追加

---

## 🔗 関連ドキュメント

- **使い方の詳細**: [../docs/guides/QUICKSTART.md](../docs/guides/QUICKSTART.md)
- **開発ガイド**: [../docs/guides/DEVELOPMENT.md](../docs/guides/DEVELOPMENT.md)
- **コマンドリファレンス**: [../.claude/COMMANDS.md](../.claude/COMMANDS.md)
- **完全実装ガイド**: [../docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md](../docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md)

---

**最終更新**: 2026-01-31
**スクリプト数**: 10個
**プロジェクトディレクトリ**: `/Users/kotafuse/Yasui/Prog/Test/carbon-relay/`
