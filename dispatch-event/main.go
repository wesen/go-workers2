package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	workers "github.com/digitalocean/go-workers2"
)

// EventData represents the structure of the event hash from Ruby
type EventData struct {
	Type          string                 `json:"type"`
	SourceService string                 `json:"source_service"`
	Data          map[string]interface{} `json:"data,omitempty"`
}

// JobArgs represents the arguments passed from the Ruby job
type JobArgs struct {
	EventHash              map[string]interface{} `json:"event_hash"`
	EventClassName         string                 `json:"event_class_name"`
	SlackAppInstallationID int                    `json:"slack_app_installation_id"`
	SlackUserID            *int                   `json:"slack_user_id,omitempty"`
}

func EventDispatcherProcessor(msg *workers.Msg) error {
	// Get the raw arguments JSON
	argsJSON := msg.Args().ToJson()

	// Parse the job arguments
	var args JobArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		log.Printf("Error parsing job arguments: %v", err)
		// Don't retry on malformed data
		return fmt.Errorf("malformed job data: %v", err)
	}

	// Validate required fields
	if args.EventHash == nil {
		return fmt.Errorf("event_hash is required")
	}
	if args.EventClassName == "" {
		return fmt.Errorf("event_class_name is required")
	}

	// Extract and validate type and source_service
	eventType, ok := args.EventHash["type"].(string)
	if !ok || eventType == "" {
		log.Printf("Event hash missing type: %v", args.EventHash)
		return nil // Match Ruby behavior of logging and returning
	}

	sourceService, ok := args.EventHash["source_service"].(string)
	if !ok || sourceService == "" {
		log.Printf("Event hash missing source_service: %v", args.EventHash)
		return nil // Match Ruby behavior of logging and returning
	}

	// Log debug information
	log.Printf("Processing event: class=%s, type=%s, source=%s, installation_id=%d",
		args.EventClassName, eventType, sourceService, args.SlackAppInstallationID)

	// TODO: Implement your actual event processing logic here
	// This would include:
	// 1. Fetching the SlackAppInstallation from your database
	// 2. Getting the bot token
	// 3. Fetching the SlackUser if SlackUserID is provided
	// 4. Hydrating the event (converting the hash to a proper event object)
	// 5. Dispatching the event to your service manager

	// Simulate some processing time
	time.Sleep(time.Millisecond * 100)

	return nil
}

func main() {
	// Configure the worker
	opts := workers.Options{
		ProcessID:  "event_dispatcher_worker",
		Namespace:  "sidekiq", // Important: Match Sidekiq's namespace
		ServerAddr: "localhost:6379",
		Database:   0,
		PoolSize:   10,
	}

	// Create a new manager
	manager, err := workers.NewManager(opts)
	if err != nil {
		log.Fatalf("Error creating manager: %v", err)
	}

	// Define middleware stack
	middlewares := workers.NewMiddlewares(
		// Logging middleware
		func(queue string, mgr *workers.Manager, next workers.JobFunc) workers.JobFunc {
			return func(msg *workers.Msg) error {
				start := time.Now()
				log.Printf("Starting EventDispatcherJob processing")
				err := next(msg)
				if err != nil {
					log.Printf("Error processing EventDispatcherJob: %v", err)
				}
				log.Printf("Finished EventDispatcherJob processing in %v", time.Since(start))
				return err
			}
		},
		workers.LogMiddleware,   // Default logging
		workers.RetryMiddleware, // Default retry handling
		workers.StatsMiddleware, // Default stats tracking
	)

	// Configure retry handling
	manager.SetRetriesExhaustedHandlers(func(queue string, msg *workers.Msg, err error) {
		log.Printf("Job exhausted retries: queue=%s, error=%v", queue, err)
	})

	// Add the worker using the new API
	manager.AddWorker("default", 10, EventDispatcherProcessor, middlewares...)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start processing
	log.Println("Starting EventDispatcher worker... Press Ctrl+C to stop")
	manager.Run(ctx)
}
