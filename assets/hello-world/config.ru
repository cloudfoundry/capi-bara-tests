app = lambda do |env|
  body = "Hello, world!"

  [ 200,
    { "content-type" => "text/plain",
      "content-length" => body.length.to_s
    },
    [body]
  ]
end

run app
