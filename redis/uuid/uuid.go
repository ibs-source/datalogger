/**
* Package uuid provides a UUID mapping mechanism to assign and manage UUIDs for endpoints and node IDs.
*/
package uuid

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"

    "github.com/google/uuid"
    "github.com/femogas/datalogger/redis"
    "github.com/sirupsen/logrus"
)

/**
* UUIDEntry represents an entry in the UUID map, containing the UUID and the associated configuration object.
*/
type UUIDEntry struct {
    UUID          string      `json:"uuid"`          // UUID associated with the key
    Configuration interface{} `json:"configuration"` // Associated configuration object
}

/**
* UUIDMapper is a thread-safe structure that maintains a map between keys and UUIDEntries.
* It uses Redis for persistence and Logrus for logging.
*/
type UUIDMapper struct {
    sync.RWMutex
    Mapping map[string]UUIDEntry // Mapping stores key-UUIDEntry pairs
    Logger  *logrus.Logger      // Logger for recording messages
    Redis   *redis.Client       // Redis client for persistence
}

/**
* NewUUIDMapper creates a new instance of UUIDMapper.
*
* @param client The Redis client for persistence.
* @param logger The logger for recording messages.
* @return A pointer to the initialized UUIDMapper.
*/
func NewUUIDMapper(client *redis.Client, logger *logrus.Logger) *UUIDMapper {
    return &UUIDMapper{
        Mapping: make(map[string]UUIDEntry),
        Redis:   client,
        Logger:  logger,
    }
}

/**
* GetUUIDEntryFromMapping retrieves the UUIDEntry from the internal map for a given key.
*
* @param key The key to search for.
* @return The UUIDEntry and a boolean indicating if it exists.
*/
func (um *UUIDMapper) GetUUIDEntryFromMapping(key string) (UUIDEntry, bool) {
    um.RLock()
    entry, exists := um.Mapping[key]
    um.RUnlock()
    return entry, exists
}

/**
* SaveMappingToRedis saves a key-UUIDEntry pair to Redis as a JSON string.
*
* @param key   The key to save.
* @param entry The UUIDEntry associated with the key.
* @return An error if the operation fails.
*/
func (um *UUIDMapper) SaveMappingToRedis(key string, entry UUIDEntry) error {
    data, err := json.Marshal(entry)
    if err != nil {
        return fmt.Errorf("error serializing UUIDEntry: %w", err)
    }
    return um.Redis.Client.HSet(context.Background(), "uuid-map", key, data).Err()
}

/**
* GenerateUUIDMap verifies the current UUID mappings against the provided configurations.
*
* @param createValidKeysFunc A function that generates a set of valid keys as map[string]interface{}.
* @return An error if the operation fails.
*/
func (um *UUIDMapper) GenerateUUIDMap(createValidKeysFunc func() map[string]interface{}) error {
    // Create a set of valid keys using the provided function
    validKeys := createValidKeysFunc()

    // Apply the valid configurations to the mappings
    if err := um.applyValidConfigurations(validKeys); err != nil {
        return fmt.Errorf("error applying valid configurations: %w", err)
    }

    return nil
}

/**
* loadCurrentMapping loads the current UUID map from Redis.
*
* @return The map of UUIDEntries and an error if the operation fails.
*/
func (um *UUIDMapper) loadCurrentMapping() (map[string]UUIDEntry, error) {
    currentMapping, err := um.Redis.Client.HGetAll(context.Background(), "uuid-map").Result()
    if err != nil {
        return nil, fmt.Errorf("error loading UUID map from Redis: %w", err)
    }
    mapping := make(map[string]UUIDEntry)
    for key, value := range currentMapping {
        var entry UUIDEntry
        if err := json.Unmarshal([]byte(value), &entry); err != nil {
            return nil, fmt.Errorf("error deserializing key %s: %w", key, err)
        }
        mapping[key] = entry
    }

    return mapping, nil
}

/**
* generateUUID generates a new UUID.
*
* @return A string representing the new UUID.
*/
func (um *UUIDMapper) generateUUID() string {
    return uuid.New().String()
}

/**
* applyValidConfigurations updates the configurations of existing valid keys in the map.
*
* @param validKeys The map of valid keys with their respective configuration objects.
* @return An error if the operation fails.
*/
func (um *UUIDMapper) applyValidConfigurations(validKeys map[string]interface{}) error {
    um.Lock()
    defer um.Unlock()

    currentMapping, err := um.loadCurrentMapping()
    if err != nil {
        return err
    }

    for key, configuration := range validKeys {
        if value, exists := currentMapping[key]; exists {
            um.Mapping[key] = value
            continue
        }
        uuid := um.generateUUID()
        updatedEntry := UUIDEntry{
            UUID:          uuid,
            Configuration: configuration,
        }
        um.Mapping[key] = updatedEntry

        // Save the updated UUIDEntry to Redis
        if err := um.SaveMappingToRedis(key, updatedEntry); err != nil {
            um.Logger.WithFields(logrus.Fields{
                "key":   key,
                "error": err,
            }).Error("Error updating UUIDEntry configuration in Redis")
            return err
        }

        um.Logger.WithFields(logrus.Fields{
            "key":  key,
            "uuid": uuid,
        }).Info("Generated configuration for UUID")
    }

    return nil
}
