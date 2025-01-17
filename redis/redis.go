/**
* Package redis provides functionalities for connecting and interacting with Redis,
* including client initialization, stream management, and message pushing.
*/
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/femogas/datalogger/appplication/configuration"
	"github.com/femogas/datalogger/redis/configuration"
	"github.com/femogas/datalogger/redis/pubsub"
	"github.com/femogas/datalogger/redis/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

/**
* Client encapsulates the Redis client and related configurations.
*/
type Client struct {
	Client        *redis.Client        // Underlying Redis client.
	Logger        *logrus.Logger       // Logger for logging messages.
	UUIDMapper    *uuid.UUIDMapper     // UUIDMapper for mapping UUIDs.
	Configuration *configuration.Redis // Redis connection settings.
	Context       context.Context      // Context for controlling operations.
}

/**
* NewClient initializes and returns a new Redis client.
*
* @param logger A logger instance for logging.
* @param ctx    The context for client operations.
* @param cancel A cancel function to stop the context.
* @return A pointer to the initialized Client and an error if any occurred.
*/
func NewClient(logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc) (*Client, error) {
	config := loadConfig()
	client, err := createRedisClient(config.RedisURL)
	if err != nil {
		return nil, err
	}
	rc := &Client{
		Client:        client,
		Logger:        logger,
		UUIDMapper:    uuid.NewUUIDMapper(logger),
		Configuration: config,
		Context:       ctx,
	}

	// Start the Redis connection monitor as a goroutine.
	go rc.redisConnectionMonitor()

	// Start the periodic UUID map synchronization as a goroutine.
	go rc.uuidMapSyncTicker()

	return rc, nil
}

/**
* loadConfig loads the Redis configuration from environment variables.
*
* @return A pointer to the Redis configuration.
*/
func loadConfig() *configuration.Redis {
	return &configuration.Redis{
		RedisURL: global.GetEnv("REDIS_URL", "redis://127.0.0.1:6379"),
	}
}

/**
* createRedisClient creates a new Redis client with the given URL.
*
* @param redisURL The Redis server URL.
* @return A pointer to the Redis client and an error if any occurred.
*/
func createRedisClient(redisURL string) (*redis.Client, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(options), nil
}

/**
* Close closes the Redis client connection after synchronizing the UUID map.
*
* @return An error if closing fails.
*/
func (rc *Client) Close() error {
	// Synchronize the UUID map before closing.
	if err := rc.syncUUIDMapToRedis(); err != nil {
		rc.Logger.WithError(err).Error("Error synchronizing uuid-map to Redis before closing")
	}

	if err := rc.Client.Close(); err != nil {
		rc.Logger.WithError(err).Error("Error closing Redis client")
		return err
	}
	return nil
}

/**
* redisConnectionMonitor monitors the Redis connection and handles reconnection.
*/
func (rc *Client) redisConnectionMonitor() {
	for {
		select {
		case <-rc.Context.Done():
			return
		default:
		}

		err := rc.checkRedisConnection()
		if err == nil {
			// Connection is healthy; no action needed.
			time.Sleep(30 * time.Second)
			continue
		}

		// Connection is lost; attempt to reconnect.
		rc.Logger.WithError(err).Error("Redis connection lost. Attempting to reconnect...")

		err = rc.retryPing(24, 2*time.Second)
		if err != nil {
			// Unable to reconnect.
			rc.Logger.WithError(err).Error("Unable to reconnect to Redis.")
			time.Sleep(30 * time.Second)
			continue
		}

		// Reconnected successfully.
		rc.Logger.Info("Redis connection reestablished.")

		// After reconnection, synchronize the UUID map.
		if err := rc.checkAndSyncUUIDMap(); err != nil {
			rc.Logger.WithError(err).Error("Error synchronizing uuid-map after reconnection")
		}

		time.Sleep(30 * time.Second)
	}
}

/**
* checkRedisConnection checks the Redis connection by sending a PING command.
*
* @return An error if the connection is lost.
*/
func (rc *Client) checkRedisConnection() error {
	return rc.Client.Ping(rc.Context).Err()
}

/**
* uuidMapSyncTicker performs periodic synchronization of the UUID map.
*/
func (rc *Client) uuidMapSyncTicker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rc.Context.Done():
			return
		case <-ticker.C:
			if err := rc.checkAndSyncUUIDMap(); err != nil {
				rc.Logger.WithError(err).Error("Error during periodic uuid-map synchronization")
			}
		}
	}
}

