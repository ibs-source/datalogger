/**
* Package redis provides functionalities for connecting and interacting with Redis,
* including client initialization, stream management, and message pushing.
*/
package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/femogas/datalogger/app/configuration"
	"github.com/femogas/datalogger/redis/configuration"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Expose PubSub type
type PubSub = redis.PubSub

// Expose Message type
type Message = redis.Message

/**
* Client encapsulates the Redis client and related configurations.
*/
type Client struct {
	Client        *redis.Client        // Client is the underlying Redis client.
	Logger        *logrus.Logger       // Logger for logging messages.
	Configuration *configuration.Redis // Configuration holds Redis connection settings.
	Context       context.Context      // Context for controlling operations.
}

/**
* NewClient initializes and returns a new Redis client.
*
* @param logger A logger instance for logging.
* @param ctx    The context for the client operations.
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
		Configuration: config,
		Context:       ctx,
	}

	// Ping Redis to ensure connection is established.
	if err := rc.Ping(); err != nil {
		cancel()
		return nil, err
	}

	// Initialize the Redis stream and consumer group.
	if err := rc.InitializeStream(); err != nil {
		return nil, err
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
		RedisURL:      global.GetEnv("REDIS_URL", "redis://127.0.0.1:6379"),
		StreamName:    global.GetEnv("REDIS_STREAM", "redis-stream"),
		ConsumerGroup: global.GetEnv("REDIS_CONSUMER", "redis-group"),
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
* Instance returns the underlying Redis client instance.
*
* @return A pointer to the Redis client.
*/
func (rc *Client) Instance() *redis.Client {
	return rc.Client
}

/**
* Close closes the Redis client connection.
*
* @return An error if closing fails.
*/
func (rc *Client) Close() error {
	err := rc.Instance().Close()
	if err != nil {
		rc.Logger.WithError(err).Error("Error closing Redis client")
		return err
	}
	return nil
}

/**
* InitializeStream initializes the Redis stream and consumer group.
*
* @return An error if initialization fails.
*/
func (rc *Client) InitializeStream() error {
	err := rc.createConsumerGroupIfNotExists()
	if err != nil {
		return err
	}
	rc.Logger.WithFields(logrus.Fields{
		"stream":        rc.Configuration.StreamName,
		"consumerGroup": rc.Configuration.ConsumerGroup,
	}).Info("Consumer Group created or already exists")
	return nil
}

/**
* createConsumerGroupIfNotExists creates a consumer group for the stream if it doesn't already exist.
*
* @return An error if the operation fails.
*/
func (rc *Client) createConsumerGroupIfNotExists() error {
	_, err := rc.Instance().XGroupCreateMkStream(
		context.Background(),
		rc.Configuration.StreamName,
		rc.Configuration.ConsumerGroup,
		"$",
	).Result()
	if err == nil || strings.Contains(err.Error(), "BUSYGROUP") {
		// BUSYGROUP error indicates the group already exists.
		return nil
	}
	rc.Logger.WithError(err).Error("Error creating Consumer Group")
	return err
}

/**
* Ping checks the connection to Redis by sending a PING command.
*
* @return An error if the ping fails.
*/
func (rc *Client) Ping() error {
	return rc.retryPing(24, 2 * time.Second)
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
		_, err = rc.Instance().Ping(rc.Context).Result()
		if err == nil {
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
* PushToRedis adds data to the Redis stream.
*
* @param data The data to be added to the stream.
* @return An error if the operation fails.
*/
func (rc *Client) PushToRedis(data map[string]interface{}) error {
	data = rc.addTimestamp(data)
	err := rc.addToStream(data)
	if err != nil {
		rc.Logger.WithError(err).Error("Error adding message to Stream")
		return err
	}
	return nil
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
* addToStream adds the data to the Redis stream.
*
* @param data The data to be added to the stream.
* @return An error if the operation fails.
*/
func (rc *Client) addToStream(data map[string]interface{}) error {
	_, err := rc.Instance().XAdd(rc.Context, &redis.XAddArgs{
		Stream: rc.Configuration.StreamName,
		ID:     "*",
		Values: data,
	}).Result()
	return err
}
