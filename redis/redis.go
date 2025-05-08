/**
 * Package redis provides functionalities for connecting and interacting with Redis,
 * including client initialization, stream management, and message pushing.
 */
package redis

import (
	"errors"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ibs-source/datalogger/application/configuration"
	"github.com/ibs-source/datalogger/redis/configuration"
	"github.com/ibs-source/datalogger/redis/pubsub"
	"github.com/ibs-source/datalogger/redis/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// streamItem encapsulates a Redis Stream XADD command for async batching.
type streamItem struct {
	Stream string
	MaxLen int64
	Values map[string]interface{}
}

/**
 * Client encapsulates the Redis client and related configurations.
 */
type Client struct {
	Client        *redis.Client        // Underlying Redis client.
	Logger        *logrus.Logger       // Logger for logging messages.
	UUIDMapper    *uuid.UUIDMapper     // UUIDMapper for mapping UUIDs.
	Configuration *configuration.Redis // Redis connection settings.
	PubSub        *redis.PubSub        // PubSub client for listening to commands.
	Context       context.Context      // Context for controlling operations.
	// Async batching fields
	asyncCh           chan streamItem  // Channel for async streamItems
	asyncWg           sync.WaitGroup   // WaitGroup for async publisher
	asyncBatchSize    int              // Number of items per batch
	asyncBatchTimeout time.Duration    // Max time before auto-flush
}


/**
 * NewClient initializes and returns a new Redis client.
 *
 * @param logger A logger instance for logging.
 * @param ctx    The context for client operations.
 * @return A pointer to the initialized Client and an error if any occurred.
 */
func NewClient(logger *logrus.Logger, ctx context.Context) (*Client, error) {
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
	if len(rc.UUIDMapper.GetMappingCopy()) > 0 {
    	go rc.uuidMapSyncTicker()
  	}
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
 * Includes timeouts to prevent blocking indefinitely during shutdown.
 *
 * @return An error if closing fails.
 */
func (rc *Client) Close() error {
    // Create a timeout context to prevent blocking forever
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    rc.Logger.Debug("Stopping Pub/Sub subscription...")
    rc.stopPubSub()
    // Synchronize the UUID map before closing.
    if len(rc.UUIDMapper.GetMappingCopy()) > 0 {
        rc.Logger.Debug("Synchronizing uuid-map to Redis (final sync)...")
        if err := rc.syncUUIDMapToRedisWithTimeout(ctx); err != nil {
            rc.Logger.WithError(err).Error("Error synchronizing uuid-map to Redis before closing")
        }
    }
    // Disable async batching if enabled
    rc.disableAsyncBatchingWithTimeout(ctx)
    rc.Logger.Debug("Closing Redis client...")
    err := rc.closeRedisClientWithTimeout(ctx)
    if err != nil {
        rc.Logger.WithError(err).Error("Error closing Redis client")
        return err
    }
    return nil
}

/**
 * closeRedisClientWorker performs the actual client closing and sends the result to the provided channel.
 *
 * @param resultChan    The channel to send the closing result to.
 */
func (rc *Client) closeRedisClientWorker(resultChan chan<- error) {
    resultChan <- rc.Client.Close()
}

/**
 * closeRedisClientWithTimeout closes the Redis client with a timeout.
 *
 * @param ctx    The context with timeout for the operation.
 * @return       An error if closing fails or times out.
 */
func (rc *Client) closeRedisClientWithTimeout(ctx context.Context) error {
    // Create channel for completion signal
    closeErr := make(chan error, 1)
    // Start closing in a separate goroutine
    go rc.closeRedisClientWorker(closeErr)
    // Wait for completion or timeout
    select {
    case err := <-closeErr:
        return err
    case <-ctx.Done():
        return fmt.Errorf("redis client close timed out: %w", ctx.Err())
    }
}

/**
 * syncUUIDMapToRedisWithTimeout synchronizes the UUID map to Redis with a timeout.
 *
 * @param ctx    The context with timeout for the operation.
 * @return       An error if synchronization fails or times out.
 */
func (rc *Client) syncUUIDMapToRedisWithTimeout(ctx context.Context) error {
    // Create channel for completion signal
    syncDone := make(chan error, 1)
    // Start synchronization in a separate goroutine
    go rc.syncToRedisWorker(syncDone)
    // Wait for completion or timeout
    select {
    case err := <-syncDone:
        return err
    case <-ctx.Done():
        return fmt.Errorf("UUID map sync timed out during shutdown: %w", ctx.Err())
    }
}

/**
 * syncToRedisWorker performs the actual synchronization and sends the result to the provided channel.
 *
 * @param resultChan    The channel to send the synchronization result to.
 */
func (rc *Client) syncToRedisWorker(resultChan chan<- error) {
    resultChan <- rc.syncUUIDMapToRedis()
}

/**
 * disableAsyncBatchingWithTimeout disables async batching with a timeout.
 *
 * @param ctx    The context with timeout for the operation.
 */
func (rc *Client) disableAsyncBatchingWithTimeout(ctx context.Context) {
    if rc.asyncCh == nil {
        return
    }
    // Close the channel to signal the publisher to stop
    close(rc.asyncCh)
    // Create channel for completion signal
    done := make(chan struct{})
    // Start waiting in a separate goroutine
    go rc.waitForAsyncBatching(done)
    // Wait for completion or timeout
    select {
    case <-done:
        // WaitGroup completed normally
        rc.Logger.Debug("Async batching disabled successfully")
    case <-ctx.Done():
        rc.Logger.Warn("Timed out waiting for async batch processing to complete")
    }
    rc.asyncCh = nil
}

/**
 * waitForAsyncBatching waits for the async batching to complete and signals on the provided channel.
 *
 * @param doneChan    The channel to signal when waiting is complete.
 */
func (rc *Client) waitForAsyncBatching(doneChan chan<- struct{}) {
    rc.asyncWg.Wait()
    close(doneChan)
}

/**
 * cancellableDelay sleeps for the given duration or returns early if the context is canceled.
 * Returns true if the full duration was waited, false if interrupted by context cancellation.
 */
func (rc *Client) cancellableDelay(duration time.Duration) bool {
    select {
    case <-time.After(duration):
        return true
    case <-rc.Context.Done():
        // Important: don't log context cancellation here to avoid duplicate logs
        return false
    }
}

/**
 * isContextCanceled checks if the context is canceled and logs a message if it is.
 *
 * @param message The message to log if the context is canceled.
 * @return True if the context is canceled, false otherwise.
 */
func (rc *Client) isContextCanceled(message string) bool {
    if rc.Context.Err() != nil {
        rc.Logger.Info(message)
        return true
    }
    return false
}

/**
 * stopPubSub closes the Pub/Sub subscription, if any.
 */
func (rc *Client) stopPubSub() {
	if rc.PubSub == nil {
		return
	}
	rc.PubSub.Close()
	rc.PubSub = nil
}

/**
 * redisConnectionMonitor monitors the Redis connection and handles reconnection.
 */
func (rc *Client) redisConnectionMonitor() {
    consecutiveFailures := 0
    for {
        if rc.isContextCanceled("Redis monitor stopping (context canceled)") {
            return
        }
        if rc.handleConnectionCheck(&consecutiveFailures) || rc.handleReconnectionFailure(&consecutiveFailures) {
            continue
        }
        rc.handleSuccessfulReconnection()
        if !rc.cancellableDelay(30 * time.Second) {
            return
        }
    }
}

/**
 * handleConnectionCheck checks the Redis connection status and resets the failure counter on success.
 *
 * @param consecutiveFailures Pointer to the failure counter.
 * @return True if the connection is healthy, false otherwise.
 */
func (rc *Client) handleConnectionCheck(consecutiveFailures *int) bool {
    err := rc.checkRedisConnection()
    if err == nil {
        // Reset counter on successful connection
        *consecutiveFailures = 0
        if !rc.cancellableDelay(30 * time.Second) {
            return true
        }
        return true
    }
    rc.Logger.WithError(err).Error("Redis connection lost. Attempting to reconnect...")
    return false
}

/**
 * handleReconnectionFailure processes a failed reconnection attempt.
 *
 * @param consecutiveFailures Pointer to the failure counter.
 * @return True if processing should continue, false if reconnection succeeded.
 */
func (rc *Client) handleReconnectionFailure(consecutiveFailures *int) bool {
    reconnectErr := rc.retryPing(24, 2*time.Second)
    if reconnectErr == nil {
        return false // Reconnection successful, don't continue this handler
    }
    *consecutiveFailures++
    if errors.Is(reconnectErr, context.Canceled) {
        return true
    }
    rc.Logger.WithError(reconnectErr).WithField("consecutiveFailures", *consecutiveFailures).Error("Unable to reconnect to Redis.")
    return rc.handleBackoffStrategy(*consecutiveFailures)
}

/**
 * handleBackoffStrategy applies exponential backoff based on consecutive failures.
 *
 * @param consecutiveFailures The current number of consecutive failures.
 * @return True to continue retrying, false if the context was canceled.
 */
func (rc *Client) handleBackoffStrategy(consecutiveFailures int) bool {
    const maxConsecutiveFailures = 30
    if consecutiveFailures >= maxConsecutiveFailures {
        rc.Logger.Error("Too many consecutive reconnection failures, backing off")
        // Exponential backoff to prevent CPU spinning
        backoffTime := time.Duration(math.Min(float64(consecutiveFailures * 5), 300)) * time.Second
        if !rc.cancellableDelay(backoffTime) {
            return false
        }
    } else if !rc.cancellableDelay(30 * time.Second) {
        return false
    }
    return true
}

/**
 * handleSuccessfulReconnection processes steps after a successful reconnection.
 */
func (rc *Client) handleSuccessfulReconnection() {
    rc.Logger.Info("Redis connection reestablished.")
    if syncErr := rc.checkAndSyncUUIDMap(); syncErr != nil {
        rc.Logger.WithError(syncErr).Error("Error synchronizing uuid-map after reconnection")
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
 * Includes frequent context checks to ensure prompt termination.
 */
func (rc *Client) uuidMapSyncTicker() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    // Create a more frequent check ticker for context cancellation
    contextCheckTicker := time.NewTicker(5 * time.Second)
    defer contextCheckTicker.Stop()
    for {
        // Immediate context check at the beginning of each loop
        if rc.Context.Err() != nil {
            rc.Logger.Debug("UUID map sync ticker stopping due to context cancellation")
            return
        }
        select {
        case <-rc.Context.Done():
            rc.Logger.Debug("UUID map sync ticker stopping due to context cancellation")
            return
        case <-ticker.C:
            if err := rc.checkAndSyncUUIDMap(); err != nil {
                rc.Logger.WithError(err).Error("Error during periodic uuid-map synchronization")
            }
        case <-contextCheckTicker.C:
            // More frequent check for context cancellation to avoid waiting a full minute
            if rc.Context.Err() != nil {
                rc.Logger.Debug("UUID map sync ticker stopping due to context cancellation (periodic check)")
                return
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
        if rc.Context.Err() != nil {
            return rc.Context.Err()
        }
        if _, err = rc.Client.Ping(rc.Context).Result(); err == nil {
            return nil
        }
        rc.Logger.WithFields(logrus.Fields{"retry": i + 1, "maxRetries": maxRetries}).
            WithError(err).
            Errorf("Error connecting to Redis. Retrying in %v...", retryDelay)
        if !rc.cancellableDelay(retryDelay) {
            return rc.Context.Err()
        }
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
    // Get a copy of the mapping
    mappingCopy := rc.UUIDMapper.GetMappingCopy()
    if len(mappingCopy) == 0 {
        rc.Logger.Warn("UUIDMapper is empty; nothing to synchronize to Redis.")
        return nil
    }
    // Extract keys for batching
    keys := rc.extractKeys(mappingCopy)
    return rc.processBatches(keys, mappingCopy)
}

/**
 * extractKeys extracts the keys from a map for batch processing.
 *
 * @param mapping The map containing the entries.
 * @return A slice of keys from the map.
 */
func (rc *Client) extractKeys(mapping map[string]uuid.UUIDEntry) []string {
    keys := make([]string, 0, len(mapping))
    for k := range mapping {
        keys = append(keys, k)
    }
    return keys
}

/**
 * processBatches processes the UUID entries in batches to avoid Redis command size limits.
 *
 * @param keys    A slice of keys to process.
 * @param mapping The mapping containing the entries.
 * @return An error if processing fails.
 */
func (rc *Client) processBatches(keys []string, mapping map[string]uuid.UUIDEntry) error {
    const batchSize = 500
    totalBatches := (len(keys) + batchSize - 1) / batchSize
    for i := 0; i < len(keys); i += batchSize {
        end := i + batchSize
        if end > len(keys) {
            end = len(keys)
        }
        currentBatch := (i / batchSize) + 1
        batchKeys := keys[i:end]
        if err := rc.processSingleBatch(batchKeys, mapping, currentBatch, totalBatches); err != nil {
            return err
        }
    }
    rc.Logger.Info("uuid-map synchronized to Redis successfully.")
    return nil
}

/**
 * processSingleBatch processes a single batch of UUID entries.
 *
 * @param batchKeys    The keys for this batch.
 * @param mapping      The full mapping.
 * @param currentBatch The current batch number.
 * @param totalBatches The total number of batches.
 * @return An error if batch processing fails.
 */
func (rc *Client) processSingleBatch(
	batchKeys []string,
	mapping map[string]uuid.UUIDEntry,
	currentBatch,
	totalBatches int,
) error {
    // Create a pipeline for this batch
    pipeline := rc.Client.Pipeline()
    // Add all entries in this batch to the pipeline
    for _, key := range batchKeys {
        entry := mapping[key]
        if err := rc.addUUIDEntryToPipeline(pipeline, key, entry); err != nil {
            return fmt.Errorf("error adding entry to pipeline: %w", err)
        }
    }
    // Execute the pipeline
    if _, err := pipeline.Exec(rc.Context); err != nil {
        return fmt.Errorf("error executing pipeline for uuid-map batch: %w", err)
    }
    // Log progress if processing multiple batches
    rc.logBatchProgress(currentBatch, totalBatches, len(batchKeys), len(mapping))
    return nil
}

/**
 * logBatchProgress logs the progress of batch processing.
 *
 * @param currentBatch The current batch number.
 * @param totalBatches The total number of batches.
 * @param batchSize    The number of items in the current batch.
 * @param totalItems   The total number of items being processed.
 */
func (rc *Client) logBatchProgress(currentBatch, totalBatches, batchSize, totalItems int) {
    if totalBatches > 1 {
        rc.Logger.WithFields(logrus.Fields{
            "batch": fmt.Sprintf("%d/%d", currentBatch, totalBatches),
            "items": fmt.Sprintf("%d items", batchSize),
            "progress": fmt.Sprintf("%.1f%%", float64(currentBatch) / float64(totalBatches) * 100),
        }).Debug("Batch synchronized to Redis")
    }
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
 * addTimestamp adds a timestamp to the data map if it does not already exist.
 *
 * @param data The data map to which the timestamp is conditionally added.
 * @return The updated data map with the timestamp, if it was not already present.
 */
func (rc *Client) addTimestamp(data map[string]interface{}) map[string]interface{} {
	if _, exists := data["timestamp"]; !exists {
		data["timestamp"] = time.Now().UnixMilli()
	}
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
 * PushToRedisAsync enqueues a stream entry for asynchronous batched XADD.
 *
 * @param uuid The name of the Redis stream.
 * @param max  The maximum number of records to retain in the stream.
 * @param data The data to be added to the stream.
 * @return An error if async batching is not enabled or context is canceled.
 */
func (rc *Client) PushToRedisAsync(uuid string, max int64, data map[string]interface{}) error {
	if rc.asyncCh == nil {
		return fmt.Errorf("async batching not enabled")
	}
	data = rc.addTimestamp(data)
	select {
	case rc.asyncCh <- streamItem{Stream: uuid, MaxLen: max, Values: data}:
		return nil
	case <-rc.Context.Done():
		return fmt.Errorf("context canceled")
	}
}

/**
 * EnableAsyncBatching enables asynchronous batched XADD operations.
 *
 * @param batchSize    The number of records per pipeline batch.
 * @param batchTimeout The maximum wait time before flushing a batch.
 */
func (rc *Client) EnableAsyncBatching(batchSize int, batchTimeout time.Duration) {
	if rc.asyncCh != nil {
		return
	}
	rc.asyncBatchSize = batchSize
	rc.asyncBatchTimeout = batchTimeout
	rc.asyncCh = make(chan streamItem, batchSize*2)
	rc.asyncWg.Add(1)
	go rc.runAsyncPublisher()
}

/**
 * runAsyncPublisher runs in a goroutine to batch and flush stream items.
 */
func (rc *Client) runAsyncPublisher() {
	defer rc.asyncWg.Done()
	// Ticker to force periodic flush
	ticker := time.NewTicker(rc.asyncBatchTimeout)
	defer ticker.Stop()
	// Batch buffer
	batch := make([]streamItem, 0, rc.asyncBatchSize)
	for {
		select {
		case it, ok := <-rc.asyncCh:
			if !ok {
				// Channel closed: flush remaining and exit
				rc.flushBatch(&batch)
				return
			}
			// Add item and flush if capacity reached
			batch = rc.appendToBatch(batch, it)
			if len(batch) >= rc.asyncBatchSize {
				rc.flushBatch(&batch)
			}
		case <-ticker.C:
			// Timeout: flush whatever's in batch
			rc.flushBatch(&batch)
		case <-rc.Context.Done():
			// Context cancelled: final flush and exit
			rc.flushBatch(&batch)
			return
		}
	}
}

/**
 * appendToBatch appends the given streamItem to the provided batch slice.
 *
 * @param batch The current slice of streamItem entries.
 * @param it    The streamItem to append to the batch.
 * @return A new slice containing all previous batch items plus the new item.
 */
func (rc *Client) appendToBatch(batch []streamItem, it streamItem) []streamItem {
    return append(batch, it)
}

/**
 * flushBatch sends all items in the batch to Redis in a single pipeline
 * and then resets the batch to an empty slice.
 *
 * @param batch A pointer to the slice of streamItem entries to flush.
 */
func (rc *Client) flushBatch(batch *[]streamItem) {
    if len(*batch) == 0 {
        return
    }
    pipe := rc.Client.Pipeline()
    for _, it := range *batch {
        pipe.XAdd(rc.Context, &redis.XAddArgs{
            Stream: it.Stream,
            ID:     "*",
            Values: it.Values,
            MaxLen: it.MaxLen,
        })
    }
    pipe.Exec(rc.Context)
    *batch = (*batch)[:0]
}

/**
 * DisableAsyncBatching disables async batching and flushes remaining items.
 */
func (rc *Client) DisableAsyncBatching() {
	if rc.asyncCh == nil {
		return
	}
	close(rc.asyncCh)
	rc.asyncWg.Wait()
	rc.asyncCh = nil
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
	// Instead of rc.UUIDMapper.Mapping = mapping, we call ReplaceMapping:
	rc.UUIDMapper.ReplaceMapping(mapping)
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
 * Includes timeout checks to prevent blocking during shutdown.
 *
 * @param startFunc The function to execute when a start command is received.
 * @param stopFunc  The function to execute when a stop command is received.
 */
func (rc *Client) ListenForCommands(startFunc, stopFunc func() error) {
    ps := pubsub.NewPubSub(rc.Logger)
    ctx := rc.Context
    // Subscribe without deferring pubsubClient.Close()
    rc.PubSub = rc.Client.Subscribe(ctx, "remote-control")
    // Retrieve the channel to read messages from
    ch := rc.PubSub.Channel()
    handlers := map[string]pubsub.CommandHandler{
        "start": startFunc,
        "stop":  stopFunc,
    }
    // Start a goroutine to force close PubSub on context cancellation
    go rc.monitorContextForPubSub()
    // Enter command listening loop with timeout checks
    rc.commandListeningLoop(ctx, ps, ch, handlers)
}

/**
 * monitorContextForPubSub monitors the context and forces PubSub closure when the context is done.
 * This ensures we don't get stuck waiting on the PubSub channel.
 */
func (rc *Client) monitorContextForPubSub() {
    <-rc.Context.Done()
    // Force closing the PubSub when context is canceled
    rc.stopPubSub()
}

/**
 * commandListeningLoop handles the main loop for listening to PubSub commands.
 * Includes regular timeout checks to ensure the application can exit properly.
 *
 * @param ctx        The context for controlling operation.
 * @param ps         The PubSub handler.
 * @param ch         The channel to read messages from.
 * @param handlers   The command handlers map.
 */
func (rc *Client) commandListeningLoop(ctx context.Context, ps *pubsub.PubSub, ch <-chan *redis.Message, handlers map[string]pubsub.CommandHandler) {
    // Create a ticker for regular timeout checks
    timeoutTicker := time.NewTicker(5 * time.Second)
    defer timeoutTicker.Stop()
    for {
        select {
        case <-ctx.Done():
            // Context is canceled: stop listening
            ps.Logger.Info("Stopping command listener (context canceled)")
            return
        case msg, ok := <-ch:
            // If the channel is closed or invalid, exit the loop to avoid panics
            if !ok {
                ps.Logger.Warn("PubSub channel closed unexpectedly")
                return
            }
            // Handle the incoming command
            ps.HandleCommand(msg.Payload, handlers)
        case <-timeoutTicker.C:
            // Regular check for context cancellation
            if ctx.Err() != nil {
                ps.Logger.Info("Command listener timeout check detected context cancellation")
                return
            }
        }
    }
}