# コレクター実装ガイド

## 📋 目次

このドキュメントは3つの主要セクションで構成されています：

### 🔴 [MUST] 必須実装ルール
すべてのコレクター実装で必須の内容。AI判断時に最優先で参照。

1. **コード品質基準** - lint/vet必須チェック、頻出エラー対策
2. **インターフェース実装** - Collector interface、依存性注入パターン
3. **データ構築ルール** - 空値の扱い、NewResource使用、型変換ルール
4. **GetColumns()実装** - helpers.GetMapValue使用必須
5. **SubCategory階層構造** - 親子関係の設計原則

### 🟡 [PATTERN] 実装パターン
頻出する実装パターンの具体例。

- 基本構造パターン
- ページネーションパターン
- 階層構造の出力順序

### 🔵 [REFERENCE] リファレンス
詳細仕様とトラブルシューティング。

- よくある問題と対策
- Lintエラー対応パターン

## 概要

本ドキュメントは`arc`の各コレクターを実装・検証する際の基準とガイドラインを定義します。

## 前提条件

- Go 1.25.4以上
- golangci-lint インストール済み
- AWS SDK for Go v2の基本的な理解

---

## 🔴 [MUST] 必須実装ルール

このセクションの内容は**すべてのコレクター実装で必須**です。

### 1. コード品質基準

#### ✅ 必須チェック項目

- [ ] `golangci-lint`でエラー・警告がないこと
- [ ] `go build`が成功すること
- [ ] `go vet`でエラーがないこと
- [ ] 未使用のimportがないこと
- [ ] rangeValCopy警告がないこと（大きな構造体はポインタまたはインデックス参照を使用）

#### 🔧 頻出lintエラーと対策

実装時に必ず対応すること:

| Linter         | エラー         | 対策                                                   | 例                                               |
| -------------- | -------------- | ------------------------------------------------------ | ------------------------------------------------ |
| `gocritic`     | `rangeValCopy` | 大きな構造体はインデックスでアクセスし、ポインタを使用 | `for i := range items { item := &items[i] }`     |
| `govet`        | `shadow`       | 変数の再宣言を避ける。別名を使用                       | `err` → `tableErr`, `imgErr` など                |
| `ineffassign`  | 未使用の代入   | 使用されない変数への代入を削除                         | `count := 0` で未使用なら削除                    |
| `revive`       | `add-constant` | マジックナンバー/文字列を定数化                        | `"false"` → `const DefaultFalseString = "false"` |
| `unused`       | 未使用の定数   | 使用されない定数を削除                                 | 未使用の`const`宣言を削除                        |
| `wastedassign` | 無駄な代入     | 再代入前の初期化を削除                                 | `x := ""; x = getValue()` → `x := getValue()`    |

**定数化が必要な典型的なマジックナンバー/文字列:**

- 数値の基数: `10` → `const DecimalBase = 10`
- デフォルトbool値: `"false"`, `"true"` → `const DefaultFalseString = "false"`
- デフォルト数値: `"0"` → `const DefaultZeroString = "0"`

### 2. インターフェース実装

各コレクターは以下のメソッドを実装すること:

```go
type Collector interface {
    Name() string
    ShouldSort() bool
    GetColumns() []Column
    Collect(ctx context.Context, region string) ([]Resource, error)
}
```

- [ ] `Name()`: コレクター名を返す（カテゴリ名と一致）
- [ ] `ShouldSort()`: ソートが必要かどうかを返す
- [ ] `GetColumns()`: CSVカラム定義を返す
- [ ] `Collect()`: リソースを収集して返す（`cfg`パラメータは不要、DI済み）

#### 依存性注入パターン（Dependency Injection Pattern）

**重要**: コレクターは`init()`関数による静的登録ではなく、`NewXxxCollector`コンストラクタを使用した明示的な初期化を行います。

##### コンストラクタ命名規則

