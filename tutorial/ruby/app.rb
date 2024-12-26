require_relative 'config/initializers/sidekiq'
require_relative 'app/jobs/go_processor_job'
require_relative 'app/jobs/ruby_processor_job'

# Example of sending a job to Go
GoProcessorJob.perform_async('Alice', 'Hello from Ruby!')

puts "Job enqueued to Go worker. Check the Go worker's output."
puts 'This Ruby process will also process jobs from Go when running with Sidekiq.'
puts 'To start processing jobs from Go, run: bundle exec sidekiq -r ./app.rb'
