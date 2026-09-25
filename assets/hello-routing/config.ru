require 'json'

app = lambda do |env|
  json = ENV['VCAP_APPLICATION']
 
  vcapApp = JSON.parse(json)

  body = "Hello, " + vcapApp['name'] + " at index: " + ENV['CF_INSTANCE_INDEX'] + "!"

  # log headers
  puts JSON.pretty_generate(env)
  $stdout.flush

  [ 200,
    { "content-type" => "text/plain",
      "content-length" => body.length.to_s,
      "set-cookie" => "JSESSIONID=12345",
    },
    [body]
  ]
end

run app
