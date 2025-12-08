package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// ARNPartsCount represents the expected number of parts in an ARN
	ARNPartsCount = 5
	// ARNPartsAccountIndex represents the index of account ID in ARN parts
	ARNPartsAccountIndex = 4
	// DecimalBase is the base for decimal number formatting
	DecimalBase = 10
	// DefaultFalseString is the default string value for false boolean values
	DefaultFalseString = "false"
	// Float32Bits is the bit size for float32 formatting
	Float32Bits = 32
	// Float64Bits is the bit size for float64 formatting
	Float64Bits = 64
)

var (
	// ErrInvalidARNFormat indicates that the provided ARN has an invalid format
	ErrInvalidARNFormat = errors.New("invalid ARN format")
)

// StringValue converts any value to its string representation.
// It safely handles pointers and nil values.
// If the value is nil or empty (for strings), it returns the first defaultValue if provided, otherwise returns "N/A".
// This follows the project's policy of using "N/A" for missing values.
func StringValue(v any, defaultValues ...string) string {
	defaultValue := "N/A"
	if len(defaultValues) > 0 {
		defaultValue = defaultValues[0]
	}

	if v == nil {
		return defaultValue
	}

	switch val := v.(type) {
	case *string:
		if val == nil {
			return defaultValue
		}
		if *val == "" {
			return defaultValue
		}
		return *val
	case string:
		if val == "" {
			return defaultValue
		}
		return val
	case *int32:
		if val == nil {
			return defaultValue
		}
		return strconv.FormatInt(int64(*val), DecimalBase)
	case *int64:
		if val == nil {
			return defaultValue
		}
		return strconv.FormatInt(*val, DecimalBase)
	case int:
		return strconv.Itoa(val)
	case int32:
		return strconv.FormatInt(int64(val), DecimalBase)
	case int64:
		return strconv.FormatInt(val, DecimalBase)
	case *float32:
		if val == nil {
			return defaultValue
		}
		return strconv.FormatFloat(float64(*val), 'g', -1, Float32Bits)
	case *float64:
		if val == nil {
			return defaultValue
		}
		return strconv.FormatFloat(*val, 'g', -1, Float64Bits)
	case float32:
		return strconv.FormatFloat(float64(val), 'g', -1, Float32Bits)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, Float64Bits)
	case *bool:
		if val == nil {
			return defaultValue
		}
		return strconv.FormatBool(*val)
	case bool:
		return strconv.FormatBool(val)
	case *time.Time:
		if val == nil {
			return defaultValue
		}
		if val.IsZero() {
			return defaultValue
		}
		return val.UTC().Format(time.RFC3339)
	case time.Time:
		if val.IsZero() {
			return defaultValue
		}
		return val.UTC().Format(time.RFC3339)
	case []string:
		if len(val) == 0 {
			return defaultValue
		}
		slices.Sort(val)
		return strings.Join(val, "\n")
	case []*string:
		if len(val) == 0 {
			return defaultValue
		}
		var strs []string
		for _, s := range val {
			if s != nil && *s != "" {
				strs = append(strs, *s)
			}
		}
		if len(strs) == 0 {
			return defaultValue
		}
		slices.Sort(strs)
		return strings.Join(strs, "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ExtractAccountID extracts the AWS account ID from an ARN
func ExtractAccountID(arn string) (string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < ARNPartsCount {
		return "", fmt.Errorf("%w: %s", ErrInvalidARNFormat, arn)
	}
	return parts[ARNPartsAccountIndex], nil
}

// ToString returns the string value of the pointer, or empty string if the pointer is nil.
func ToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// NormalizeRawData processes the raw data map and replaces nil or empty string values with "N/A".
// It uses StringValue to handle various types consistently.
func NormalizeRawData(data map[string]any) map[string]any {
	for k, v := range data {
		data[k] = StringValue(v)
	}
	return data
}

// GetMapValue retrieves a string value for a key from a raw-data map.
// It uses StringValue with an empty default so absent or nil values
// return the empty string (preferred for CSV output).
func GetMapValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	return StringValue(data[key], "")
}

// FormatJSONIndent converts a value to an indented JSON string with 2-space indentation.
// If val is a string, it treats it as JSON and formats it.
// If val is any other type, it marshals the value directly.
// Returns error if marshaling/unmarshaling fails.
func FormatJSONIndent(val any) (string, error) {
	if val == nil {
		return "", nil
	}

	var data any
	if str, ok := val.(string); ok {
		// If it's a string, treat it as JSON and unmarshal first
		if str == "" {
			return "", nil
		}
		if err := json.Unmarshal([]byte(str), &data); err != nil {
			return "", fmt.Errorf("failed to unmarshal JSON string: %w", err)
		}
	} else {
		// Otherwise, use the value directly
		data = val
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(jsonBytes), nil
}
