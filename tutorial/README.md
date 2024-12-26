# Go-Ruby Sidekiq Integration Tutorial

This tutorial demonstrates bidirectional job processing between Go and Ruby using Sidekiq and go-workers2.

## Prerequisites

- Go 1.19+
- Ruby 2.5+
- Redis server running on localhost:6379
- Bundler gem installed (`gem install bundler`)

## Directory Structure

```
tutorial/
├── go/
│   ├── go.mod
│   └── main.go
└── ruby/
    ├── Gemfile
    ├── app.rb
    ├── config/
    │   └── initializers/
    │       └── sidekiq.rb
    └── app/
        └── jobs/
            ├── go_processor_job.rb
            └── ruby_processor_job.rb
```

## Setup

1. Start Redis server:
```bash
redis-server
```

2. Set up the Go worker:
```bash
cd go
go mod tidy
go run main.go
```

3. Set up the Ruby worker (in a new terminal):
```bash
cd ruby
bundle install
bundle exec sidekiq -r ./app.rb
```

4. Send a test message from Ruby to Go (in a new terminal):
```bash
cd ruby
bundle exec ruby app.rb
```

## What to Expect

1. The Go worker will:
   - Listen for jobs on the "ruby_jobs" queue
   - Send a job to the "ruby_sidekiq_queue" queue
   - Display stats at http://localhost:8080/stats

2. The Ruby worker will:
   - Process jobs from the "ruby_sidekiq_queue" queue
   - Be able to send jobs to the "ruby_jobs" queue

3. You should see:
   - Go worker logging messages received from Ruby
   - Ruby worker logging messages received from Go
   - Stats available in the Sidekiq web UI and Go stats endpoint

## Troubleshooting

1. Make sure Redis is running and accessible
2. Check that the queue names match between Go and Ruby
3. Verify that the job class names match exactly
4. Check the logs for both Go and Ruby workers 