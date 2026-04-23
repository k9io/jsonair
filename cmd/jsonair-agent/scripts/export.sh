#!/bin/bash

export JSONAIR_PAT="ACCESS_TOKEN"
export JSONAIR_URL="https://example.com"
export JSONAIR_TYPE="jsonair"
export JSONAIR_NAME="suricata.config"

export RUNAS="nobody"
export CONFIG_FILE="/etc/suricata/suricata.yaml"
export PRUNE=5
export MASK=0644
export SLEEP=300


./jsonair-agent 


