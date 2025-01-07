package workers

import (
	"fmt"
	"reflect"
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

// Dispatch routes a message to its registered handler
func (d *JobDispatcher) Dispatch(msg *Msg) error {
	class := msg.Class()
	handlerInfo, ok := d.handlers[class]
	if !ok {
		return fmt.Errorf("no handler registered for job class: %s", class)
	}

	args := msg.Args()
	if args == nil {
		return fmt.Errorf("no arguments received for job class: %s", class)
	}

	// Create a new instance of the args struct
	argsValue := reflect.New(handlerInfo.argsType.Elem())
	argsInterface := argsValue.Interface()
	// Decode the arguments
	if err := DecodeSidekiqArgs(args.Json, argsInterface); err != nil {
		return fmt.Errorf("failed to decode job args for class %s: %v", class, err)
	}

	// Call the handler
	return handlerInfo.handler.HandleJob(argsInterface)
}
