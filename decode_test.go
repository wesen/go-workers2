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

	type NestedStruct struct {
		Name    string
		Details BasicTypes
	}

	type PointerTypes struct {
		StringPtr *string
		IntPtr    *int
		StructPtr *BasicTypes
	}

	type MapTypes struct {
		StringMap    map[string]string
		IntMap       map[string]int
		StructMap    map[string]BasicTypes
		InterfaceMap map[string]interface{}
	}

	type SliceTypes struct {
		Strings  []string
		Ints     []int
		Int64s   []int64
		Float64s []float64
		Bools    []bool
	}

	type MixedTypes struct {
		String   string
		Ints     []int
		Float64  float64
		Bools    []bool
		LastBool bool
	}

	type PartialStruct struct {
		String string
		skip   string // unexported field
		Int    int
	}

	str := "test"
	num := 42

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
			name:    "nested struct",
			jsonStr: `[{"name": "test", "details": {"String": "inner", "Int": 1, "Int64": 2, "Float64": 1.23, "Bool": true}}]`,
			target: &struct {
				Data NestedStruct
			}{},
			expected: &struct {
				Data NestedStruct
			}{
				Data: NestedStruct{
					Name: "test",
					Details: BasicTypes{
						String:  "inner",
						Int:     1,
						Int64:   2,
						Float64: 1.23,
						Bool:    true,
					},
				},
			},
		},
		{
			name:    "pointer types",
			jsonStr: `["test", 42, {"String": "inner", "Int": 1, "Int64": 2, "Float64": 1.23, "Bool": true}]`,
			target:  &PointerTypes{},
			expected: &PointerTypes{
				StringPtr: &str,
				IntPtr:    &num,
				StructPtr: &BasicTypes{
					String:  "inner",
					Int:     1,
					Int64:   2,
					Float64: 1.23,
					Bool:    true,
				},
			},
		},
		{
			name:    "map types",
			jsonStr: `[{"key": "value"}, {"num": 42}, {"struct": {"String": "test", "Int": 1, "Int64": 2, "Float64": 1.23, "Bool": true}}, {"mixed": {"str": "value", "num": 42}}]`,
			target:  &MapTypes{},
			expected: &MapTypes{
				StringMap: map[string]string{"key": "value"},
				IntMap:    map[string]int{"num": 42},
				StructMap: map[string]BasicTypes{
					"struct": {
						String:  "test",
						Int:     1,
						Int64:   2,
						Float64: 1.23,
						Bool:    true,
					},
				},
				InterfaceMap: map[string]interface{}{
					"mixed": map[string]interface{}{
						"str": "value",
						"num": float64(42),
					},
				},
			},
		},
		{
			name:    "slice types all fields",
			jsonStr: `[["a", "b"], [1, 2], [3, 4], [1.1, 2.2], [true, false]]`,
			target:  &SliceTypes{},
			expected: &SliceTypes{
				Strings:  []string{"a", "b"},
				Ints:     []int{1, 2},
				Int64s:   []int64{3, 4},
				Float64s: []float64{1.1, 2.2},
				Bools:    []bool{true, false},
			},
		},
		{
			name:    "mixed types with slices",
			jsonStr: `["hello", [1, 2, 3], 3.14, [true, false], true]`,
			target:  &MixedTypes{},
			expected: &MixedTypes{
				String:   "hello",
				Ints:     []int{1, 2, 3},
				Float64:  3.14,
				Bools:    []bool{true, false},
				LastBool: true,
			},
		},
		{
			name:    "empty slices",
			jsonStr: `[[], [], [], [], []]`,
			target:  &SliceTypes{},
			expected: &SliceTypes{
				Strings:  []string{},
				Ints:     []int{},
				Int64s:   []int64{},
				Float64s: []float64{},
				Bools:    []bool{},
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
		{
			name:    "type conversion error - non-array to slice",
			jsonStr: `[42]`,
			target: &struct {
				Slice []int
			}{},
			expectError: true,
		},
		{
			name:    "type conversion error - wrong element type in slice",
			jsonStr: `[["not a number"]]`,
			target: &struct {
				Ints []int
			}{},
			expectError: true,
		},
		{
			name:    "type conversion error - mixed types in slice",
			jsonStr: `[[1, "not a number", 3]]`,
			target: &struct {
				Ints []int
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
