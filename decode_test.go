package workers

import (
	"testing"

	"github.com/bitly/go-simplejson"
	"github.com/stretchr/testify/assert"
)

func TestDecodeSidekiqArgs(t *testing.T) {
	type BasicTypes struct {
		String  string
		Int     int
		Int64   int64
		Float64 float64
		Bool    bool
	}

	type PartialStruct struct {
		String string
		skip   string // unexported field
		Int    int
	}

	tests := []struct {
		name        string
		jsonStr     string
		target      interface{}
		expected    interface{}
		expectError bool
	}{
		{
			name:    "basic types all fields",
			jsonStr: `["hello", 42, 64, 3.14, true]`,
			target:  &BasicTypes{},
			expected: &BasicTypes{
				String:  "hello",
				Int:     42,
				Int64:   64,
				Float64: 3.14,
				Bool:    true,
			},
		},
		{
			name:    "partial struct with unexported field",
			jsonStr: `["hello", 42]`,
			target:  &PartialStruct{},
			expected: &PartialStruct{
				String: "hello",
				Int:    42,
			},
		},
		{
			name:        "nil pointer",
			jsonStr:     `["hello"]`,
			target:      nil,
			expectError: true,
		},
		{
			name:        "non-pointer",
			jsonStr:     `["hello"]`,
			target:      BasicTypes{},
			expectError: true,
		},
		{
			name:        "pointer to non-struct",
			jsonStr:     `["hello"]`,
			target:      new(string),
			expectError: true,
		},
		{
			name:    "type conversion error - string to int",
			jsonStr: `["not a number", "hello"]`,
			target: &struct {
				Number int
				Text   string
			}{},
			expectError: true,
		},
		{
			name:    "type conversion error - bool to string",
			jsonStr: `[true, "hello"]`,
			target: &struct {
				Text1 string
				Text2 string
			}{},
			expectError: true,
		},
		{
			name:    "type conversion error - number to bool",
			jsonStr: `[42]`,
			target: &struct {
				Bool bool
			}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js, err := simplejson.NewJson([]byte(tt.jsonStr))
			assert.NoError(t, err)

			err = DecodeSidekiqArgs(js, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tt.target)
			}
		})
	}
}
