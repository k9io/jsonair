#!/bin/bash

export RUNAS="nobody"

export MYSQL_USERNAME="sql_user_name"
export MYSQL_PASSWORD="sql_password"
export MYSQL_HOST="127.0.0.1"
export MYSQL_PORT=3306
export MYSQL_DATABASE="jsonair"
export MYSQL_TABLE="configurations"
export MYSQL_TLS=false

export HTTP_TLS=false
export HTTP_LISTEN=":9191"
export HTTP_CERT="/etc/letsencrypt/live/YOURSITE/fullchain.pem"
export HTTP_KEY="/etc/letsencrypt/live/YOURSITE/privkey.pem"
export HTTP_MODE="release"


./jsonair


