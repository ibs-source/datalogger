/**
* Package producer provides the implementation for the main application logic,
* including signal handling and connector management.
*/
package producer

import (
    "os"
    "os/signal"
    "syscall"
    "context"

    "github.com/femogas/datalogger/redis"
    "github.com/femogas/datalogger/redis/uuid"
    "github.com/femogas/datalogger/app/configuration"
    "github.com/sirupsen/logrus"
)

/**
* Connector is an interface that defines methods for setting up,
* starting, stopping, and checking the status of a connector.
*/
type Connector interface {
    /**
    * Sets up the connector with the given global configuration.
    *
    * @param globalconfig A pointer to the global configuration.
    * @return An error if setup fails, otherwise nil.
    */
    Setup(globalconfig *global.GlobalConfiguration) error

    /**
    * Starts the connector.
    *
    * @return An error if the connector fails to start, otherwise nil.
    */
    Start() error

    /**
    * Retrieves the status of the connector.
    *
    * @return The current status of the connector.
    */
    Status() Status

    /**
    * Stops the connector.
    *
    * @return An error if the connector fails to stop, otherwise nil.
    */
    Stop() error

    /**
    * Creates a map of valid keys from the given configurations.
    *
    * @param configurations A slice of configurations.
    * @return A map where each key is a string, and the value is an interface.
    */
    CreateValidKeys() map[string]interface{}
}

/**
* Status represents the overall status of the connector,
* including the status of each endpoint.
*/
type Status struct {
    Connector []StatusEndpoint `json:"connector"`
}

/**
* StatusEndpoint represents the status of an individual endpoint.
*/
type StatusEndpoint struct {
    Endpoint string `json:"endpoint"`
    Status   bool   `json:"status"`
}

/**
* Main contains the main components of the application,
* including context, connectors, logger, and Redis client.
*/
type Main struct {
    Context     context.Context
    Cancel      context.CancelFunc
    Connector   Connector
    Logger      *logrus.Logger
    RedisClient *redis.Client
    UUIDMapper  *uuid.UUIDMapper
}

/**
* Sets up signal handling for graceful shutdown of the application.
* Listens for SIGINT and SIGTERM signals and initiates shutdown procedures.
*
* @param app       A pointer to the Main application structure.
* @param connector The connector to be managed during shutdown.
*/
func SetupSignalHandling(app *Main, connector Connector) {
    signals := make(chan os.Signal, 1)
    // Notify the channel on receiving SIGINT or SIGTERM signals.
    signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
    // Start a goroutine to handle signals.
    go handleSignals(signals, app, connector)
}

/**
* Handles incoming OS signals for graceful shutdown.
* Stops the connector and cancels the application context.
*
* @param signals   A channel receiving OS signals.
* @param app       A pointer to the Main application structure.
* @param connector The connector to be stopped.
*/
func handleSignals(signals chan os.Signal, app *Main, connector Connector) {
    // Wait for an OS signal.
    sig := <-signals
    app.Logger.WithField("signal", sig).Info("Termination signal received, shutting down safely...")

    // Attempt to stop the connector.
    if err := connector.Stop(); err != nil {
        app.Logger.WithError(err).Error("Error stopping connector")
    }

    // Cancel the application context to trigger shutdown.
    app.Cancel()
}