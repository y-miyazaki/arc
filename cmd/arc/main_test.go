package main

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/y-miyazaki/arc/internal/aws/resources"
	"github.com/y-miyazaki/arc/internal/logger"
)

func TestCollectionOptions(t *testing.T) {
	// Test that CollectionOptions struct can be created and fields accessed
	opts := &CollectionOptions{
		Region:     "us-east-1",
		Profile:    "default",
		OutputDir:  "/tmp/output",
		Categories: "ec2,s3",
		HTML:       true,
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
}

// fakeCollector is a small test helper implementing Collector
type fakeCollector struct {
	name        string
	shouldError bool
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

func TestCollectResources_AggregatesErrors(t *testing.T) {
	// Create two collectors: one succeeds, one fails
	collectors := map[string]resources.Collector{
		"ok":  &fakeCollector{name: "ok", shouldError: false},
		"bad": &fakeCollector{name: "bad", shouldError: true},
	}

	// Logger with discarded output
	l := logger.NewDefault()
	l.SetOutput(io.Discard)

	ctx := context.Background()
	results, failed := collectResources(ctx, l, collectors, []string{"r1"}, &CollectionOptions{MaxConcurrency: 2})

	if _, ok := results["ok"]; !ok {
		t.Fatalf("expected ok results, got %v", results)
	}
	if _, ok := failed["bad"]; !ok {
		t.Fatalf("expected bad in failed map, got %v", failed)
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

func TestErrInvalidOutputPath(t *testing.T) {
	// Test that the error variable is defined
	expectedMsg := "invalid output file path"
	if ErrInvalidOutputPath == nil {
		t.Errorf("ErrInvalidOutputPath should not be nil")
	}
	if ErrInvalidOutputPath.Error() != expectedMsg {
		t.Errorf("Expected ErrInvalidOutputPath message to be %q, got %q", expectedMsg, ErrInvalidOutputPath.Error())
	}
}

func TestInitializeRegions(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected []string
	}{
		{
			name:     "global service region",
			region:   GlobalServiceRegion,
			expected: []string{GlobalServiceRegion},
		},
		{
			name:     "non-global region",
			region:   "us-west-2",
			expected: []string{"us-west-2", GlobalServiceRegion},
		},
		{
			name:     "eu region",
			region:   "eu-west-1",
			expected: []string{"eu-west-1", GlobalServiceRegion},
		},
		{
			name:     "empty region",
			region:   "",
			expected: []string{GlobalServiceRegion},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initializeRegions expects a slice of regions; pass single-element slice
			result := initializeRegions([]string{tt.region})
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