/**
* retryPing attempts to ping Redis multiple times with a delay between attempts.
*
* @param maxRetries The maximum number of retries.
* @param retryDelay The delay between retries.
* @return An error if unable to connect after max retries.
*/
func (rc *Client) retryPing(maxRetries int, retryDelay time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if _, err = rc.Client.Ping(rc.Context).Result(); err == nil {
			return nil
		}
		rc.Logger.WithFields(logrus.Fields{
			"retry":      i + 1,
			"maxRetries": maxRetries,
		}).WithError(err).Errorf("Error connecting to Redis. Retrying in %v...", retryDelay)
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("unable to connect to Redis after %d attempts: %w", maxRetries, err)
}

/**
* checkAndSyncUUIDMap checks the uuid-map in Redis, verifies its integrity, and synchronizes it with the in-memory UUIDMapper if necessary.
*
* @return An error if the operation fails.
*/
func (rc *Client) checkAndSyncUUIDMap() error {
	exists, err := rc.uuidMapExistsInRedis()
	if err != nil {
		return err
	}

	rc.UUIDMapper.Lock()
	defer rc.UUIDMapper.Unlock()

	if !exists {
		rc.Logger.Warn("uuid-map does not exist in Redis. Synchronizing from in-memory UUIDMapper to Redis.")
		return rc.syncUUIDMapToRedis()
	}

	redisUUIDMap, err := rc.loadUUIDMapFromRedis()
	if err != nil {
		return err
	}

	if err := rc.verifyAndProcessUUIDMapDifferences(redisUUIDMap); err != nil {
		return err
	}

	return nil
}

/**
* uuidMapExistsInRedis checks if the uuid-map exists in Redis.
*
* @return A boolean indicating existence and an error if any occurred.
*/
func (rc *Client) uuidMapExistsInRedis() (bool, error) {
	exists, err := rc.Client.Exists(rc.Context, "uuid-map").Result()
	if err != nil {
		return false, fmt.Errorf("error checking existence of uuid-map in Redis: %w", err)
	}
	return exists > 0, nil
}

/**
* loadUUIDMapFromRedis loads the uuid-map from Redis.
*
* @return A map of the UUID mappings and an error if the operation fails.
*/
func (rc *Client) loadUUIDMapFromRedis() (map[string]uuid.UUIDEntry, error) {
	result, err := rc.Client.HGetAll(rc.Context, "uuid-map").Result()
	if err != nil {
		return nil, fmt.Errorf("error loading uuid-map from Redis: %w", err)
	}

	create := make(map[string]uuid.UUIDEntry)
	for key, value := range result {
		var entry uuid.UUIDEntry
		if err := json.Unmarshal([]byte(value), &entry); err != nil {
			rc.Logger.WithError(err).Errorf("Error deserializing UUIDEntry for key %s", key)
			continue
		}
		create[key] = entry
	}
	return create, nil
}

/**
* verifyAndProcessUUIDMapDifferences compares the in-memory UUIDMapper with the Redis uuid-map, and processes differences if found.
*
* @param redisUUIDMap The uuid-map loaded from Redis.
* @return An error if the operation fails.
*/
func (rc *Client) verifyAndProcessUUIDMapDifferences(redisUUIDMap map[string]uuid.UUIDEntry) error {
	if rc.UUIDMapper.Equals(redisUUIDMap) {
		rc.Logger.Info("uuid-map in Redis is consistent with in-memory UUIDMapper. No action required.")
		return nil
	}

	rc.Logger.Warn("Discrepancies found between in-memory UUIDMapper and Redis uuid-map. Processing differences.")

	// Overwrite the uuid-map in Redis with the in-memory UUIDMapper
	if err := rc.syncUUIDMapToRedis(); err != nil {
		return fmt.Errorf("error synchronizing uuid-map to Redis: %w", err)
	}
	return nil
}

/**
* syncUUIDMapToRedis synchronizes the uuid-map from the in-memory UUIDMapper to Redis.
*
* @return An error if the operation fails.
*/
func (rc *Client) syncUUIDMapToRedis() error {
	if len(rc.UUIDMapper.Mapping) == 0 {
		rc.Logger.Warn("UUIDMapper is empty; nothing to synchronize to Redis.")
		return nil
	}

	pipeline := rc.Client.Pipeline()
	if err := rc.addUUIDMapToPipeline(pipeline); err != nil {
		return err
	}

	if _, err := pipeline.Exec(rc.Context); err != nil {
		return fmt.Errorf("error synchronizing uuid-map to Redis: %w", err)
	}

	rc.Logger.Info("uuid-map synchronized to Redis successfully.")
	return nil
}

/**
* addUUIDMapToPipeline adds the UUIDMapper's mappings to the provided pipeline.
*
* @param pipeline The Redis pipeline.
* @return An error if the operation fails.
*/
func (rc *Client) addUUIDMapToPipeline(pipeline redis.Pipeliner) error {
	for key, entry := range rc.UUIDMapper.Mapping {
		if err := rc.addUUIDEntryToPipeline(pipeline, key, entry); err != nil {
			return err
		}
	}
	return nil
}

