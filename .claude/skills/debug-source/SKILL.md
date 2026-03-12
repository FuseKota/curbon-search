---
name: debug-source
description: 特定ソースが 0 件を返す原因を診断する（サイト構造変更・セレクタ不一致・Bot保護等）
allowed-tools: Bash, Read, WebFetch, AskUserQuestion
---

# Carbon Relay: ソースデバッグスキル

`[WARN] {source} returned 0 headlines` などの問題が発生した際に、原因を特定して診断レポートを出力する。

---

## Step 1: 引数解析

`$ARGUMENTS` からソース名を取得する。

- 引数が指定されている場合 → そのまま SOURCE として使用
- 引数が未指定の場合 → `AskUserQuestion` でソース名を確認する

> どのソースをデバッグしますか？（例: `carbonherald`, `rmi`, `euractiv`）

---

## Step 2: ビルド確認

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
go build -o pipeline ./cmd/pipeline 2>&1 && echo "BUILD_OK" || echo "BUILD_FAILED"
```

`BUILD_FAILED` が出力された場合は即座に中断し、エラー内容を表示する。

---

## Step 3: DEBUG_SCRAPING で実行

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
DEBUG_SCRAPING=1 ./pipeline -sources={SOURCE} -perSource=1 2>&1
```

出力をそのまま保持する（URLアクセス結果・セレクタのマッチ状況が含まれる）。

---

## Step 4: ソース実装ファイルを特定・表示

ソース名から対応する実装ファイルを検索する：

```bash
grep -rn "{SOURCE}" /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay/internal/pipeline/ --include="*.go" -l
```

該当ファイルを `Read` で開き、以下の情報を抽出する：
- 収集関数名（`collectHeadlines*`）
- アクセスURL / フィードURL
- セレクタ（CSS / XPath / JSONパス）
- キーワードフィルタ（存在する場合）

---

## Step 5: 対象サイトの現状確認

Step 4 で特定した URL に `WebFetch` でアクセスする。

確認ポイント：
- HTTP ステータス（200 / 403 / 404 / 429 等）
- ページに記事一覧が存在するか
- 実装で使用しているセレクタ・フィードURLが現在も有効か
- Bot保護（Cloudflare / Imperva / CAPTCHA）の兆候がないか

---

## Step 6: 診断レポート出力

以下の形式でまとめて表示する：

```
=== debug-source: {SOURCE} ===

【DEBUG出力】
（Step 3 の出力をそのまま掲載）

【現在の実装】
ファイル: internal/pipeline/sources_xxx.go
URL: https://...
セレクタ / フィードURL: ...
キーワードフィルタ: あり / なし
（関数の主要部分を抜粋）

【サイト現状（WebFetch）】
HTTP: 200 OK / エラー
（記事一覧付近のHTML / JSONを抜粋）

【診断】
- URLにアクセスできるか      → ○ / ✗
- セレクタが一致しているか    → ○ / ✗（目視確認が必要）
- キーワードフィルタで除外されているか → ○ / ✗
- 考えられる原因: サイトリニューアル / Bot保護 / RSS URL変更 / キーワード不一致 など

【次のステップ】
→ セレクタ変更が必要な場合: internal/pipeline/sources_xxx.go を編集
→ 新しいURL/フィードへの切り替えが必要な場合: 該当ファイルのURLを更新
→ Bot保護で取得不可の場合: fetchViaCurl() の使用を検討、または /add-carbon-source で別方式に変更
→ キーワードフィルタが原因の場合: 該当ソース関数内のキーワード配列を確認・更新
```

---

## 注意事項

- `DEBUG_SCRAPING=1` を設定するとHTTPリクエストのURL・ステータス・セレクタのマッチ状況が詳細に出力される
- arXiv は IPベースのレート制限（429）が厳しく、0件になることがある（一時的な問題の可能性が高い）
- Bot保護（Cloudflare 等）がある場合は `fetchViaCurl()` でTLSフィンガープリントを偽装することで回避できる場合がある（Nature Communicationsの実装例を参照）
- キーワードフィルタが有効なソース（JRI、Mizuho R&T、IOP Science、Nature Eco&Evo、ScienceDirect）は、フィルタで全記事が除外されている可能性もある
