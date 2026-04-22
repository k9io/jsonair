#!/bin/bash

export RUNAS="nobody"

export JSONAIR_PAT="12560ace2aba2a52eae1cc3812b9ffdf94ae1246fc227171a6381752"
export JSONAIR_URL="https://key9.dev:9191"
export JSONAIR_TYPE="highvolt"
#export JSONAIR_NAME="aws-s3.config"
export JSONAIR_NAME="suricata.config"

export JSONAIR_CONFIG_URL="https://key9.dev:9191/api/v1/jsonair/config"
export JSONAIR_AUTH_URL="https://key9.dev:9191/api/v1/jsonair/auth/token"

export CONFIG_FILE="/tmp/something.json"
export SLEEP=300


./jsonair-agent 


