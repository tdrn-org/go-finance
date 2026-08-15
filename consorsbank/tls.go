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

const skipVerify bool = true

func TLSSkipVerifiy() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: skipVerify,
	}
}

func TLSVerifyRoot(file string) (*tls.Config, error) {
	return nil, nil
}

func TLSRootFromPEM(data []byte) (*tls.Config, error) {
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" || len(rest) > 0 {
		return nil, fmt.Errorf("invald certificate data")
	}
	root, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate (cause: %w)", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: skipVerify,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			rawCertsLen := len(rawCerts)
			if rawCertsLen != 1 {
				return fmt.Errorf("unexpected raw certificate count %d", rawCertsLen)
			}
			peerCert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse peer certificate (cause: %w)", err)
			}
			verifyOpts := &x509.VerifyOptions{
				Roots: roots,
			}
			_, err = peerCert.Verify(*verifyOpts)
			if err != nil {
				return fmt.Errorf("failed to verifiy peer certificate (cause: %w)", err)
			}
			return nil
		},
	}
	return tlsConfig, nil
}

func TLSRootFromFile(file string) (*tls.Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file '%s' (cause: %w)", file, err)
	}
	return TLSRootFromPEM(data)
}
