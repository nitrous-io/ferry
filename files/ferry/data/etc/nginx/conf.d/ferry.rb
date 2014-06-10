#!/usr/bin/env ruby
require 'rubygems'
require 'erb'
require 'ostruct'

ferry_conf = <<-EOF
map $http_upgrade $connection_upgrade {
  default upgrade;
  ''      close;
}

upstream ferry {
  server 0.0.0.0:<%= listen_port %>;
}

server {
  listen 80 default_server;

  # TODO: enable this once we have custom TLS certs for boxhosts
  # listen 443 default_server ssl;

  <% (start_port..end_port).each do |port| %>
  listen <%= port %>;
  <% end %>

  # injected at build time
  # TODO: enable this once we have custom TLS certs for boxhosts
  # ssl_certificate /etc/ssl/certs/nitrouspro.crt;
  # ssl_certificate_key /etc/ssl/private/nitrouspro.key;

  location / {
    proxy_pass http://ferry;
    proxy_redirect   off;

    proxy_set_header    Host              $host:$server_port;
    proxy_set_header    X-Real-IP         $remote_addr;
    proxy_set_header    X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_http_version  1.1;
    proxy_set_header    Upgrade $http_upgrade;
    proxy_set_header    Connection $connection_upgrade;
    proxy_pass_header   Server;
  }
}
EOF

class Ferry < OpenStruct
  def render(template)
    ERB.new(template).result(binding)
  end
end

et = Ferry.new({
  listen_port: 65343,
  start_port: 1024,
  end_port: 9999,
})

File.open(ARGV[0], "w") do |f|
  f.write et.render(ferry_conf)
end
