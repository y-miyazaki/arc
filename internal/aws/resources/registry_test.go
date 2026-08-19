package resources

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockCollector is a mock implementation of the Collector interface for testing
type MockCollector struct {
	name        string
	shouldSort  bool
	columns     []Column
	collectFunc func(ctx context.Context, region string) ([]Resource, error)
}

func NewMockCollector(name string, shouldSort bool) *MockCollector {
	return &MockCollector{
		name:       name,
		shouldSort: shouldSort,
		columns: []Column{
			{Header: "Category", Value: func(r Resource) string { return r.Category }},
			{Header: "Name", Value: func(r Resource) string { return r.Name }},
			{Header: "Region", Value: func(r Resource) string { return r.Region }},
		},
		collectFunc: func(ctx context.Context, region string) ([]Resource, error) {
			return []Resource{
				{
					Category: "test",
					Name:     "test-resource",
					Region:   region,
					RawData: helpers.NormalizeRawData(map[string]any{
						"Status": "active",
					}),
				},
			}, nil
		},
	}
}

func (m *MockCollector) Name() string {
	return m.name
}

func (m *MockCollector) ShouldSort() bool {
	return m.shouldSort
}

func (m *MockCollector) GetColumns() []Column {
	return m.columns
}

func (m *MockCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	return m.collectFunc(ctx, region)
}

func TestNewResource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *ResourceInput
		want Resource
	}{
		{
			name: "copies fields and normalizes raw data",
			in: &ResourceInput{
				Category:     "test-category",
				SubCategory1: "test-subcategory",
				Name:         "test-name",
				Region:       "us-east-1",
				ARN:          "arn:aws:test:us-east-1:123456789012:test/test-name",
				RawData: map[string]any{
					"Status":      "active",
					"CreatedDate": "2023-01-01T00:00:00Z",
					"Count":       42,
				},
			},
			want: Resource{
				Category:     "test-category",
				SubCategory1: "test-subcategory",
				Name:         "test-name",
				Region:       "us-east-1",
				ARN:          "arn:aws:test:us-east-1:123456789012:test/test-name",
				RawData: map[string]any{
					"Status":      "active",
					"CreatedDate": "2023-01-01T00:00:00Z",
					"Count":       "42",
				},
			},
		},
		{
			name: "nil optional fields become empty strings",
			in: &ResourceInput{
				Category:     "test-category",
				SubCategory1: nil,
				SubCategory2: nil,
				Name:         "test-name",
				Region:       "us-east-1",
				ARN:          nil,
				RawData:      map[string]any{},
			},
			want: Resource{
				Category: "test-category",
				Name:     "test-name",
				Region:   "us-east-1",
				ARN:      "",
				RawData:  map[string]any{},
			},
		},
		{
			name: "subcategory3 is copied",
			in: &ResourceInput{
				Category:     "test-category",
				SubCategory3: "leaf",
				Name:         "test-name",
				Region:       "us-east-1",
				RawData:      map[string]any{},
			},
			want: Resource{
				Category:     "test-category",
				SubCategory3: "leaf",
				Name:         "test-name",
				Region:       "us-east-1",
				RawData:      map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewResource(tt.in)
			assert.Equal(t, tt.want.Category, got.Category)
			assert.Equal(t, tt.want.SubCategory1, got.SubCategory1)
			assert.Equal(t, tt.want.SubCategory2, got.SubCategory2)
			assert.Equal(t, tt.want.SubCategory3, got.SubCategory3)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Region, got.Region)
			assert.Equal(t, tt.want.ARN, got.ARN)
			assert.Equal(t, tt.want.RawData, got.RawData)
		})
	}
}

func TestRegister(t *testing.T) {
	// Clear the registry before test
	originalCollectors := make(map[string]Collector)
	maps.Copy(originalCollectors, collectors)
	collectors = make(map[string]Collector)
	defer func() {
		collectors = originalCollectors
	}()

	// Test registering a collector
	mockCollector := NewMockCollector("test-collector", true)
	Register("test", mockCollector)

	assert.Contains(t, collectors, "test")
	assert.Equal(t, mockCollector, collectors["test"])
}

func TestGetCollectors(t *testing.T) {
	// Clear the registry before test
	originalCollectors := make(map[string]Collector)
	maps.Copy(originalCollectors, collectors)
	collectors = make(map[string]Collector)
	defer func() {
		collectors = originalCollectors
	}()

	// Register some test collectors
	mockCollector1 := NewMockCollector("collector1", true)
	mockCollector2 := NewMockCollector("collector2", false)

	Register("test1", mockCollector1)
	Register("test2", mockCollector2)

	result := GetCollectors()

	assert.Len(t, result, 2)
	assert.Contains(t, result, "test1")
	assert.Contains(t, result, "test2")
	assert.Equal(t, mockCollector1, result["test1"])
	assert.Equal(t, mockCollector2, result["test2"])

	result["injected"] = NewMockCollector("injected", false)
	again := GetCollectors()
	assert.NotContains(t, again, "injected")
	assert.Len(t, again, 2)
}

