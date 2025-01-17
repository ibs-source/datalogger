/**
 * Package certificate provides functionality for generating X.509 certificates
 * and corresponding private keys, with support for both RSA and ECDSA.
 */
package producer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strings"
	"time"
)

/**
 * GenerateCert creates a self-signed certificate (not a CA) and a private key for the provided hosts.
 * By default, it generates an RSA key if rsaBits > 0. The certificate is valid for the duration specified in validFor.
 *
 * The certificate includes DNS/IP and URI Subject Alternative Names (SANs) based on the provided hosts.
 *
 * @param host     A comma-separated list of hosts (DNS names, IP addresses, or URIs).
 * @param rsaBits  The size of the RSA key in bits. If 0, an RSA 2048-bit key is used by default.
 * @param validFor The validity period of the certificate.
 * @return The generated certificate (PEM), the generated key (PEM), and an error if any occurred.
 */
func GenerateCert(host string, rsaBits int, validFor time.Duration) (certPEM, keyPEM []byte, err error) {
	if len(host) == 0 {
		return nil, nil, errors.New("missing required host parameter")
	}
	if rsaBits == 0 {
		rsaBits = 2048
	}

	// Generate an RSA private key by default.
	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// This certificate is intended for server and client authentication, but it's NOT a CA.
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Datalogger"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		// Typical usage for a leaf certificate (not a CA).
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Parse the comma-separated hosts.
	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
		// Attempt to parse h as a URI; only valid URIs will be appended.
		if parsedURI, uriErr := url.Parse(h); uriErr == nil && parsedURI.Scheme != "" && parsedURI.Host != "" {
			template.URIs = append(template.URIs, parsedURI)
		}
	}

	// Create the certificate.
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Convert to PEM format.
	certPEMBlock := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	certPEM = pem.EncodeToMemory(certPEMBlock)

	keyPEMBlock, err := pemBlockForKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(keyPEMBlock)

	return certPEM, keyPEM, nil
}

/**
 * GenerateECDSACert creates an ECDSA self-signed certificate and a private key for the provided hosts.
 * The certificate is valid for the specified duration. Use this function if ECDSA keys are required.
 *
 * @param host     A comma-separated list of hosts (DNS names, IP addresses, or URIs).
 * @param curve    The elliptic curve to use (e.g., elliptic.P256()).
 * @param validFor The validity period of the certificate.
 * @return The generated certificate (PEM), the generated key (PEM), and an error if any occurred.
 */
func GenerateECDSACert(host string, curve elliptic.Curve, validFor time.Duration) (certPEM, keyPEM []byte, err error) {
	if len(host) == 0 {
		return nil, nil, errors.New("missing required host parameter")
	}

	// Generate an ECDSA private key with the specified curve.
	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDSA private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Datalogger"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
		if parsedURI, uriErr := url.Parse(h); uriErr == nil && parsedURI.Scheme != "" && parsedURI.Host != "" {
			template.URIs = append(template.URIs, parsedURI)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ECDSA certificate: %w", err)
	}

	certPEMBlock := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	certPEM = pem.EncodeToMemory(certPEMBlock)

	keyPEMBlock, err := pemBlockForKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal ECDSA private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(keyPEMBlock)

	return certPEM, keyPEM, nil
}

/**
 * publicKey extracts the public key portion from the given private key object.
 *
 * @param priv The private key interface (RSA or ECDSA).
 * @return The corresponding public key.
 */
func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

/**
 * pemBlockForKey marshals the private key into a PEM block. Supports RSA and ECDSA.
 *
 * @param priv The private key interface (RSA or ECDSA).
 * @return The corresponding PEM block and an error if any occurred.
 */
func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		// RSA in PKCS#1 format
		return &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}, nil
	case *ecdsa.PrivateKey:
		// ECDSA in SEC 1 format
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		return &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: b,
		}, nil
	default:
		return nil, errors.New("unsupported private key type")
	}
}
