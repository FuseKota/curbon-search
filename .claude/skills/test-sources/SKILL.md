---
name: test-sources
description: 全ソース（または指定ソース）をテスト実行し収集結果を確認する
allowed-tools: Bash, AskUserQuestion
disable-model-invocation: true
---

# Carbon Relay: ソーステスト実行スキル

引数 `$ARGUMENTS` を解析し、パイプラインをビルド・実行して結果をまとめて表示する。

---

## Step 1: 引数解析

`$ARGUMENTS` を以下のルールで解析する：

| 引数パターン | `-sources` | `-perSource` |
|-------------|-----------|-------------|
| （なし） | `all-free` | `3` |
| 数値のみ（例: `5`） | `all-free` | その数値 |
| ソース名のみ（例: `carbonherald`） | そのソース名 | `5` |
| ソース名 + 数値（例: `carbonherald 10`） | そのソース名 | その数値 |
| カンマ区切り（例: `carbonherald,rmi`） | `carbonherald,rmi` | `5` |

## Step 2: ビルド確認

```bash
cd /Users/kotafuse/Work/Yasui/Prog/Test/carbon-relay
go build -o pipeline ./cmd/pipeline 2>&1 && echo "BUILD_OK" || echo "BUILD_FAILED"
```

`BUILD_FAILED` が出力された場合は即座に中断し、エラー内容を表示する。

## Step 3: モード確認

`AskUserQuestion` でユーザーに確認する：

> Notion に挿入しますか？
>
> 1. 収集のみ（`-notionClip` なし）
> 2. Notion にも挿入（`-notionClip` フラグを追加）

回答に応じてモードを決定する：
- `収集のみ` → `-notionClip` なし
- `Notion にも挿入` → `-notionClip` フラグを追加

## Step 4: テスト実行

出力ファイルを一時パスに保存し、STDERRも別ファイルに保存する。

**収集のみ**（Step 3 で「収集のみ」を選択した場合）：

```bash
OUT=/tmp/test_sources_$(date +%s).json
ERR=/tmp/test_sources_err_$(date +%s).txt

./pipeline -sources={SOURCES} -perSource={PER_SOURCE} -out="$OUT" 2>"$ERR"

echo "OUT=$OUT"
echo "ERR=$ERR"
```

**Notion にも挿入**（Step 3 で「Notion にも挿入」を選択した場合）：

```bash
OUT=/tmp/test_sources_$(date +%s).json
ERR=/tmp/test_sources_err_$(date +%s).txt

./pipeline -sources={SOURCES} -perSource={PER_SOURCE} -out="$OUT" -notionClip 2>"$ERR"

echo "OUT=$OUT"
echo "ERR=$ERR"
```

## Step 5: 結果表示

### 全体件数

```bash
echo "=== 総収集件数 ==="
jq 'length' "$OUT"
```

### ソース別件数サマリー

```bash
echo "=== ソース別件数 ==="
jq -r 'group_by(.source) | map({source: .[0].source, count: length}) | sort_by(.source)[] | "\(.source): \(.count)件"' "$OUT"
```

### ソース別先頭タイトル

```bash
echo "=== ソース別先頭タイトル ==="
jq -r 'group_by(.source) | map({source: .[0].source, title: .[0].title}) | sort_by(.source)[] | "[\(.source)] \(.title)"' "$OUT"
```

### エラー・警告の抽出

```bash
echo "=== エラー/警告 ==="
grep -E "ERROR|WARN" "$ERR" || echo "（エラーなし）"
```

### Notion挿入結果（Step 3 で「Notion にも挿入」を選択した場合のみ）

STDERRからNotion関連のログを抽出して表示する：

```bash
echo "=== Notion挿入結果 ==="
grep -E "Notion|notion|clip|Clip|inserted|created|page" "$ERR" || echo "（Notionログなし）"
```

---

## 表示例

### 収集のみ

```
=== 総収集件数 ===
87

=== ソース別件数 ===
carbonherald: 3件
rmi: 3件
...

=== ソース別先頭タイトル ===
[carbonherald] Carbon Markets Weekly: ...
[rmi] The Case for Clean Energy in ...
...

=== エラー/警告 ===
（エラーなし）
```

### Notion挿入あり

```
=== 総収集件数 ===
15

=== ソース別件数 ===
carbonherald: 5件
rmi: 5件
...

=== ソース別先頭タイトル ===
[carbonherald] Carbon Markets Weekly: ...
...

=== エラー/警告 ===
（エラーなし）

=== Notion挿入結果 ===
INFO: Clipping 15 headlines to Notion...
INFO: Created page: Carbon Markets Weekly: ...
INFO: Created page: The Case for Clean Energy in ...
...
INFO: Notion clip complete: 15 pages created
```

---

## 注意事項

- `all-free` は `config.go` の `DefaultSources` を使用するため、新ソース追加後は自動で対象に含まれる
- arXiv は IPベースのレート制限（429）が厳しいため、他ソースと同時テスト時に失敗することがある
- テスト後の一時ファイルは `/tmp/test_sources_*.json` に残る（手動削除可）
