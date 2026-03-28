#!/bin/bash
# Fake agentledger binary for demo recording
# Simulates realistic output without needing real API keys

DIR="$(cd "$(dirname "$0")" && pwd)"

case "$1" in
  serve)
    bash "$DIR/demo.sh" start
    # Keep running until Ctrl+C
    trap 'exit 0' INT
    while true; do sleep 1; done
    ;;
  costs)
    if [[ "$*" == *"--by agent"* ]]; then
      bash "$DIR/demo.sh" costs-agent
    else
      bash "$DIR/demo.sh" costs
    fi
    ;;
  --version)
    echo "agentledger 0.1.0"
    ;;
esac