/**
* addUUIDEntryToPipeline adds a UUIDEntry HSet command to the provided pipeline.
*
* @param pipeline  The Redis pipeline.
* @param key   The key of the entry.
* @param entry The UUIDEntry to add.
* @return An error if the operation fails.
*/
func (rc *Client) addUUIDEntryToPipeline(pipeline redis.Pipeliner, key string, entry uuid.UUIDEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error serializing UUIDEntry: %w", err)
	}
	pipeline.HSet(rc.Context, "uuid-map", key, data)
	return nil
}

/**
* PushToRedis adds data to a specified Redis stream with approximate pruning.
*
* @param uuid The name of the Redis stream.
* @param max  The maximum number of records to retain in the stream.
* @param data The data to be added to the stream.
* @return An error if the operation fails.
*/
func (rc *Client) PushToRedis(uuid string, max int64, data map[string]interface{}) error {
	data = rc.addTimestamp(data)
	return rc.addToStream(uuid, max, data)
}

/**
* addTimestamp adds a timestamp to the data map.
*
* @param data The data map to which the timestamp is added.
* @return The updated data map with the timestamp.
*/
func (rc *Client) addTimestamp(data map[string]interface{}) map[string]interface{} {
	data["timestamp"] = time.Now().Format(time.RFC3339)
	return data
}

/**
* addToStream adds the data to the specified Redis stream with approximate pruning.
*
* @param uuid The name of the Redis stream.
* @param max  The maximum number of records to retain in the stream.
* @param data The data to be added to the stream.
* @return An error if the operation fails.
*/
func (rc *Client) addToStream(uuid string, max int64, data map[string]interface{}) error {
	_, err := rc.Client.XAdd(rc.Context, &redis.XAddArgs{
		Stream: uuid,
		ID:     "*",
		Values: data,
		MaxLen: max,
	}).Result()
	if err != nil {
		rc.Logger.WithError(err).Error("Error adding message to stream")
	}
	return err
}

/**
* GenerateUUIDMap verifies the current UUID mappings against the provided configurations.
*
* @param createValidKeysFunc A function that generates a set of valid keys as map[string]interface{}.
* @return An error if the operation fails.
*/
func (rc *Client) GenerateUUIDMap(createValidKeysFunc func() map[string]interface{}) error {
	validKeys := createValidKeysFunc()
	if err := rc.applyValidConfigurations(validKeys); err != nil {
		return fmt.Errorf("error applying valid configurations: %w", err)
	}

	return nil
}

/**
* applyValidConfigurations updates the configurations of existing valid keys in the map by calling
* the UUIDMapper's method to get the UUIDEntry to send to Redis.
*
* @param validKeys The map of valid keys with their respective configuration objects.
* @return An error if the operation fails.
*/
func (rc *Client) applyValidConfigurations(validKeys map[string]interface{}) error {
	mapping, err := rc.loadUUIDMapFromRedis()
	if err != nil {
		return err
	}
	rc.UUIDMapper.Mapping = mapping

	pipeline := rc.Client.Pipeline()

	for key, configuration := range validKeys {
		// Call the method in UUIDMapper to get the UUIDEntry
		entry, err := rc.UUIDMapper.UpsertUUIDEntry(key, configuration)
		if err != nil {
			rc.Logger.WithError(err).Errorf("Error processing key %s", key)
			return err
		}

		// Add the UUIDEntry to the pipeline to be sent to Redis
		if err := rc.addUUIDEntryToPipeline(pipeline, key, entry); err != nil {
			return err
		}
	}

	// Execute the pipeline after adding all commands.
	if _, err := pipeline.Exec(rc.Context); err != nil {
		return fmt.Errorf("error executing pipeline: %w", err)
	}

	return nil
}

/**
* ListenForCommands subscribes to the Redis channel and listens for start and stop commands.
*
* @param startFunc The function to execute when a start command is received.
* @param stopFunc  The function to execute when a stop command is received.
*/
func (rc *Client) ListenForCommands(startFunc, stopFunc func() error) {
	ps := pubsub.NewPubSub(rc.Logger)
	ctx := rc.Context

	pubsubClient := rc.Client.Subscribe(ctx, "remote-control")
	defer pubsubClient.Close()

	ch := pubsubClient.Channel()
	handlers := map[string]pubsub.CommandHandler{
		"start": startFunc,
		"stop":  stopFunc,
	}

	for {
		select {
		case <-ctx.Done():
			ps.Logger.Info("Stopping command listener")
			return
		case msg := <-ch:
			ps.HandleCommand(msg.Payload, handlers)
		}
	}
}