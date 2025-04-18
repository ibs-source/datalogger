# Guide to Integrating the Datalogger Custom Connector

Welcome! This guide will help you integrate the Datalogger Custom Connector package into your project. We’ll provide an overview of the main features and guide you through the necessary steps to develop, configure, and integrate your connector.

## Introduction

This project provides a framework for creating connectors that interact with a Redis-based system, manage unique UUIDs for nodes or endpoints, and utilize Pub/Sub mechanisms for inter-process communication. This guide is intended for developers looking to extend the system's functionality by creating and integrating custom connectors.

## Prerequisites

- **Go** version 1.16 or higher.
- **Redis** installed and running.
- Familiarity with Go language and concepts like contexts, logging, and error handling.

## Installation

To use the Datalogger Custom Connector in your project, add it as a dependency in your `go.mod` file:

```bash
go get github.com/ibs-source/datalogger
```

Ensure that the package is properly installed and available in your project.

## Project Structure

The project is divided into several key packages:

- **Global**: Manages global application configuration.
- **PubSub**: Manages Pub/Sub communication through Redis.
- **UUID**: Manages UUID mapping and persistence.
- **Redis**: Contains functions to interact with Redis and manage the stream.
- **Server**: Provides an HTTP server for the connector interface.
- **App**: Contains the main application logic.
- **Configuration**: Manages loading connector configurations.

## Global Configuration

To initialize global configuration, use the `InitializeGlobalConfiguration()` function from the `global` package. This function reads environment variables and command-line flags to set configuration values.

```go
import "github.com/ibs-source/datalogger/application/configuration"

config := configuration.InitializeGlobalConfiguration()
```

This configuration is essential for setting up the connector correctly and should be integrated at the beginning of your application.

## Logger Initialization

The logger is essential for monitoring the operation of your connector. Initialize it using `InitializeLogger(debug int)` from the `global` package, where `debug` represents the desired logging level:

- `0`: DebugLevel (all logs)
- `1`: InfoLevel (informative and higher level logs)
- `2`: ErrorLevel (errors only)

```go
import "github.com/ibs-source/datalogger/application"

logger := application.InitializeLogger(config.DebugMode)
```

The logger will help you track the connector's activities and debug issues effectively.

## Connecting to Redis

To interact with Redis, create a new client using `NewClient(logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc)` from the `redis` package.

```go
import (
    "context"
    "github.com/ibs-source/datalogger/redis"
)

ctx, cancel := context.WithCancel(context.Background())
redisClient, err := redis.NewClient(logger, ctx, cancel)
if err != nil {
    logger.Fatalf("Error initializing Redis client: %v", err)
}
defer redisClient.Close()
```

Proper context management ensures the Redis client connection is closed when it is no longer needed.

## UUID Management

UUIDs are used to uniquely identify nodes or endpoints. The UUID management process is now handled automatically within the Redis client, making it easier to manage and integrate UUIDs for your nodes.

```go
// Generate UUID map
err = redisClient.GenerateUUIDMap(connector.CreateValidKeys)
if err != nil {
    logger.WithError(err).Fatal("Failed to generate UUID map")
}
```

This step is essential for ensuring each node is uniquely identified and can be tracked throughout the system.

## Connector Implementation

To create a custom connector, implement the `Connector` interface provided by the Datalogger package. This interface includes methods for setting up, starting, stopping, and checking the connector's status.

### Connector Interface

Below is a description of the `Connector` interface:

```go
type Connector interface {
    Setup(globalconfig *global.GlobalConfiguration) error
    Start() error
    Status() Status
    Stop() error
    CreateValidKeys() map[string]struct{}
}
```

You must implement these methods to define how your connector should behave during its lifecycle.

### Example Usage

To use the connector, you need to create an instance of your connector struct and initialize it with the necessary dependencies, such as Redis and global configuration.

