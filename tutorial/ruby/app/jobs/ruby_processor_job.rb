require 'sidekiq'

class RubyProcessorJob
  include Sidekiq::Worker
  sidekiq_options queue: 'ruby_sidekiq_queue'

  def perform(name, message)
    # Log received parameters
    logger.info 'Received message from Go worker...'
    logger.info "Parameters: name=#{name.inspect}, message=#{message.inspect}"

    # Process the message
    logger.info "Processing message from Go: #{name} says #{message}"

    # Log completion
    logger.info 'Finished processing message from Go worker'
  end
end
