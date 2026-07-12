#!/usr/bin/with-contenv bashio
# Entrypoint for the GrowRig add-on. Generates a Supervisor-mode Grow Core
# config from the add-on options, then hands off to the binary as PID 1.
set -e

CONTROL_INTERVAL="$(bashio::config 'control_interval')"

bashio::log.info "Starting Grow Core (control interval: ${CONTROL_INTERVAL})"

# Grow Core expands ${SUPERVISOR_TOKEN} at load, so the token stays out of the
# file. /data is the add-on's persistent volume. The escaped $ keeps the token
# reference literal here — Grow Core resolves it, not bash.
cat > /tmp/growcore.yaml <<EOF
server:
  addr: ":8080"
storage:
  path: /data/growcore.db
control:
  interval: ${CONTROL_INTERVAL}
adapter:
  type: homeassistant
homeassistant:
  url: http://supervisor/core
  token: \${SUPERVISOR_TOKEN}
EOF

exec /usr/bin/growcore -config /tmp/growcore.yaml
