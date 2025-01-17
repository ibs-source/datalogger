package uuid

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

/**
 * Package uuid provides a UUID mapping mechanism to assign and manage UUIDs for endpoints and node IDs.
 */

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
 * normalizeValue recursively normalizes an interface{}.
 * - If v is a map[string]interface{}, it sorts the keys and normalizes each value.
 * - If v is a []interface{}, it normalizes each element.
 * - Otherwise, it returns the primitive value (string, float64, bool, nil, etc.) as is.
 */
func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// Sort the keys to ensure deterministic output
		sortedKeys := make([]string, 0, len(val))
		for k := range val {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		normalized := make(map[string]interface{}, len(val))
		for _, k := range sortedKeys {
			normalized[k] = normalizeValue(val[k])
		}
		return normalized

	case []interface{}:
		normalized := make([]interface{}, len(val))
		for i, elem := range val {
			normalized[i] = normalizeValue(elem)
		}
		return normalized

	default:
		// Primitive type (string, float64, bool, nil, etc.)
		return val
	}
}

/**
 * marshalCanonical performs:
 * 1) A first json.Marshal to get a byte representation of v.
 * 2) A json.Unmarshal into an interface{} to convert to a generic structure.
 * 3) normalizeValue on the resulting structure to get a deterministic layout.
 * 4) A final json.Marshal of the normalized structure.
 */
func marshalCanonical(v interface{}) ([]byte, error) {
	// First pass: get a generic JSON representation
	tmp, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Unmarshal into an interface{}
	var decoded interface{}
	if err := json.Unmarshal(tmp, &decoded); err != nil {
		return nil, err
	}

	// Recursively normalize the structure
	normalized := normalizeValue(decoded)

	// Final Marshal of the normalized structure
	return json.Marshal(normalized)
}

/**
 * EqualConfiguration compares the Configuration of the UUIDEntry with another configuration.
 * It generates a "canonical JSON" for both configurations (recursively normalized) and compares their SHA256 hash.
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
