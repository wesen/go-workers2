require 'sidekiq'
require 'json'

puts 'Loading RubyProcessorJob...'
puts 'Queue name: ruby_sidekiq'

class RubyProcessorJob
  include Sidekiq::Worker

  # Make sure retry is enabled and set queue name
  sidekiq_options queue: 'ruby_sidekiq', retry: true

  def self.queue_name
    get_sidekiq_options['queue']
  end

  puts "RubyProcessorJob configured with queue: #{queue_name}"

  def perform(name, message)
    puts '=' * 50
    puts 'RubyProcessorJob#perform called'
    puts 'Arguments:'
    puts "  name: #{name.inspect}"
    puts "  message: #{message.inspect}"

    # Log received parameters with more detail
    logger.info '=' * 50
    logger.info 'Received message from Go worker...'
    logger.info "Job ID: #{jid}"
    logger.info "Queue: #{self.class.queue_name}"
    logger.info 'Parameters:'
    logger.info "  - name: #{name.inspect}"
    logger.info "  - message: #{message.inspect}"

    # Process the message
    logger.info "Processing message from Go: #{name} says #{message}"

    # Log completion
    logger.info 'Finished processing message from Go worker'
    logger.info '=' * 50
  rescue StandardError => e
    logger.error "Error processing message: #{e.message}"
    logger.error e.backtrace.join("\n")
    raise # Re-raise to trigger Sidekiq retry
  end
end