```go
// 標準的な命名パターン: New<Service>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*ServiceCollector, error)
func NewACMCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*ACMCollector, error) {
    clients, err := helpers.CreateRegionalClients(cfg, regions,
        func(cfg *aws.Config, region string) *acm.Client {
            return acm.NewFromConfig(*cfg, func(o *acm.Options) {
                o.Region = region
            })
        })
    if err != nil {
        return nil, fmt.Errorf("failed to create ACM clients: %w", err)
    }
    return &ACMCollector{
        clients:      clients,
        nameResolver: nameResolver,
    }, nil
}
```

#### コレクター構造体

```go
// 各リージョン用のAWS SDKクライアントをマップで保持
// nameResolverは全コレクターで共有されるリソース名解決インスタンス
type ACMCollector struct {
    clients      map[string]*acm.Client
    nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}
```

#### Collect実装

```go
func (c *ACMCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
    // 注入されたクライアントを取得
    client, ok := c.clients[region]
    if !ok {
        return nil, fmt.Errorf("no ACM client found for region: %s", region)
    }
    // clientを使用してリソース収集...
}
```

#### 初期化フロー

`main.go`で以下の順序で初期化:

```go
// 1. AWSクライアント設定読み込み
cfg, err := awsClient.GetAWSConfig(ctx, awsprofile, region)

// 2. リージョンリスト取得
regionsToCheck := getRegionsToCheck(ctx, cfg, regions)

// 3. 全コレクターを明示的に初期化（リフレクションベース）
// InitializeCollectorsは内部でNameResolverを作成し、全コレクターに注入します
if err := resources.InitializeCollectors(&cfg, regionsToCheck); err != nil {
    return err
}

// 4. 登録済みコレクターを取得して実行
collectors := resources.GetCollectors()
```

**注**: `InitializeCollectors`は以下を自動的に実行します:
1. 単一の`NameResolver`インスタンスを作成（全リージョンのEC2/KMSクライアント付き）
2. 全コレクターのコンストラクタを呼び出し（リフレクション）
3. 各コレクターに`NameResolver`を注入
4. コレクターをグローバルレジストリに登録

#### コレクターの登録

新しいコレクターを追加する際は、`registry.go`の`InitializeCollectors`内で登録します:

```go
func InitializeCollectors(cfg *aws.Config, regions []string) error {
    // NameResolver作成
    nameResolver, err := helpers.NewNameResolver(cfg, regions)
    if err != nil {
        return fmt.Errorf("failed to create NameResolver: %w", err)
    }

    // コレクターを登録
    RegisterConstructor("acm", NewACMCollector)
    RegisterConstructor("apigateway", NewAPIGatewayCollector)  // 新規追加例
    // ... 他のコレクター

    // 自動初期化
    for name := range collectorConstructors {
        collector, collErr := createCollector(name, cfg, regions, nameResolver)
        if collErr != nil {
            return fmt.Errorf("failed to initialize %s collector: %w", name, collErr)
        }
        Register(name, collector)
    }
    return nil
}
```

#### グローバルサービスの扱い

- **US East 1専用サービス** (例: IAM, CloudFront, Route53):
  - `us-east-1`がリージョンリストに含まれない場合、空のクライアントマップを返す
  - `Collect`で該当リージョンがない場合は空リストを返す（エラーではない）

#### S3の特殊ケース

S3は複数リージョンで存在しますが、クライアントはバケットリージョンに基づいて動的に作成する必要があります。コンストラクタでは基本設定のみ保持:

```go
type S3Collector struct {
    cfg *aws.Config  // 基本設定を保持、動的にクライアント作成
}
```

#### テスト用インターフェース定義

モックテストを容易にするため、必要に応じてAWS SDKクライアントのインターフェースを定義:

```go
// テスト用インターフェース（必要なメソッドのみ定義）
type ACMClient interface {
    ListCertificates(ctx context.Context, params *acm.ListCertificatesInput,
        optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
    DescribeCertificate(ctx context.Context, params *acm.DescribeCertificateInput,
        optFns ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error)
}
```

## 実装ルール

### 空値の扱い

