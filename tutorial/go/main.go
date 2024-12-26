package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	workers "github.com/digitalocean/go-workers2"
)

// RubyJobArgs represents the structure of arguments received from Ruby
type RubyJobArgs struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// processRubyJob handles jobs coming from Ruby
func processRubyJob(msg *workers.Msg) error {
	// Ruby sends arguments as an array of values
	args := msg.Args()
	if args == nil {
		return fmt.Errorf("no arguments received")
	}

	// Extract arguments from the array
	name, err := args.GetIndex(0).String()
	if err != nil {
		return fmt.Errorf("failed to get name: %v", err)
	}

	message, err := args.GetIndex(1).String()
	if err != nil {
		return fmt.Errorf("failed to get message: %v", err)
	}

	// Process the job
	log.Printf("Received message from Ruby: %s says %s\n",
		name, message)
	return nil
}

// enqueueRubyJob sends a job to be processed by Ruby
func enqueueRubyJob(producer *workers.Producer) error {
	// Send arguments as individual values (not as a map)
	name := "Bob"
	message := "Hello from Go!"

	// Enqueue the job with separate arguments
	jid, err := producer.Enqueue("ruby_sidekiq_queue", "RubyProcessorJob", []interface{}{name, message})
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %v", err)
	}

	log.Printf("Successfully enqueued job to Ruby with JID: %s", jid)
	return nil
}

func main() {
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

	// Create a producer for sending jobs to Ruby
	producer := manager.Producer()

	// Start the stats server first
	go workers.StartAPIServer(8080)

	// Give the API server a moment to start
	time.Sleep(100 * time.Millisecond)

	log.Printf("Worker starting. Stats will be available at http://localhost:8080/stats")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to periodically send jobs to Ruby (for demo purposes)
	go func() {
		// Give Redis and the manager a moment to connect
		time.Sleep(500 * time.Millisecond)

		// Enqueue one job right away
		if err := enqueueRubyJob(producer); err != nil {
			log.Printf("Error enqueueing job: %v", err)
		}
	}()

	// Start processing jobs
	manager.Run()
}
