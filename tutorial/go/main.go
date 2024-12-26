package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	workers "github.com/digitalocean/go-workers2"
	"golang.org/x/sync/errgroup"
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
	// Create a context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create an error group with the context
	g, ctx := errgroup.WithContext(ctx)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	// Create a sync.Once for stopping the manager
	var stopOnce sync.Once

	// Handle shutdown signal
	g.Go(func() error {
		select {
		case <-sigChan:
			log.Println("Received shutdown signal")
			cancel() // This will trigger all other goroutines to stop
			stopOnce.Do(func() {
				log.Println("Stopping manager...")
				manager.Stop()
			})
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Start the stats server
	g.Go(func() error {
		log.Printf("Starting stats server at http://localhost:8080/stats")
		go workers.StartAPIServer(8080)
		<-ctx.Done()
		workers.StopAPIServer()
		return nil
	})

	// Start the job sender
	g.Go(func() error {
		// Give Redis and the manager a moment to connect
		time.Sleep(500 * time.Millisecond)
		return runJobSender(ctx, producer)
	})

	// Start the manager in a goroutine
	g.Go(func() error {
		go func() {
			<-ctx.Done()
			stopOnce.Do(func() {
				log.Println("Stopping manager...")
				manager.Stop()
			})
		}()
		manager.Run()
		return nil
	})

	// Wait for all goroutines to complete or for an error
	if err := g.Wait(); err != nil && err != context.Canceled {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
