require 'sidekiq'
require 'sidekiq/api'

puts 'Configuring Sidekiq...'

# Explicitly set the queues to process
Sidekiq.options[:queues] = %w[ruby_sidekiq]

Sidekiq.configure_server do |config|
  config.redis = { url: 'redis://localhost:6379/0' }
  puts "Sidekiq server configured with queues: #{Sidekiq[:queues].inspect}"
end

Sidekiq.configure_client do |config|
  config.redis = { url: 'redis://localhost:6379/0' }
end

puts "Sidekiq queues configured: #{Sidekiq[:queues].inspect}"
