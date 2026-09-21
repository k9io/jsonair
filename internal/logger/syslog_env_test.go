package logger

import "testing"

func TestParseSyslogEnv(t *testing.T) {

	tests := []struct {
		host, proto         string
		wantHost, wantProto string
		wantErr             bool
	}{
		{"", "", "", "", false},
		{"", "tcp", "", "", false}, /* proto is ignored for local syslog */
		{"local", "", "", "", false},
		{"10.0.0.5:514", "", "10.0.0.5:514", "udp", false},
		{"10.0.0.5:514", "udp", "10.0.0.5:514", "udp", false},
		{"logs.internal:6514", "tcp", "logs.internal:6514", "tcp", false},
		{"[::1]:514", "tcp", "[::1]:514", "tcp", false},
		{"10.0.0.5", "", "", "", true},         /* missing port */
		{"10.0.0.5:514", "http", "", "", true}, /* bad proto */
		{"10.0.0.5:514", "UDP", "", "", true},  /* case sensitive, like net.Dial's callers expect lowercase */
	}

	for _, tt := range tests {

		host, proto, err := ParseSyslogEnv(tt.host, tt.proto)

		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSyslogEnv(%q, %q) error = %v, wantErr %v", tt.host, tt.proto, err, tt.wantErr)
			continue
		}

		if host != tt.wantHost || proto != tt.wantProto {
			t.Errorf("ParseSyslogEnv(%q, %q) = (%q, %q), want (%q, %q)", tt.host, tt.proto, host, proto, tt.wantHost, tt.wantProto)
		}
	}

}
