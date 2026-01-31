# carbon-relay テストサマリー

**テスト実施日**: 2026-01-02
**総合評価**: ✅ 合格（本番環境使用可能）

---

## 📊 テスト結果一覧

| 機能 | 結果 | 詳細 |
|------|------|------|
| Carbon Pulse収集 | ✅ | 3件、excerpt付き |
| QCI収集 | ✅ | 3件取得 |
| CarbonCredits.jp | ✅ | 全文1,689文字 |
| Carbon Herald | ✅ | 全文3,734文字 |
| Climate Home News | ⚠️ | タイムアウト（サイト側問題） |
| CarbonCredits.com | ✅ | 全文12,724文字 |
| OpenAI Web検索 | ✅ | URL抽出＋タイトル生成 |
| マッチングエンジン | ✅ | スコア0.86達成 |
| Notion DB作成 | ✅ | 既存DB検出 |
| Notion記事保存 | ✅ | 全文保存（AI Summary） |
| メール送信 | ✅ | 6件送信成功 |
| .env自動読み込み | ✅ | godotenv動作 |

**成功率**: 92% (11/12)

---

## ✅ 動作確認済み機能

### ヘッドライン収集
- ✅ Carbon Pulse（有料見出し + excerpt）
- ✅ QCI（有料見出し）
- ✅ CarbonCredits.jp（無料全文、日本語）
- ✅ Carbon Herald（無料全文）
- ✅ CarbonCredits.com（無料全文、12K文字対応）

### Web検索 & マッチング
- ✅ OpenAI Responses API統合
- ✅ URL抽出（正規表現）
- ✅ 疑似タイトル生成
- ✅ IDF計算
- ✅ 類似度スコアリング（titleSim=0.94）
- ✅ シグナル抽出（topic=1.00）
- ✅ topK選定（3件）

### Notion統合
- ✅ データベース検出・接続
- ✅ AI Summaryフィールドに全文保存
- ✅ 複数ブロック分割（2000文字/ブロック）
- ✅ ページ本文保存（paragraph blocks）
- ✅ 日本語対応

### メール送信
- ✅ Notionからデータ取得（6件）
- ✅ プレーンテキスト生成
- ✅ Gmail SMTP送信
- ✅ App Password認証
- ✅ 日本語・英語混在対応

### その他
- ✅ .env自動読み込み（godotenv）
- ✅ エラーハンドリング
- ✅ 長文処理（12K文字以上）

---

## 📈 パフォーマンス

| 処理 | 時間 |
|------|------|
| ヘッドライン収集 | 2-3秒/件 |
| 全文取得 | 4秒/件 |
| OpenAI検索 | 10秒/クエリ |
| Notion保存 | 2.5秒/件 |
| メール送信 | 5秒 |

**メモリ使用量**: 50-80MB（リークなし）

---

## ⚠️ 既知の問題

### Climate Home News タイムアウト
- **原因**: サイト側のレスポンス遅延
- **影響**: 1ソースが利用不可（他5ソースは正常）
- **対応**: タイムアウト値調整検討（20秒→30秒）
- **優先度**: 低

---

## 🎯 テスト実証データ

### マッチングスコア例
```json
{
  "score": 0.8645141215139844,
  "reason": "overlap=1.00 titleSim=0.94 topic=1.00 sharedTokens=14"
}
```

### 全文取得例
- **CarbonCredits.jp**: 1,689文字（日本語）
- **Carbon Herald**: 3,734文字
- **CarbonCredits.com**: 12,724文字

### Notion保存
- **AI Summary**: 複数ブロック分割（2000文字×N）
- **ページ本文**: paragraphブロック（2000文字×N）
- **日本語**: 正常処理

---

## ✨ 結論

**すべての主要機能が正常に動作しています。**

carbon-relayプロジェクトは本番環境で使用可能な状態です。

---

詳細は [TEST_REPORT.md](TEST_REPORT.md) を参照してください。