- 構造体のフィールドとして存在するが、値が設定されていない（`nil`または`""`）場合は、`"N/A"`を出力
- 構造体のフィールドとして存在しない（または取得対象外）場合は、空文字`""`を出力
- **重要**: Booleanフィールドも含め、すべてのnil値は`"N/A"`に統一。カスタムデフォルト値は使用しない

#### ⭐ NewResourceファクトリ関数の使用

**必須**: リソースの作成には必ず`NewResource`関数を使用

`NewResource`は自動的に以下の処理を実行:

- すべてのフィールド（`Name`、`ARN`、`Category`など）に`helpers.StringValue`を適用
- `RawData`に`helpers.NormalizeRawData`を適用

**利点**:

- `helpers.StringValue`や`helpers.NormalizeRawData`の呼び出し忘れを防止
- コードの重複を削減
- 一貫性のあるデータ処理を保証

**使用例**:

```go
// Good: NewResource を使用
resources = append(resources, NewResource(&ResourceInput{
    Category:    "iam",
    SubCategory: "Role",
    Name:        role.RoleName,        // ポインタをそのまま渡せる
    Region:      "Global",             // 文字列リテラルを直接渡す
    ARN:         role.Arn,             // ポインタをそのまま渡せる
    RawData: map[string]any{
        "Path":        role.Path,       // helpers.StringValue 不要
        "CreatedDate": role.CreateDate, // helpers.StringValue 不要
        "Enabled":     role.Enabled,    // *bool もそのまま渡す
    },
}))

// Bad: 古いパターン（使用しないこと）
resources = append(resources, Resource{
    Category:    "iam",
    SubCategory: "Role",
    Name:        helpers.StringValue(role.RoleName), // 冗長
    Region:      "Global",
    ARN:         helpers.StringValue(role.Arn),      // 冗長
    RawData: helpers.NormalizeRawData(map[string]any{
        "Path":        helpers.StringValue(role.Path),       // 冗長
        "CreatedDate": helpers.StringValue(role.CreateDate), // 冗長
    }),
})
```

### helpers.StringValueの直接使用を避ける

- `NewResource`の引数（`Name`、`ARN`、`Category`、`Region`）には`helpers.StringValue`を**使用しない**
- `NewResource`が自動的に変換するため、冗長で不要
- 例外: `RawData`以外で明示的な変換が必要な場合（配列の結合など）のみ使用可能

```go
// Good: StringValue を配列処理で使用
aliases := helpers.StringValue(dist.Aliases.Items)  // []string → "\n" 区切り文字列

// Bad: NewResource の引数で使用
Name: helpers.StringValue(role.RoleName)  // 不要、NewResource が処理する
```

### AWS SDK v2のaws.ToStringを避ける

- `NewResource`のフィールドでは`aws.ToString()`を**使用しない**
- `NewResource`が`helpers.StringValue`で自動変換するため、冗長
- ポインタをそのまま渡すことで、nilチェックと変換が自動的に行われる

```go
// Good: ポインタをそのまま渡す
Name: nat.NatGatewayId,
ARN:  pool.Id,

// Bad: aws.ToString を使用
Name: aws.ToString(nat.NatGatewayId),  // 不要
ARN:  aws.ToString(pool.Id),           // 不要
```

### 一時変数の削減

- **可読性を損なわない範囲で**、不要な一時変数を削減
- **匿名関数は使用しない**（可読性が大きく低下するため）

避けるべき削減:

```go
// Bad: 匿名関数は使用しない（可読性が低下）
"BillingMode": func() *string {
    if table.BillingModeSummary != nil {
        return aws.String(string(table.BillingModeSummary.BillingMode))
    }
    return nil
}(),

// Good: 一時変数を使用（可読性を維持）
var billingMode *string
if table.BillingModeSummary != nil {
    billingMode = table.BillingModeSummary.BillingMode
}
"BillingMode": billingMode,
```

#### 📦 RawDataの構築（最重要）

**重要原則**: `NewResource()`を使用する場合、`NormalizeRawData()`が自動的にすべての値を文字列化するため、\*\*ほとんどの型で手動変換は不要\*\*

##### 🔑 型変換が不要な型一覧

以下の型は**そのままRawDataに渡す**（手動変換不要）:

