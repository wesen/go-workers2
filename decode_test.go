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

	type ComplexSliceTypes struct {
		NestedStrings [][]string
		SlicePointers []*string
		StructSlices  []BasicTypes
		MixedSlice    []interface{}
	}

	type ComplexMapTypes struct {
		NestedMap   map[string]map[string]string
		MapPointers map[string]*BasicTypes
		SliceMap    map[string][]string
	}

	type TaggedStruct struct {
		Struct struct {
			RenamedField string `json:"renamed"`
			IgnoredField string `json:"-"`
			EmptyOmitted string `json:"omitted,omitempty"`
		}
	}

	type NullableTypes struct {
		NullString    *string
		NullStruct    *BasicTypes
		NullSlice     []string
		NullMap       map[string]string
		NullInterface interface{}
	}

	// Helper function to compare string pointer slices
	compareStringPointerSlices := func(t *testing.T, expected, actual []*string) {
		assert.Equal(t, len(expected), len(actual), "slice lengths should match")
		for i := range expected {
			if expected[i] == nil {
				assert.Nil(t, actual[i], "element %d should be nil", i)
			} else {
				assert.NotNil(t, actual[i], "element %d should not be nil", i)
				assert.Equal(t, *expected[i], *actual[i], "element %d values should match", i)
			}
		}
	}

	// Helper function to compare maps with BasicTypes pointers
	compareBasicTypesPointerMap := func(t *testing.T, expected, actual map[string]*BasicTypes) {
		assert.Equal(t, len(expected), len(actual), "map lengths should match")
		for k, expectedVal := range expected {
			actualVal, ok := actual[k]
			if !ok {
				t.Errorf("key %q missing from actual map", k)
				continue
			}
			if expectedVal == nil {
				assert.Nil(t, actualVal, "value for key %q should be nil", k)
			} else {
				assert.NotNil(t, actualVal, "value for key %q should not be nil", k)
				assert.Equal(t, *expectedVal, *actualVal, "values for key %q should match", k)
			}
		}
	}

	str := "test"
	num := 42
	testStr1 := "test"
	testStr2 := "test2"
	value := "value"

	tests := []struct {
		name        string
		jsonStr     string
		target      interface{}
		expected    interface{}
		expectError bool
		compare     func(t *testing.T, expected, actual interface{}) // optional custom comparison
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
			target:  &BasicTypes{},
			expected: &BasicTypes{
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
		{
			name:    "complex slices",
			jsonStr: `[[["a", "b"], ["c", "d"]], ["test", "test2"], [{"String": "s1", "Int": 1}, {"String": "s2", "Int": 2}], ["string", 42, true]]`,
			target:  &ComplexSliceTypes{},
			expected: &ComplexSliceTypes{
				NestedStrings: [][]string{{"a", "b"}, {"c", "d"}},
				SlicePointers: []*string{&testStr1, &testStr2},
				StructSlices: []BasicTypes{
					{String: "s1", Int: 1},
					{String: "s2", Int: 2},
				},
				MixedSlice: []interface{}{"string", float64(42), true},
			},
			compare: func(t *testing.T, expected, actual interface{}) {
				e := expected.(*ComplexSliceTypes)
				a := actual.(*ComplexSliceTypes)

				// Compare non-pointer fields normally
				assert.Equal(t, e.NestedStrings, a.NestedStrings)
				assert.Equal(t, e.StructSlices, a.StructSlices)
				assert.Equal(t, e.MixedSlice, a.MixedSlice)

				// Compare pointer slice specially
				compareStringPointerSlices(t, e.SlicePointers, a.SlicePointers)
			},
		},
		{
			name:    "complex maps",
			jsonStr: `[{"key1": {"inner": "value"}}, {"key": {"String": "test"}}, {"key": ["a", "b"]}]`,
			target:  &ComplexMapTypes{},
			expected: &ComplexMapTypes{
				NestedMap: map[string]map[string]string{
					"key1": map[string]string{
						"inner": "value",
					},
				},
				MapPointers: map[string]*BasicTypes{
					"key": {String: "test"},
				},
				SliceMap: map[string][]string{
					"key": []string{"a", "b"},
				},
			},
			compare: func(t *testing.T, expected, actual interface{}) {
				e := expected.(*ComplexMapTypes)
				a := actual.(*ComplexMapTypes)

				// Compare nested maps
				assert.Equal(t, len(e.NestedMap), len(a.NestedMap), "NestedMap lengths should match")
				for k1, expectedInner := range e.NestedMap {
					actualInner, ok := a.NestedMap[k1]
					if !ok {
						t.Errorf("key %q missing from NestedMap", k1)
						continue
					}
					assert.Equal(t, expectedInner, actualInner, "inner maps for key %q should match", k1)
				}

				// Compare pointer maps
				compareBasicTypesPointerMap(t, e.MapPointers, a.MapPointers)

				// Compare slice maps
				assert.Equal(t, len(e.SliceMap), len(a.SliceMap), "SliceMap lengths should match")
				for k, expectedSlice := range e.SliceMap {
					actualSlice, ok := a.SliceMap[k]
					if !ok {
						t.Errorf("key %q missing from SliceMap", k)
						continue
					}
					assert.Equal(t, expectedSlice, actualSlice, "slices for key %q should match", k)
				}
			},
		},
		{
			name:    "json tags",
			jsonStr: `[{"renamed": "new name", "ignored": "should not set", "omitted": ""}]`,
			target:  &TaggedStruct{},
			expected: &TaggedStruct{
				Struct: struct {
					RenamedField string `json:"renamed"`
					IgnoredField string `json:"-"`
					EmptyOmitted string `json:"omitted,omitempty"`
				}{
					RenamedField: "new name",
					IgnoredField: "", // Should remain empty
					EmptyOmitted: "", // Should be included even though empty
				},
			},
			compare: func(t *testing.T, expected, actual interface{}) {
				e := expected.(*TaggedStruct)
				a := actual.(*TaggedStruct)
				assert.Equal(t, e.Struct.RenamedField, a.Struct.RenamedField, "renamed field should match")
				assert.Equal(t, e.Struct.IgnoredField, a.Struct.IgnoredField, "ignored field should match")
				assert.Equal(t, e.Struct.EmptyOmitted, a.Struct.EmptyOmitted, "empty field should match")
			},
		},
		{
			name:    "null values",
			jsonStr: `[null, null, null, null, null]`,
			target:  &NullableTypes{},
			expected: &NullableTypes{
				NullString:    nil,
				NullStruct:    nil,
				NullSlice:     nil,
				NullMap:       nil,
				NullInterface: nil,
			},
		},
		{
			name:    "mixed null and non-null",
			jsonStr: `["value", {"String": "test"}, ["item"], {"key": "value"}, 42]`,
			target:  &NullableTypes{},
			expected: &NullableTypes{
				NullString:    &value,
				NullStruct:    &BasicTypes{String: "test"},
				NullSlice:     []string{"item"},
				NullMap:       map[string]string{"key": "value"},
				NullInterface: float64(42),
			},
			compare: func(t *testing.T, expected, actual interface{}) {
				e := expected.(*NullableTypes)
				a := actual.(*NullableTypes)
				// Compare string pointers by value
				if e.NullString == nil {
					assert.Nil(t, a.NullString)
				} else {
					assert.NotNil(t, a.NullString)
					assert.Equal(t, *e.NullString, *a.NullString)
				}
				// Compare struct pointers by value
				if e.NullStruct == nil {
					assert.Nil(t, a.NullStruct)
				} else {
					assert.NotNil(t, a.NullStruct)
					assert.Equal(t, *e.NullStruct, *a.NullStruct)
				}
				// Compare remaining fields normally
				assert.Equal(t, e.NullSlice, a.NullSlice)
				assert.Equal(t, e.NullMap, a.NullMap)
				assert.Equal(t, e.NullInterface, a.NullInterface)
			},
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
				if tt.compare != nil {
					tt.compare(t, tt.expected, tt.target)
				} else {
					assert.Equal(t, tt.expected, tt.target)
				}
			}
		})
	}
}
