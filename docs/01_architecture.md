# アーキテクチャ設計

## 概要

`arc`は既存の`aws_get_resources.sh`シェルスクリプトをGoで書き換えたCLIツールです。

### 目的

- メンテナンス性の向上
- 並行処理によるパフォーマンス改善
- 拡張性の向上

## プロジェクト構造

```
.
├── cmd
│   └── arc
│       └── main.go          # エントリーポイント
├── internal
│   ├── aws
│   │   ├── client.go        # AWS SDKクライアント初期化
│   │   ├── helpers          # ヘルパー関数
│   │   │   ├── helpers.go   # 時刻フォーマット等
│   │   │   └── name_resolvers.go  # リソース名解決（KMS、VPCなど）
│   │   └── resources        # リソースコレクター（リソース毎に1ファイル）
│   │       ├── registry.go  # コレクターレジストリ
│   │       ├── acm.go       # ACMコレクター
│   │       ├── ec2.go       # EC2コレクター
│   │       └── ...          # その他コレクター
│   ├── config
│   │   └── config.go        # 設定構造体とパース
│   ├── exporter
│   │   └── csv.go           # CSVエクスポートロジック
│   ├── validation
│   │   └── precheck.go      # 実行前検証（AWS認証、依存関係）
│   └── logger
│       └── logger.go        # ロガー設定
├── go.mod
└── go.sum
```

### ディレクトリガイドライン

- **internal/aws/resources**: リソース収集ロジックのみを含む。各ファイルは特定のAWSリソースタイプの`Collector`インターフェースを実装
- **internal/aws/helpers**: ヘルパー関数とユーティリティ（時刻フォーマット、ARNパース、リソース名解決など）
- **internal/validation**: 実行前検証関数（AWS認証検証、依存関係チェック）

## 技術スタック

### CLIフレームワーク

`github.com/urfave/cli/v2`を使用してコマンドライン引数とフラグを解析

### AWS SDK

AWS SDK for Go v2 (`github.com/aws/aws-sdk-go-v2`) を使用

**プロファイル対応**: `--profile`フラグでAWSプロファイルを指定可能

**SSOセッション管理**: AWS SSOを使用する場合、SSOセッションがアクティブであることを確認。エラー時は`aws sso login`でセッションを更新。アプリケーションはSTS GetCallerIdentityで認証検証を実施

### ロギング

`github.com/y-miyazaki/go-common/pkg/logger`のカスタムロガーを使用（`slog_logger.go`）

## アーキテクチャ

### インターフェース

全リソースコレクターが実装する`Collector`インターフェース:

```go
type Collector interface {
    Name() string
    Collect(ctx context.Context, cfg aws.Config, region string) ([]Resource, error)
}

type Resource struct {
    Category        string
    SubCategory     string
    Name            string
    Region          string
    // ... その他共通フィールド
    RawData         map[string]interface{} // CSVカラム用
}
```

### リソースコレクター（自己登録パターン）

各AWSリソース（EC2、S3、RDSなど）は`internal/aws/resources/`内で`Collector`インターフェースを実装

**拡張性設計**:

- `internal/aws/resources/registry.go`にグローバルレジストリ（map）が存在
- 各リソースファイル（例: `acm.go`）は`init()`関数でコレクターインスタンスをレジストリに登録
- メインアプリケーションはレジストリを反復処理して全コレクターを実行
- **結果**: 新しいリソース追加時は`internal/aws/resources/`に新規ファイルを作成するだけ。`main.go`等の修正不要

### 並行処理

Goの並行処理機能（goroutineとchannel）を活用し、異なるカテゴリとリージョンからのリソース収集を並列実行

### 出力生成

- **CSV**: `{outputDir}/{accountID}/resources/{category}.csv`に各カテゴリ別CSVファイルを出力
- **all.csv**: 全カテゴリCSVをアルファベット順（A-Z）に結合し、各カテゴリ間に1行の空白行を挿入。`{outputDir}/{accountID}/resources/all.csv`に出力
- **HTML**: `--html`フラグ指定時、HTMLインデックスを`html/template`で生成

### 出力フォーマット標準

#### ヘッダー命名規則

CSVヘッダー名はcamelCaseを使用（アンダースコア不可）

例: `RequestDate`（`Request_Date`は不可）、`KeyAlgorithm`（`key_algorithm`は不可）

#### ヘッダー順序

CSVヘッダーカラムの優先順位:

1. Category
2. SubCategory
3. SubSubCategory
4. Name
5. Region
6. ARN
7. Status
8. その他フィールド（`KeyAlgorithm`、`InUse`、`Exported`など）
9. 日付フィールド（`RequestDate`、`IssuedDate`、`ExpirationDate`、`CreatedDate`）は最後に配置

#### フィールド値処理ポリシー

CSVフィールド値はnull/空値処理のルールに従う:

1. **空文字列 (`""`)**: コレクターが意図的に取得しないフィールド
   - 例: Clusterリソースの`PortMappings`フィールド（クラスタにポートマッピングは存在しない）
