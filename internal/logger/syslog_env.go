/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

package logger

import (
	"fmt"
	"net"
)

// ParseSyslogEnv validates the SYSLOG_HOST and SYSLOG_PROTO environment
// variables and returns the values to pass to Init_Logger.
//
// An empty host or "local" means the local syslog daemon (the default), and
// proto is ignored. Otherwise host must be "address:port" of a remote syslog
// server, and proto must be "udp" (the default) or "tcp".
func ParseSyslogEnv(host string, proto string) (string, string, error) {

	if host == "" || host == "local" {
		return "", "", nil
	}

	if _, _, err := net.SplitHostPort(host); err != nil {
		return "", "", fmt.Errorf("SYSLOG_HOST must be 'local' or 'address:port' (for example '10.0.0.5:514'): %v", err)
	}

	if proto == "" {
		proto = "udp"
	}

	if proto != "udp" && proto != "tcp" {
		return "", "", fmt.Errorf("invalid SYSLOG_PROTO '%s'; valid values are 'udp' and 'tcp'", proto)
	}

	return host, proto, nil

}
