package validation_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/y-miyazaki/arc/internal/validation"
)

func TestCheckAWSCredentials(t *testing.T) {
	tests := []struct {
		name        string
		setupCfg    func() *aws.Config
		expectError bool
	}{
		{
			name: "test with default config",
			setupCfg: func() *aws.Config {
				return &aws.Config{}
			},
			expectError: true, // Will fail in test environment due to no credentials
		},
		{
			name: "test with nil config",
			setupCfg: func() *aws.Config {
				return nil
			},
			expectError: true, // Should return error for nil config
		},
		{
			name: "test with config having nil credentials",
			setupCfg: func() *aws.Config {
				return &aws.Config{
					Credentials: nil,
				}
			},
			expectError: true, // Will fail due to no credentials
		},
		{
			name: "test with config having empty region",
			setupCfg: func() *aws.Config {
				return &aws.Config{
					Region: "",
				}
			},
			expectError: true, // Will fail due to no credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := tt.setupCfg()

			_, err := validation.CheckAWSCredentials(ctx, cfg)

			if tt.expectError {
				if err == nil {
					t.Errorf("CheckAWSCredentials() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("CheckAWSCredentials() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckAWSCredentials_ErrorMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("nil config returns ErrNilConfig", func(t *testing.T) {
		_, err := validation.CheckAWSCredentials(ctx, nil)
		if err != validation.ErrNilConfig {
			t.Errorf("Expected ErrNilConfig, got %v", err)
		}
	})
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrEmptyARN",
			err:         validation.ErrEmptyARN,
			expectedMsg: "aws credentials are not set or invalid: empty ARN",
		},
		{
			name:        "ErrMissingTools",
			err:         validation.ErrMissingTools,
			expectedMsg: "missing required tools",
		},
		{
			name:        "ErrNilConfig",
			err:         validation.ErrNilConfig,
			expectedMsg: "aws config is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("Expected error message %q, got %q", tt.expectedMsg, tt.err.Error())
			}
		})
	}
}
