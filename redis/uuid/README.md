# UUID

## Functions Summary

### `func NewUUIDMapper(client *redis.Client, logger *logrus.Logger) *UUIDMapper`

**NewUUIDMapper** creates a new instance of `UUIDMapper`.

- **Parameters:**
  - `client` — The Redis client for persistence.
  - `logger` — The logger for logging messages.
- **Returns:** A pointer to the initialized `UUIDMapper`.

### `func (um *UUIDMapper) GetUUIDEntryFromMapping(key string) (UUIDEntry, bool)`

**GetUUIDEntryFromMapping** retrieves the `UUIDEntry` from the internal map for a given key.

- **Parameters:**
  - `key` — The key to search for.
- **Returns:**
  - `UUIDEntry` — The entry associated with the key.
  - `bool` — Indicates whether the entry exists.

### `func (um *UUIDMapper) GenerateUUIDMap(createValidKeysFunc func() map[string]interface{}) error`

**GenerateUUIDMap** verifies the current UUID mappings against the provided configurations and updates them accordingly.

- **Parameters:**
  - `createValidKeysFunc` — A function that generates a set of valid keys as `map[string]interface{}`.
- **Returns:** An error if the operation fails.