require 'sidekiq'

class GoProcessorJob
  include Sidekiq::Worker

  # The queue name must match the one in Go
  sidekiq_options queue: 'ruby_jobs'

  def perform(name, message)
    # Log before sending
    logger.info 'Preparing to send message to Go worker...'
    logger.info "Parameters: name=#{name.inspect}, message=#{message.inspect}"

    # The arguments will be automatically JSON encoded and sent to Go
    logger.info "Sending message to Go: #{name} says #{message}"

    # Log after sending
    logger.info 'Message sent to Go worker queue'
  end
end
