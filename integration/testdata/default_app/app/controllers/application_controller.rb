class ApplicationController < ActionController::Base
  def index
    render text: 'Hello World', content_type: 'text/plain'
  end
end
