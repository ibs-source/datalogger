/**
 * Package uuid provides a UUID mapping mechanism to assign and manage UUIDs for endpoints and node IDs.
 */
package uuid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// UUIDEntry represents an entry in the UUID map, containing the UUID and the associated configuration object.
type UUIDEntry struct {
	UUID          string      `json:"uuid"`          // UUID associated with the key
	Configuration interface{} `json:"configuration"` // Associated configuration object
}

// UUIDMapperInterface defines the methods that a UUID mapper should implement.
type UUIDMapperInterface interface {
	ApplyValidConfigurations(validKeys map[string]interface{}) error
}

// UUIDMapper is a thread-safe structure that maintains a map between keys and UUIDEntries.
// It uses a Logger for logging messages.
type UUIDMapper struct {
	sync.RWMutex
	Mapping map[string]UUIDEntry // Mapping stores key-UUIDEntry pairs
	Logger  *logrus.Logger       // Logger for recording messages
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
 * marshalCanonical creates a deterministic JSON representation of a value.
 * The function normalizes the structure by sorting map keys and preserving array order.
 *
 * @param v The value to convert to canonical JSON.
 * @return A byte slice containing the canonical JSON representation.
 * @return An error if serialization fails.
 */
func marshalCanonical(v interface{}) ([]byte, error) {
    // First pass: marshal to get a generic JSON representation
    rawJSON, err := json.Marshal(v)
    if err != nil {
        return nil, fmt.Errorf("initial marshaling failed: %w", err)
    }
    // Unmarshal into an interface{} to get a generic structure
    var decoded interface{}
    if err := json.Unmarshal(rawJSON, &decoded); err != nil {
        return nil, fmt.Errorf("unmarshaling failed: %w", err)
    }
    // Apply normalization to create a deterministic structure
    normalized := normalizeValue(decoded)
    // Final marshal of the normalized structure
    result, err := json.Marshal(normalized)
    if err != nil {
        return nil, fmt.Errorf("final marshaling failed: %w", err)
    }
    return result, nil
}

/**
 * normalizeValue recursively normalizes an interface{} value.
 * Maps have their keys sorted, arrays preserve order but normalize their elements.
 *
 * @param v The value to normalize.
 * @return A normalized version of the input value.
 */
func normalizeValue(v interface{}) interface{} {
    switch val := v.(type) {
    case map[string]interface{}:
        // Get a sorted list of keys
        keys := make([]string, 0, len(val))
        for k := range val {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        // Create a new map with sorted keys
        normalized := make(map[string]interface{}, len(val))
        for _, k := range keys {
            normalized[k] = normalizeValue(val[k])
        }
        return normalized
    case []interface{}:
        // Normalize each element in the array
        normalized := make([]interface{}, len(val))
        for i, elem := range val {
            normalized[i] = normalizeValue(elem)
        }
        return normalized
    default:
        // Primitive types are returned as is
        return val
    }
}

/**
 * EqualConfiguration compares the Configuration of the UUIDEntry with another configuration.
 * It generates a "canonical JSON" for both configurations and compares them.
 *
 * @param other The other configuration to compare with.
 * @return True if the configurations are effectively equal, false otherwise.
 */
func (entry *UUIDEntry) EqualConfiguration(other interface{}) bool {
	entryJSON, entryErr := marshalCanonical(entry.Configuration)
	otherJSON, otherErr := marshalCanonical(other)
	if entryErr != nil || otherErr != nil {
		return false
	}
	return bytes.Equal(entryJSON, otherJSON)
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
 * Equals compares the current UUIDMapper's Mapping with another map of UUIDEntry.
 *
 * @param other The other map to compare with.
 * @return True if the mappings are equal, false otherwise.
 */
func (um *UUIDMapper) Equals(other map[string]UUIDEntry) bool {
	um.RLock()
	defer um.RUnlock()
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
	// Generate new UUID for new keys.
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

/**
 * ReplaceMapping replaces the entire internal Mapping with the provided newMap,
 * protecting the access with a Lock().
 *
 * @param newMap The new map of UUIDEntry to set.
 */
func (um *UUIDMapper) ReplaceMapping(newMap map[string]UUIDEntry) {
	um.Lock()
	defer um.Unlock()
	um.Mapping = newMap
}

/**
 * GetMappingCopy returns a copy of the internal map, protecting the access with RLock().
 *
 * @return A copy of the internal map of UUIDEntries.
 */
func (um *UUIDMapper) GetMappingCopy() map[string]UUIDEntry {
	um.RLock()
	defer um.RUnlock()
	copyMap := make(map[string]UUIDEntry, len(um.Mapping))
	for k, v := range um.Mapping {
		copyMap[k] = v
	}
	return copyMap
}