# UUID Package Documentation Update

This package provides a UUID mapping mechanism for assigning and managing UUIDs for endpoints and node IDs.

## Package Overview

The `uuid` package allows you to maintain a mapping between unique keys and UUIDs, along with their associated configuration objects. It includes features such as generating new UUIDs, updating existing configurations, and comparing mappings to verify changes.

## Structs

### `UUIDEntry`

Represents an entry in the UUID map, containing the UUID and the associated configuration object.

- **Fields:**
  - `UUID` (string): The UUID associated with the key.
  - `Configuration` (interface{}): The associated configuration object.

### `UUIDMapper`

A thread-safe structure that maintains a map between keys and `UUIDEntry` objects.

- **Fields:**
  - `Mapping` (map[string]UUIDEntry): Stores key-UUIDEntry pairs.
  - `Logger` (*logrus.Logger): Logger for recording messages.

## Interfaces

### `UUIDMapperInterface`

Defines the methods that a UUID mapper should implement.

- **Methods:**
  - `ApplyValidConfigurations(validKeys map[string]interface{}) error`

## Functions

### `func NewUUIDMapper(logger *logrus.Logger) *UUIDMapper`

**NewUUIDMapper** creates a new instance of `UUIDMapper`.

- **Parameters:**
  - `logger`: The logger for recording messages.
- **Returns:** A pointer to the initialized `UUIDMapper`.

### `func (entry *UUIDEntry) EqualConfiguration(other interface{}) bool`

**EqualConfiguration** compares the `Configuration` of the `UUIDEntry` with another configuration.

- **Parameters:**
  - `other`: The other configuration to compare with.
- **Returns:** `true` if the configurations are equal, `false` otherwise.

### `func (entry *UUIDEntry) Equals(other UUIDEntry) bool`

**Equals** compares the current `UUIDEntry` with another `UUIDEntry`.

- **Parameters:**
  - `other`: The other `UUIDEntry` to compare with.
- **Returns:** `true` if the entries are equal, `false` otherwise.

### `func (um *UUIDMapper) Equals(other map[string]UUIDEntry) bool`

**Equals** compares the current `UUIDMapper`'s `Mapping` with another map of `UUIDEntry`.

- **Parameters:**
  - `other`: The other map to compare with.
- **Returns:** `true` if the mappings are equal, `false` otherwise.

### `func (um *UUIDMapper) GetUUIDEntryFromMapping(key string) (UUIDEntry, bool)`

**GetUUIDEntryFromMapping** retrieves the `UUIDEntry` from the internal map for a given key.

- **Parameters:**
  - `key`: The key to search for.
- **Returns:**
  - `UUIDEntry`: The entry associated with the key.
  - `bool`: Indicates whether the entry exists.

### `func (um *UUIDMapper) GenerateUUID() string`

**GenerateUUID** generates a new UUID.

- **Returns:** A string representing the new UUID.

### `func (um *UUIDMapper) UpsertUUIDEntry(key string, configuration interface{}) (UUIDEntry, error)`

**UpsertUUIDEntry** updates an existing `UUIDEntry` or creates a new one if it doesn't exist.

- **Parameters:**
  - `key`: The key for the `UUIDEntry`.
  - `configuration`: The configuration object.
- **Returns:**
  - `UUIDEntry`: The updated or newly created `UUIDEntry`.
  - `error`: An error if any occurred.