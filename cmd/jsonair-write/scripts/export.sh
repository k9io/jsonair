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


export RUNAS="nobody"		# What jsonair-write should "drop privs" to.

export MYSQL_USERNAME="MYSQL_WRITE_USERNAME"
export MYSQL_PASSWORD="MYSQL_SECURE_PASSWORD"
export MYSQL_HOST="127.0.0.1"
export MYSQL_PORT=3306
export MYSQL_DATABASE="jsonair"
export MYSQL_TLS=false

# Optional: send logs to a remote syslog server ("local" is the default).
#export SYSLOG_HOST="10.0.0.5:514"
#export SYSLOG_PROTO="udp"

# jsonair-write should only ever listen on an internal/restricted interface.

export HTTP_TLS=true
export HTTP_LISTEN="10.0.0.5:9192"
export HTTP_CERT="/etc/jsonair/write-server.pem"
export HTTP_KEY="/etc/jsonair/write-server.key"
export HTTP_CLIENT_CA="/etc/jsonair/clients-ca.pem"	# Optional. If set, callers must present a certificate signed by this CA (mTLS).
export HTTP_MODE="production"

# Note:  WRITE_JWT_TOKEN_SECRET, WRITE_TOKEN_HMAC_SECRET and CONFIG_ENCRYPT_SECRET
# must be _separately generated_ strings (at least 32 characters).  They should
# _never_ share values.  These names differ from jsonair's on purpose: do not
# reuse jsonair's JWT_TOKEN_SECRET or TOKEN_HMAC_SECRET here.
#
# CONFIG_ENCRYPT_SECRET is the exception to "different from jsonair": it must be
# the _same_ value jsonair uses, or the read API cannot decrypt what is written.

export WRITE_JWT_TOKEN_SECRET="REALLYLONGSTRING"	# Generate with `openssl rand -hex 32`
export WRITE_JWT_TOKEN_EXPIRE=15

export WRITE_TOKEN_HMAC_SECRET="REALLYLONGSTRING"	# Generate with `openssl rand -hex 32`
export CONFIG_ENCRYPT_SECRET="SAME_AS_JSONAIR"

# Execute jsonair-write.

./jsonair-write
