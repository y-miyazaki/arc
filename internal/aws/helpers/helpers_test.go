package helpers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStringValue_PrimitivesAndPointers(t *testing.T) {
	// nil
	assert.Equal(t, "N/A", StringValue(nil))

	// string
	s := "hello"
	assert.Equal(t, "hello", StringValue(s))
	empty := ""
	assert.Equal(t, "N/A", StringValue(empty))

	// *string
	assert.Equal(t, "hello", StringValue(&s))
	assert.Equal(t, "N/A", StringValue(&empty))

	// int family
	assert.Equal(t, "42", StringValue(42))
	var i32 int32 = 7
	assert.Equal(t, "7", StringValue(i32))
	var i64 int64 = 900
	assert.Equal(t, "900", StringValue(i64))

	// floats
	var f32 float32 = 1.5
	var f64 float64 = 2.25
	assert.Equal(t, "1.5", StringValue(f32))
	assert.Equal(t, "2.25", StringValue(f64))

	// bool
	assert.Equal(t, "true", StringValue(true))
	var bptr *bool
	assert.Equal(t, "N/A", StringValue(bptr))

	// time
	z := time.Time{}
	assert.Equal(t, "N/A", StringValue(z))
	now := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	assert.Equal(t, "2020-01-02T03:04:05Z", StringValue(now))

	// slices
	arr := []string{"a", "b"}
	assert.Equal(t, "a\nb", StringValue(arr))
	emptyArr := []string{}
	assert.Equal(t, "N/A", StringValue(emptyArr))

	// []*string
	s1 := "x"
	s2 := ""
	ptrs := []*string{&s1, &s2, nil}
	assert.Equal(t, "x", StringValue(ptrs))
}

func TestStringValue_DefaultOverride(t *testing.T) {
	assert.Equal(t, "default", StringValue(nil, "default"))
}

func TestExtractAccountID(t *testing.T) {
	arn := "arn:aws:iam::123456789012:role/test"
	acct, err := ExtractAccountID(arn)
	assert.NoError(t, err)
	assert.Equal(t, "123456789012", acct)

	// invalid
	_, err = ExtractAccountID("invalid-arn")
	assert.Error(t, err)
}

func TestToString(t *testing.T) {
	var p *string
	assert.Equal(t, "", ToString(p))
	s := "ok"
	assert.Equal(t, "ok", ToString(&s))
}

func TestNormalizeRawDataAndGetMapValue(t *testing.T) {
	data := map[string]any{
		"a": nil,
		"b": 123,
		"c": "",
	}
	out := NormalizeRawData(data)
	// after normalization nil and empty string become "N/A"
	assert.Equal(t, "N/A", out["a"])
	assert.Equal(t, "123", out["b"])
	assert.Equal(t, "N/A", out["c"])

	// GetMapValue returns empty string when default is empty
	assert.Equal(t, "", GetMapValue(out, "missing"))
	// but existing keys return their string values
	assert.Equal(t, "N/A", GetMapValue(out, "a"))
}

func TestFormatJSONIndent(t *testing.T) {
	// nil
	s, err := FormatJSONIndent(nil)
	assert.NoError(t, err)
	assert.Equal(t, "", s)

	// empty string
	s2, err := FormatJSONIndent("")
	assert.NoError(t, err)
	assert.Equal(t, "", s2)

	// valid json string
	in := `{"k":1}`
	out, err := FormatJSONIndent(in)
	assert.NoError(t, err)
	assert.Contains(t, out, "\n  \"k\": 1")

	// invalid json string -> error
	_, err = FormatJSONIndent("not-json")
	assert.Error(t, err)

	// map input
	m := map[string]any{"x": "y"}
	out2, err := FormatJSONIndent(m)
	assert.NoError(t, err)
	assert.Contains(t, out2, "\"x\": \"y\"")
}