| 型                 | 例                                       | 自動変換結果                        | 備考              |
| ------------------ | ---------------------------------------- | ----------------------------------- | ----------------- |
| `*string`          | `user.Name`                              | `"value"` or `"N/A"`                | nilは"N/A"        |
| `*int32`, `*int64` | `table.ItemCount`                        | `"123"` or `"N/A"`                  | nilは"N/A"        |
| `*bool`            | `config.Enabled`                         | `"true"`, `"false"`, `"N/A"`        | nilは"N/A"        |
| `*time.Time`       | `cert.NotAfter`                          | `"2024-01-01T00:00:00Z"` or `"N/A"` | RFC3339形式       |
| `[]string`         | `dist.Aliases.Items`                     | `"item1\nitem2\nitem3"`             | 改行区切り        |
| `[]*string`        | `policy.Actions`                         | `"action1\naction2"`                | 改行区切り        |
| Enum型             | `cert.Status` (types.CertificateStatus)  | `"ISSUED"`                          | fmt.Sprintf("%v") |
| String型Enum       | `config.HttpVersion` (types.HttpVersion) | `"http2"`                           | string型ベース    |
| 文字列リテラル     | `"Global"`                               | `"Global"`                          | そのまま          |

**使用例**:

```go
RawData: map[string]any{
    // ポインタ型: そのまま渡す
    "Name":        user.Name,              // *string
    "Count":       table.ItemCount,        // *int32
    "Enabled":     config.Enabled,         // *bool
    "CreatedDate": role.CreateDate,        // *time.Time

    // 配列型: そのまま渡す（自動的に改行区切りで連結）
    "Aliases":     dist.Aliases.Items,     // []string
    "Tags":        resource.Tags,          // []*string

    // Enum型: そのまま渡す
    "Status":      cert.Status,            // types.CertificateStatus
    "HttpVersion": config.HttpVersion,     // types.HttpVersion (string型)

    // 文字列リテラル: そのまま渡す
    "Region":      "Global",
}
```

#### Enum配列型の扱い

**重要**: Enum型の配列（`[]types.Method`など）も**そのまま渡す**

```go
// Good: Enum配列をそのまま渡す（NormalizeRawDataが自動変換）
var allowedMethods any
if behavior.AllowedMethods != nil {
    allowedMethods = behavior.AllowedMethods.Items  // []types.Method
}
RawData: map[string]any{
    "AllowedMethods": allowedMethods,  // 自動的に "GET\nPOST\nHEAD" に変換される
}

// Bad: 手動で文字列変換（不要）
var methods []string
for _, m := range behavior.AllowedMethods.Items {
    methods = append(methods, string(m))
}
allowedMethods := strings.Join(methods, ",")  // 不要な処理
```

#### 型変換が必要な場合

以下の場合のみ、手動変換が必要:

1. **条件分岐後にポインタ型変数へ格納する場合**

```go
// String型EnumをポインタとTして格納する場合
var httpVersion *string
if config.HttpVersion != "" {
    httpVersion = aws.String(string(config.HttpVersion))  // aws.String()が必要
}
RawData: map[string]any{
    "HttpVersion": httpVersion,
}
```

2. **複雑な文字列加工が必要な場合**

```go
// 複数フィールドを組み合わせる場合
originConfig := fmt.Sprintf("HTTP=%d HTTPS=%d Protocol=%s",
    aws.ToInt32(origin.CustomOriginConfig.HTTPPort),
    aws.ToInt32(origin.CustomOriginConfig.HTTPSPort),
    origin.CustomOriginConfig.OriginProtocolPolicy)
RawData: map[string]any{
    "Config": &originConfig,
}
```

### 名前解決の共通化

VPC名、Subnet名などのID→名前解決には`helpers.ResolveNameFromMap()`を使用。手動でマップルックアップを書かない

```go
// Good: 共通関数を使用
vpcName := helpers.ResolveNameFromMap(instance.VpcId, vpcNames)

// Bad: 手動でマップルックアップ
vpcID := helpers.StringValue(instance.VpcId)
vpcName := vpcID
if n, ok := vpcNames[vpcID]; ok {
    vpcName = n
}
```

