# carbon-relay プロジェクトステータス

**最終更新日：** 2025-12-29

---

## 🎯 プロジェクトの現状

### MVP完成度：✅ 80%

**動作状況：**
- ✅ Carbon Pulse / QCI からヘッドライン収集
- ✅ OpenAI web_search でURL探索
- ✅ URLから疑似タイトル生成
- ✅ IDF + シグナルベースマッチング
- ✅ relatedFree として一次情報を出力

**品質：**
- 🟡 検索結果の精度：中（タイトル生成が疑似的なため）
- ✅ マッチングの精度：高
- 🟡 一次情報の優先度：中（site:演算子で改善済み、さらに要調整）

---

## 📊 実績データ（2025-12-29テスト）

### テストケース：Carbon Pulse 5件

| 見出し | relatedFree数 | トップスコア | 一次情報割合 |
|-------|-------------|-----------|-----------|
| Climate litigation marks turning point | 3 | 0.79 | 67% (PDF 2件) |
| US DOE expands 45V hydrogen credits | 3 | 0.81 | 33% (energy.gov) |
| Hawaii court cruise ship climate levy | 3 | 0.80 | 100% (現地メディア) |
| CP Daily Newsletter (無関係) | 1 | 0.64 | 0% |
| CP News Ticker (無関係) | 1 | 0.72 | 100% (PDF) |

**総合評価：**
- carbon/climate関連の見出しでは**優秀な結果**
- ニュースレター系の見出しは検索結果の質が低い（期待通り）
- 一次情報（.gov, .pdf, 現地メディア）の取得率：**60%**

---

## 🚨 既知の問題・制約

### 重大（Critical）

#### 1. OpenAI Responses API の構造化データ欠如
**問題：**
```json
// 期待：
{"results": [{"title": "...", "url": "...", "snippet": "..."}]}

// 実際：
{"results": []}  // 常に空
```

**影響：**
- URLからタイトルを推測するしかない
- 精度が低下（特にランダムファイル名のPDF）

**対策：**
- ✅ 疑似タイトル生成で暫定対応
- 🔲 Brave Search API導入（推奨）

---

### 中程度（Medium）

#### 2. 無関係な見出しでノイズが多い
**問題：**
- "CP Daily Newsletter" → Christian Postの記事が返される

**対策：**
- 検索クエリにドメイン除外を追加？（`-site:christianpost.com`）
- 見出しフィルタの強化

#### 3. 一部の地域・市場でカバレッジ不足
**問題：**
- アジア（日本・中国）の一次情報が少ない
- 新興市場（マレーシア、ベトナム等）の情報が取れない

**対策：**
- 地域別サイトリストの拡充
- ローカル言語での検索クエリ生成

---

### 軽微（Minor）

#### 4. スクレイピングの安定性
**問題：**
- サイトレイアウト変更でスクレイピング失敗のリスク

**対策：**
- 複数のセレクタパターンを用意
- エラー時のフォールバック実装

#### 5. パフォーマンス
**問題：**
- 10見出し処理に約60秒かかる

**対策：**
- 並列化（goroutine）
- クエリ数の動的調整

---

## 🔧 技術的負債

| 項目 | 優先度 | 説明 |
|-----|-------|------|
| ユニットテスト未実装 | 中 | matcher.go, search_queries.go にテストなし |
| エラーハンドリング不十分 | 低 | 一部のエラーを握りつぶしている |
| ログ出力の体系化 | 低 | DEBUG_OPENAI 以外のログがない |
| 設定ファイル対応 | 低 | すべてコマンドライン引数 |
| キャッシュ機構 | 低 | 同じクエリを再検索している |

---

## 📈 次のマイルストーン

### Phase 2: 精度向上（推奨期間：1週間）

- [ ] Brave Search API 統合
  - [ ] search_brave.go 実装
  - [ ] main.go で provider 切り替え
  - [ ] パフォーマンス比較（OpenAI vs Brave）

- [ ] 検索クエリ最適化
  - [ ] 企業名自動抽出（NER風）
  - [ ] 時間範囲フィルタ（after:2024-01-01）
  - [ ] ドメイン除外リスト（-site:...）

### Phase 3: プロダクション準備（推奨期間：2週間）

- [ ] ユニットテスト実装（カバレッジ70%以上）
- [ ] 統合テスト実装
- [ ] エラーハンドリング強化
- [ ] ログ出力体系化（structured logging）
- [ ] 設定ファイル対応（YAML/TOML）
- [ ] Docker対応
- [ ] CI/CD構築

### Phase 4: UI・定期実行（推奨期間：3週間）

