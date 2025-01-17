/**
 * Package producer provides the implementation for the main application logic,
 * including signal handling and connector management.
 */
package producer

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/femogas/datalogger/application/configuration"
	"github.com/femogas/datalogger/redis"
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
	Context   context.Context
	Cancel    context.CancelFunc
	Connector Connector
	Logger    *logrus.Logger
	Redis     *redis.Client
}

/**
 * Sets up signal handling for graceful shutdown of the application.
 * Listens for SIGINT and SIGTERM signals and initiates shutdown procedures.
 *
 * @param application  A pointer to the Main application structure.
 * @param connector    The connector to be managed during shutdown.
 */
func SetupSignalHandling(application *Main, connector Connector) {
	signals := make(chan os.Signal, 1)
	// Notify the channel on receiving SIGINT or SIGTERM signals.
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	// Start a goroutine to handle signals.
	go handleSignals(signals, application, connector)
}

/**
 * ParseHostFromEndpointURL extracts the hostname from an endpoint URL string.
 *
 * @param endpointURL The URL string representing the endpoint.
 *
 * @return string     The hostname extracted from the endpoint URL.
 * @return error      An error if the URL cannot be parsed.
 */
func ParseHostFromEndpointURL(endpointURL string) (string, error) {
	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return "", fmt.Errorf("invalid endpointURL '%s': %w", endpointURL, err)
	}
	return parsedURL.Hostname(), nil
}

/**
 * Handles incoming OS signals for graceful shutdown.
 * Stops the connector and cancels the application context.
 *
 * @param signals       A channel receiving OS signals.
 * @param application   A pointer to the Main application structure.
 * @param connector     The connector to be stopped.
 */
func handleSignals(signals chan os.Signal, application *Main, connector Connector) {
	// Wait for an OS signal.
	sig := <-signals
	application.Logger.WithField("signal", sig).Info("Termination signal received, shutting down safely...")

	// Attempt to stop the connector.
	if err := connector.Stop(); err != nil {
		application.Logger.WithError(err).Error("Error stopping connector")
	}

	// Cancel the application context to trigger shutdown.
	application.Cancel()
}
