# Producer

This document describes the main components and interfaces involved in the producer logic and how they interact with the rest of the application. It also includes documentation for the `certificate` package, which generates self-signed certificates for TLS/HTTPS usage.

## `func SetupSignalHandling(application *Main, connector Connector)`

Sets up signal handling for graceful shutdown of the application. Listens for SIGINT and SIGTERM signals and initiates shutdown procedures.

- **Parameters:**
  - `application` — A pointer to the Main application structure.
  - `connector` — The connector to be managed during shutdown.

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

---

# Certificate

The `certificate` package provides functionality for generating self-signed X.509 certificates for both RSA and ECDSA keys. Below is an overview of the main functions.

## `func GenerateCert(host string, rsaBits int, validFor time.Duration) (certPEM, keyPEM []byte, err error)`

Creates a self-signed X.509 certificate (not a CA) using an RSA key. By default, it generates a 2048-bit RSA key if `rsaBits` is `0`. The resulting certificate is valid for the time span specified by `validFor`.  
The function parses `host` (a comma-separated list of DNS names, IP addresses, or URIs) to create the Subject Alternative Names (SANs).

- **Parameters:**

  - `host` — A comma-separated list of hosts (DNS names, IP addresses, or URIs).
  - `rsaBits` — The size of the RSA key in bits. If `0`, a 2048-bit key is used.
  - `validFor` — The duration for which the certificate is valid.

- **Returns:**
  - `certPEM` — The generated certificate in PEM format.
  - `keyPEM` — The generated private key in PEM format.
  - `err` — Error, if any occurred during certificate generation.

## `func GenerateECDSACert(host string, curve elliptic.Curve, validFor time.Duration) (certPEM, keyPEM []byte, err error)`

Creates a self-signed X.509 certificate (not a CA) using an ECDSA key on the specified elliptic curve (e.g., `elliptic.P256()`). The certificate is valid for the duration specified by `validFor`.  
Like `GenerateCert`, it uses `host` to populate the certificate's SAN fields for DNS, IP, and URI.

- **Parameters:**

  - `host` — A comma-separated list of hosts (DNS names, IP addresses, or URIs).
  - `curve` — The elliptic curve (e.g., `elliptic.P256()`).
  - `validFor` — The duration for which the certificate is valid.

- **Returns:**
  - `certPEM` — The generated certificate in PEM format.
  - `keyPEM` — The generated private key in PEM format.
  - `err` — Error, if any occurred during certificate generation.

## `func publicKey(priv interface{}) interface{}`

Extracts the public key portion from the given private key interface.

- **Parameters:**

  - `priv` — The private key, which can be of type `*rsa.PrivateKey` or `*ecdsa.PrivateKey`.

- **Returns:**  
  The corresponding public key object.

## `func pemBlockForKey(priv interface{}) (*pem.Block, error)`

Marshals the provided private key into a PEM block. Supports both RSA and ECDSA.

- **Parameters:**

  - `priv` — The private key, which can be of type `*rsa.PrivateKey` or `*ecdsa.PrivateKey`.

- **Returns:**
  - `*pem.Block` — The corresponding PEM block containing the private key.
  - `error` — If the key type is unsupported or marshaling fails.
