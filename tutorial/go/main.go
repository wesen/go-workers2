package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	workers "github.com/digitalocean/go-workers2"
	"golang.org/x/sync/errgroup"
)

// RubyJobArgs without tags
type RubyJobArgs struct {
	Name    string
	Message string
}

// GoJobArgs without tags
type GoJobArgs struct {
	Name    string
	Message string
}

// GoJobHandler handles jobs coming from Ruby
type GoJobHandler struct{}

func (h *GoJobHandler) HandleJob(args interface{}) error {
	jobArgs := args.(*GoJobArgs)
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
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Send first job immediately
	if err := enqueueRubyJob(producer); err != nil {
		log.Printf("Error enqueueing job: %v", err)
	}

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

	// Create and configure the job dispatcher
	dispatcher := workers.NewJobDispatcher()
	if err := dispatcher.RegisterHandler("GoProcessorJob", &GoJobHandler{}, &GoJobArgs{}); err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	// Register the worker to process jobs using the dispatcher
	manager.AddWorker("ruby_jobs", 10, dispatcher.Dispatch, middlewares...)

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
	g.Go(func() error {
		// Create a producer for sending jobs to Ruby
		producer := manager.Producer()
		// Give Redis and the manager a moment to connect
		time.Sleep(500 * time.Millisecond)
		err := runJobSender(ctx, producer)
		log.Println("Job sender stopped")
		return err
	})

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
