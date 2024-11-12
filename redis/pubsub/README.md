# PubSub

## `func NewPubSub(client *redis.Client, logger *logrus.Logger) *PubSub`

NewPubSub creates a new PubSub instance with the given Redis client and logger.

 * **Parameters:**
   * `client` — A pointer to the Redis client.
   * `logger` — A pointer to the logger.
 * **Returns:** A pointer to the newly created PubSub instance.

## `func (ps *PubSub) ListenForCommands(ctx context.Context, startFunc, stopFunc func() error)`

ListenForCommands subscribes to the Redis channel and listens for start and stop commands.

 * **Parameters:**
   * `ctx` — The context for managing cancellation.
   * `startFunc` — The function to execute when a start command is received.
   * `stopFunc` — The function to execute when a stop command is received.