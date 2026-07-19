#!/bin/sh
set -e

# Start Go API on :8081
PORT=8081 /usr/local/bin/redpolitika &

# Start Next.js on port 3000
cd /nextjs
HOSTNAME=0.0.0.0 PORT=3000 node server.js &

# Start Caddy (foreground — Docker needs one foreground process)
exec caddy run --config /etc/caddy/Caddyfile
