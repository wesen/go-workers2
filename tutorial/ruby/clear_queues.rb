require_relative 'config/initializers/sidekiq'
require 'sidekiq/api'

puts 'Clearing all Sidekiq queues...'

# Clear specific queues
Sidekiq::Queue.new('ruby_sidekiq').clear
Sidekiq::Queue.new('ruby_jobs').clear

# Clear retry set
Sidekiq::RetrySet.new.clear

# Clear scheduled jobs
Sidekiq::ScheduledSet.new.clear

# Clear dead jobs
Sidekiq::DeadSet.new.clear

puts 'All queues cleared!'
