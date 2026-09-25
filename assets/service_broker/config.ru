$: << File.expand_path("../.", __FILE__)

require "service_broker"
require "stringio"

# Rack 3 no longer provides a rewindable rack.input. Read and cache the raw
# request body before any other middleware (e.g. rack-protection) touches it,
# then replace rack.input with a fresh StringIO so downstream code can read
# the body as many times as it likes.
use Rack::Builder do
  use(Class.new do
    def initialize(app)
      @app = app
    end

    def call(env)
      raw = env["rack.input"].read
      env["rack.input"] = StringIO.new(raw)
      env["rack.body_cache"] = raw
      @app.call(env)
    end
  end)
end

run ServiceBroker
