# Bidirectional Job Processing Between Go and Ruby with Sidekiq

## Overview
This tutorial demonstrates how to create a bidirectional job processing system between Go and Ruby applications using Sidekiq. The system allows both languages to enqueue and process jobs, enabling seamless communication between Go and Ruby services. This is particularly useful in polyglot environments where you want to leverage both Go's performance and Ruby's rich ecosystem.

The system uses Redis as a message broker, with Sidekiq providing the job queue infrastructure. The Go side uses the go-workers2 library, which implements Sidekiq's protocol, allowing it to interact with Sidekiq queues directly. This enables Go services to both consume jobs from Ruby applications and enqueue jobs for Ruby workers to process.

This tutorial demonstrates how to:
1. Set up a Go worker that can process jobs from Ruby's Sidekiq
2. Set up Ruby to enqueue jobs to be processed by Go
3. Set up Go to enqueue jobs to be processed by Ruby
4. Handle job arguments and error cases in both directions
5. Implement proper shutdown handling and monitoring

## Prerequisites
- Go 1.19+
- Ruby 2.5+
- Redis server running (default: localhost:6379)
- Bundler gem installed (`gem install bundler`)

## Directory Structure
The project is organized into two main parts: the Go worker and the Ruby components. This separation keeps the concerns of each language isolated while allowing them to communicate through Redis.

```
tutorial/
├── go/
│   ├── go.mod
│   └── main.go
└── ruby/
    ├── Gemfile
    ├── app.rb
    ├── clear_queues.rb
    ├── config.ru
    ├── config/
    │   └── initializers/
    │       └── sidekiq.rb
    └── app/
        └── jobs/
            ├── go_processor_job.rb
            └── ruby_processor_job.rb
```

The Go side consists of a single worker application that both processes jobs from Ruby and sends jobs to Ruby. The Ruby side includes Sidekiq job classes, configuration, and utility scripts for monitoring and managing the queues.

## Part 1: Setting Up the Go Worker

The Go implementation serves as both a worker and a job producer. It uses go-workers2, a Sidekiq-compatible library that allows Go programs to interact with Sidekiq queues. The implementation includes proper shutdown handling using errgroup for goroutine management and context for cancellation.

Key components of the Go implementation:
- A worker that processes jobs from Ruby's "ruby_jobs" queue
- A job producer that sends jobs to Ruby's "ruby_sidekiq" queue
- A stats server for monitoring
- Graceful shutdown handling

The worker uses a manager pattern where a single Manager instance coordinates multiple components:
- Job processing workers
- Job producer
- Stats server
- Shutdown handling

Here's the essential Go implementation:

```go
// go.mod
module tutorial

go 1.19

require (
    github.com/digitalocean/go-workers2 v0.10.0
    golang.org/x/sync v0.3.0
)
```

```go
// main.go - Key components
func processRubyJob(msg *workers.Msg) error {
    args := msg.Args()
    name, err := args.GetIndex(0).String()
    if err != nil {
        return fmt.Errorf("failed to get name: %v", err)
    }
    message, err := args.GetIndex(1).String()
    if err != nil {
        return fmt.Errorf("failed to get message: %v", err)
    }
    log.Printf("Received message from Ruby: %s says %s\n", name, message)
    return nil
}

func enqueueRubyJob(producer *workers.Producer) error {
    name := "Bob"
    message := fmt.Sprintf("Hello from Go! Sent at %s", time.Now().Format(time.RFC3339))
    jid, err := producer.Enqueue("ruby_sidekiq", "RubyProcessorJob", []interface{}{name, message})
    if err != nil {
        return fmt.Errorf("failed to enqueue job: %v", err)
    }
    log.Printf("Successfully enqueued job to Ruby with JID: %s", jid)
    return nil
}

// Manager setup in main()
manager, err := workers.NewManager(workers.Options{
    ServerAddr: "localhost:6379",
    Database:   0,
    PoolSize:   30,
    ProcessID:  "1",
})
manager.AddWorker("ruby_jobs", 10, processRubyJob, middlewares...)
```

## Part 2: Setting Up the Ruby Side

The Ruby implementation uses Sidekiq, a robust background job processor. It's set up to both process jobs from Go and send jobs to Go. The implementation includes job classes, queue configuration, monitoring tools, and a web interface for observing the system.

Key components of the Ruby implementation:
- Sidekiq configuration for Redis connection
- Job classes for sending and receiving messages
- Queue monitoring utilities
- Web UI for system observation

