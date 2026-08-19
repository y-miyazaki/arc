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
	t.Parallel()

	tests := []struct {
		name string
		opts CollectionOptions
		want CollectionOptions
	}{
		{
			name: "fields round-trip",
			opts: CollectionOptions{
				Region:     "us-east-1",
				Profile:    "default",
				OutputDir:  "/tmp/output",
				Categories: "ec2,s3",
				HTML:       true,
				Timeout:    5 * time.Minute,
			},
			want: CollectionOptions{
				Region:     "us-east-1",
				Profile:    "default",
				OutputDir:  "/tmp/output",
				Categories: "ec2,s3",
				HTML:       true,
				Timeout:    5 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.opts.Region != tt.want.Region {
				t.Fatalf("CollectionOptions.Region = %q, want %q", tt.opts.Region, tt.want.Region)
			}
			if tt.opts.Profile != tt.want.Profile {
				t.Fatalf("CollectionOptions.Profile = %q, want %q", tt.opts.Profile, tt.want.Profile)
			}
			if tt.opts.OutputDir != tt.want.OutputDir {
				t.Fatalf("CollectionOptions.OutputDir = %q, want %q", tt.opts.OutputDir, tt.want.OutputDir)
			}
			if tt.opts.Categories != tt.want.Categories {
				t.Fatalf("CollectionOptions.Categories = %q, want %q", tt.opts.Categories, tt.want.Categories)
			}
			if tt.opts.HTML != tt.want.HTML {
				t.Fatalf("CollectionOptions.HTML = %v, want %v", tt.opts.HTML, tt.want.HTML)
			}
			if tt.opts.Timeout != tt.want.Timeout {
				t.Fatalf("CollectionOptions.Timeout = %v, want %v", tt.opts.Timeout, tt.want.Timeout)
			}
		})
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

func TestCollectResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		collectors      map[string]resources.Collector
		regions         []string
		maxConcurrency  int
		wantResultKeys  []string
		wantFailedKey   string
		wantFailedCount int
		wantResCount    int
	}{
		{
			name: "aggregates collector errors",
			collectors: map[string]resources.Collector{
				"ok":  &fakeCollector{name: "ok"},
				"bad": &fakeCollector{name: "bad", shouldError: true},
			},
			regions:         []string{"r1"},
			maxConcurrency:  2,
			wantResultKeys:  []string{"ok"},
			wantFailedKey:   "bad",
			wantFailedCount: 1,
		},
		{
			name: "preserves failures per region",
			collectors: map[string]resources.Collector{
				"bad": &fakeCollector{name: "bad", shouldError: true},
			},
			regions:         []string{"r1", "r2"},
			maxConcurrency:  2,
			wantFailedKey:   "bad",
			wantFailedCount: 2,
		},
		{
			name: "merges resources from multiple regions",
			collectors: map[string]resources.Collector{
				"test": &fakeCollector{name: "test"},
			},
			regions:        []string{"r1", "r2"},
			maxConcurrency: 2,
			wantResultKeys: []string{"test"},
			wantResCount:   2,
		},
		{
			name: "respects concurrency limit",
			collectors: map[string]resources.Collector{
				"test1": &fakeCollector{name: "test1"},
				"test2": &fakeCollector{name: "test2"},
			},
			regions:        []string{"r1", "r2"},
			maxConcurrency: 1,
			wantResultKeys: []string{"test1", "test2"},
		},
		{
			name: "zero uses default concurrency",
			collectors: map[string]resources.Collector{
				"ok": &fakeCollector{name: "ok"},
			},
			regions:        []string{"r1"},
			maxConcurrency: 0,
			wantResultKeys: []string{"ok"},
		},
		{
			name: "negative uses default concurrency",
			collectors: map[string]resources.Collector{
				"ok": &fakeCollector{name: "ok"},
			},
			regions:        []string{"r1"},
			maxConcurrency: -1,
			wantResultKeys: []string{"ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := logger.NewSlogLogger(&logger.SlogConfig{
				Output: io.Discard,
			})
			results, failed := collectResources(context.Background(), l, tt.collectors, tt.regions, &CollectionOptions{MaxConcurrency: tt.maxConcurrency})

			if len(tt.wantResultKeys) != len(results) && tt.wantFailedKey == "" {
				t.Fatalf("collectResources(...) results = %v, want keys %v", results, tt.wantResultKeys)
			}
			for _, key := range tt.wantResultKeys {
				if _, ok := results[key]; !ok {
					t.Fatalf("collectResources(...) results = %v, want key %q", results, key)
				}
			}
			if tt.wantFailedKey != "" {
				regionFailures, ok := failed[tt.wantFailedKey]
				if !ok {
					t.Fatalf("collectResources(...) failed = %v, want key %q", failed, tt.wantFailedKey)
				}
				if len(regionFailures) != tt.wantFailedCount {
					t.Fatalf("collectResources(...) failed[%q] = %v, want count %d", tt.wantFailedKey, regionFailures, tt.wantFailedCount)
				}
				seen := make(map[string]bool, len(regionFailures))
				for _, failure := range regionFailures {
					seen[failure.Region] = true
					if failure.Err == nil {
						t.Fatalf("collectResources(...) failure.Err = nil, want error")
					}
				}
				for _, region := range tt.regions {
					if !seen[region] {
						t.Fatalf("collectResources(...) failed regions = %v, want %q", regionFailures, region)
					}
				}
			} else if len(failed) != 0 {
				t.Fatalf("collectResources(...) failed = %v, want empty", failed)
			}
			if tt.wantResCount > 0 {
				got := results[tt.wantResultKeys[0]]
				if len(got.resources) != tt.wantResCount {
					t.Fatalf("collectResources(...) resource count = %d, want %d", len(got.resources), tt.wantResCount)
				}
			}
		})
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

func TestCreateRunContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		timeout   time.Duration
		wantDone  bool
		wantErr   error
		waitAfter time.Duration
	}{
		{
			name:      "positive timeout cancels with deadline exceeded",
			timeout:   20 * time.Millisecond,
			wantDone:  true,
			wantErr:   context.DeadlineExceeded,
			waitAfter: time.Second,
		},
		{
			name:      "zero timeout does not expire",
			timeout:   0,
			wantDone:  false,
			waitAfter: 20 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := createRunContext(context.Background(), tt.timeout)
			defer cancel()

			select {
			case <-ctx.Done():
				if !tt.wantDone {
					t.Fatalf("createRunContext(%v) context done unexpectedly: %v", tt.timeout, ctx.Err())
				}
			case <-time.After(tt.waitAfter):
				if tt.wantDone {
					t.Fatal("context did not time out")
				}
			}

			if tt.wantErr != nil && ctx.Err() != tt.wantErr {
				t.Fatalf("createRunContext(%v) error = %v, want %v", tt.timeout, ctx.Err(), tt.wantErr)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "log key category", got: LogKeyCategory, want: "category"},
		{name: "log key error", got: LogKeyError, want: "error"},
		{name: "log key file", got: LogKeyFile, want: "file"},
		{name: "default dir perm", got: DefaultDirPerm, want: 0o750},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Fatalf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestCollectionError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  CollectionError
		want string
	}{
		{
			name: "reports generic collection failure",
			err: CollectionError{
				Details: map[string][]CollectionFailure{
					"category1": {{Err: fmt.Errorf("error1"), Region: "r1"}},
					"category2": {{Err: fmt.Errorf("error2"), Region: "r2"}},
				},
			},
			want: "failed to collect one or more categories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("CollectionError.Error() = %q, want %q", got, tt.want)
			}
		})
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
