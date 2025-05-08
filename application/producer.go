/**
 * Package producer provides the implementation for the main application logic,
 * including signal handling and connector management.
 */
package producer

import (
	"context"
	"fmt"
	"time"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ibs-source/datalogger/application/configuration"
	"github.com/ibs-source/datalogger/redis"
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
 * Ensures all resources are properly closed and context is canceled.
 * Implements timeouts to prevent indefinite blocking during shutdown.
 *
 * @param signals     A channel receiving OS signals.
 * @param application A pointer to the Main application structure.
 * @param connector   The connector to be stopped.
 */
func handleSignals(signals chan os.Signal, application *Main, connector Connector) {
    // Wait for signal
    sig := <-signals
    logger := application.Logger.WithField("signal", sig)
    logger.Info("Termination signal received, shutting down safely...")
    // Create a deadline for shutdown
    shutdownDeadline := time.Now().Add(30 * time.Second)
    shutdownCtx, cancel := context.WithDeadline(context.Background(), shutdownDeadline)
    defer cancel()
    // Track any errors during shutdown
    var shutdownErrors []error
    // Stop connector if available
    if connector != nil {
        logger.Debug("Stopping connector...")
        err := stopConnectorWithTimeout(shutdownCtx, connector)
        if err != nil {
            shutdownErrors = append(shutdownErrors, fmt.Errorf("connector shutdown: %w", err))
            logger.WithError(err).Error("Error stopping connector")
        } else {
            logger.Debug("Connector stopped successfully")
        }
    }
    // Close Redis connection if available
    if application.Redis != nil {
        logger.Debug("Closing Redis connection...")
        err := closeRedisWithTimeout(shutdownCtx, application.Redis)
        if err != nil {
            shutdownErrors = append(shutdownErrors, fmt.Errorf("redis shutdown: %w", err))
            logger.WithError(err).Error("Error closing Redis connection")
        } else {
            logger.Debug("Redis connection closed successfully")
        }
    }
    // Cancel the application context
    if application.Cancel != nil {
        logger.Debug("Canceling application context...")
        application.Cancel()
        logger.Debug("Application context canceled")
    }
    // Log shutdown status
    if len(shutdownErrors) > 0 {
        logger.WithField("errorCount", len(shutdownErrors)).
            Warn("Shutdown completed with errors")
    } else {
        logger.Info("Shutdown completed successfully")
    }
    // Force exit if we've reached the max shutdown time
    timeRemaining := time.Until(shutdownDeadline)
    if timeRemaining <= 0 {
        logger.Warn("Shutdown deadline exceeded, forcing exit")
        os.Exit(1)
    }
}

/**
 * stopConnectorWithTimeout attempts to stop the connector with a timeout to prevent blocking.
 *
 * @param ctx       The context with timeout for the operation.
 * @param connector The connector to be stopped.
 * @return          An error if stopping fails or times out.
 */
func stopConnectorWithTimeout(ctx context.Context, connector Connector) error {
    // Create channels for result communication
    connectorDone := make(chan error, 1)
    // Start connector stopping operation
    go stopConnectorWorker(connector, connectorDone)
    // Wait for completion or timeout
    select {
    case err := <-connectorDone:
        return err
    case <-ctx.Done():
        return fmt.Errorf("connector shutdown timed out: %w", ctx.Err())
    }
}

/**
 * stopConnectorWorker performs the actual connector stopping and sends the result to the provided channel.
 *
 * @param connector    The connector to be stopped.
 * @param resultChan   The channel to send the stopping result to.
 */
func stopConnectorWorker(connector Connector, resultChan chan<- error) {
    resultChan <- connector.Stop()
}

/**
 * closeRedisWithTimeout attempts to close the Redis connection with a timeout to prevent blocking.
 *
 * @param ctx       The context with timeout for the operation.
 * @param redisClient The Redis client to be closed.
 * @return          An error if closing fails or times out.
 */
func closeRedisWithTimeout(ctx context.Context, redisClient *redis.Client) error {
    // Create channels for result communication
    redisDone := make(chan error, 1)
    // Start Redis closing operation
    go closeRedisWorker(redisClient, redisDone)
    // Wait for completion or timeout
    select {
    case err := <-redisDone:
        return err
    case <-ctx.Done():
        return fmt.Errorf("redis shutdown timed out: %w", ctx.Err())
    }
}

/**
 * closeRedisWorker performs the actual Redis client closing and sends the result to the provided channel.
 *
 * @param redisClient The Redis client to be closed.
 * @param resultChan  The channel to send the closing result to.
 */
func closeRedisWorker(redisClient *redis.Client, resultChan chan<- error) {
    resultChan <- redisClient.Close()
}