package workers

import (
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

	t := v.Type()
	currentIdx := 0

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the value at the current index
		jsonVal := args.GetIndex(currentIdx)
		fieldValue := v.Field(i)

		// Handle different field types
		switch fieldValue.Kind() {
		case reflect.String:
			str, err := jsonVal.String()
			if err != nil {
				return fmt.Errorf("failed to decode string for field %s: %v", field.Name, err)
			}
			fieldValue.SetString(str)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			num, err := jsonVal.Int64()
			if err != nil {
				return fmt.Errorf("failed to decode int for field %s: %v", field.Name, err)
			}
			fieldValue.SetInt(num)
		case reflect.Float32, reflect.Float64:
			num, err := jsonVal.Float64()
			if err != nil {
				return fmt.Errorf("failed to decode float for field %s: %v", field.Name, err)
			}
			fieldValue.SetFloat(num)
		case reflect.Bool:
			b, err := jsonVal.Bool()
			if err != nil {
				return fmt.Errorf("failed to decode bool for field %s: %v", field.Name, err)
			}
			fieldValue.SetBool(b)
		default:
			return fmt.Errorf("unsupported type %v for field %s", fieldValue.Kind(), field.Name)
		}

		currentIdx++
	}

	return nil
}