2. **`"N/A"` 文字列**: コレクターが取得を試みたが値がnil、欠落、または取得不可
   - 例: 存在すべきだがAPIが返さなかったTaskDefinition ARN
3. **`<nil>`、`null`、Goのnilデフォルト表現は出力しない**: 常に空文字列または`"N/A"`を使用

このポリシーは元の`aws_get_resources.sh`の動作との一貫性を保ち、CSV解析時の混乱を防ぐ

#### テスト・開発時の出力ファイル配置

コレクターのテスト・開発時は、テストファイルを常に`tmp/`ディレクトリに出力

例: `./arc -c ecs -o tmp/ecs-test.csv`

`tmp/`と`output/`ディレクトリは`.gitignore`に含まれている

#### 行順序

親子関係のあるリソース（VPC → Subnet）の出力行は`SubCategory`と`SubSubCategory`でネストして順序付け。元の`aws_get_resources.sh`の動作に一致

## 設定マッピング

| Bashフラグ         | Goフラグ       | 説明                         |
| :----------------- | :------------- | :--------------------------- |
| (New)              | `--profile`    | 使用するAWSプロファイル      |
| `-v, --verbose`    | `--verbose`    | デバッグログを有効化         |
| `-o, --output`     | `--output`     | 出力ファイル名               |
| `-D, --output-dir` | `--output-dir` | 出力ディレクトリ             |
| `-r, --region`     | `--region`     | 対象AWSリージョン            |
| `-c, --categories` | `--categories` | カテゴリのカンマ区切りリスト |
| `-H, --html`       | `--html`       | HTMLインデックスを生成       |

## 実装状況

| リソースカテゴリ  | ステータス | 備考                                                          |
| :---------------- | :--------- | :------------------------------------------------------------ |
| ACM               | 実装済み   |                                                               |
| APIGateway        | 実装済み   | REST (v1) と HTTP (v2) API対応                                |
| Batch             | 実装済み   |                                                               |
| CloudFormation    | 実装済み   |                                                               |
| CloudFront        | 実装済み   |                                                               |
| CloudWatch Alarms | 実装済み   |                                                               |
| CloudWatch Logs   | 実装済み   |                                                               |
| Cognito           | 実装済み   | User PoolsとIdentity Pools対応                                |
| DynamoDB          | 実装済み   |                                                               |
| EC2               | 実装済み   |                                                               |
| ECR               | 実装済み   |                                                               |
| ECS               | 実装済み   |                                                               |
| EFS               | 実装済み   |                                                               |
| ElastiCache       | 実装済み   |                                                               |
| ELBv2             | 実装済み   |                                                               |
| EventBridge       | 実装済み   | RulesとScheduler対応                                          |
| Glue              | 実装済み   | DatabasesとJobs対応                                           |
| IAM               | 実装済み   |                                                               |
| Kinesis           | 実装済み   | StreamsとFirehose対応                                         |
| KMS               | 実装済み   |                                                               |
| Lambda            | 実装済み   |                                                               |
| QuickSight        | 実装済み   | Data SourcesとAnalyses対応                                    |
| RDS               | 実装済み   |                                                               |
| Redshift          | 実装済み   |                                                               |
| Route53           | 実装済み   |                                                               |
| S3                | 実装済み   |                                                               |
| SecretsManager    | 実装済み   |                                                               |
| SNS               | 実装済み   |                                                               |
| SQS               | 実装済み   |                                                               |
| TransferFamily    | 実装済み   |                                                               |
| SES               | `ses`      | Identities, configuration sets, templates, sending statistics |
| WAF               | 実装済み   | WAFv2（Regional & Global）対応                                |

### SES (Simple Email Service)

SES はメール送信・受信に関わる各種設定を収集します。実装では以下の項目を想定して収集します。

#### 収集対象

- Email Identities（ドメイン・メールアドレス）
- Configuration Sets（送信設定）
- Templates（テンプレート）
- Sending statistics（送信結果・送信量の要約）

#### 使用する主なAPI（SESv2を優先）

- `sesv2:ListEmailIdentities`, `sesv2:GetEmailIdentity`
- `sesv2:ListConfigurationSets`, `sesv2:GetConfigurationSet`
- `sesv2:ListEmailTemplates`, `sesv2:GetEmailTemplate`
- 必要に応じて旧API（SES Classic）の `ses:ListIdentities`, `ses:GetIdentityDkimAttributes` など

#### IAM 権限（例）

```json
{
    "Effect": "Allow",
    "Action": [
        "sesv2:ListEmailIdentities",
        "sesv2:GetEmailIdentity",
        "sesv2:ListConfigurationSets",
        "sesv2:GetConfigurationSet",
        "sesv2:ListEmailTemplates",
        "sesv2:GetEmailTemplate",
        "ses:ListIdentities",
        "ses:GetIdentityDkimAttributes"
    ],
    "Resource": "*"
}
```

#### 注意点

- 一部APIはグローバル（us-east-1等）またはリージョン毎に異なる動作をするため、実行時は`--region`指定に注意すること。
- 送信統計（CloudWatchやEventデータ）を深掘りする場合は追加の権限が必要になる可能性がある。
