require_relative 'boot'

require 'rails/all'

# Require the gems listed in Gemfile, including any gems
# you've limited to :test, :development, or :production.
Bundler.require(*Rails.groups)

module DefaultApp
  class Application < Rails::Application
    # Initialize configuration defaults for originally generated Rails version.
    config.load_defaults 6.0

    # Settings in config/environments/* take precedence over those specified here.
    # Application configuration can go into files in config/initializers
    # -- all .rb files in that directory are automatically loaded after loading
    # the framework and any gems in your application.
    config.secret_key_base = "69b48fcf7457a5481541cc46e948f4efa22e83807868f91b1c27e65eac6df451ebf43ad8f798d704e2d7691c8351273a05bad9ac230bdee350f2745a9d9e63d0"
  end
end