#### 📝 変数の省略

`RawData`の構築時に、可能な限り一時変数の定義を避け、直接値を渡すことを推奨

**複雑なロジックが必要な場合**: `nil`チェック後のネストしたフィールドへのアクセスや、条件分岐が必要な場合は、可読性のために一時変数を使用可能

```go
var zoneComment *string
if zone.Config != nil {
    zoneComment = zone.Config.Comment
}
RawData: map[string]any{ "Comment": zoneComment }
```

**複数回使用される値**: 同じ値を複数箇所で使用する場合は、一時変数を使用可能

**文字列操作が必要な場合**: `strings.Join()`、`fmt.Sprintf()`などの結果は一時変数に格納可能

### 4. GetColumns()の実装

**必須**: `GetColumns()`では、`RawData`からの値取得に必ず`helpers.GetMapValue(r.RawData, "key")`を使用

**理由**:

- `helpers.GetMapValue()`は存在しないキーに対して空文字列`""`を返す
- `NewResource()`が`NormalizeRawData()`を呼び出すため、`RawData`の全ての値は既に文字列化されている

**実装例**:

```go
func (_c *MyCollector) GetColumns() []Column {
    return []Column{
        {Header: "Category", Value: func(r Resource) string { return r.Category }},
        {Header: "Name", Value: func(r Resource) string { return r.Name }},
        {Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
        {Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
    }
}
```

### 5. SubCategory階層構造の設計

#### 基本原則

- **SubCategory1, SubCategory2, SubCategory3にはリソースタイプを設定する**（実際の値は設定しない）
- **Name フィールドには実際のリソース識別子（名前、ID など）を設定する**
- **親リソースの名前はRawDataに格納する**（トレーサビリティのため）
- **親子関係のある場合、SubCategory1は空文字列とする**（視覚的にネストしていることを表現）
  - CSV/HTML出力で表として見た際に、インデントされた階層構造がわかりやすくなる
  - 例: UserPool配下のGroup、Group配下のUserなど

#### CSV出力の制御

- `GetColumns()`で定義されたカラムのみがCSV出力される
- SubCategory2, SubCategory3が不要な場合は、`GetColumns()`から除外する
- ただし、`Resource`構造体には全てのSubCategoryフィールドを保持する（データの一貫性のため）

```go
// SubCategory2/3を使用しないコレクターの例
func (*S3Collector) GetColumns() []Column {
    return []Column{
        {Header: "Category", Value: func(r Resource) string { return r.Category }},
        {Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
        // SubCategory2, SubCategory3はCSV出力に含めない
        {Header: "Name", Value: func(r Resource) string { return r.Name }},
        {Header: "Region", Value: func(r Resource) string { return r.Region }},
        // ... その他のカラム
    }
}
```

#### 実装例: Cognito

Cognitoのような階層構造を持つリソースでは、以下のように実装します:

