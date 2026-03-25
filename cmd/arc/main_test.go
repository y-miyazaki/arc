package main

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/y-miyazaki/arc/internal/aws/resources"
	"github.com/y-miyazaki/go-common/pkg/logger"
)

func TestCollectionOptions(t *testing.T) {
	// Test that CollectionOptions struct can be created and fields accessed
	opts := &CollectionOptions{
		Region:     "us-east-1",
		Profile:    "default",
		OutputDir:  "/tmp/output",
		Categories: "ec2,s3",
		HTML:       true,
		Timeout:    5 * time.Minute,
	}

	if opts.Region != "us-east-1" {
		t.Errorf("Expected Region to be 'us-east-1', got %s", opts.Region)
	}
	if opts.Profile != "default" {
		t.Errorf("Expected Profile to be 'default', got %s", opts.Profile)
	}
	if opts.OutputDir != "/tmp/output" {
		t.Errorf("Expected OutputDir to be '/tmp/output', got %s", opts.OutputDir)
	}
	if opts.Categories != "ec2,s3" {
		t.Errorf("Expected Categories to be 'ec2,s3', got %s", opts.Categories)
	}
	if opts.HTML != true {
		t.Errorf("Expected HTML to be true, got %t", opts.HTML)
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Expected Timeout to be 5m, got %s", opts.Timeout)
	}
}

// fakeCollector is a small test helper implementing Collector
type fakeCollector struct {
	name        string
	shouldError bool
}

type blockingCollector struct {
	calls atomic.Int32
	name  string
}

func (f *fakeCollector) Name() string { return f.name }
func (f *fakeCollector) GetColumns() []resources.Column {
	return []resources.Column{{Header: "h", Value: func(r resources.Resource) string { return r.Name }}}
}
func (f *fakeCollector) ShouldSort() bool { return false }
func (f *fakeCollector) Collect(ctx context.Context, region string) ([]resources.Resource, error) {
	if f.shouldError {
		return nil, fmt.Errorf("collector %s failed", f.name)
	}
	return []resources.Resource{{Category: f.name, Name: f.name + "-r", Region: region}}, nil
}

