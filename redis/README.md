# Redis Documentation

This package provides functionalities for connecting and interacting with Redis, including client initialization, stream management, and message pushing.

## Environment Variables

| Parameter   | Type     | Description                  | Default                  |
| ----------- | -------- | ---------------------------- | ------------------------ |
| `REDIS_URL` | `string` | The URL of the Redis server. | `redis://127.0.0.1:6379` |

## Functions

### `func NewClient(logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc) (*Client, error)`

Initializes and returns a new Redis client.

- **Parameters:**
  - `logger`: A logger instance for logging.
  - `ctx`: The context for client operations.
  - `cancel`: A cancel function to stop the context.
- **Returns:** A pointer to the initialized `Client` and an error if any occurred.

### `func (rc *Client) Close() error`

Closes the Redis client connection after synchronizing the UUID map.

- **Returns:** An error if closing fails.

### `func (rc *Client) PushToRedis(uuid string, max int64, data map[string]interface{}) error`

Adds data to a specified Redis stream with approximate pruning.

- **Parameters:**
  - `uuid`: The name of the Redis stream.
  - `max`: The maximum number of records to retain in the stream.
  - `data`: The data to be added to the stream.
- **Returns:** An error if the operation fails.

### `func (rc *Client) ListenForCommands(startFunc, stopFunc func() error)`

Subscribes to the Redis channel and listens for start and stop commands.

- **Parameters:**
  - `startFunc`: The function to execute when a start command is received.
  - `stopFunc`: The function to execute when a stop command is received.

### `func (rc *Client) GenerateUUIDMap(createValidKeysFunc func() map[string]interface{}) error`

Verifies the current UUID mappings against the provided configurations.

- **Parameters:**
  - `createValidKeysFunc`: A function that generates a set of valid keys as `map[string]interface{}`.
- **Returns:** An error if the operation fails.

## Structs

### `type Client`

Encapsulates the Redis client and related configurations.

- **Fields:**
  - `Client`: Underlying Redis client.
  - `Logger`: Logger for logging messages.
  - `UUIDMapper`: UUIDMapper for mapping UUIDs.
  - `Configuration`: Redis connection settings.
  - `Context`: Context for controlling operations.