```go
// IdentityPool: 最上位リソース
NewResource(&ResourceInput{
    Category:     "cognito_identity",
    SubCategory1: "IdentityPool",  // リソースタイプ
    Name:         identityPoolId,   // 実際のID
    Region:       region,
    ARN:          identityPoolArn,
    RawData: map[string]any{
        "IdentityPoolName": identityPoolName,
        // ... その他のデータ
    },
})

// UserPool: 最上位リソース
NewResource(&ResourceInput{
    Category:     "cognito_user_pool",
    SubCategory1: "UserPool",      // リソースタイプ
    Name:         userPoolId,       // 実際のID
    Region:       region,
    ARN:          userPoolArn,
    RawData: map[string]any{
        "UserPoolName": userPoolName,
        "MfaConfiguration": mfaConfig,
        // ... その他のデータ
    },
})

// Group: UserPoolの子リソース（親子関係の視覚的表現のためSubCategory1は空）
NewResource(&ResourceInput{
    Category:     "cognito_user_pool",
    SubCategory1: "",              // 空 = 親リソースにネストしていることを視覚的に表現
    SubCategory2: "Group",         // 自身のリソースタイプ
    Name:         groupName,        // 実際のグループ名
    Region:       region,
    ARN:          groupName,
    RawData: map[string]any{
        "UserPoolId":   userPoolId,   // 親リソースのIDを保存
        "Description":  description,
        "AttachedUsers": usernames,
        // ... その他のデータ
    },
})

// User with Group: Groupの子リソース（親子関係の視覚的表現のためSubCategory1は空）
NewResource(&ResourceInput{
    Category:     "cognito_user_pool",
    SubCategory1: "",              // 空 = 親リソースにネストしていることを視覚的に表現
    SubCategory2: "Group",         // 親のリソースタイプ
    SubCategory3: "User",          // 自身のリソースタイプ
    Name:         username,         // 実際のユーザー名
    Region:       region,
    ARN:          username,
    RawData: map[string]any{
        "UserPoolId": userPoolId,   // 最上位親のIDを保存
        "GroupName":  groupName,    // 親のグループ名を保存
        "Attributes": attributes,
        // ... その他のデータ
    },
})

// User without Group: UserPoolの直接の子リソース（親子関係の視覚的表現のためSubCategory1は空）
NewResource(&ResourceInput{
    Category:     "cognito_user_pool",
    SubCategory1: "",              // 空 = 親リソースにネストしていることを視覚的に表現
    SubCategory2: "User",          // 自身のリソースタイプ
    Name:         username,         // 実際のユーザー名
    Region:       region,
    ARN:          username,
    RawData: map[string]any{
        "UserPoolId": userPoolId,   // 親リソースのIDを保存
        "Attributes": attributes,
        // ... その他のデータ
    },
})
```

#### 階層構造の判断基準

以下の場合にSubCategory2, SubCategory3を使用します:

1. **論理的な親子関係が存在する場合**
   - 例: UserPool → Group → User
   - 例: VPC → Subnet → NetworkInterface

2. **リソースが複数レベルのコンテナに属する場合**
   - 例: ECS Cluster → Service → Task

3. **サブリソースの分類が必要な場合**
   - 例: APIGateway → RestAPI → Resource → Method

#### SubCategory不要な場合

以下の場合はSubCategory1のみを使用します:

1. **フラットな構造のリソース**: S3 Bucket, EC2 Instance, Lambda Function など
2. **単一のリソースタイプのみ**: 複数のサブタイプが存在しない場合
3. **親子関係が不要**: リソース間に明確な階層がない場合

### r.RawData["somekey"]が存在しない場合の扱い

#### 方針

リソース側（各コレクター）で個々に対応

#### 実装方法

1. **NewResource()の使用**
   - `NewResource()`を使用してリソースを作成
   - `RawData`に含まれる値は自動的に`NormalizeRawData()`で正規化される
   - nilや空文字列の値は`"N/A"`に変換される
2. **GetColumns()でのGetMapValue()の使用**
   - `helpers.GetMapValue(r.RawData, "key")`を使用
   - キーが存在する場合: 正規化された値（`"N/A"`または実際の値）を返す
   - キーが存在しない場合: 空文字列`""`を返す

#### 実装例

```go
// CloudFormation Stack の場合
RawData: map[string]any{
    "Description": stack.Description,  // nil の場合 "N/A"、値がある場合はその値
    "Type":        "Stack",
    "Outputs":     outputs,
    "Parameters":  params,
    "Resources":   stackResources,
    "CreatedDate": stack.CreationTime,
    "UpdatedDate": stack.LastUpdatedTime,  // Stack には存在
    "DriftStatus": stack.DriftInformation.StackDriftStatus,
    "Status":      stack.StackStatus,
}

// CloudFormation StackSet の場合
RawData: map[string]any{
    "Description": ss.Description,
    "Type":        "StackSet",
    "Parameters":  params,
    "Status":      ss.Status,
    // "Outputs", "Resources", "CreatedDate", "UpdatedDate", "DriftStatus" は設定しない
}

// GetColumns() での使用
{Header: "UpdatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "UpdatedDate") }}
// → Stack の場合: 値または "N/A"
// → StackSet の場合: "" (空文字列)
```

