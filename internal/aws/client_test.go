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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(ctx, tt.region, tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.region != "" && cfg.Region != tt.region {
					t.Errorf("NewConfig() region = %v, want %v", cfg.Region, tt.region)
				}
				// Verify that config has credentials (basic check)
				if cfg.Credentials == nil {
					t.Error("NewConfig() credentials should not be nil")
				}
			}
		})
	}
}

func TestNewConfigWithInvalidProfile(t *testing.T) {
	ctx := context.Background()

	// Test with a non-existent profile - this should fail in most environments
	_, err := NewConfig(ctx, "us-east-1", "non-existent-profile")
	if err == nil {
		t.Error("NewConfig() with invalid profile should return error")
	}
}
