package configdata

import (
	"encoding/base64"
	"errors"
	"testing"

	cry "github.com/k9io/jsonair/internal/crypto"
)

func TestValidate(t *testing.T) {

	tests := []struct {
		name    string
		format  string
		data    string
		wantErr bool
	}{
		{"json ok", "json", `{"a":1}`, false},
		{"json bad", "json", `{"a":`, true},
		{"json plain text", "json", `hello`, true},
		{"xml ok", "xml", `<a><b>1</b></a>`, false},
		{"xml unclosed", "xml", `<a><b>1</a>`, true},
		{"xml plain text", "xml", `just text`, true},
		{"yaml ok", "yaml", "a: 1\nb:\n  - x\n", false},
		{"yaml bad", "yaml", "a: [1, 2\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.format, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}

}

func TestValidateUnknownFormat(t *testing.T) {

	for _, f := range []string{"", "toml", "JSON"} {
		if err := Validate(f, []byte("x")); !errors.Is(err, ErrUnknownFormat) {
			t.Errorf("Validate(%q) = %v, want ErrUnknownFormat", f, err)
		}
	}

}

func TestEncodeRoundTrip(t *testing.T) {

	key := cry.DeriveKey([]byte("test-secret"))
	raw := "line one\nline two: {\"k\":\"v\"}\n"

	enc, err := Encode(raw, key)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	/* The stored value is encrypt(base64(raw)) — this is what the read API decrypts. */

	dec, err := cry.Decrypt(enc, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	got, err := base64.StdEncoding.DecodeString(string(dec))
	if err != nil {
		t.Fatalf("base64: %v", err)
	}

	if string(got) != raw {
		t.Errorf("round trip = %q, want %q", got, raw)
	}

}
