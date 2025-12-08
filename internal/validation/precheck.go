// Package validation provides pre-execution validation utilities.
package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var (
	// ErrEmptyARN is returned when the caller identity ARN is empty
	ErrEmptyARN = errors.New("aws credentials are not set or invalid: empty ARN")
	// ErrMissingTools is returned when required tools are not found in PATH
	ErrMissingTools = errors.New("missing required tools")
	// ErrNilConfig is returned when the AWS config is nil
	ErrNilConfig = errors.New("aws config is nil")
)

// CheckAWSCredentials validates AWS credentials by calling STS GetCallerIdentity.
// Returns the caller identity ARN on success, or an error if credentials are invalid.
func CheckAWSCredentials(ctx context.Context, cfg *aws.Config) (string, error) {
	if cfg == nil {
		return "", ErrNilConfig
	}
	svc := sts.NewFromConfig(*cfg)
	result, err := svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		// Check if the error is related to SSO session expiration
		if strings.Contains(err.Error(), "sso session has expired") ||
			strings.Contains(err.Error(), "SSO session has expired") {
			return "", fmt.Errorf("aws sso session has expired. please run 'aws sso login' to refresh your session: %w", err)
		}
		return "", fmt.Errorf("aws credentials are not set or invalid: %w", err)
	}

	arn := aws.ToString(result.Arn)
	if arn == "" {
		return "", ErrEmptyARN
	}

	return arn, nil
}
