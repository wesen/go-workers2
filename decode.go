package workers

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bitly/go-simplejson"
)

// DecodeSidekiqArgs decodes a SimpleJSON array into a struct's public fields in order
func DecodeSidekiqArgs(args *simplejson.Json, target interface{}) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to a struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	// Get the raw JSON array
	arr, err := args.Array()
	if err != nil {
		return fmt.Errorf("failed to decode JSON array: %v", err)
	}

	// Create a map of field names to values
	values := make(map[string]interface{})
	currentIdx := 0

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		if currentIdx >= len(arr) {
			break
		}

		values[field.Name] = arr[currentIdx]
		currentIdx++
	}

	// Marshal the map back to JSON
	jsonBytes, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal intermediate JSON: %v", err)
	}

	// Unmarshal into the target struct
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal into target struct: %v", err)
	}

	return nil
}