func TestMockCollector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		collector  *MockCollector
		wantName   string
		wantSort   bool
		wantCols   []string
		wantRegion string
	}{
		{
			name:       "name sort columns and collect",
			collector:  NewMockCollector("test-name", true),
			wantName:   "test-name",
			wantSort:   true,
			wantCols:   []string{"Category", "Name", "Region"},
			wantRegion: "us-west-2",
		},
		{
			name:      "should sort false",
			collector: NewMockCollector("test", false),
			wantName:  "test",
			wantSort:  false,
			wantCols:  []string{"Category", "Name", "Region"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantName, tt.collector.Name())
			assert.Equal(t, tt.wantSort, tt.collector.ShouldSort())
			columns := tt.collector.GetColumns()
			require.Len(t, columns, len(tt.wantCols))
			for i, header := range tt.wantCols {
				assert.Equal(t, header, columns[i].Header)
			}
			if tt.wantRegion == "" {
				return
			}
			resources, err := tt.collector.Collect(context.Background(), tt.wantRegion)
			require.NoError(t, err)
			require.Len(t, resources, 1)
			assert.Equal(t, "test", resources[0].Category)
			assert.Equal(t, "test-resource", resources[0].Name)
			assert.Equal(t, tt.wantRegion, resources[0].Region)
			assert.Equal(t, "active", resources[0].RawData["Status"])
		})
	}
}

func TestInitializeCollectors(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "us-west-2"}

	originalCollectors := maps.Clone(collectors)
	originalConstructors := maps.Clone(collectorConstructors)
	collectors = make(map[string]Collector)
	t.Cleanup(func() {
		collectors = originalCollectors
		collectorConstructors = originalConstructors
	})

	require.NoError(t, InitializeCollectors(cfg, regions))
	registeredCollectors := GetCollectors()
	require.NotEmpty(t, registeredCollectors)

	expectedCollectors := []string{"acm", "ec2", "s3_bucket", "iam_role"}
	for _, name := range expectedCollectors {
		assert.Contains(t, registeredCollectors, name)
		assert.NotNil(t, registeredCollectors[name])
	}

	registeredCollectors["injected"] = NewMockCollector("injected", false)
	again := GetCollectors()
	assert.NotContains(t, again, "injected")
}

func TestRegisterConstructor(t *testing.T) {
	// Clear the registry before test
	originalConstructors := make(map[string]any)
	maps.Copy(originalConstructors, collectorConstructors)
	collectorConstructors = make(map[string]any)
	defer func() {
		collectorConstructors = originalConstructors
	}()

	// Test registering a constructor
	RegisterConstructor("test", NewMockCollector)

	assert.Contains(t, collectorConstructors, "test")
	assert.NotNil(t, collectorConstructors["test"])
}

func TestCreateCollector(t *testing.T) {
	originalConstructors := make(map[string]any)
	maps.Copy(originalConstructors, collectorConstructors)
	collectorConstructors = make(map[string]any)
	defer func() {
		collectorConstructors = originalConstructors
	}()

	RegisterConstructor("acm", NewACMCollector)

	cfg := &aws.Config{Region: "us-east-1"}
	regions := []string{"us-east-1"}
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	tests := []struct {
		name      string
		collector string
		wantName  string
		wantErr   error
	}{
		{name: "known constructor", collector: "acm", wantName: "acm"},
		{name: "unknown collector", collector: "unknown", wantErr: ErrUnknownCollector},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mutates package constructor map; omit t.Parallel() (TBL-06).
			got, createErr := createCollector(tt.collector, cfg, regions, nameResolver)
			if tt.wantErr != nil {
				require.Error(t, createErr)
				assert.ErrorIs(t, createErr, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, createErr)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.Name())
		})
	}
}

func notCollectorConstructor(_ *aws.Config, _ []string, _ *helpers.NameResolver) (string, error) {
	return "not-a-collector", nil
}

func invalidErrorTypeConstructor(_ *aws.Config, _ []string, _ *helpers.NameResolver) (*ACMCollector, any) {
	return nil, "not-an-error"
}

var errTestConstructor = errors.New("constructor failed")

func failingCollectorConstructor(_ *aws.Config, _ []string, _ *helpers.NameResolver) (*ACMCollector, error) {
	return nil, errTestConstructor
}

func TestCreateCollector_InvalidReturnTypes(t *testing.T) {
	originalConstructors := make(map[string]any)
	maps.Copy(originalConstructors, collectorConstructors)
	collectorConstructors = make(map[string]any)
	defer func() {
		collectorConstructors = originalConstructors
	}()

	cfg := &aws.Config{Region: "us-east-1"}
	regions := []string{"us-east-1"}
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	tests := []struct {
		name        string
		constructor any
		wantErr     error
	}{
		{name: "invalid collector type", constructor: notCollectorConstructor, wantErr: ErrInvalidCollectorType},
		{name: "invalid error type", constructor: invalidErrorTypeConstructor, wantErr: ErrInvalidErrorType},
		{name: "constructor error", constructor: failingCollectorConstructor, wantErr: errTestConstructor},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			RegisterConstructor("invalid", tt.constructor)
			collector, createErr := createCollector("invalid", cfg, regions, nameResolver)
			require.Error(t, createErr)
			assert.ErrorIs(t, createErr, tt.wantErr)
			assert.Nil(t, collector)
		})
	}
}
