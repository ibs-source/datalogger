/**
* Package uuid provides a UUID mapping mechanism to assign and manage UUIDs for endpoints and node IDs.
*/
package uuid

import (
    "context"
    "fmt"
    "sync"

    "github.com/google/uuid"
    "github.com/femogas/datalogger/redis"
    "github.com/sirupsen/logrus"
)

/**
* UUIDMapper is a thread-safe structure that maintains a mapping between keys and UUIDs.
*/
type UUIDMapper struct {
    sync.RWMutex
    Mapping map[string]string // Mapping stores the key-UUID pairs.
    Logger  *logrus.Logger    // Logger for logging messages.
    Redis   *redis.Client     // Redis client for persistence.
}

/**
* NewUUIDMapper creates a new UUIDMapper instance.
*
* @param client The Redis client for persistence.
* @param logger The logger for logging messages.
* @return A pointer to the initialized UUIDMapper.
*/
func NewUUIDMapper(client *redis.Client, logger *logrus.Logger) *UUIDMapper {
    return &UUIDMapper{
        Mapping: make(map[string]string),
        Redis:   client,
        Logger:  logger,
    }
}

/**
* GetOrCreateUUID retrieves an existing UUID for the given endpoint and ID,
* or creates a new one if it doesn't exist.
*
* @param endpoint The endpoint string.
* @param ID   The node ID string.
* @return The UUID string and an error if any occurred.
*/
func (um *UUIDMapper) GetOrCreateUUID(endpoint, ID string) (string, error) {
    key := endpoint + ":" + ID
    if uuidStr, exists := um.getUUIDFromMapping(key); exists {
        return uuidStr, nil
    }
    return um.createUUIDForKey(key)
}

/**
* getUUIDFromMapping retrieves the UUID from the internal mapping for a given key.
*
* @param key The key to look up.
* @return The UUID string and a boolean indicating if it exists.
*/
func (um *UUIDMapper) getUUIDFromMapping(key string) (string, bool) {
    um.RLock()
    uuidStr, exists := um.Mapping[key]
    um.RUnlock()
    return uuidStr, exists
}

/**
* createUUIDForKey generates a new UUID for a given key and saves it to the mapping and Redis.
*
* @param key The key for which to create a UUID.
* @return The new UUID string and an error if any occurred.
*/
func (um *UUIDMapper) createUUIDForKey(key string) (string, error) {
    um.Lock()
    // Double-check to avoid race conditions.
    if uuidStr, exists := um.Mapping[key]; exists {
        um.Unlock()
        return uuidStr, nil
    }
    newUUID := uuid.New().String()
    um.Mapping[key] = newUUID
    um.Unlock()

    // Save the new mapping to Redis.
    if err := um.SaveMappingToRedis(key, newUUID); err != nil {
        return "", err
    }
    um.Logger.WithFields(logrus.Fields{
        "key":  key,
        "uuid": newUUID,
    }).Info("Created new UUID mapping")
    return newUUID, nil
}

/**
* SaveMappingToRedis saves a key-UUID pair to Redis.
*
* @param key     The key to save.
* @param uuidStr The UUID associated with the key.
* @return An error if the operation fails.
*/
func (um *UUIDMapper) SaveMappingToRedis(key, uuidStr string) error {
    return um.Redis.Client.HSet(context.Background(), "uuid_map", key, uuidStr).Err()
}

/**
* LoadMappingFromRedis loads the UUID mapping from Redis into memory.
*
* @return An error if the operation fails.
*/
func (um *UUIDMapper) LoadMappingFromRedis() error {
    result, err := um.Redis.Client.HGetAll(context.Background(), "uuid_map").Result()
    if err != nil {
        return err
    }
    um.Lock()
    um.Mapping = result
    um.Unlock()
    return nil
}

/*
* VerifyAndCleanUUIDMap checks the current UUID mappings against the provided configurations,
* removes invalid entries, and resolves duplicate UUIDs.
*
* @param createValidKeysFunc A function that generates a set of valid keys as map[string]struct{}.
* @return An error if the operation fails.
*/
func (um *UUIDMapper) VerifyAndCleanUUIDMap(createValidKeysFunc func() map[string]struct{}) error {
    // Create a set of valid keys using the provided function.
    validKeys := createValidKeysFunc()

    // Retrieve the current mappings from Redis.
    currentMapping, err := um.Redis.Client.HGetAll(context.Background(), "uuid_map").Result()
    if err != nil {
        return fmt.Errorf("error loading UUID mapping from Redis: %w", err)
    }

    // Remove any invalid entries.
    um.removeInvalidEntries(currentMapping, validKeys)

    // Get the count of UUID occurrences to detect duplicates.
    uuidCount := um.getUUIDCount()

    // Resolve any duplicate UUIDs.
    um.resolveDuplicateUUIDs(uuidCount)

    return nil
}

/**
* removeInvalidEntries removes entries from the mapping that are not in the set of valid keys.
*
* @param currentMapping The current key-UUID mappings.
* @param validKeys      The set of valid keys.
*/
func (um *UUIDMapper) removeInvalidEntries(currentMapping map[string]string, validKeys map[string]struct{}) {
    for key, uuidStr := range currentMapping {
        if _, exists := validKeys[key]; exists {
            continue
        }
        // Remove the invalid key from Redis.
        if err := um.Redis.Client.HDel(context.Background(), "uuid_map", key).Err(); err != nil {
            um.Logger.WithFields(logrus.Fields{
                "key":   key,
                "uuid":  uuidStr,
                "error": err,
            }).Error("Error removing entry from UUID mapping")
            continue
        }
        um.Logger.WithFields(logrus.Fields{
            "key":  key,
            "uuid": uuidStr,
        }).Info("Removed invalid UUID mapping")

        // Remove the invalid key from the in-memory mapping.
        um.Lock()
        delete(um.Mapping, key)
        um.Unlock()
    }
}

/**
* getUUIDCount counts the occurrences of each UUID to detect duplicates.
*
* @return A map where keys are UUIDs and values are their counts.
*/
func (um *UUIDMapper) getUUIDCount() map[string]int {
    uuidCount := make(map[string]int)
    um.RLock()
    for _, uuidStr := range um.Mapping {
        uuidCount[uuidStr]++
    }
    um.RUnlock()
    return uuidCount
}

/**
* resolveDuplicateUUIDs resolves duplicate UUIDs by generating new UUIDs for keys with duplicates.
*
* @param uuidCount A map of UUID counts to identify duplicates.
*/
func (um *UUIDMapper) resolveDuplicateUUIDs(uuidCount map[string]int) {
    for key, uuidStr := range um.Mapping {
        if uuidCount[uuidStr] <= 1 {
            continue
        }
        // Generate a new UUID to replace the duplicate.
        newUUID := uuid.New().String()
        if err := um.SaveMappingToRedis(key, newUUID); err != nil {
            um.Logger.WithFields(logrus.Fields{
                "key":     key,
                "oldUUID": uuidStr,
                "newUUID": newUUID,
                "error":   err,
            }).Error("Error updating UUID mapping to avoid duplicates")
            continue
        }

        // Update the in-memory mapping with the new UUID.
        um.Lock()
        um.Mapping[key] = newUUID
        um.Unlock()

        um.Logger.WithFields(logrus.Fields{
            "key":     key,
            "oldUUID": uuidStr,
            "newUUID": newUUID,
        }).Info("Updated UUID mapping to avoid duplicate")

        // Update the UUID counts.
        uuidCount[uuidStr]--
        uuidCount[newUUID]++
    }
}
