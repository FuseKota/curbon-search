# Carbon Pulse / QCI 有料ソース削除レポート

**日付**: 2026年2月4日
**作業者**: Claude Code
**ステータス**: 完了

---

## 概要

Carbon RelayプロジェクトからCarbon PulseとQCI（有料ニュースソース）を完全に削除し、関連するドキュメントとコードを更新しました。これに伴い、「モード2: 有料記事マッチングモード」の機能参照もすべて削除しました。

---

## 削除の理由

Carbon PulseとQCIは有料記事サイトであり、システムで使用する必要がなくなったため削除されました。

---

## 変更されたファイル一覧

### コアコードファイル（以前の会話で更新済み）

| ファイル | 変更内容 |
|---------|---------|
| `internal/pipeline/headlines.go` | sourceCollectorsからcarbonpulse/qciを削除、HeadlineSourceConfigから有料ソース用フィールドを削除 |
| `internal/pipeline/sources_paid.go` | **削除** - Carbon Pulse/QCIスクレイピング実装 |
| `internal/pipeline/notion.go` | Notionソースオプションからcarbon pulse/qciを削除 |
| `cmd/pipeline/headlines.go` | extractExcerptFromContext関数からCarbon Pulse固有コードを削除 |

### 主要ドキュメント（以前の会話で更新済み）

| ファイル | 変更内容 |
|---------|---------|
| `CLAUDE.md` | 2つの運用モードを単一モードに変更、ソース数を18から16に更新 |
| `README.md` | モード2セクション削除、コマンド例更新 |
| `.claude/PROJECT_CONTEXT.md` | モード2削除、OpenAI参照削除、データソースリスト更新 |
| `docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md` | セクション3.1（有料ソース）削除、セクション番号再割り当て |

### スクリプトファイル（以前の会話で更新済み）

| ファイル | 変更内容 |
|---------|---------|
| `scripts/README.md` | 有料ソース参照とモード2例を削除 |
| `scripts/collect_headlines_only.sh` | all-freeソースを使用するよう完全書き換え |
| `scripts/collect_and_view.sh` | デフォルトソースをall-freeに変更 |
| `scripts/full_pipeline.sh` | OpenAI/検索機能を削除し完全書き換え |
| `scripts/run_examples.sh` | 新しいソース名で例を完全書き換え |

### ガイドドキュメント（本セッションで更新）

| ファイル | 変更内容 |
|---------|---------|
| `docs/guides/VIEWING_GUIDE.md` | Carbon Pulse/QCIの例をall-free/Carbon Herald/JRI等に変更 |
| `docs/guides/NOTION_INTEGRATION.md` | 有料ソース参照削除、OpenAI要件削除、例を更新 |
| `docs/guides/HEADLINES_ONLY.md` | 完全書き換え - 16無料ソースの収集ガイドに変更 |
| `docs/guides/QUICKSTART.md` | 完全書き換え - 新しいソースとコマンドに対応 |
| `docs/guides/DEVELOPMENT.md` | 完全書き換え - アーキテクチャ図とソース一覧を更新 |

---

## 削除された機能

### 1. 有料ソース収集

以下のソースのスクレイピング機能が削除されました：

- **Carbon Pulse** (`carbonpulse`)
  - Daily Timeline ページ
  - Newsletters カテゴリ

- **QCI** (`qci`)
  - Carbon セクション

### 2. モード2: 有料記事マッチングモード

以下の機能参照が削除されました：

- 有料記事ヘッドラインからの無料記事検索
- OpenAI Web Search API 統合
- IDF（逆文書頻度）マッチング
- スコアリングアルゴリズム（市場/トピック/地域シグナル）
- `relatedFree` フィールドの生成

### 3. 削除されたコマンドラインオプション参照

ドキュメントから以下のオプション使用例が削除されました：

- `-sources=carbonpulse`
- `-sources=qci`
- `-sources=carbonpulse,qci`
- `-queriesPerHeadline=3` または `5`（検索用）
- `-resultsPerQuery=10` または `20`
- `-topK=3`
- `-minScore=0.32`

---

## 現在のシステム構成

### 利用可能なソース（16個）

#### 日本ソース（7個）
1. `jri` - 日本総研
2. `env-ministry` - 環境省
3. `meti` - 経産省 審議会
4. `pwc-japan` - PwC Japan
5. `mizuho-rt` - みずほリサーチ＆テクノロジーズ
6. `jpx` - 日本取引所グループ
7. `carboncredits.jp` - カーボンクレジット.jp

#### 国際ソース（9個）
1. `carbonherald` - Carbon Herald
2. `carbon-brief` - Carbon Brief
3. `sandbag` - Sandbag
4. `icap` - ICAP
5. `ieta` - IETA
6. `politico-eu` - Politico EU
7. `iisd` - IISD SDG Knowledge Hub
8. `unfccc` - UNFCCC News
9. `gef` - Global Environment Facility

### 標準実行コマンド

```bash
# 全無料ソースから収集
./pipeline -sources=all-free -perSource=10 -queriesPerHeadline=0 -out=headlines.json

# 日本ソースのみ
./pipeline -sources=jri,env-ministry,meti -perSource=10 -queriesPerHeadline=0

# 国際ソースのみ
./pipeline -sources=carbonherald,carbon-brief,sandbag -perSource=10 -queriesPerHeadline=0
```

---

## 変更されていないファイル

以下のファイルは変更を必要としませんでした：

- `docs/reports/` 配下の過去のレポート（履歴として保持）
- `go.mod`, `go.sum`（依存関係に変更なし）
- Lambda/メール送信関連のコード（有料ソースに依存していない）

---

## 注意事項

1. **過去のレポートは変更していません**
   - `docs/reports/` 配下の過去のテストレポートは履歴として保持
   - 当時のテスト結果を反映した状態を維持

2. **環境変数の変更**
   - `OPENAI_API_KEY` は現在のシステムでは不要
   - `NOTION_TOKEN` はNotionクリップ機能を使用する場合のみ必要

3. **後方互換性**
   - 古いコマンド（`-sources=carbonpulse`等）は「unknown source」エラーとなります
   - 新しいソース名（`all-free`, `carbonherald`等）を使用してください

---

## 確認コマンド

変更後のシステムが正常に動作することを確認するコマンド：

```bash
# ビルド
go build -o pipeline ./cmd/pipeline

# クイックテスト（全ソース各1件）
./pipeline -sources=all-free -perSource=1 -queriesPerHeadline=0

# 日本ソーステスト
./pipeline -sources=jri -perSource=3 -queriesPerHeadline=0

# デバッグモードでのテスト
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=2 -queriesPerHeadline=0
```

---

## 関連ドキュメント

- [COMPLETE_IMPLEMENTATION_GUIDE.md](../architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) - 実装詳細
- [QUICKSTART.md](../guides/QUICKSTART.md) - クイックスタートガイド
- [HEADLINES_ONLY.md](../guides/HEADLINES_ONLY.md) - ヘッドライン収集ガイド

---

**レポート作成**: 2026年2月4日
**最終確認**: Claude Code
