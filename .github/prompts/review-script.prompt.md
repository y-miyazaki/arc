---
name: "review-script"
description: "Shell Script正確性・セキュリティ・保守性・ベストプラクティス準拠レビュー"
tools: ["context7", "fetch_webpage"]
---

# Shell Script Review Prompt

Shell Script ベストプラクティス精通エキスパート。正確性・セキュリティ・保守性・業界標準準拠レビュー。
validate_all_scripts.sh 自動化検証前提。MCP: github, context7, fetch_webpage。レビューコメント日本語。

## Review Guidelines (ID Based)

### 1. Global / Base (G)

- G-01: validate_all_scripts.sh 検証合格（shebang・権限・構文・shellcheck・関数分析）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-02: Shebang 明記（#!/bin/bash 必須）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-03: set -euo pipefail 必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-04: SCRIPT_DIR 設定+lib/all.sh source
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-05: 機密情報ハードコーディング禁止
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-06: 命名規則統一（snake_case/UPPER_SNAKE_CASE）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-07: 関数順序遵守（show_usage/parse_arguments→a-z 順 →main 最後）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-08: デッドコード削除
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-09: error_exit 利用エラーハンドリング
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- G-10: スクリプト冪等性
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 2. Code Standards (CODE)

- CODE-01: 引用符適切（変数"$var"、リテラル'string'）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-02: コマンド置換$()推奨
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-03: test [[]]推奨
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-04: 算術演算$(())推奨
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-05: 配列適切利用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-06: グローバル変数最小化（local 宣言）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-07: パイプライン set -o pipefail
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-08: Here document 適切利用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-09: Process substitution 適切利用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- CODE-10: 関数単一責任・引数明示
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 3. Function Design (FUNC)

- FUNC-01: 関数順序遵守（show_usage/parse_arguments→a-z 順 →main 最後）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-02: 関数 50 行以下推奨
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-03: parse_arguments 標準化（case 文・getopts）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-04: show_usage 実装（Usage/Description/Options/Examples・exit 0）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-05: 戻り値設計（return code・echo・エラー伝播）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-06: local 変数宣言
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-07: 共通ライブラリ活用（error_exit/log_message 等）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-08: error_exit 利用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-09: validate_dependencies 関数（コマンド存在確認）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- FUNC-10: main 関数実装
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 4. Error Handling (ERR)

- ERR-01: set -e 必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-02: set -u 必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-03: set -o pipefail 必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-04: trap 設定（EXIT・ERR・INT・TERM）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-05: 終了コード確認
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-06: エラーメッセージ明確
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-07: クリーンアップ処理
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-08: リトライ戦略
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-09: 部分的失敗許容（set +e 一時解除）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- ERR-10: エラーログ記録
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 5. Security (SEC)

- SEC-01: 機密情報環境変数化
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-02: 入力値検証
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-03: コマンドインジェクション対策（引用符）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-04: パス traversal 対策
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-05: 一時ファイル mktemp+trap 削除
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-06: 権限チェック
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-07: ログ機密情報マスク
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-08: 外部コマンド検証
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-09: 環境変数汚染回避
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- SEC-10: セキュアデフォルト（umask 027）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 6. Performance (PERF)

- PERF-01: 外部コマンド最小化
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-02: サブシェル削減
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-03: ファイル I/O 最適化
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-04: ループ効率化（while read）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-05: 文字列処理最適化
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-06: 条件分岐最適化
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-07: 並列実行活用（&・xargs -P）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-08: キャッシュ戦略
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-09: リソース制限（ulimit）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- PERF-10: プロファイリング（set -x・time）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 7. Testing (TEST)

- TEST-01: validate_all_scripts.sh 検証必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- TEST-02: shellcheck 警告対応
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- TEST-03: 単体テスト実装
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- TEST-04: Bats テスト関数 a-z 順（setup 除く）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- TEST-05: CI/CD 統合
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 8. Documentation (DOC)

- DOC-01: ヘッダー標準形式（Description/Usage/Design Rules）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-02: show_usage 必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-03: 関数#######区切り+コメント
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-04: 複雑ロジックコメント
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-05: 変数説明
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-06: 英語コメント統一
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-07: README.md 整備
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-08: エラーメッセージ文書化
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-09: 変更履歴 CHANGELOG
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DOC-10: Main エントリポイント実装
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 9. Dependencies (DEP)

- DEP-01: lib/all.sh 活用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DEP-02: validate_dependencies 利用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DEP-03: SCRIPT_DIR 設定必須
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DEP-04: 必須コマンド明示
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- DEP-05: コマンド存在確認
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

### 10. Logging (LOG)

- LOG-01: log_message/echo_section 活用
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- LOG-02: stdout/stderr 分離
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- LOG-03: ログレベル実装（INFO・WARN・ERROR）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- LOG-04: 構造化ログ（timestamp・レベル・メッセージ）
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- LOG-05: 機密情報マスク
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- LOG-06: echo_section セクション区切り
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加
- LOG-07: verbose 実装
  - Problem: 記載不足
  - Impact: 見落としリスク
  - Recommendation: 明確な記載追加

## Output Format

レビュー結果リスト形式、簡潔説明+推奨修正案。

**Checks**: 全項目表示、✅=Pass / ❌=Fail
**Issues**: 問題ありのみ表示

## Example Output

### ✅ All Pass

```markdown
# Shell Script Review Result

## Issues

None ✅
```

### ❌ Issues Found

```markdown
# Shell Script Review Result

## Issues

1. G-04 共通ライブラリ未読込

   - Problem: lib/all.sh 未 source
   - Impact: error_exit/validate_dependencies 等共通関数利用不可・コード重複
   - Recommendation: `source "${SCRIPT_DIR}/../lib/all.sh"`追加

2. G-07 関数順序違反
   - Problem: main 関数が show_usage より前
   - Impact: 可読性低下・プロジェクト標準違反
   - Recommendation: show_usage/parse_arguments→a-z 順 →main 順
```
