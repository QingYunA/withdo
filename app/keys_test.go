package main

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestDecodeLegacySecret(t *testing.T) {
	const key = "sk-test-ключ\nline2"
	cases := []struct {
		name, raw, want string
		legacy, wantErr bool
	}{
		{"plain", "sk-abc", "sk-abc", false, false},
		{"plain with newline", "sk-abc\n", "sk-abc", false, false},
		{"base64", legacyBase64Prefix + base64.StdEncoding.EncodeToString([]byte(key)), key, true, false},
		{"hex", legacyHexPrefix + hex.EncodeToString([]byte(key)), key, true, false},
		{"empty base64", legacyBase64Prefix, "", true, false},
		{"bad base64", legacyBase64Prefix + "!!!", "", true, true},
		{"bad hex", legacyHexPrefix + "zz", "", true, true},
	}
	for _, c := range cases {
		got, legacy, err := decodeLegacySecret(c.raw)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, want error %v", c.name, err, c.wantErr)
			continue
		}
		if !c.wantErr && (got != c.want || legacy != c.legacy) {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", c.name, got, legacy, c.want, c.legacy)
		}
		if c.wantErr && !legacy {
			t.Errorf("%s: a decode error should still say legacy", c.name)
		}
	}
}
