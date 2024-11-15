# Producer

## `func SetupSignalHandling(app *Main, connector Connector)`

Sets up signal handling for graceful shutdown of the application. Listens for SIGINT and SIGTERM signals and initiates shutdown procedures.

 * **Parameters:**
   * `app` — A pointer to the Main application structure.
   * `connector` — The connector to be managed during shutdown.

## `type Connector interface`

Defines methods for setting up, starting, stopping, and checking the status of a connector.

 * **Methods:**
   * `Setup(globalconfig *global.GlobalConfiguration) error` — Sets up the connector with the given global configuration.
   * `Start() error` — Starts the connector.
   * `Status() Status` — Retrieves the status of the connector.
   * `Stop() error` — Stops the connector.
   * `CreateValidKeys() map[string]interface{}` — Creates a map of valid keys from the given configurations.

## `type Status`

Represents the overall status of the connector, including the status of each endpoint.

 * **Fields:**
   * `Connector []StatusEndpoint` — List of endpoint statuses.

## `type StatusEndpoint`

Represents the status of an individual endpoint.

 * **Fields:**
   * `Endpoint string` — The endpoint identifier.
   * `Status bool` — The status of the endpoint.

## `type Main`

Contains the main components of the application, including context, connectors, logger, and Redis client.

 * **Fields:**
   * `Context context.Context` — Application context.
   * `Cancel context.CancelFunc` — Function to cancel the application context.
   * `Connector Connector` — The connector instance.
   * `Logger *logrus.Logger` — Logger for logging application events.
   * `RedisClient *redis.Client` — Redis client for data handling.
   * `UUIDMapper *uuid.UUIDMapper` — UUID mapper for managing UUID mappings.