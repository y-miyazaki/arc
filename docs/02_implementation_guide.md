# コレクター実装ガイド

## 概要

本ドキュメントは`arc`の各コレクターを実装・検証する際の基準とガイドラインを定義します。

## 前提条件

- Go 1.25.4以上
- golangci-lint インストール済み
- AWS SDK for Go v2の基本的な理解

## コード品質基準

### 必須チェック項目

- [ ] `golangci-lint`でエラー・警告がないこと
- [ ] `go build`が成功すること
- [ ] `go vet`でエラーがないこと
- [ ] 未使用のimportがないこと
- [ ] rangeValCopy警告がないこと（大きな構造体はポインタまたはインデックス参照を使用）

### 頻出lintエラーと対策

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

## インターフェース実装

各コレクターは以下のメソッドを実装すること:

```go
type Collector interface {
    Name() string
    ShouldSort() bool
    GetColumns() []Column
    Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error)
}
```

- [ ] `Name()`: コレクター名を返す（カテゴリ名と一致）
- [ ] `ShouldSort()`: ソートが必要かどうかを返す
- [ ] `GetColumns()`: CSVカラム定義を返す
- [ ] `Collect()`: リソースを収集して返す

### init()関数

`init()`関数で`Register`を呼び出してコレクターを登録すること:

```go
func init() { //nolint:gochecknoinits
    Register("category_name", &CategoryCollector{})
}
```

## 実装ルール

### 空値の扱い

- 構造体のフィールドとして存在するが、値が設定されていない（`nil`または`""`）場合は、`"N/A"`を出力
- 構造体のフィールドとして存在しない（または取得対象外）場合は、空文字`""`を出力
- **重要**: Booleanフィールドも含め、すべてのnil値は`"N/A"`に統一。カスタムデフォルト値は使用しない

### NewResourceファクトリ関数の使用

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

良い削減例:

```go
// Good: 配列を直接 strings.Join で結合
"AvailabilityZone": strings.Join(azs, "\n"),
"SecurityGroup":    strings.Join(sgs, "\n"),

// Bad: 不要な中間変数
lbAZ := strings.Join(azs, "\n")
"AvailabilityZone": lbAZ,
```

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
    billingMode = aws.String(string(table.BillingModeSummary.BillingMode))
}
"BillingMode": billingMode,
```

### 配列型データの処理

- `helpers.StringValue`は配列型（`[]string`, `[]*string`）に対応
- 配列は自動的に`\n`（改行）で連結される
- `strings.Join`の代わりに`helpers.StringValue`を使用可能

```go
// Good: StringValue が配列を処理
"AlternateDomain": helpers.StringValue(dist.Aliases.Items),

// Also Good: 明示的な strings.Join
"AlternateDomain": strings.Join(dist.Aliases.Items, "\n"),
```

### RawDataの構築

#### ポインタ型

ポインタ型（`*string`、`*int32`、`*bool`、`*time.Time`など）はそのまま渡す。`NewResource`が自動的に処理

#### Enum型

AWS SDKのenum型（`types.InstanceType`、`types.CertificateStatus`など）も`string()`キャストせずにそのまま渡す。`NormalizeRawData`が`fmt.Sprintf("%v", val)`で文字列化

```go
// Bad: "Status": string(cert.Status)
// Good: "Status": cert.Status
```

#### Boolean型

`*bool`型もそのまま渡す。nilの場合は`"N/A"`、true/falseの場合は`"true"`/`"false"`に変換される

```go
// Bad: "Enabled": helpers.StringValue(config.Enabled, "false")
// Good: "Enabled": config.Enabled
```

#### デフォルト値のカスタマイズ

- **原則**: デフォルト値は常に`"N/A"`を使用
- **例外**: プロジェクト全体で合意された特別な理由がある場合のみ、`helpers.StringValue`でカスタムデフォルト値を指定可能
- 現在、カスタムデフォルト値の使用は推奨されない

#### 文字列リテラルの渡し方

文字列リテラル（`"Global"`、`""`など）は直接渡す。`aws.String()`や`&`は不要

```go
// Bad: Region: aws.String("Global")
// Bad: Region: &region
// Good: Region: "Global"
// Good: Region: region
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

### 変数の省略

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

### GetColumns()の実装

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
    "Outputs":     strings.Join(outputs, "\n"),
    "Parameters":  strings.Join(params, "\n"),
    "Resources":   strings.Join(stackResources, "\n"),
    "CreatedDate": stack.CreationTime,
    "UpdatedDate": stack.LastUpdatedTime,  // Stack には存在
    "DriftStatus": stack.DriftInformation.StackDriftStatus,
    "Status":      stack.StackStatus,
}

// CloudFormation StackSet の場合
RawData: map[string]any{
    "Description": ss.Description,
    "Type":        "StackSet",
    "Parameters":  strings.Join(params, "\n"),
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

## 基本構造パターン

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

## ページネーションパターン

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

## 階層構造の出力順序

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

## よくある問題と対策

### ビルド・実行時エラー

| 問題                  | 対策                                                         |
| --------------------- | ------------------------------------------------------------ |
| `undefined: Resource` | パッケージ全体でビルド (`./internal/aws/resources/...`)      |
| 空の出力              | グローバルサービスのリージョンチェック、ページネーション確認 |
| 順序の違い            | `ShouldSort()`の戻り値、収集順序を確認                       |
| フィールド値の違い    | `strconv.FormatBool`、`fmt.Sprintf`の使用を確認              |

### Lintエラー対応パターン

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
