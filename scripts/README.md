# Carbon Relay Scripts

このディレクトリには、Carbon Relayプロジェクトのスクリプトが含まれています。

日常的なヘッドライン収集・Notionクリップ・メール送信は `./pipeline` コマンドで直接実行できます。
コマンドリファレンスは [../.claude/COMMANDS.md](../.claude/COMMANDS.md) を参照してください。

## 📜 スクリプト一覧

### `build_lambda.sh`

AWS Lambda用のデプロイパッケージをビルドします。

```bash
./scripts/build_lambda.sh
```

**出力**: `carbon-relay-lambda.zip` (Lambda関数としてアップロード可能)

### `view_headlines.sh`

収集済みのヘッドラインJSONファイルを見やすく表示します。

```bash
./scripts/view_headlines.sh headlines_20260131_120000.json
```

**用途**: 収集結果の確認（`jq` による整形表示）

---

## 🔗 関連ドキュメント

- **コマンドリファレンス**: [../.claude/COMMANDS.md](../.claude/COMMANDS.md)
- **使い方ガイド**: [../docs/guides/QUICKSTART.md](../docs/guides/QUICKSTART.md)
- **完全実装ガイド**: [../docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md](../docs/architecture/COMPLETE_IMPLEMENTATION_GUIDE.md)

---

**最終更新**: 2026-03-03
**スクリプト数**: 2個
