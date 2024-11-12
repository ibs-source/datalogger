/**
* Package pubsub provides functionality for subscribing to Redis channels
* and handling commands received via Pub/Sub messaging.
*/
package pubsub

import (
    "context"

    "github.com/femogas/datalogger/redis"
    "github.com/sirupsen/logrus"
)

const (
    // Channel is the Redis channel used for remote control commands.
    Channel = "remote-control"
)

const (
    // Start command string.
    Start = "start"
    // Stop command string.
    Stop = "stop"
)

/**
* CommandHandler defines a function type that handles a command and returns an error if any.
*/
type CommandHandler func() error

/**
* PubSub encapsulates the Redis client and logger for handling Pub/Sub messages.
*/
type PubSub struct {
    Client *redis.Client
    Logger *logrus.Logger
}

/**
* NewPubSub creates a new PubSub instance with the given Redis client and logger.
*
* @param client A pointer to the Redis client.
* @param logger A pointer to the logger.
* @return A pointer to the newly created PubSub instance.
*/
func NewPubSub(client *redis.Client, logger *logrus.Logger) *PubSub {
    return &PubSub{
        Client: client,
        Logger: logger,
    }
}

/**
* ListenForCommands subscribes to the Redis channel and listens for start and stop commands.
*
* @param ctx       The context for managing cancellation.
* @param startFunc The function to execute when a start command is received.
* @param stopFunc  The function to execute when a stop command is received.
*/
func (ps *PubSub) ListenForCommands(ctx context.Context, startFunc, stopFunc func() error) {
    pubsub := ps.subscribeToChannel(ctx)
    defer pubsub.Close()

    ch := pubsub.Channel()
    handlers := ps.initializeHandlers(startFunc, stopFunc)

    ps.listen(ctx, ch, handlers)
}

/**
* Subscribes to the Redis channel.
*
* @param ctx The context for managing cancellation.
* @return A pointer to the Redis PubSub instance.
*/
func (ps *PubSub) subscribeToChannel(ctx context.Context) *redis.PubSub {
    return ps.Client.Client.Subscribe(ctx, Channel)
}

/**
* Initializes the command handlers for start and stop commands.
*
* @param startFunc The function to execute when a start command is received.
* @param stopFunc  The function to execute when a stop command is received.
* @return A map of command strings to their corresponding handlers.
*/
func (ps *PubSub) initializeHandlers(startFunc, stopFunc func() error) map[string]CommandHandler {
    return map[string]CommandHandler{
        Start: startFunc,
        Stop:  stopFunc,
    }
}

/**
* Listens for incoming commands from the Redis channel and handles them accordingly.
*
* @param ctx      The context for managing cancellation.
* @param ch       The channel to receive messages from.
* @param handlers A map of command strings to their corresponding handlers.
*/
func (ps *PubSub) listen(ctx context.Context, ch <-chan *redis.Message, handlers map[string]CommandHandler) {
    for {
        select {
        case <-ctx.Done():
            ps.handleContextDone()
            return
        case msg := <-ch:
            ps.handleCommand(msg.Payload, handlers)
        }
    }
}

/**
* Handles the context cancellation event.
*/
func (ps *PubSub) handleContextDone() {
    ps.Logger.Info("Stopping command listener")
}

/**
* Handles an incoming command by executing the corresponding handler.
*
* @param command  The command string received.
* @param handlers A map of command strings to their corresponding handlers.
*/
func (ps *PubSub) handleCommand(command string, handlers map[string]CommandHandler) {
    ps.logReceivedCommand(command)

    handler, exists := handlers[command]
    if !exists {
        ps.logUnknownCommand(command)
        return
    }

    if err := ps.executeHandler(handler); err != nil {
        ps.logHandlerError(err, command)
        return
    }

    ps.logCommandSuccess(command)
}

/**
* Executes the provided command handler.
*
* @param handler The CommandHandler function to execute.
* @return An error if the handler execution fails, otherwise nil.
*/
func (ps *PubSub) executeHandler(handler CommandHandler) error {
    return handler()
}

/**
* Logs that a command has been received.
*
* @param command The command string received.
*/
func (ps *PubSub) logReceivedCommand(command string) {
    ps.Logger.WithField("message", command).Info("Received command")
}

/**
* Logs that an unknown command has been received.
*
* @param command The command string received.
*/
func (ps *PubSub) logUnknownCommand(command string) {
    ps.Logger.WithField("command", command).Warn("Unknown command received")
}

/**
* Logs an error that occurred while executing a command handler.
*
* @param err     The error that occurred.
* @param command The command string being handled.
*/
func (ps *PubSub) logHandlerError(err error, command string) {
    ps.Logger.WithError(err).Errorf("Error executing %q command", command)
}

/**
* Logs that a command has been successfully executed.
*
* @param command The command string that was executed.
*/
func (ps *PubSub) logCommandSuccess(command string) {
    ps.Logger.Infof("Connector %s via command", command)
}
