/**
* Package uuid provides a UUID mapping mechanism to assign and manage UUIDs for endpoints and node IDs.
*/
package uuid

import (
    "sync"
    "crypto/sha256"
    
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
    "github.com/gibson042/canonicaljson-go"
)

/**
* UUIDEntry represents an entry in the UUID map, containing the UUID and the associated configuration object.
*/
type UUIDEntry struct {
    UUID          string      `json:"uuid"`          // UUID associated with the key
    Configuration interface{} `json:"configuration"` // Associated configuration object
}

/**
* UUIDMapperInterface defines the methods that a UUID mapper should implement.
*/
type UUIDMapperInterface interface {
    ApplyValidConfigurations(validKeys map[string]interface{}) error
}

/**
* UUIDMapper is a thread-safe structure that maintains a map between keys and UUIDEntries.
* It uses a Logger for logging messages.
*/
type UUIDMapper struct {
    sync.RWMutex
    Mapping map[string]UUIDEntry // Mapping stores key-UUIDEntry pairs
    Logger  *logrus.Logger       // Logger for recording messages
}

/**
* EqualConfiguration compares the Configuration of the UUIDEntry with another configuration.
*
* @param other The other configuration to compare with.
* @return True if the configurations are equal, false otherwise.
*/
func (entry *UUIDEntry) EqualConfiguration(other interface{}) bool {
    entryJSON, entryErr := canonicaljson.Marshal(entry.Configuration)
    otherJSON, otherErr := canonicaljson.Marshal(other)
    if entryErr != nil || otherErr != nil {
        return false
    }
    entryHash := sha256.Sum256(entryJSON)
    otherHash := sha256.Sum256(otherJSON)
    return entryHash == otherHash
}

/**
* Equals compares the current UUIDEntry with another UUIDEntry.
*
* @param other The other UUIDEntry to compare with.
* @return True if the entries are equal, false otherwise.
*/
func (entry *UUIDEntry) Equals(other UUIDEntry) bool {
	if entry.UUID != other.UUID {
		return false
	}
	return entry.EqualConfiguration(other.Configuration)
}

/**
* NewUUIDMapper creates a new instance of UUIDMapper.
*
* @param logger The logger for recording messages.
* @return A pointer to the initialized UUIDMapper.
*/
func NewUUIDMapper(logger *logrus.Logger) *UUIDMapper {
    return &UUIDMapper{
        Mapping: make(map[string]UUIDEntry),
        Logger:  logger,
    }
}

/**
* Equals compares the current UUIDMapper's Mapping with another map of UUIDEntry.
*
* @param other The other map to compare with.
* @return True if the mappings are equal, false otherwise.
*/
func (um *UUIDMapper) Equals(other map[string]UUIDEntry) bool {
	if len(um.Mapping) != len(other) {
		return false
	}
	for key, entry := range um.Mapping {
		if otherEntry, exists := other[key]; !exists || !entry.Equals(otherEntry) {
			return false
		}
	}
	return true
}

/**
* GetUUIDEntryFromMapping retrieves the UUIDEntry from the internal map for a given key.
*
* @param key The key to search for.
* @return The UUIDEntry and a boolean indicating if it exists.
*/
func (um *UUIDMapper) GetUUIDEntryFromMapping(key string) (UUIDEntry, bool) {
    um.RLock()
    defer um.RUnlock()
    entry, exists := um.Mapping[key]
    return entry, exists
}

/**
* GenerateUUID generates a new UUID.
*
* @return A string representing the new UUID.
*/
func (um *UUIDMapper) GenerateUUID() string {
    return uuid.New().String()
}

/**
* UpsertUUIDEntry updates an existing UUIDEntry or creates a new one if it doesn't exist.
*
* @param key           The key for the UUIDEntry.
* @param configuration The configuration object.
* @return The updated or newly created UUIDEntry, and an error if any occurred.
*/
func (um *UUIDMapper) UpsertUUIDEntry(key string, configuration interface{}) (UUIDEntry, error) {
	um.Lock()
	defer um.Unlock()

	if entry, exists := um.Mapping[key]; exists {
		if !entry.EqualConfiguration(configuration) {
			entry.Configuration = configuration
			um.Mapping[key] = entry
			um.Logger.WithField("key", key).Info("Updated configuration for existing UUIDEntry")
		}
		return entry, nil
	}

	// Generate new UUIDs for new keys.
	uuidStr := um.GenerateUUID()
	newEntry := UUIDEntry{
		UUID:          uuidStr,
		Configuration: configuration,
	}

	um.Mapping[key] = newEntry
	um.Logger.WithFields(logrus.Fields{
		"key":  key,
		"uuid": uuidStr,
	}).Info("Generated new UUIDEntry")

	return newEntry, nil
}