```go
import (
    "context"
    "sync"
    "github.com/ibs-source/datalogger/application"
    "github.com/ibs-source/datalogger/application/configuration"
    "github.com/ibs-source/datalogger/redis"
    "github.com/ibs-source/datalogger/server"
    "github.com/sirupsen/logrus"
)

/**
* @struct YourNewConnector
* @brief Manages client connections and configurations.
*
* This struct handles the setup, validation, and management of clients based on
* the provided configurations. It ensures thread-safe operations using mutex locks.
*/
type YourNewConnector struct {
    clients       []*YourNewConnectorClient // Slice of active multiple clients
    mutex         sync.Mutex                // Mutex for synchronizing access to the connector
    running       bool                      // Indicates if the connector is currently running
    Logger        *logrus.Logger            // Logger for logging activities and errors
    Configuration *configuration.Connector  // Connector configurations
    Redis         *redis.Client             // Redis client for data storage and retrieval
}

/**
* @brief The main function initializes the application components and starts the server.
*
* It performs the following steps:
* 1. Initializes global configuration and logger.
* 2. Loads configurations from the specified config file.
* 3. Initializes the Redis client.
* 4. Sets up the UUID mapper and connector.
* 5. Starts the Pub/Sub listener for commands.
* 6. Starts the HTTP server.
* 7. Handles graceful shutdown upon receiving termination signals.
*/
func main() {
    globalConfig := configuration.InitializeGlobalConfiguration()
    logger := application.InitializeLogger(globalConfig.DebugMode)

    // Load configurations and handle potential errors.
    configurations, err := configuration.LoadConfigurations(globalConfig.ConfigFilePath)
    if err != nil {
        logger.Fatalf("Error loading configurations: %v", err)
    }

    // Create a cancellable context for managing lifecycle.
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Initialize Redis client
    redisClient, err := redis.NewClient(logger, ctx, cancel)
    if err != nil {
        logger.WithError(err).Fatal("Failed to initialize Redis client")
    }
    defer redisClient.Close()

    // Create your custom connector instance
    connector := &YourNewConnector{
        Redis:         redisClient,
        Logger:        logger,
        Configuration: &configuration.Connector{Configurations: configurations},
    }

    // Generate UUID map
    err = redisClient.GenerateUUIDMap(connector.CreateValidKeys)
    if err != nil {
        logger.WithError(err).Fatal("Failed to generate UUID map")
    }

    // Setup the connector with global configurations and handle setup errors.
    if err := connector.Setup(globalConfig); err != nil {
        logger.Fatalf("Error setting up connector: %v", err)
    }

    go redisClient.ListenForCommands(connector.Start, connector.Stop)

    // Initialize and start the HTTP server, handling any startup errors.
    httpServer := server.NewServer(appInstance, logger)
    if err := httpServer.Start(); err != nil {
        logger.Fatalf("Error starting server: %v", err)
    }

    // Setup signal handling for graceful shutdown.
    producer.SetupSignalHandling(appInstance, connector)

    // Wait for the context to be canceled (e.g., via a termination signal).
    <-ctx.Done()
}

/**
* @brief Creates a map of valid keys by hashing endpoint URLs and node IDs.
*
* This function iterates through all configurations and nodes, generating a unique key
* for each combination of endpoint URL and node ID. This logic can be customized as needed.
*
* @return A map with unique keys pointing to their respective node configurations.
*/
func (connector *YourNewConnector) CreateValidKeys() map[string]interface{} {
    validKeys := make(map[string]interface{})
    for item := range connector.Configuration.Configurations {
        configuration := &connector.Configuration.Configurations[item]
        for index := range configuration.Nodes {
            node := &configuration.Nodes[index]
            // Custom logic for key generation, e.g., using node ID and endpoint URL
            key := node.ID + "-" + configuration.EndpointURL
            validKeys[key] = node
        }
    }
    return validKeys
}
```

This example demonstrates how to integrate the connector lifecycle with Redis and HTTP server functionalities.

## Signal Handling

To ensure a clean application shutdown, use `SetupSignalHandling` from the `producer` package. This will listen for system signals like `SIGINT` and `SIGTERM` and call appropriate methods to stop the connector.

```go
import "github.com/ibs-source/datalogger/application"

producer.SetupSignalHandling(appInstance, connector)
```

Proper signal handling ensures that the application resources are released gracefully.

## Running the Server

The Datalogger Custom Connector includes an HTTP server for integration purposes. Use `NewServer` from the `server` package to initialize and run the HTTP server:

```go
import "github.com/ibs-source/datalogger/server"

httpServer := server.NewServer(appInstance, logger)
if err := httpServer.Start(); err != nil {
    logger.Fatalf("Error starting server: %v", err)
}
```

The server provides a simple interface to monitor and control the connector, facilitating integration with other components like a Node.js application.

## Pub/Sub Listener

The Redis client includes a listener for commands that control the connector. Use the `ListenForCommands` method to start or stop the connector based on messages received.

```go
go redisClient.ListenForCommands(connector.Start, connector.Stop)
```

This setup allows you to control the connector dynamically, responding to external requests for starting or stopping operations.

## Versioning

We use [SemVer](https://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/ibs-source/datalogger/tags).

## Authors

- **Paolo Fabris** - _Initial work_ - [ubyte.it](https://ubyte.it/)

See also the list of [contributors](https://github.com/ibs-source/datalogger/blob/main/CONTRIBUTORS.md) who participated in this project.

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/ibs-source/datalogger/blob/main/LICENSE) file for details.
