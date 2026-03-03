# ヘッドライン収集ガイド

## 🎯 概要

このモードでは、**39の無料ソース**からカーボン関連ニュースのヘッドラインと記事要約を収集します。

- ✅ OpenAI API不要（OPENAI_API_KEY不要）
- ✅ 各種スクレイピング方式（WordPress API、HTML、RSSフィード）
- ✅ 記事の要約も自動取得
- ✅ 高速（検索処理なし）

---

## 🚀 クイックスタート

### 方法1: コマンドライン（最もシンプル）

```bash
# ビルド（初回のみ）
go build -o pipeline ./cmd/pipeline

# 全無料ソースからヘッドライン収集
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -out=headlines.json
```

---

## 📋 出力フォーマット

```json
[
  {
    "source": "Carbon Herald",
    "title": "EU carbon price hits record high amid supply concerns",
    "url": "https://carbonherald.com/article/...",
    "excerpt": "EU carbon prices reached a new record...",
    "isHeadline": true
  },
  {
    "source": "JRI",
    "title": "カーボンニュートラル達成に向けた政策動向",
    "url": "https://www.jri.co.jp/page.jsp?id=...",
    "isHeadline": true
  }
]
```

---

## 🔧 オプション

| オプション | デフォルト | 説明 |
|----------|----------|------|
| `-sources` | `all-free` | 収集元（カンマ区切りまたはall-free） |
| `-perSource` | `30` | 各ソースから収集する最大件数 |
| `-out` | - | 出力ファイルパス（未指定で標準出力） |
| `-hoursBack` | `0` | 指定時間以内の記事のみ（0で制限なし） |

---

## 📊 実行例

### 全無料ソースから収集
```bash
./pipeline \
  -sources=all-free \
  -perSource=10 \
  -out=all_headlines.json
```

### 日本ソースのみ
```bash
./pipeline \
  -sources=jri,env-ministry,meti,pwc-japan,mizuho-rt,jpx,carboncredits.jp \
  -perSource=20 \
  -out=japan_headlines.json
```

### 国際ソースのみ
```bash
./pipeline \
  -sources=carbonherald,carbon-brief,sandbag,icap,ieta,politico-eu \
  -perSource=20 \
  -out=international_headlines.json
```

### 標準出力（パイプで利用）
```bash
./pipeline \
  -sources=carbonherald \
  -perSource=5 \
  | jq -r '.[].title'
```

### 過去24時間の記事のみ
```bash
./pipeline \
  -sources=all-free \
  -perSource=30 \
  -hoursBack=24 \
  -out=recent_headlines.json
```

---

## 📰 利用可能なソース（39ソース）

### 日本ソース（5ソース）
| ソース名 | 説明 |
|---------|------|
| `jri` | 日本総研 |
| `pwc-japan` | PwC Japan |
| `mizuho-rt` | みずほリサーチ＆テクノロジーズ |
| `jpx` | 日本取引所グループ |
| `carboncredits.jp` | カーボンクレジット.jp |

### WordPress REST APIソース（6ソース）
| ソース名 | 説明 |
|---------|------|
| `carbonherald` | Carbon Herald |
| `climatehomenews` | Climate Home News |
| `carboncredits.com` | CarbonCredits.com |
| `sandbag` | Sandbag |
| `ecosystem-marketplace` | Ecosystem Marketplace |
| `carbon-brief` | Carbon Brief |

### HTMLスクレイピングソース（6ソース）
| ソース名 | 説明 |
|---------|------|
| `icap` | ICAP |
| `ieta` | IETA |
| `energy-monitor` | Energy Monitor |
| `world-bank` | World Bank |
| `newclimate` | NewClimate Institute |
| `carbon-knowledge-hub` | Carbon Knowledge Hub |

### VCM認証団体（4ソース）
| ソース名 | 説明 |
|---------|------|
| `verra` | Verra |
| `gold-standard` | Gold Standard |
| `acr` | American Carbon Registry |
| `car` | Climate Action Reserve |

### 国際機関（2ソース）
| ソース名 | 説明 |
|---------|------|
| `iisd` | IISD ENB |
| `climate-focus` | Climate Focus |

### 地域ETS（5ソース）
| ソース名 | 説明 |
|---------|------|
| `eu-ets` | EU ETS |
| `uk-ets` | UK ETS |
| `carb` | カリフォルニア大気資源局 |
| `rggi` | RGGI |
| `australia-cer` | オーストラリアCER |

### RSSフィード（3ソース）
| ソース名 | 説明 |
|---------|------|
| `politico-eu` | Politico EU |
| `euractiv` | Euractiv（RSS + 記事ページスクレイピングで全文取得） |
| `carbon-market-watch` | Carbon Market Watch |

### 学術・研究機関（2ソース）
| ソース名 | 説明 |
|---------|------|
| `arxiv` | arXiv |
| `oies` | オックスフォードエネルギー研究所 |

### CDR関連（2ソース）
| ソース名 | 説明 |
|---------|------|
| `puro-earth` | Puro.earth |
| `isometric` | Isometric |

---

## ⚠️ 注意事項

1. **スクレイピング制約**
   - サイトのレイアウト変更で動作しなくなる可能性があります
   - 過度なアクセスは避けてください

2. **日本語ソースのキーワードフィルタ**
   - JRI、Mizuho R&Tはカーボン関連キーワードでフィルタリングされます

3. **無意味なリンクは自動除外**
   - "Read more", "Click here"等は除外されます
   - 10文字未満のタイトルも除外されます

---

## 🆘 トラブルシューティング

### エラー: "no headlines collected"
```bash
# サイトがブロックしている可能性
# → User-Agentを確認
# → 手動でサイトにアクセスできるか確認
```

### ヘッドライン数が少ない
```bash
# perSource を増やす
./pipeline -sources=all-free -perSource=50
```

### 特定ソースがエラーになる
```bash
# デバッグモードで確認
DEBUG_SCRAPING=1 ./pipeline -sources=carbonherald -perSource=1
```

---

## 📚 関連ドキュメント

- **README.md** - プロジェクト全体の説明
- **QUICKSTART.md** - 5分で始めるガイド
- **VIEWING_GUIDE.md** - 収集結果の確認方法

---

**Have fun collecting! 📰**
