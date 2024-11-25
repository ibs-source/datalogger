# PubSub

## `func NewPubSub(logger *logrus.Logger) *PubSub`

NewPubSub creates a new PubSub instance with the given logger.

- **Parameters:**
  - `logger` — A pointer to the logger.
- **Returns:** A pointer to the newly created PubSub instance.

## `func (ps *PubSub) HandleCommand(command string, handlers map[string]CommandHandler)`

HandleCommand handles an incoming command by executing the corresponding handler.

- **Parameters:**
  - `command` — The command string received.
  - `handlers` — A map of command strings to their corresponding handlers.