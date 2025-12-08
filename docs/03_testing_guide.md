# テスト方針

## 概要

本ドキュメントは`arc`のテスト戦略と方針を定義します。

## テストカバレッジ戦略

### 現在のアプローチ

プロジェクトは最も重要でテスト可能なコンポーネントに焦点を当てる:

#### 1. GetColumnsメソッドのテスト（主要フォーカス）

- カラムヘッダーが正しく定義されていることをテスト
- Value関数がResourceから正しくデータを抽出することをテスト
- データ変換ロジックを検証
- 高価値/労力比

#### 2. ヘルパー関数のテスト

- `internal/aws/helpers`のユーティリティ関数をテスト
- データ抽出とフォーマットに重要

### なぜCollectメソッドをテストしないのか

**Collectメソッドは意図的にユニットテストしていません**。理由:

1. **薄いAWS APIラッパー**: CollectメソッドはAWS SDK呼び出しの薄いラッパーで、最小限のビジネスロジックしか含まない
2. **高いメンテナンスコスト**: 30以上のコレクターに対してインターフェースとモックを作成すると、大きなメンテナンスコストが発生
3. **AWS SDKは既にテスト済み**: AWS SDK v2はAWSによって徹底的にテストされている
4. **より良いテスト代替手段**:
   - LocalStackを使用した統合テスト
   - ステージング環境でのE2Eテスト
   - 実際のAWS認証情報を使用した手動テスト

## テスト構造

各コレクターは以下のテストを持つべき:

```go
func TestXXXCollector_Basic(t *testing.T) {
    collector := &XXXCollector{}
    assert.Equal(t, "xxx", collector.Name())
    assert.True(t, collector.ShouldSort()) // or False depending on collector
}

func TestXXXCollector_GetColumns(t *testing.T) {
    collector := &XXXCollector{}
    columns := collector.GetColumns()

    // Test 1: Verify column headers
    expectedHeaders := []string{
        "Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
        // ... other columns
    }

    assert.Len(t, columns, len(expectedHeaders))
    for i, column := range columns {
        assert.Equal(t, expectedHeaders[i], column.Header)
    }

    // Test 2: Verify Value functions with sample resource
    sampleResource := Resource{
        Category:       "Security",
        SubCategory:    "XXX",
        SubSubCategory: "YYY",
        Name:           "test-resource",
        Region:         "us-east-1",
        ARN:            "arn:aws:xxx:us-east-1:123456789012:resource/test",
        RawData: map[string]interface{}{
            "Field1": "value1",
            "Field2": "value2",
            // ... test data
        },
    }

    expectedValues := []string{
        "Security", "XXX", "YYY", "test-resource", "us-east-1",
        "arn:aws:xxx:us-east-1:123456789012:resource/test",
        "value1", "value2",
        // ... expected extracted values
    }

    for i, column := range columns {
        assert.Equal(t, expectedValues[i], column.Value(sampleResource),
            "Column %d (%s) value mismatch", i, column.Header)
    }
}
```

## カバレッジ目標

- **現在のカバレッジ**: `internal/aws/resources`で約27.0%
- **フォーカスエリア**:
  - ✅ GetColumnsメソッド: 十分にカバー
  - 🎯 ヘルパー関数: 28.2%から改善
  - 🎯 メインエントリポイント: 6.1%から改善

## ベストプラクティス

1. **データ抽出ロジックをテスト**: AWS APIレスポンスを変換するValue関数に焦点を当てる
2. **現実的なテストデータを使用**: サンプルリソースは実際のAWSレスポンスを模倣すべき
3. **エッジケースをテスト**: 空値、欠落フィールド、特殊文字
4. **一貫性を維持**: 確立されたテストパターンを全コレクターで踏襲