#### 利点

- リソースタイプごとに異なるフィールドセットを柔軟に扱える
- CSV出力で不要なフィールドは空欄となり、見やすい
- コードの保守性が向上する

---

## 🟡 [PATTERN] 実装パターン

このセクションでは頻出する実装パターンを提供します。

### 基本構造パターン

```go
package resources

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/<service>"
)

type <Service>Collector struct{}

func init() { //nolint:gochecknoinits
    Register("<category>", &<Service>Collector{})
}

func (*<Service>Collector) Name() string {
    return "<category>"
}

func (*<Service>Collector) ShouldSort() bool {
    return true
}

func (*<Service>Collector) GetColumns() []Column {
    return []Column{
        // 標準カラム
        {Header: "Category", Value: func(r Resource) string { return r.Category }},
        {Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
        {Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
        {Header: "Name", Value: func(r Resource) string { return r.Name }},
        {Header: "Region", Value: func(r Resource) string { return r.Region }},
        {Header: "ARN", Value: func(r Resource) string { return r.ARN }},
        // カスタムカラム
        {Header: "CustomField", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CustomField") }},
    }
}

func (*<Service>Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
    svc := <service>.NewFromConfig(*cfg, func(o *<service>.Options) {
        o.Region = region
    })

    var resources []Resource

    // リソース収集ロジック
    // ...

    return resources, nil
}
```

### ページネーションパターン

```go
paginator := <service>.New<Operation>Paginator(svc, &<service>.<Operation>Input{})
for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list resources: %w", err)
    }
    for i := range page.Items {
        item := &page.Items[i]
        // 処理
    }
}
```

### 階層構造の出力順序

親子関係があるリソース（RDS Cluster→Instance、ECS Cluster→Service→Task）の場合:

```go
// 親リソースを追加
resources = append(resources, parentResource)

// 直後に子リソースを追加
for _, child := range parent.Children {
    resources = append(resources, childResource)
}
```

**重要**: シェルスクリプトの出力順序を確認し、同じ順序で出力すること

---

## 🔵 [REFERENCE] リファレンス

このセクションは詳細仕様とトラブルシューティング情報です。

### よくある問題と対策

#### ビルド・実行時エラー

| 問題                  | 対策                                                         |
| --------------------- | ------------------------------------------------------------ |
| `undefined: Resource` | パッケージ全体でビルド (`./internal/aws/resources/...`)      |
| 空の出力              | グローバルサービスのリージョンチェック、ページネーション確認 |
| 順序の違い            | `ShouldSort()`の戻り値、収集順序を確認                       |
| フィールド値の違い    | `strconv.FormatBool`、`fmt.Sprintf`の使用を確認              |

#### Lintエラー対応パターン

**gocritic: rangeValCopy**

```go
// ❌ Bad: 大きな構造体を値でコピー
for _, item := range items {
    process(item)
}

// ✅ Good: インデックスでアクセスし、ポインタを使用
for i := range items {
    item := &items[i]
    process(item)
}
```

**govet: shadow**

```go
// ❌ Bad: 変数errを再宣言
result, err := operation1()
if err != nil {
    return err
}
data, err := operation2() // errがシャドウイング

// ✅ Good: 別名を使用
result, err := operation1()
if err != nil {
    return err
}
data, opErr := operation2()
if opErr != nil {
    return opErr
}
```

**revive: add-constant**

```go
// ❌ Bad: マジックナンバー/文字列
value := strconv.FormatInt(num, 10)
status := helpers.StringValue(enabled, "false")

// ✅ Good: 定数化
const (
    DecimalBase = 10
    DefaultFalseString = "false"
)
value := strconv.FormatInt(num, DecimalBase)
status := helpers.StringValue(enabled, DefaultFalseString)
```

**ineffassign / wastedassign**

```go
// ❌ Bad: 未使用または無駄な代入
count := 0
// countが使用されない、または
count = len(items) // 初期化が無駄

// ✅ Good: 必要な代入のみ
count := len(items)
```
