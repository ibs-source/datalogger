/**
 * Package producer provides functionality for generating X.509 certificates
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
 * generateRSAPrivateKey generates an RSA private key with the specified bit size.
 *
 * @param rsaBits    The number of bits for the RSA key.
 *
 * @return           A pointer to the generated RSA private key.
 * @return           An error if key generation fails.
 */
func generateRSAPrivateKey(rsaBits int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaBits)
}

/**
 * generateECDSAPrivateKey generates an ECDSA private key using the provided curve.
 *
 * @param curve      The elliptic curve to use for key generation.
 *
 * @return           A pointer to the generated ECDSA private key.
 * @return           An error if key generation fails.
 */
func generateECDSAPrivateKey(curve elliptic.Curve) (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(curve, rand.Reader)
}

/**
 * createCertificate builds and signs an X.509 certificate based on the provided
 * private key, host information, and validity duration.
 *
 * @param privateKey The private key to use for signing the certificate.
 * @param host       A comma-separated list of hostnames or IP addresses
 *                   included in the certificate's SAN.
 * @param validFor   The duration for which the certificate is valid.
 *
 * @return certPEM   The PEM-encoded certificate.
 * @return keyPEM    The PEM-encoded private key.
 * @return err       An error if the certificate creation fails.
 */
func createCertificate(privateKey interface{}, host string, validFor time.Duration) ([]byte, []byte, error) {
	template := buildCertificateTemplate(validFor)
	parseHostsIntoCertificate(host, &template)

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, extractPublicKey(privateKey), privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return encodeCertificateAndKey(derBytes, privateKey)
}

/**
 * buildCertificateTemplate constructs a default certificate template,
 * setting the NotBefore and NotAfter fields based on the validFor duration.
 *
 * @param validFor   The duration for which the certificate is valid.
 *
 * @return           An x509.Certificate with default settings for usage,
 *                   subject, and validity.
 */
func buildCertificateTemplate(validFor time.Duration) x509.Certificate {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return x509.Certificate{
		IsCA:                  false,
		SerialNumber:          serialNumber,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validFor),
		Subject:               pkix.Name{Organization: []string{"datalogger"}},
		KeyUsage:              x509.KeyUsageContentCommitment | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
}

/**
 * parseHostsIntoCertificate parses a comma-separated list of hosts and updates
 * the certificate template's DNSNames, IPAddresses, and URIs accordingly.
 *
 * @param host       A comma-separated list of hostnames or IP addresses.
 * @param template   A pointer to the x509.Certificate to update.
 */
func parseHostsIntoCertificate(host string, template *x509.Certificate) {
	for _, h := range strings.Split(host, ",") {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
		if parsedURL, err := url.Parse(h); err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
			template.URIs = append(template.URIs, parsedURL)
		}
	}
}

/**
 * encodeCertificateAndKey encodes the certificate and private key into PEM blocks.
 *
 * @param derBytes   The DER-encoded certificate bytes.
 * @param privateKey The private key corresponding to the certificate.
 *
 * @return certPEM   The PEM-encoded certificate.
 * @return keyPEM    The PEM-encoded private key.
 * @return err       An error if the key cannot be encoded.
 */
func encodeCertificateAndKey(derBytes []byte, privateKey interface{}) ([]byte, []byte, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyBlock, err := pemBlockForPrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}

	keyPEM := pem.EncodeToMemory(keyBlock)
	return certPEM, keyPEM, nil
}

/**
 * extractPublicKey extracts the public key from the given private key.
 * Supports both RSA and ECDSA private keys.
 *
 * @param privateKey The private key from which to extract the public key.
 *
 * @return           The corresponding public key, or nil if the key type is unsupported.
 */
func extractPublicKey(privateKey interface{}) interface{} {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return &key.PublicKey
	case *ecdsa.PrivateKey:
		return &key.PublicKey
	default:
		return nil
	}
}

/**
 * pemBlockForPrivateKey wraps the private key in a PEM block.
 * Supports both RSA and ECDSA private keys.
 *
 * @param privateKey The private key to wrap.
 *
 * @return           A *pem.Block containing the private key.
 * @return           An error if the key type is unsupported or cannot be marshaled.
 */
func pemBlockForPrivateKey(privateKey interface{}) (*pem.Block, error) {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}, nil
	case *ecdsa.PrivateKey:
		ecBytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		return &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: ecBytes,
		}, nil
	default:
		return nil, errors.New("unsupported private key type")
	}
}

/**
 * GenerateCertificate creates an X.509 certificate and its corresponding
 * RSA private key. If rsaBits is 0, it defaults to 2048 bits.
 *
 * @param host       A comma-separated list of hostnames or IP addresses
 *                   included in the certificate's SAN (Subject Alternative Name).
 * @param rsaBits    The number of bits for the RSA key. If 0, defaults to 2048.
 * @param validFor   The duration for which the certificate is valid.
 *
 * @return certPEM   The PEM-encoded certificate.
 * @return keyPEM    The PEM-encoded private key.
 * @return err       An error if the certificate or private key generation fails.
 */
func GenerateCertificate(host string, rsaBits int, validFor time.Duration) ([]byte, []byte, error) {
	if len(host) == 0 {
		return nil, nil, errors.New("missing required host parameter")
	}
	if rsaBits == 0 {
		rsaBits = 2048
	}

	privateKey, err := generateRSAPrivateKey(rsaBits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return createCertificate(privateKey, host, validFor)
}

/**
 * GenerateECDSACertificate creates an X.509 certificate and its corresponding
 * ECDSA private key.
 *
 * @param host       A comma-separated list of hostnames or IP addresses
 *                   included in the certificate's SAN (Subject Alternative Name).
 * @param curve      The elliptic curve to use (e.g., elliptic.P256()).
 * @param validFor   The duration for which the certificate is valid.
 *
 * @return certPEM   The PEM-encoded certificate.
 * @return keyPEM    The PEM-encoded private key.
 * @return err       An error if the certificate or private key generation fails.
 */
func GenerateECDSACertificate(host string, curve elliptic.Curve, validFor time.Duration) ([]byte, []byte, error) {
	if len(host) == 0 {
		return nil, nil, errors.New("missing required host parameter")
	}

	privateKey, err := generateECDSAPrivateKey(curve)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	return createCertificate(privateKey, host, validFor)
}

/**
 * DecodeCertificate decodes a PEM-encoded certificate and returns its raw bytes.
 *
 * @param pemData    The PEM-encoded certificate data.
 *
 * @return []byte    The raw certificate bytes.
 * @return error     An error if decoding fails or the block type is not "CERTIFICATE".
 */
func DecodeCertificate(pemData []byte) ([]byte, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for certificate")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected PEM block type: got %s, expected CERTIFICATE", block.Type)
	}
	return block.Bytes, nil
}

/**
 * DecodeRSAPrivateKey decodes a PEM-encoded RSA private key. It supports both
 * PKCS#1 ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE KEY") formats.
 *
 * @param pemData       The PEM-encoded private key data.
 *
 * @return *rsa.Key     The decoded RSA private key.
 * @return error        An error if decoding fails or the key is not RSA.
 */
func DecodeRSAPrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for RSA private key")
	}

	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS#8: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil

	case "RSA PRIVATE KEY":
		rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS#1: %w", err)
		}
		return rsaKey, nil

	default:
		return nil, fmt.Errorf("unsupported key type %s", block.Type)
	}
}
