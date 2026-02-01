# 新規ソース追加 実装レポート

**実施日**: 2026年1月31日
**ステータス**: 完了（13ソース有効、6ソース一時無効化）

---

## 概要

モード1（無料記事収集）用に19の新規ソースを実装。そのうち13ソースが正常動作、6ソースはWebサイト側の制限により一時無効化。

---

## 新規作成ファイル

| ファイル | 説明 | 関数数 |
|----------|------|--------|
| `cmd/pipeline/sources_academic.go` | 学術・研究機関ソース | 3 |
| `cmd/pipeline/sources_regional_ets.go` | 地域ETSソース | 4 |

---

## 拡張したファイル

| ファイル | 追加内容 | 関数数 |
|----------|----------|--------|
| `cmd/pipeline/sources_html.go` | VCM認証団体、国際機関、CDR関連 | 9 |
| `cmd/pipeline/sources_rss.go` | EU政策RSS | 2 |
| `cmd/pipeline/headlines.go` | sourceCollectorsマップ更新 | - |

---

## 動作確認済みソース（13個）

### VCM認証団体（4ソース）

| 識別子 | ソース名 | URL | 実装方法 |
|--------|----------|-----|----------|
| `verra` | Verra | https://verra.org/news/ | HTML |
| `gold-standard` | Gold Standard | https://www.goldstandard.org/newsroom | HTML |
| `acr` | American Carbon Registry | https://acrcarbon.org/news/ | HTML |
| `car` | Climate Action Reserve | https://climateactionreserve.org/updates/ | HTML |

### 国際機関（2ソース）

| 識別子 | ソース名 | URL | 実装方法 |
|--------|----------|-----|----------|
| `iisd` | IISD ENB | https://enb.iisd.org/ | HTML |
| `climate-focus` | Climate Focus | https://climatefocus.com/publications/ | HTML |

### 学術・研究機関（2ソース）

| 識別子 | ソース名 | URL | 実装方法 |
|--------|----------|-----|----------|
| `arxiv` | arXiv | http://export.arxiv.org/api/query | XML API |
| `nature-comms` | Nature Communications | https://www.nature.com/ncomms.rss | RSS + キーワードフィルタ |

### 地域ETS（4ソース）

| 識別子 | ソース名 | URL | 実装方法 |
|--------|----------|-----|----------|
| `eu-ets` | EU ETS (EC) | https://climate.ec.europa.eu/news-other-reads/news_en | HTML |
| `carb` | California CARB | https://ww2.arb.ca.gov/news | HTML |
| `rggi` | RGGI | https://www.rggi.org/news-releases/rggi-releases | HTML (Table) |
| `australia-cer` | Australia CER | https://cer.gov.au/news-and-media/news | HTML |

### CDR関連（1ソース）

| 識別子 | ソース名 | URL | 実装方法 |
|--------|----------|-----|----------|
| `isometric` | Isometric | https://isometric.com/writing | HTML |

---

## 一時無効化ソース（6個）

| 識別子 | ソース名 | 無効化理由 | 対応策 |
|--------|----------|------------|--------|
| `euractiv` | Euractiv | Cloudflare保護 | ヘッドレスブラウザ必要 |
| `uk-ets` | UK ETS | Atomフィードが空 | 別のフィードURL調査 |
| `unfccc` | UNFCCC | Incapsula保護 | ヘッドレスブラウザ必要 |
| `oies` | OIES | JavaScript描画必須 | ヘッドレスブラウザ必要 |
| `puro-earth` | Puro.earth | 構造化ニュースなし | 別ページ調査 |
| `carbon-market-watch` | Carbon Market Watch | 403 Forbidden | 既存問題 |

---

## テスト結果

```
$ ./pipeline -sources=verra,gold-standard,acr,car,iisd,climate-focus,isometric,arxiv,nature-comms,eu-ets,carb,rggi,australia-cer -perSource=2 -queriesPerHeadline=0

結果: 25 articles collected
```

### 個別テスト結果

```
verra               : 2 articles ✅
gold-standard       : 2 articles ✅
acr                 : 2 articles ✅
car                 : 2 articles ✅
iisd                : 2 articles ✅
climate-focus       : 2 articles ✅
isometric           : 2 articles ✅
arxiv               : 2 articles ✅
nature-comms        : 1 articles ✅ (キーワードフィルタのため少なめ)
eu-ets              : 2 articles ✅
carb                : 2 articles ✅
rggi                : 2 articles ✅
australia-cer       : 2 articles ✅
```

---

## プロジェクト全体のソース数

| カテゴリ | 有効 | 無効 |
|----------|------|------|
| 有料ソース | 2 | 0 |
| WordPress API | 7 | 0 |
| HTMLスクレイピング | 18 | 2 |
| 日本語ソース | 6 | 0 |
| RSS/Atom | 1 | 2 |
| 学術・研究 | 2 | 1 |
| 地域ETS | 4 | 0 |
| **合計** | **35** | **6** |

---

## 使用方法

### 新規ソースの個別テスト

```bash
# 単一ソース
./pipeline -sources=verra -perSource=5 -queriesPerHeadline=0 -out=/tmp/test.json

# デバッグモード
DEBUG_SCRAPING=1 ./pipeline -sources=verra -perSource=1 -queriesPerHeadline=0
```

### 新規ソース一括テスト

```bash
./pipeline -sources=verra,gold-standard,acr,car,iisd,climate-focus,isometric,arxiv,nature-comms,eu-ets,carb,rggi,australia-cer -perSource=3 -queriesPerHeadline=0 -out=/tmp/test_new.json
```

---

## 未コミットの変更

```
modified:   cmd/pipeline/headlines.go
modified:   cmd/pipeline/sources_html.go
modified:   cmd/pipeline/sources_rss.go
new file:   cmd/pipeline/sources_academic.go
new file:   cmd/pipeline/sources_regional_ets.go
```

### コミットコマンド

```bash
git add cmd/pipeline/sources_academic.go \
        cmd/pipeline/sources_regional_ets.go \
        cmd/pipeline/headlines.go \
        cmd/pipeline/sources_html.go \
        cmd/pipeline/sources_rss.go

git commit -m "feat: Add 13 new carbon news sources (VCM, regional ETS, academic)

- VCM certification bodies: Verra, Gold Standard, ACR, CAR
- International orgs: IISD ENB, Climate Focus
- Academic: arXiv (XML API), Nature Communications (RSS)
- Regional ETS: EU ETS, California CARB, RGGI, Australia CER
- CDR: Isometric

6 sources temporarily disabled due to bot protection or JS rendering

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## 今後の課題

1. **ヘッドレスブラウザ対応**: UNFCCC、Euractiv、OIESはPlaywright等での対応が必要
2. **UK ETS**: 正しいフィードURLの調査
3. **Puro.earth**: プレスリリースページの調査
4. **定期監視**: 各ソースの構造変更を検知する仕組み

---

## 関連ドキュメント

- [COMPLETE_IMPLEMENTATION_GUIDE.md](../architecture/COMPLETE_IMPLEMENTATION_GUIDE.md) - 全体実装ガイド
- [CLAUDE.md](../../CLAUDE.md) - プロジェクト固有指示

---

**作成者**: Claude Opus 4.5
**最終更新**: 2026年1月31日
