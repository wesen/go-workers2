
# Bidirectional Job Processing Between Go and Ruby with Sidekiq

## Overview
This tutorial will show you how to:
1. Set up a Go worker that can process jobs from Ruby's Sidekiq
2. Set up Ruby to enqueue jobs to be processed by Go
3. Set up Go to enqueue jobs to be processed by Ruby
4. Handle job arguments and error cases in both directions

## Prerequisites
- Go 1.13+
- Ruby 2.5+
- Redis server running (default: localhost:6379)
- Sidekiq gem installed in Ruby
- go-workers2 package installed in Go

## Part 1: Setting Up the Go Worker

First, let's create a Go worker that can process jobs from Ruby. The worker will use the `go-workers2` package which is Sidekiq-compatible.

```go
package main

import (
    "fmt"
    "encoding/json"
    workers "github.com/digitalocean/go-workers2"
)

// Define a struct that matches your Ruby job arguments
type RubyJobArgs struct {
    Name    string `json:"name"`
    Message string `json:"message"`
}

func processRubyJob(msg *workers.Msg) error {
    // Get the raw arguments JSON
    argsJSON := msg.Args().ToJson()

    // Parse the job arguments
    var args RubyJobArgs
    if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
        return fmt.Errorf("malformed job data: %v", err)
    }

    // Process the job
    fmt.Printf("Received message from Ruby: %s says %s\n", 
        args.Name, args.Message)
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
        panic(err)
    }

    // Add middleware for logging and retries
    middlewares := workers.DefaultMiddlewares()

    // Register the worker to process jobs from the "ruby_jobs" queue
    manager.AddWorker("ruby_jobs", 10, processRubyJob, middlewares...)

    // Start the stats server (optional)
    go workers.StartAPIServer(8080)

    // Start processing jobs
    manager.Run()
}
```

## Part 2: Ruby Side - Enqueuing Jobs to Go

In your Ruby application, you'll need to configure Sidekiq and create a job that can be processed by the Go worker:

```ruby
# config/initializers/sidekiq.rb
Sidekiq.configure_client do |config|
  config.redis = { url: 'redis://localhost:6379/0' }
end

# app/jobs/go_processor_job.rb
class GoProcessorJob
  include Sidekiq::Worker
  
  # The queue name must match the one in Go
  sidekiq_options queue: 'ruby_jobs'

  def perform(name, message)
    # The method name must match the "class" field expected by go-workers2
    # The arguments will be automatically JSON encoded
  end
end

# To enqueue a job to be processed by Go:
GoProcessorJob.perform_async("Alice", "Hello from Ruby!")
```

## Part 3: Go Side - Enqueuing Jobs to Ruby

Now let's see how to enqueue jobs from Go to be processed by Ruby:

```go
func enqueueRubyJob(producer *workers.Producer) error {
    // Create the job arguments
    jobArgs := map[string]interface{}{
        "name": "Bob",
        "message": "Hello from Go!",
    }

    // Enqueue the job
    // The class name must match your Ruby job class
    err := producer.Enqueue("ruby_sidekiq_queue", "RubyProcessorJob", jobArgs)
    if err != nil {
        return fmt.Errorf("failed to enqueue job: %v", err)
    }

    return nil
}

// In your main function or wherever you need to enqueue:
producer := manager.Producer()
if err := enqueueRubyJob(producer); err != nil {
    log.Printf("Error enqueueing job: %v", err)
}
```

## Part 4: Ruby Side - Processing Go Jobs

Create a Ruby job class to process the jobs enqueued by Go:

```ruby
# app/jobs/ruby_processor_job.rb
class RubyProcessorJob
  include Sidekiq::Worker
  sidekiq_options queue: 'ruby_sidekiq_queue'

  def perform(args)
    name = args['name']
    message = args['message']
    
    Rails.logger.info "Received message from Go: #{name} says #{message}"
  end
end
```

## Important Concepts and Best Practices

1. **Redis Connection**
   - Both Go and Ruby must connect to the same Redis instance
   - Use the same database number in both languages
   - Consider using sentinel for production environments

2. **Job Arguments**
   - Always use JSON-serializable data types
   - Define clear structures for job arguments
   - Handle missing or malformed data gracefully

3. **Error Handling**
   - Implement proper error handling in both languages
   - Use retries for transient failures
   - Log errors appropriately

4. **Queues**
   - Use different queues for different types of jobs
   - Consider queue priorities
   - Monitor queue sizes and processing rates

5. **Monitoring**
   - Use the built-in stats server in go-workers2 (`:8080/stats`)
   - Monitor Redis memory usage
   - Set up proper logging in both languages

## Common Gotchas and Solutions

1. **Job Class Names**
   - Go: Use the exact Ruby class name when enqueueing
   - Ruby: The job class must exist and match the name used in Go

2. **Argument Serialization**
   - Go: Use `map[string]interface{}` for flexible argument structures
   - Ruby: Use simple data types that can be JSON serialized

3. **Redis Connection Pool**
   - Configure appropriate pool sizes based on your concurrency needs
   - Monitor for connection exhaustion

4. **Process Management**
   - Use proper process supervision (systemd, supervisord, etc.)
   - Implement graceful shutdown handlers

This setup allows for seamless bidirectional job processing between Go and Ruby applications, leveraging the power of Sidekiq's reliable queueing system and the performance of Go workers.