The Ruby side is organized into several components:
- Sidekiq initializer for configuration
- Job classes in the app/jobs directory
- Utility scripts for monitoring and queue management
- Web interface for real-time monitoring

Essential Ruby implementation:

```ruby
# Gemfile
source 'https://rubygems.org'
gem 'sidekiq', '~> 6.5'
gem 'redis', '~> 4.8'
gem 'sinatra'  # For Sidekiq Web UI
gem 'rack'
```

```ruby
# config/initializers/sidekiq.rb
require 'sidekiq'
require 'sidekiq/api'

Sidekiq.options[:queues] = %w[ruby_sidekiq]

Sidekiq.configure_server do |config|
  config.redis = { url: 'redis://localhost:6379/0' }
end
```

```ruby
# app/jobs/ruby_processor_job.rb
class RubyProcessorJob
  include Sidekiq::Worker
  sidekiq_options queue: 'ruby_sidekiq', retry: true

  def perform(name, message)
    logger.info "Processing message from Go: #{name} says #{message}"
  rescue StandardError => e
    logger.error "Error processing message: #{e.message}"
    raise # Re-raise to trigger Sidekiq retry
  end
end
```

```ruby
# app/jobs/go_processor_job.rb
class GoProcessorJob
  include Sidekiq::Worker
  sidekiq_options queue: 'ruby_jobs'

  def perform(name, message)
    logger.info "Sending message to Go: #{name} says #{message}"
  end
end
```

```ruby
# Queue monitoring example
def monitor_queues
  puts "Available queues: #{Sidekiq::Queue.all.map(&:name).inspect}"
  Sidekiq::Queue.all.each do |queue|
    puts "  #{queue.name}: #{queue.size} jobs"
  end
end
```

## Running the System

The system operates with several processes working together:
1. Redis server acts as the message broker
2. Go worker processes jobs and sends new ones
3. Sidekiq worker processes Go's jobs
4. Optional web UI for monitoring

Each component plays a crucial role:
- Redis stores the job queues and manages message delivery
- Go worker continuously processes Ruby jobs and sends new ones
- Sidekiq worker handles jobs from Go
- Web UI provides visibility into the system's operation

```bash
# Start Redis
redis-server

# Start Go worker
cd go
go run main.go

# Start Sidekiq worker
cd ruby
bundle exec sidekiq -r ./app.rb -v -q ruby_sidekiq -q ruby_jobs

# Start Web UI (optional)
bundle exec rackup config.ru -p 3000

# Monitor queues
bundle exec ruby app.rb
```

## Monitoring

The system provides multiple ways to monitor its operation, each offering different insights:

1. **Sidekiq Web UI** provides a comprehensive view of the Ruby side:
   - Real-time queue monitoring
   - Job processing statistics
   - Retry and failure management
   - Historical data

2. **Go Stats API** offers metrics from the Go worker:
   - Current queue sizes
   - Processing statistics
   - Worker status
   - Error rates

3. **Redis CLI** allows direct inspection of the queues:
   - Raw queue contents
   - Queue lengths
   - Real-time monitoring
   - Debugging capabilities

## Shutdown Handling

The system implements graceful shutdown to ensure no jobs are lost when stopping components. The Go implementation uses several mechanisms working together:

- `errgroup` coordinates multiple goroutines
- `context` propagates cancellation signals
- `sync.Once` ensures clean manager shutdown
- Signal handling catches system signals

This ensures that:
- In-progress jobs complete
- No new jobs start during shutdown
- Resources are properly released
- Workers stop gracefully

## Error Handling

The system implements comprehensive error handling on both sides:

Both Go and Ruby implementations include:
- Automatic job retries with backoff
- Dead letter queues for failed jobs
- Detailed error logging
- Queue monitoring

This ensures that:
- Transient failures are handled automatically
- Persistent failures are captured for investigation
- System state is always observable
- No jobs are lost

## Best Practices

The system follows several best practices for reliable operation:

1. **Queue Names**
   - Consistent naming between languages
   - Clear documentation
   - Regular monitoring
   This ensures reliable message routing and system observability.

2. **Job Arguments**
   - Simple, serializable types
   - Strict validation
   - Error handling
   This prevents serialization issues and ensures reliable processing.

3. **Monitoring**
   - Multiple monitoring points
   - Real-time observation
   - Historical data
   This provides complete system visibility and aids in troubleshooting.

4. **Error Handling**
   - Automatic retries
   - Detailed logging
   - Dead letter queues
   This ensures reliable operation and aids in problem resolution.

