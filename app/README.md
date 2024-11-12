# Producer

## `func SetupSignalHandling(app *Main, connector Connector)`

Sets up signal handling for graceful shutdown of the application. Listens for SIGINT and SIGTERM signals and initiates shutdown procedures.

 * **Parameters:**
   * `app` — A pointer to the Main application structure.
   * `connector` — The connector to be managed during shutdown.