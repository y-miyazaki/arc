package aws

import (
	"context"
	"testing"
)

func TestNewConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		region  string
		profile string
		wantErr bool
	}{
		{
			name:    "valid config with region",
			region:  "us-east-1",
			profile: "",
			wantErr: false,
		},
		{
			name:    "empty region",
			region:  "",
			profile: "",
			wantErr: false, // Empty region is allowed, AWS SDK will use default region
		},
		{
			name:    "invalid profile",
			region:  "us-east-1",
			profile: "non-existent-profile",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(ctx, tt.region, tt.profile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewConfig(%q, %q) error = %v, wantErr %v", tt.region, tt.profile, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.region != "" && cfg.Region != tt.region {
				t.Fatalf("NewConfig(%q, %q) region = %v, want %v", tt.region, tt.profile, cfg.Region, tt.region)
			}
			if cfg.Credentials == nil {
				t.Fatalf("NewConfig(%q, %q) credentials = nil, want non-nil", tt.region, tt.profile)
			}
		})
	}
}
