package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// CertAndKey contains an x509 certificate (PEM format) and ECDSA private key (also PEM format)
type CertAndKey struct {
	Cert []byte
	Key  []byte
}

func GenerateGPTScriptCert() (CertAndKey, error) {
	return GenerateSelfSignedCert("gptscript server")
}

func GenerateSelfSignedCert(name string) (CertAndKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return CertAndKey{}, fmt.Errorf("failed to generate ECDSA key: %v", err)
	}

	marshalledPrivateKey, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return CertAndKey{}, fmt.Errorf("failed to marshal ECDSA key: %v", err)
	}

	marshalledPrivateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: marshalledPrivateKey})

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: name,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0), // a year from now
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		IsCA:        false,
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return CertAndKey{}, fmt.Errorf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})

	return CertAndKey{Cert: certPEM, Key: marshalledPrivateKeyPEM}, nil
}
