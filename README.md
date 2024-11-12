
# Guide to Creating a Custom Connector

Welcome! This guide will help you create your own custom connector using our project as a base. We’ll provide an overview of the main features and guide you through the necessary steps to develop and integrate your connector.

## Introduction

This project provides a framework for creating connectors that interact with a Redis-based system, manage unique UUIDs for nodes or endpoints, and utilize Pub/Sub mechanisms for inter-process communication. This guide is intended for developers looking to extend the system's functionality by creating custom connectors.

## Prerequisites

- **Go** version 1.16 or higher.
- **Redis** installed and running.
- Familiarity with Go language and concepts like contexts, logging, and error handling.

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
import "github.com/femogas/datalogger/app/configuration"

config := configuration.InitializeGlobalConfiguration()
```

## Logger Initialization

The logger is essential for monitoring the operation of your connector. Initialize it using `InitializeLogger(debug int)` from the `global` package, where `debug` represents the desired logging level:

- `0`: DebugLevel (all logs)
- `1`: InfoLevel (informative and higher level logs)
- `2`: ErrorLevel (errors only)

```go
import "github.com/femogas/datalogger/app"

logger := app.InitializeLogger(config.DebugMode)
```

## Connecting to Redis

To interact with Redis, create a new client using `NewClient(logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc)` from the `redis` package.

```go
import (
    "context"
    "github.com/femogas/datalogger/redis"
)

ctx, cancel := context.WithCancel(context.Background())
redisClient, err := redis.NewClient(logger, ctx, cancel)
if err != nil {
    logger.Fatalf("Error initializing Redis client: %v", err)
}
defer redisClient.Close()
```

Ensure proper context management and close the connection when it's no longer needed.

## UUID Management

UUIDs are used to uniquely identify nodes or endpoints. Use `NewUUIDMapper` from the `uuid` package to create a UUID handler.

```go
import "github.com/femogas/datalogger/redis/uuid"

uuidMapper := uuid.NewUUIDMapper(redisClient, logger)
```

Load any existing mappings from Redis:

```go
err = uuidMapper.LoadMappingFromRedis()
if err != nil {
    logger.Fatalf("Error loading UUID mappings from Redis: %v", err)
}
```

To verify and clean the UUID map:

```go
import "github.com/femogas/datalogger/app/configuration"

configurations, err := configuration.LoadConfigurations(globalConfig.ConfigFilePath)
if err != nil {
    logger.Fatalf("Error loading configurations: %v", err)
}

if err := uuidMapper.VerifyAndCleanUUIDMap(configurations); err != nil {
    logger.Fatalf("Error verifying and cleaning UUID mapping: %v", err)
}
```

## Connector Implementation

The connector is defined through the `Connector` interface. This interface includes methods for setting up, starting, stopping, and checking the connector's status. Below is a description of the `Connector` interface:

```go
type Connector interface {
    Setup(globalconfig *global.GlobalConfiguration) error
    Start() error
    Status() Status
    Stop() error
    CreateValidKeys() map[string]struct{}
}
```

## Signal Handling

To ensure a clean application shutdown, use `SetupSignalHandling` from the `producer` package.

```go
import "github.com/femogas/datalogger/app"

producer.SetupSignalHandling(appInstance, connector)
```

This listens for system signals like `SIGINT` and `SIGTERM` and calls appropriate methods to stop the connector.

## Running the Server

The connector includes an HTTP server for integration with the node.js part; initialize it using `NewServer` from the `server` package.

```go
import "github.com/femogas/datalogger/server"

httpServer := server.NewServer(appInstance, logger)
if err := httpServer.Start(); err != nil {
    logger.Fatalf("Error starting server: %v", err)
}
```

## Versioning

We use [SemVer](https://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/femogas/datalogger/tags). 

## Authors

* **Paolo Fabris** - *Initial work* - [ubyte.it](https://ubyte.it/)

See also the list of [contributors](https://github.com/femogas/datalogger/blob/main/CONTRIBUTORS.md) who participated in this project.

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/femogas/datalogger/blob/main/LICENSE) file for details.