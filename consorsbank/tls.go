/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consorsbank

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// TLSSkipVerify returns a TLS configuration that disables certificate
// verification entirely. It is intended only for endpoints whose identity
// cannot be pinned (e.g. an ephemeral self-signed certificate).
func TLSSkipVerify() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 -- intentionally insecure, see doc comment
	}
}

// TLSCAFromPEM returns a TLS configuration that pins the connection to the
// single certificate provided in PEM format.
//
// InsecureSkipVerify is enabled so that the self-signed peer certificate is
// validated by VerifyPeerCertificate instead of the standard PKI chain (which
// would reject it as an unknown authority before the callback ever runs).
func TLSCAFromPEM(data []byte) (*tls.Config, error) {
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" || len(rest) > 0 {
		return nil, fmt.Errorf("invalid certificate data")
	}
	root, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate (cause: %w)", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	return &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 -- self-signed cert is pinned via VerifyPeerCertificate below
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) != 1 {
				return fmt.Errorf("unexpected raw certificate count %d", len(rawCerts))
			}
			peerCert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse peer certificate (cause: %w)", err)
			}
			verifyOpts := &x509.VerifyOptions{
				Roots: roots,
			}
			if _, err := peerCert.Verify(*verifyOpts); err != nil {
				return fmt.Errorf("failed to verify peer certificate (cause: %w)", err)
			}
			return nil
		},
	}, nil
}

func TLSCAFromFile(file string) (*tls.Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file '%s' (cause: %w)", file, err)
	}
	return TLSCAFromPEM(data)
}
