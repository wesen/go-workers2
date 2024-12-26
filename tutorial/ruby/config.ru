require 'sidekiq'
require 'sidekiq/web'

require_relative 'config/initializers/sidekiq'

# Optional: Add basic auth
# Sidekiq::Web.use(Rack::Auth::Basic) do |user, password|
#   [user, password] == ["admin", "password"]
# end

run Sidekiq::Web
