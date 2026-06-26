package apiclient

import "crypto/tls"

// GetTLSConfig returns a TLS configuration with certificate verification disabled
func GetTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}
