# Rails 6.1 Example

This example was upgraded from a Rails 6.0 fixture using the following steps adapted from the [Rails upgrade guide](https://guides.rubyonrails.org/upgrading_ruby_on_rails.html):

1. Bump `rails` gem to `~> 6.1.0` in the `Gemfile` and run `bundle update`
2. Bump `webpacker` gem to `~> 6.0.0.rc.6` in the `Gemfile` and run `bundle update`
3. Run `bundle exec rails app:update`
4. Run `bundle exec rails webpacker:install`
5. Follow the directions in the [Upgrading from Webpacker 5 to 6 docs](https://github.com/rails/webpacker/blob/master/docs/v6_upgrade.md)