func (b *blockingCollector) Name() string { return b.name }
func (b *blockingCollector) GetColumns() []resources.Column {
	return []resources.Column{{Header: "h", Value: func(r resources.Resource) string { return r.Name }}}
}
func (b *blockingCollector) ShouldSort() bool { return false }
func (b *blockingCollector) Collect(ctx context.Context, region string) ([]resources.Resource, error) {
	callNumber := b.calls.Add(1)
	if callNumber == 1 {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return []resources.Resource{{Category: b.name, Name: b.name + "-r", Region: region}}, nil
}

func TestCollectResources_AggregatesErrors(t *testing.T) {
	// Create two collectors: one succeeds, one fails
	collectors := map[string]resources.Collector{
		"ok":  &fakeCollector{name: "ok", shouldError: false},
		"bad": &fakeCollector{name: "bad", shouldError: true},
	}

	// Logger with discarded output
	l := logger.NewSlogLogger(&logger.SlogConfig{
		Output: io.Discard,
	})

	ctx := context.Background()
	results, failed := collectResources(ctx, l, collectors, []string{"r1"}, &CollectionOptions{MaxConcurrency: 2})

	if _, ok := results["ok"]; !ok {
		t.Fatalf("expected ok results, got %v", results)
	}
	if _, ok := failed["bad"]; !ok {
		t.Fatalf("expected bad in failed map, got %v", failed)
	}
	if len(failed["bad"]) != 1 {
		t.Fatalf("expected one bad failure, got %v", failed["bad"])
	}
	if failed["bad"][0].Region != "r1" {
		t.Fatalf("expected failure region r1, got %q", failed["bad"][0].Region)
	}
}

func TestCollectResources_PreservesFailuresPerRegion(t *testing.T) {
	collectors := map[string]resources.Collector{
		"bad": &fakeCollector{name: "bad", shouldError: true},
	}

	l := logger.NewSlogLogger(&logger.SlogConfig{
		Output: io.Discard,
	})

	ctx := context.Background()
	_, failed := collectResources(ctx, l, collectors, []string{"r1", "r2"}, &CollectionOptions{MaxConcurrency: 2})

	regionFailures, ok := failed["bad"]
	if !ok {
		t.Fatalf("expected bad in failed map, got %v", failed)
	}
	if len(regionFailures) != 2 {
		t.Fatalf("expected two bad failures, got %v", regionFailures)
	}

	seen := make(map[string]bool, len(regionFailures))
	for _, failure := range regionFailures {
		seen[failure.Region] = true
		if failure.Err == nil {
			t.Fatal("expected region failure error to be set")
		}
	}

	if !seen["r1"] || !seen["r2"] {
		t.Fatalf("expected failures from both regions, got %v", regionFailures)
	}
}

func TestCollectResources_MultipleRegions(t *testing.T) {
	// Test that resources from multiple regions are merged correctly
	collectors := map[string]resources.Collector{
		"test": &fakeCollector{name: "test", shouldError: false},
	}

	l := logger.NewSlogLogger(&logger.SlogConfig{
		Output: io.Discard,
	})

	ctx := context.Background()
	results, failed := collectResources(ctx, l, collectors, []string{"r1", "r2"}, &CollectionOptions{MaxConcurrency: 2})

	if len(failed) != 0 {
		t.Fatalf("expected no failures, got %v", failed)
	}

	result, ok := results["test"]
	if !ok {
		t.Fatalf("expected test results, got %v", results)
	}

	// Should have resources from both regions
	if len(result.resources) != 2 {
		t.Fatalf("expected 2 resources (one per region), got %d", len(result.resources))
	}

	// Check that both regions are represented
	regions := make(map[string]bool)
	for _, res := range result.resources {
		regions[res.Region] = true
	}

	if !regions["r1"] || !regions["r2"] {
		t.Errorf("expected resources from both r1 and r2, got regions %v", regions)
	}
}

func TestCollectResources_ConcurrencyLimit(t *testing.T) {
	// Test that concurrency limit is respected (hard to test directly, but we can verify it doesn't panic)
	collectors := map[string]resources.Collector{
		"test1": &fakeCollector{name: "test1", shouldError: false},
		"test2": &fakeCollector{name: "test2", shouldError: false},
	}

	l := logger.NewSlogLogger(&logger.SlogConfig{
		Output: io.Discard,
	})

	ctx := context.Background()
	results, failed := collectResources(ctx, l, collectors, []string{"r1", "r2"}, &CollectionOptions{MaxConcurrency: 1})

	if len(failed) != 0 {
		t.Fatalf("expected no failures, got %v", failed)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 successful results, got %d", len(results))
	}
}

func TestCollectResources_RespectsContextCancelWhileWaitingForSemaphore(t *testing.T) {
	collector := &blockingCollector{name: "blocking"}
	l := logger.NewSlogLogger(&logger.SlogConfig{
		Output: io.Discard,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_, _ = collectResources(ctx, l, map[string]resources.Collector{"blocking": collector}, []string{"r1", "r2"}, &CollectionOptions{MaxConcurrency: 1})
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for collector.calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first collection to start")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("collectResources did not stop after context cancellation")
	}

	if collector.calls.Load() != 1 {
		t.Fatalf("expected only one collect call before cancellation, got %d", collector.calls.Load())
	}
}

func TestCreateRunContext_Timeout(t *testing.T) {
	ctx, cancel := createRunContext(context.Background(), 20*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("context did not time out")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestConstants(t *testing.T) {
	// Test that constants are defined and have expected values
	if LogKeyCategory != "category" {
		t.Errorf("Expected LogKeyCategory to be 'category', got %s", LogKeyCategory)
	}
	if LogKeyError != "error" {
		t.Errorf("Expected LogKeyError to be 'error', got %s", LogKeyError)
	}
	if LogKeyFile != "file" {
		t.Errorf("Expected LogKeyFile to be 'file', got %s", LogKeyFile)
	}
	if DefaultDirPerm != 0750 {
		t.Errorf("Expected DefaultDirPerm to be 0750, got %d", DefaultDirPerm)
	}
}

func TestCollectionError_Error(t *testing.T) {
	ce := CollectionError{
		Details: map[string][]CollectionFailure{
			"category1": {{Err: fmt.Errorf("error1"), Region: "r1"}},
			"category2": {{Err: fmt.Errorf("error2"), Region: "r2"}},
		},
	}

	expected := "failed to collect one or more categories"
	if ce.Error() != expected {
		t.Errorf("CollectionError.Error() = %v, want %v", ce.Error(), expected)
	}
}

func TestInitializeRegions(t *testing.T) {
	tests := []struct {
		name        string
		userRegions []string
		expected    []string
	}{
		{
			name:        "single region same as global",
			userRegions: []string{"us-east-1"},
			expected:    []string{"us-east-1"},
		},
		{
			name:        "multiple regions including global",
			userRegions: []string{"us-east-1", "us-west-2"},
			expected:    []string{"us-east-1", "us-west-2"},
		},
		{
			name:        "global service region already included",
			userRegions: []string{"us-east-1", GlobalServiceRegion},
			expected:    []string{"us-east-1"},
		},
		{
			name:        "empty regions",
			userRegions: []string{},
			expected:    []string{GlobalServiceRegion},
		},
		{
			name:        "empty string in regions",
			userRegions: []string{"us-east-1", "", "us-west-2"},
			expected:    []string{"us-east-1", "us-west-2"},
		},
		{
			name:        "different region",
			userRegions: []string{"eu-west-1"},
			expected:    []string{"eu-west-1", GlobalServiceRegion},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := initializeRegions(tt.userRegions)
			if len(result) != len(tt.expected) {
				t.Errorf("initializeRegions() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, region := range result {
				if region != tt.expected[i] {
					t.Errorf("initializeRegions()[%d] = %v, want %v", i, region, tt.expected[i])
				}
			}
		})
	}
}

func TestParseCommaList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "us-east-1",
			expected: []string{"us-east-1"},
		},
		{
			name:     "multiple values",
			input:    "us-east-1,us-west-2,eu-west-1",
			expected: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:     "values with spaces",
			input:    "us-east-1, us-west-2 , eu-west-1",
			expected: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:     "empty values",
			input:    "us-east-1,,us-west-2,",
			expected: []string{"us-east-1", "us-west-2"},
		},
		{
			name:     "duplicates",
			input:    "us-east-1,us-west-2,us-east-1",
			expected: []string{"us-east-1", "us-west-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseCommaList() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, region := range result {
				if region != tt.expected[i] {
					t.Errorf("parseCommaList()[%d] = %v, want %v", i, region, tt.expected[i])
				}
			}
		})
	}
}
