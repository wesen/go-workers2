require 'sidekiq'
require 'sidekiq/api'
require_relative 'config/initializers/sidekiq'
require_relative 'app/jobs/go_processor_job'
require_relative 'app/jobs/ruby_processor_job'

def monitor_queues
  puts "\nMonitoring Sidekiq queues..."
  puts "Available queues: #{Sidekiq::Queue.all.map(&:name).inspect}"
  puts 'Queue sizes:'
  Sidekiq::Queue.all.each do |queue|
    puts "  #{queue.name}: #{queue.size} jobs"
    next unless queue.size > 0

    puts "  Contents of #{queue.name}:"
    queue.each do |job|
      puts "    - Class: #{job.klass}, Args: #{job.args.inspect}"
    end
  end
  puts "\nRetry set size: #{Sidekiq::RetrySet.new.size}"
  puts "Dead set size: #{Sidekiq::DeadSet.new.size}"
end

# Monitor queues before sending job
monitor_queues

# Example of sending a job to Go
GoProcessorJob.perform_async('Alice', 'Hello from Ruby!')

# Monitor queues after sending job
puts "\nAfter enqueueing job:"
monitor_queues

puts "\nJob enqueued to Go worker. Check the Go worker's output."
puts 'This Ruby process will also process jobs from Go when running with Sidekiq.'
puts 'To start processing jobs from Go, run:'
puts '  bundle exec sidekiq -r ./app.rb -v -q ruby_sidekiq -q ruby_jobs'
