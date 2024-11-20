# Redis Client Documentation

| Parameter        | Type     | Description                                    | Default                  |
| ---------------- | -------- | ---------------------------------------------- | ------------------------ |
| `REDIS_URL`      | `string` | The URL of the Redis server.                   | `redis://127.0.0.1:6379` |

## `func NewClient(logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc) (*Client, error)`

NewClient initializes and returns a new Redis client.

- **Parameters:**
  - `logger` — A logger instance for logging.
  - `ctx` — The context for the client operations.
  - `cancel` — A cancel function to stop the context.
- **Returns:** A pointer to the initialized Client and an error if any occurred.

## `func (rc *Client) Instance() *redis.Client`

Instance returns the underlying Redis client instance.

- **Returns:** A pointer to the Redis client.

## `func (rc *Client) Close() error`

Close closes the Redis client connection.

- **Returns:** An error if closing fails.

## `func (rc *Client) Ping() error`

Ping checks the connection to Redis by sending a PING command.

- **Returns:** An error if the ping fails.

## `func (rc *Client) PushToRedis(uuid string, max int64, data map[string]interface{}) error`

PushToRedis adds data to a specified Redis stream with approximate pruning.

- **Parameters:**
  - `uuid` — The name of the Redis stream.
  - `max` — The maximum number of records to retain in the stream.
  - `data` — The data to be added to the stream.
- **Returns:** An error if the operation fails.