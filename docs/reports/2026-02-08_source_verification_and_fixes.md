# ソース検証・修正レポート（2026-02-08）

## 概要

各ソースの記事取得状況を1つずつ検証し、本文抽出・日付取得・Notion送信の問題を修正した。
検証済みソースはNotionデータベースへの実際の送信まで確認完了。

**作業期間**: 2026-02-08
**実装者**: Claude Code

---

## 1. 検証・修正したソース

### 1.1 RMI（Rocky Mountain Institute）

**問題**: WordPress REST APIの`content.rendered`が途中で切れる
**原因**: RMIはGutenbergブロック（Datawrapperチャート等）を多用しており、APIではインタラクティブ要素以降のテキストが返されない

**修正内容**:
- WordPress APIから`content`フィールドを除外し、記事一覧（タイトル・URL・日付）のみ取得
- 各記事ページを直接スクレイピングして全文取得
- 2つのテンプレートに対応:
  - 新テンプレート: `div.my-12.single_news_content-wrapper`
  - 旧テンプレート: `div.single_news_content-wrapper`（タイトル・ソーシャル・メタ情報を除去）
- `cleanExtractedText()`ヘルパー関数を追加（タブ・空白行の整理）
- script/style/iframe/svg要素を除去

**結果**: 記事本文が2,119文字 → 7,361文字に改善（「What is a BYO Tariff?」セクション含む全文取得成功）

### 1.2 Sandbag

**問題**: Excerptに`[et_pb_section fb_built="1" ...]`等のDiviショートコードが混入
**原因**: SandbagのWordPressがDiviページビルダーを使用しており、REST APIがショートコードを未レンダリングで返す

**修正内容**:
- `cleanHTMLTags()`関数に`reShortcodes`正規表現を追加
- パターン: `\[/?[a-z_]+[^\]]*\]` で全WordPressショートコードを除去
- 全WordPress系ソースに恩恵がある汎用修正

**結果**: 19,408文字（ショートコード含む）→ 9,994文字（クリーンテキスト）

### 1.3 Carbon Herald

**問題なし**: 記事3件、タイトル・日付・本文すべて正常取得

### 1.4 CarbonCredits.com

**問題なし**: 記事3件、タイトル・日付・本文（7,000〜8,500文字）すべて正常取得

### 1.5 JPX（日本取引所グループ）

**問題1**: キーワードフィルタにより0件（現在のRSSフィードにカーボン関連記事なし）
**問題2**: RSSフィードにdescriptionが含まれず、本文が空

**修正内容**:
- 記事ページから`p.component-text`セレクタで本文をスクレイピング
- フィルタは正常動作（カーボン関連記事がある場合のみ収集）

**結果**: テスト時（フィルタ一時無効）に本文526〜1,077文字を正常取得

### 1.6 Carbon Knowledge Hub

**問題**: 日付なし、Excerpt 0文字
**原因**: Next.jsアプリでコンテンツがクライアントサイドレンダリング。従来のHTMLスクレイピングではカテゴリ情報のみ取得

**修正内容**:
- `__NEXT_DATA__` scriptタグからfrontMatter JSON（date, description）を抽出
- SSRプリレンダリング済み`div#__next`から本文テキストを取得
- 1行目（パンくずリスト+メタ情報結合行）をスキップ

**結果**: 日付取得成功、本文3,154〜5,548文字を正常取得

### 1.7 World Bank

**問題**: 2件のみ取得、日付なし、1件はExcerpt 0文字
**原因**: HTMLスクレイピングのセレクタ（`div.featured`, `div.research`）がページ構造変更で無効化

**修正内容**:
- World Bank News Search API（`search.worldbank.org/api/v2/news`）に切り替え
- APIからURL・日付を取得、各ページから`<h1>`（タイトル）と`<p>`タグ（本文）をスクレイピング
- 英語記事のみフィルタ、日付順ソート

**結果**: 記事3件、日付・本文（2,732〜10,755文字）すべて正常取得

---

## 2. 並行作業で修正したソース

### 2.1 ICAP
- コンテンツセレクタを`.paragraph--type--text`に更新

### 2.2 IETA
- コンテンツセレクタを`.section-news-detail .intro, .section-news-detail section.bg-white`に更新

### 2.3 NewClimate Institute
- 記事ページから日付（`.event-details__name--calendar`）と本文（`.node__content`）をスクレイピング

### 2.4 PwC Japan
- 記事ページから`div.text-component`で本文を取得

### 2.5 Mizuho R&T
- `fetchMizuhoArticleDetail()`関数を追加
- `.report-detail_post`から本文、`<time>`タグから日付を抽出

### 2.6 JRI（日本総合研究所）
- コンテンツ抽出を`div.cont03`/`article#main`に改善
- キーワードフィルタをタイトル+本文に再有効化
- キーワードに「サステナビリティ」「エネルギー転換」「再生可能エネルギー」「グリーン」を追加

### 2.7 Notion（createContentBlocks）
- `maxBlockCount=100`制限を追加（Notion API AppendChildrenの上限対応）

---

## 3. Notion送信確認結果

| ソース | 件数 | 日付 | 本文 | Notion送信 |
|--------|------|------|------|------------|
| RMI | 3 | OK | OK (7,182〜36,222文字) | OK |
| Carbon Herald | 3 | OK | OK (1,826〜3,186文字) | OK |
| CarbonCredits.com | 3 | OK | OK (7,168〜8,455文字) | OK |
| Sandbag | 3 | OK | OK (2,186〜9,994文字) | OK |
| JPX | 3* | OK | OK (54〜1,077文字) | OK |
| Carbon Knowledge Hub | 3 | OK | OK (3,154〜5,548文字) | OK |
| World Bank | 3 | OK | OK (2,732〜10,755文字) | OK |

*JPXはテスト時にキーワードフィルタを一時無効化して確認

---

## 4. コミット

```
1805706 fix: Strip WordPress/Divi shortcodes and improve RMI scraping
624c0c2 fix: Improve content extraction for multiple sources
```

---

## 5. 未検証ソース（次回以降）

以下のソースは今回未検証:
- carboncredits.jp, climatehomenews, ecosystem-marketplace, carbon-brief
- icap, ieta, energy-monitor, newclimate
- jri, env-ministry, meti, pwc-japan, mizuho-rt
- politico-eu, euractiv, arxiv, oies, iopscience, nature-ecoevo, sciencedirect
- verra, gold-standard, acr, car, iisd, climate-focus
- eu-ets, uk-ets, carb, rggi, australia-cer, puro-earth, isometric
