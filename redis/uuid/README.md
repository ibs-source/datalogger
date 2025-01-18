# UUID Package

This package provides a UUID mapping mechanism for assigning and managing UUIDs for endpoints and node IDs.

## Package Overview

The `uuid` package allows you to maintain a mapping between unique keys and UUIDs, along with their associated configuration objects. It includes features such as generating new UUIDs, updating existing configurations, and comparing mappings to verify changes.

It also provides thread-safety by using internal locks (`sync.RWMutex`) when reading or writing the mapping.

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
  - `Logger` (\*logrus.Logger): Logger for recording messages.

## Interfaces

### `UUIDMapperInterface`

Defines the methods that a UUID mapper should implement.

- **Methods:**
  - `ApplyValidConfigurations(validKeys map[string]interface{}) error`

## Functions

### `func NewUUIDMapper(logger *logrus.Logger) *UUIDMapper`

Creates a new instance of `UUIDMapper`.

- **Parameters:**

  - `logger`: The logger for recording messages.

- **Returns:**  
  A pointer to the initialized `UUIDMapper`.

### `func (entry *UUIDEntry) EqualConfiguration(other interface{}) bool`

Compares the `Configuration` of the `UUIDEntry` with another configuration.

- **Parameters:**

  - `other`: The other configuration to compare with.

- **Returns:**  
  `true` if the configurations are equal, `false` otherwise.

### `func (entry *UUIDEntry) Equals(other UUIDEntry) bool`

Compares the current `UUIDEntry` with another `UUIDEntry`.

- **Parameters:**

  - `other`: The other `UUIDEntry` to compare with.

- **Returns:**  
  `true` if the entries are equal, `false` otherwise.

### `func (um *UUIDMapper) Equals(other map[string]UUIDEntry) bool`

Compares the current `UUIDMapper`'s `Mapping` with another map of `UUIDEntry`.

- **Parameters:**

  - `other`: The other map to compare with.

- **Returns:**  
  `true` if the mappings are equal, `false` otherwise.

### `func (um *UUIDMapper) GetUUIDEntryFromMapping(key string) (UUIDEntry, bool)`

Retrieves the `UUIDEntry` from the internal map for a given key.

- **Parameters:**

  - `key`: The key to search for.

- **Returns:**
  - `UUIDEntry`: The entry associated with the key.
  - `bool`: Indicates whether the entry exists.

### `func (um *UUIDMapper) GenerateUUID() string`

Generates a new UUID.

- **Returns:**  
  A string representing the new UUID.

### `func (um *UUIDMapper) UpsertUUIDEntry(key string, configuration interface{}) (UUIDEntry, error)`

Updates an existing `UUIDEntry` or creates a new one if it doesn't exist.

- **Parameters:**

  - `key`: The key for the `UUIDEntry`.
  - `configuration`: The configuration object.

- **Returns:**
  - `UUIDEntry`: The updated or newly created `UUIDEntry`.
  - `error`: An error if any occurred.

### `func (um *UUIDMapper) ReplaceMapping(newMap map[string]UUIDEntry)`

Replaces the entire internal `Mapping` with the provided new map.  
Access is protected by a write lock (`Lock`).

- **Parameters:**
  - `newMap`: A `map[string]UUIDEntry` that replaces the existing mapping.

### `func (um *UUIDMapper) GetMappingCopy() map[string]UUIDEntry`

Returns a copy of the internal `Mapping`.  
Access is protected by a read lock (`RLock`).

- **Returns:**  
  A copy of the mapping as `map[string]UUIDEntry`.
