#!/bin/bash

##
## Copyright (C) 2026 Key9, Inc <k9.io>
## Copyright (C) 2026 Champ Clark III <cclark@k9.io>
##
## This file is part of the JSONAir.
##
## This source code is licensed under the MIT license found in the
## LICENSE file in the root directory of this source tree.
##


export RUNAS="nobody"		# What JSONAir should "drop privs" to.

export MYSQL_USERNAME="MYSQL_READONLY_USERNAME"
export MYSQL_PASSWORD="MYSQL_SECURE_PASSWORD"
export MYSQL_HOST="127.0.0.1"
export MYSQL_PORT=3306
export MYSQL_DATABASE="jsonair"
export MYSQL_TLS=false

export HTTP_TLS=false
export HTTP_LISTEN=":9191"
export HTTP_CERT="/etc/letsencrypt/live/YOURSITE/fullchain.pem"
export HTTP_KEY="/etc/letsencrypt/live/YOURSITE/privkey.pem"
export HTTP_MODE="production"

# Note:  JWT_TOKEN_SECRET, TOKEN_HMAC_SECRET and CONFIG_ENCRYPT_SECRET should be 
# _separately generated_ strings.  They should _never_ share values. 

export JWT_TOKEN_SECRET="REALLYLONGSTRING"   # Generated with `openssl rand -hex 32`
export JWT_TOKEN_EXPIRE=15

export TOKEN_HMAC_SECRET="REALLYLONGSTRING"	 # Generate with `openssl rand -hex 32`
export CONFIG_ENCRYPT_SECRET="REALLYLONGSTRING"  # Generate with `openssl rand -hex 32`

# Execute jsonair. 

./jsonair