- [ ] Web UI実装（見出し表示 + relatedFree）
- [ ] API サーバー化（REST/GraphQL）
- [ ] 定期実行スクリプト（cron）
- [ ] 結果の永続化（DB）
- [ ] ユーザー認証

---

## 💾 データベース設計（Phase 4用）

```sql
-- headlines テーブル
CREATE TABLE headlines (
    id SERIAL PRIMARY KEY,
    source VARCHAR(50) NOT NULL,  -- 'Carbon Pulse' / 'QCI'
    title TEXT NOT NULL,
    url TEXT UNIQUE NOT NULL,
    collected_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_collected_at (collected_at)
);

-- related_free テーブル
CREATE TABLE related_free (
    id SERIAL PRIMARY KEY,
    headline_id INT REFERENCES headlines(id),
    source VARCHAR(50) NOT NULL,  -- 'OpenAI' / 'Brave'
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    score FLOAT NOT NULL,
    reason TEXT,
    matched_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_headline_id (headline_id),
    INDEX idx_score (score)
);

-- search_logs テーブル（デバッグ用）
CREATE TABLE search_logs (
    id SERIAL PRIMARY KEY,
    query TEXT NOT NULL,
    provider VARCHAR(20) NOT NULL,  -- 'openai' / 'brave'
    results_count INT,
    latency_ms INT,
    executed_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🧪 テスト環境

### 開発環境
- Go 1.23
- macOS Darwin 24.4.0
- OpenAI API（gpt-4o-mini）

### テストデータ
```bash
# 小規模テスト（デバッグ用）
./carbon-relay -sources=carbonpulse -perSource=2

# 中規模テスト（品質確認用）
./carbon-relay -sources=carbonpulse,qci -perSource=10

# 大規模テスト（パフォーマンス確認用）
./carbon-relay -sources=carbonpulse,qci -perSource=50
```

### パフォーマンスベンチマーク（参考）
```
2見出し処理：   ~15秒
10見出し処理：  ~60秒
50見出し処理：  ~300秒（5分）
```

---

## 📚 ドキュメント構成

| ファイル | 対象読者 | 内容 |
|---------|---------|------|
| README.md | すべて | プロジェクト概要・実行方法 |
| DEVELOPMENT.md | 開発者 | アーキテクチャ・アルゴリズム詳細 |
| STATUS.md | PM/開発者 | 現状・課題・次のステップ |
| go.mod / go.sum | 開発者 | 依存関係 |

---

## 🔐 環境変数一覧

```bash
# 必須
export OPENAI_API_KEY="sk-..."

# オプション（Phase 2以降）
export BRAVE_API_KEY="..."        # Brave Search API
export DATABASE_URL="..."         # PostgreSQL接続文字列

# デバッグ用
export DEBUG_OPENAI=1             # OpenAI検索のデバッグ出力
export DEBUG_OPENAI_FULL=1        # APIレスポンス全体を出力
export LOG_LEVEL=debug            # ログレベル（未実装）
```

---

## 🎓 学んだこと・知見

### OpenAI Responses API について

1. **web_searchツールは検索エンジンではない**
   - 検索結果を返すのではなく、検索を「代行」してテキストにまとめる
   - 構造化データの取得には向かない

2. **includeパラメータは期待通り動かない**
   - `include: ["web_search_call.results"]` を指定しても空
   - デフォルトでも同じ

3. **max_output_tokensは小さくても問題ない**
   - 検索自体は実行される
   - messageの生成が制限されるだけ

### マッチングアルゴリズムについて

1. **IDF加重が非常に重要**
   - 単純なJaccard類似度では "the", "a", "in" などが邪魔
   - IDF加重でレアなトークン（"biochar", "CORSIA"等）を重視

2. **シグナル（market/topic/geo）の併用が効果的**
   - 語彙的類似度だけでは不十分
   - ドメイン特有の知識を組み込むことで精度向上

3. **ドメイン品質スコアが決め手**
   - .gov / .pdf に +0.18 するだけで一次情報が上位に

### スクレイピングについて

1. **無意味なリンクテキストが想像以上に多い**
   - "Read more", "Click here", "Continue reading"
   - 最小文字数チェック（len < 10）が有効

2. **Carbon Pulseのリンク構造は安定している**
   - `/数字/` パターンは長期間不変
   - ただし将来的に変わる可能性あり

---

## 📞 サポート・連絡先

- **Issues**: （GitHubリポジトリのIssuesページ）
- **Email**: （メールアドレス）
- **Slack**: （チャンネル名）

---

## 📄 ライセンス

（プロジェクトライセンス）

---

**最終更新者：** Claude Sonnet 4.5
**最終更新日：** 2025-12-29
