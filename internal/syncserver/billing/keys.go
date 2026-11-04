package billing

import (
	"crypto/rsa"
	"crypto/x509"
	_ "embed" // the Waffo webhook public keys are embedded below
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Waffo's webhook public keys (not secret): one fixed key pair per environment, shared by all
// stores. A test key cannot verify production events and the other way round.
var (
	//go:embed waffo-test.pub.pem
	waffoTestPub []byte
	//go:embed waffo-live.pub.pem
	waffoLivePub []byte
)

// Webhook modes, as the payload's "mode" and WAFFO_MODE spell them.
const (
	ModeTest = "test"
	ModeProd = "prod"
)

// embeddedWebhookKeys parses the committed public keys, by mode.
func embeddedWebhookKeys() (map[string]*rsa.PublicKey, error) {
	out := map[string]*rsa.PublicKey{}
	for mode, raw := range map[string][]byte{ModeTest: waffoTestPub, ModeProd: waffoLivePub} {
		key, err := parsePublicKey(raw)
		if err != nil {
			return nil, fmt.Errorf("billing: embedded %s webhook key: %w", mode, err)
		}
		out[mode] = key
	}
	return out, nil
}

func parsePublicKey(raw []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("not PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}
	return key, nil
}

// LoadPrivateKey reads the API key's RSA private key from a PEM file: PKCS#8 ("PRIVATE KEY"), or
// PKCS#1 ("RSA PRIVATE KEY"), which Waffo's docs name. The error never includes the file's content.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("billing: read the Waffo key file: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("billing: %s is not a PEM file", path)
	}
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		key, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("billing: %s is not an RSA key", path)
		}
		return key, nil
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("billing: %s is not a PKCS#8 or PKCS#1 RSA private key", path)
	}
	return key, nil
}
