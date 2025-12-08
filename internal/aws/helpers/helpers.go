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
	// ARNPartsCount represents the expected number of parts in an ARN (arn:partition:service:region:account:resource)
	ARNPartsCount = 6
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
	// Int64Bits is the bit size for int64 parsing
	Int64Bits = 64
	// MillisThreshold is the minimum integer value that indicates a millisecond epoch.
	// Values >= MillisThreshold are treated as milliseconds since epoch rather than seconds.
	MillisThreshold = 1000000000000 // 1_000_000_000_000
)

var (
	// ErrInvalidARNFormat indicates that the provided ARN has an invalid format
	ErrInvalidARNFormat = errors.New("invalid ARN format")
)

// ExtractAccountID extracts the AWS account ID from an ARN
func ExtractAccountID(arn string) (string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < ARNPartsCount {
		return "", fmt.Errorf("%w: %s", ErrInvalidARNFormat, arn)
	}
	return parts[ARNPartsAccountIndex], nil
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

// GetMapValue retrieves a string value for a key from a raw-data map.
// It uses StringValue with an empty default so absent or nil values
// return the empty string (preferred for CSV output).
func GetMapValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	return StringValue(data[key], "")
}

// NormalizeRawData processes the raw data map and replaces nil or empty string values with "N/A".
// It uses StringValue to handle various types consistently.
func NormalizeRawData(data map[string]any) map[string]any {
	for k, v := range data {
		data[k] = StringValue(v)
	}
	return data
}

// ParseTimestamp tries to convert a timestamp string into either *time.Time or the original string.
// Supported inputs:
// - epoch seconds (e.g. "1695601655")
// - epoch milliseconds (e.g. "1695601655000")
// - RFC3339 strings (e.g. "2023-08-10T09:00:00Z")
// Returns a *time.Time when parsing succeeds, otherwise returns the original string.
func ParseTimestamp(val string) any {
	if val == "" {
		return val
	}

	// Try epoch numeric parse
	if n, err := strconv.ParseInt(val, DecimalBase, Int64Bits); err == nil {
		// Decide whether value is seconds or milliseconds by threshold
		// Any value >= MillisThreshold we treat as milliseconds
		if n >= MillisThreshold {
			t := time.Unix(0, n*int64(time.Millisecond)).UTC()
			return &t
		}
		t := time.Unix(n, 0).UTC()
		return &t
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		t = t.UTC()
		return &t
	}

	// fallback to original string
	return val
}

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
		// avoid mutating the caller's slice: make a copy before sorting
		tmp := slices.Clone(val)
		slices.Sort(tmp)
		return strings.Join(tmp, "\n")
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

// ToString returns the string value of the pointer, or empty string if the pointer is nil.
func ToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
