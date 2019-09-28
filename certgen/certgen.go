// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package certgen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path"
	"time"
)

func GenerateCertificate(dir string, cn string) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	privFile, err := os.Create(path.Join(dir, cn+".key"))
	if err != nil {
		return err
	}
	err = pem.Encode(privFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	if err != nil {
		return err
	}
	privFile.Close()

	maxSerial := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, maxSerial)
	if err != nil {
		return err
	}
	cert := x509.Certificate{
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 3650),
		Subject: pkix.Name{
			CommonName: cn,
		},
		Issuer: pkix.Name{
			CommonName: cn,
		},

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	bytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, priv.Public(), priv)
	if err != nil {
		return err
	}

	pubFile, err := os.Create(path.Join(dir, cn+".crt"))
	if err != nil {
		return err
	}
	err = pem.Encode(pubFile, &pem.Block{Type: "CERTIFICATE", Bytes: bytes})
	if err != nil {
		return err
	}
	pubFile.Close()
	return nil
}
