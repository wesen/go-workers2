package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/bitly/go-simplejson"
	workers "github.com/digitalocean/go-workers2"
	"golang.org/x/sync/errgroup"
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

// Simplified RubyJobArgs without tags
type RubyJobArgs struct {
	Name    string
	Message string
}

// processRubyJob handles jobs coming from Ruby
func processRubyJob(msg *workers.Msg) error {
	args := msg.Args()
	if args == nil {
		return fmt.Errorf("no arguments received")
	}

	var jobArgs RubyJobArgs
	if err := DecodeSidekiqArgs(args.Json, &jobArgs); err != nil {
		return fmt.Errorf("failed to decode job args: %v", err)
	}

	log.Printf("Received message from Ruby: %s says %s\n",
		jobArgs.Name, jobArgs.Message)
	return nil
}

// enqueueRubyJob sends a job to be processed by Ruby
func enqueueRubyJob(producer *workers.Producer) error {
	// Send arguments as individual values (not as a map)
	name := "Bob"
	message := fmt.Sprintf("Hello from Go! Sent at %s", time.Now().Format(time.RFC3339))

	log.Printf("Enqueueing job to Ruby with message: %s says %s", name, message)

	// Enqueue the job with separate arguments
	jid, err := producer.Enqueue("ruby_sidekiq", "RubyProcessorJob", []interface{}{name, message})
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %v", err)
	}

	log.Printf("Successfully enqueued job to Ruby with JID: %s", jid)
	return nil
}

func runJobSender(ctx context.Context, producer *workers.Producer) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Send first job immediately
	if err := enqueueRubyJob(producer); err != nil {
		log.Printf("Error enqueueing job: %v", err)
	}

	// Then send a job every 3 seconds
	for {
		select {
		case <-ticker.C:
			if err := enqueueRubyJob(producer); err != nil {
				log.Printf("Error enqueueing job: %v", err)
			}
		case <-ctx.Done():
			log.Println("Stopping job sender...")
			return nil
		}
	}
}

func main() {
	// Create a context that will be canceled on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create an error group with the context
	g, ctx := errgroup.WithContext(ctx)

	// Create a manager for the workers
	manager, err := workers.NewManager(workers.Options{
		ServerAddr: "localhost:6379",
		Database:   0,
		PoolSize:   30,
		ProcessID:  "1",
	})
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Add middleware for logging and retries
	middlewares := workers.DefaultMiddlewares()

	// Register the worker to process jobs from the "ruby_jobs" queue
	manager.AddWorker("ruby_jobs", 10, processRubyJob, middlewares...)

	// Start the stats server
	g.Go(func() error {
		log.Printf("Starting stats server at http://localhost:8080/stats")
		go workers.StartAPIServer(8080)
		<-ctx.Done()
		workers.StopAPIServer()
		log.Println("Stats server stopped")
		return nil
	})

	// Start the job sender
	// g.Go(func() error {
	// Create a producer for sending jobs to Ruby
	// producer := manager.Producer()
	// 	// Give Redis and the manager a moment to connect
	// 	time.Sleep(500 * time.Millisecond)
	// 	err := runJobSender(ctx, producer)
	// 	log.Println("Job sender stopped")
	// 	return err
	// })

	// Start the manager in a goroutine
	g.Go(func() error {
		manager.Run(ctx)
		log.Println("Manager exited")
		return nil
	})

	// Wait for all goroutines to complete or for an error
	if err := g.Wait(); err != nil && err != context.Canceled {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
