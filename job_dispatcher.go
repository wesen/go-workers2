package workers

import (
	"fmt"
	"log"
	"reflect"

	"github.com/bitly/go-simplejson"
)

// JobHandler interface defines the contract for job handlers
type JobHandler interface {
	HandleJob(args interface{}) error
}

// JobDispatcher manages job handlers and routes messages to them
type JobDispatcher struct {
	handlers map[string]struct {
		handler  JobHandler
		argsType reflect.Type
	}
}

// NewJobDispatcher creates a new JobDispatcher instance
func NewJobDispatcher() *JobDispatcher {
	return &JobDispatcher{
		handlers: make(map[string]struct {
			handler  JobHandler
			argsType reflect.Type
		}),
	}
}

// RegisterHandler registers a handler for a specific job class
func (d *JobDispatcher) RegisterHandler(class string, handler JobHandler, argsType interface{}) error {
	t := reflect.TypeOf(argsType)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("argsType must be a pointer to a struct")
	}

	d.handlers[class] = struct {
		handler  JobHandler
		argsType reflect.Type
	}{
		handler:  handler,
		argsType: t,
	}
	return nil
}

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

// Dispatch routes a message to its registered handler
func (d *JobDispatcher) Dispatch(msg *Msg) error {
	log.Printf("Dispatching message: %v", msg)
	class := msg.Class()
	handlerInfo, ok := d.handlers[class]
	if !ok {
		log.Printf("No handler registered for job class: %s", class)
		return fmt.Errorf("no handler registered for job class: %s", class)
	}
	log.Printf("Handler found for class: %s", class)

	args := msg.Args()
	if args == nil {
		return fmt.Errorf("no arguments received for job class: %s", class)
	}
	log.Printf("Arguments received for class: %s", class)

	// Create a new instance of the args struct
	argsValue := reflect.New(handlerInfo.argsType.Elem())
	argsInterface := argsValue.Interface()
	// Decode the arguments
	if err := DecodeSidekiqArgs(args.Json, argsInterface); err != nil {
		log.Printf("Failed to decode job args for class %s: %v", class, err)
		return fmt.Errorf("failed to decode job args for class %s: %v", class, err)
	}

	// Call the handler
	log.Printf("Calling handler for class: %s", class)
	return handlerInfo.handler.HandleJob(argsInterface)
}
