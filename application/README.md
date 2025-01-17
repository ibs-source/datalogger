# Producer

This document describes the main components and interfaces involved in the producer logic and how they interact with the rest of the application. It also includes documentation for the `certificate` package, which generates self-signed certificates for TLS/HTTPS usage.

---

## `func SetupSignalHandling(application *Main, connector Connector)`

Sets up signal handling for graceful shutdown of the application. Listens for SIGINT and SIGTERM signals and initiates shutdown procedures.

- **Parameters:**
  - `application` — A pointer to the Main application structure.
  - `connector` — The connector to be managed during shutdown.

## `func ParseHostFromEndpointURL(endpointURL string) (string, error)`

Parses an endpoint URL and extracts the hostname.

- **Parameters:**

  - `endpointURL`: A string representing the URL endpoint (e.g., `"https://example.com:443"`).

- **Returns:**
  - `string`: The extracted hostname (e.g., `example.com`).
  - `error`: An error if the URL cannot be parsed.

## `type Connector interface`

Defines methods for setting up, starting, stopping, and checking the status of a connector.

- **Methods:**
  - `Setup(globalconfig *global.GlobalConfiguration) error`  
    Sets up the connector with the given global configuration. Returns an error if setup fails.
  - `Start() error`  
    Starts the connector. Returns an error if the connector fails to start.
  - `Status() Status`  
    Retrieves the status of the connector.
  - `Stop() error`  
    Stops the connector. Returns an error if the connector fails to stop.
  - `CreateValidKeys() map[string]interface{}`  
    Creates a map of valid keys from the given configurations.

## `type Status`

Represents the overall status of the connector, including the status of each endpoint.

- **Fields:**
  - `Connector []StatusEndpoint`  
    List of endpoint statuses.

## `type StatusEndpoint`

Represents the status of an individual endpoint.

- **Fields:**
  - `Endpoint string`  
    The endpoint identifier.
  - `Status bool`  
    The status of the endpoint.

## `type Main`

Contains the main components of the application, including context, connectors, logger, and Redis client.

- **Fields:**
  - `Context context.Context`  
    Application context.
  - `Cancel context.CancelFunc`  
    Function to cancel the application context.
  - `Connector Connector`  
    The connector instance.
  - `Logger *logrus.Logger`  
    Logger for logging application events.
  - `Redis *redis.Client`  
    Redis client for data handling.

# Certificate Generation

The `producer` package (formerly referred to as `certificate`) provides functionality for generating and handling self-signed X.509 certificates for both RSA and ECDSA keys, as well as several utility functions for decoding and parsing certificates and keys. Below is an overview of the main functions.

## `func GenerateCertificate(host string, rsaBits int, validFor time.Duration) (certPEM, keyPEM []byte, err error)`

Creates a self-signed X.509 certificate (not a CA) using an RSA key. By default, it generates a 2048-bit RSA key if `rsaBits` is `0`. The resulting certificate is valid for the duration specified by `validFor`.

- **Parameters:**

  - `host`: A comma-separated list of hostnames, IP addresses, or URIs to include in the Subject Alternative Name (SAN).
  - `rsaBits`: The size of the RSA key in bits. If `0`, a 2048-bit key is used.
  - `validFor`: The duration for which the certificate is valid.

- **Returns:**
  - `certPEM`: The generated certificate in PEM format.
  - `keyPEM`: The generated private key in PEM format.
  - `err`: An error if certificate or key generation fails.

**Example Usage**

```go
certPEM, keyPEM, err := GenerateCertificate("localhost,127.0.0.1", 2048, 24*time.Hour)
if err != nil {
    log.Fatalf("Error generating certificate: %v", err)
}
```

## `func GenerateECDSACertificate(host string, curve elliptic.Curve, validFor time.Duration) (certPEM, keyPEM []byte, err error)`

Creates a self-signed X.509 certificate (not a CA) using an ECDSA private key on the specified elliptic curve (for example, `elliptic.P256()`). The certificate is valid for the duration specified by `validFor`.

- **Parameters:**

  - `host`: A comma-separated list of hostnames, IP addresses, or URIs to include in the SAN.
  - `curve`: The elliptic curve to use (e.g., `elliptic.P256()`).
  - `validFor`: The duration for which the certificate is valid.

- **Returns:**
  - `certPEM`: The generated certificate in PEM format.
  - `keyPEM`: The generated private key in PEM format.
  - `err`: An error if certificate or key generation fails.

**Example Usage**

```go
certPEM, keyPEM, err := GenerateECDSACertificate("example.com,10.0.0.1", elliptic.P256(), 48*time.Hour)
if err != nil {
    log.Fatalf("Error generating ECDSA certificate: %v", err)
}
```

## `func DecodeCertificate(pemData []byte) ([]byte, error)`

Decodes a PEM-encoded certificate and returns its raw DER bytes.

- **Parameters:**

  - `pemData`: The PEM-encoded certificate data.

- **Returns:**
  - `[]byte`: The raw certificate bytes in DER format.
  - `error`: An error if decoding fails or if the PEM block is not of type `"CERTIFICATE"`.

## `func DecodeRSAPrivateKey(pemData []byte) (*rsa.PrivateKey, error)`

Decodes a PEM-encoded RSA private key. It supports both PKCS#1 (`"RSA PRIVATE KEY"`) and PKCS#8 (`"PRIVATE KEY"`) formats.

- **Parameters:**

  - `pemData`: The PEM-encoded private key data.

- **Returns:**
  - `*rsa.PrivateKey`: The decoded RSA private key.
  - `error`: An error if decoding fails or if the key is not an RSA key.

## Internal Utility Functions

### `func extractPublicKey(priv interface{}) interface{}`

Extracts the public key portion from the given private key interface. Supports both RSA and ECDSA keys.

- **Parameters:**

  - `priv`: The private key (`*rsa.PrivateKey` or `*ecdsa.PrivateKey`).

- **Returns:**
  - The corresponding public key object, or `nil` if the key type is unsupported.

### `func pemBlockForPrivateKey(priv interface{}) (*pem.Block, error)`

Marshals the provided private key into a PEM block. Supports both RSA and ECDSA.

- **Parameters:**

  - `priv`: The private key (`*rsa.PrivateKey` or `*ecdsa.PrivateKey`).

- **Returns:**
  - `*pem.Block`: The corresponding PEM block containing the private key.
  - `error`: An error if the key type is unsupported or marshaling fails.

## Usage Example

```go
package main

import (
    "crypto/elliptic"
    "log"
    "time"

    "github.com/femogas/datalogger/application"
)

func main() {
    // Generate an RSA certificate
    rsaCert, rsaKey, err := producer.GenerateCertificate("localhost,127.0.0.1", 2048, time.Hour*24)
    if err != nil {
        log.Fatalf("Error generating RSA certificate: %v", err)
    }
    log.Printf("RSA Certificate:\n%s", rsaCert)
    log.Printf("RSA Private Key:\n%s", rsaKey)

    // Generate an ECDSA certificate
    ecdsaCert, ecdsaKey, err := producer.GenerateECDSACertificate("example.com", elliptic.P256(), time.Hour*24)
    if err != nil {
        log.Fatalf("Error generating ECDSA certificate: %v", err)
    }
    log.Printf("ECDSA Certificate:\n%s", ecdsaCert)
    log.Printf("ECDSA Private Key:\n%s", ecdsaKey)
}
```
