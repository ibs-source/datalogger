# UUID

## `func NewUUIDMapper(client *redis.Client, logger *logrus.Logger) *UUIDMapper`

NewUUIDMapper creates a new UUIDMapper instance.

 * **Parameters:**
   * `client` — The Redis client for persistence.
   * `logger` — The logger for logging messages.
 * **Returns:** A pointer to the initialized UUIDMapper.

## `func (um *UUIDMapper) GetOrCreateUUID(endpoint, ID string) (string, error)`

GetOrCreateUUID retrieves an existing UUID for the given endpoint and ID, or creates a new one if it doesn't exist.

 * **Parameters:**
   * `endpoint` — The endpoint string.
   * `ID` — The node ID string.
 * **Returns:** The UUID string and an error if any occurred.

## `func (um *UUIDMapper) SaveMappingToRedis(key, uuidStr string) error`

SaveMappingToRedis saves a key-UUID pair to Redis.

 * **Parameters:**
   * `key` — The key to save.
   * `uuidStr` — The UUID associated with the key.
 * **Returns:** An error if the operation fails.

## `func (um *UUIDMapper) LoadMappingFromRedis() error`

LoadMappingFromRedis loads the UUID mapping from Redis into memory.

 * **Returns:** An error if the operation fails.