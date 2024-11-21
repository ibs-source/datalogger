/**
* Package pubsub provides functionality for handling Pub/Sub commands.
*/
package pubsub

import (
    "github.com/sirupsen/logrus"
)

/**
* CommandHandler defines a function type that handles a command and returns an error if any.
*/
type CommandHandler func() error

/**
* PubSubInterface defines the methods that a PubSub should implement.
*/
type PubSubInterface interface {
    HandleCommand(command string, handlers map[string]CommandHandler)
}

/**
* PubSub encapsulates the logger and manages command handling.
*/
type PubSub struct {
    Logger *logrus.Logger
}

/**
* NewPubSub creates a new PubSub instance with the given logger.
*
* @param logger A pointer to the logger.
* @return A pointer to the newly created PubSub instance.
*/
func NewPubSub(logger *logrus.Logger) *PubSub {
    return &PubSub{
        Logger: logger,
    }
}

/**
* HandleCommand handles an incoming command by executing the corresponding handler.
*
* @param command  The command string received.
* @param handlers A map of command strings to their corresponding handlers.
*/
func (ps *PubSub) HandleCommand(command string, handlers map[string]CommandHandler) {
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