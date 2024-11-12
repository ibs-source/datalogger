/**
* Package configuration provides the structures and settings
* necessary for configuring the application's Redis connection.
*/
package configuration

/**
* Redis holds the configuration settings for connecting to a Redis instance.
*/
type Redis struct {
	/**
	* RedisURL is the URL of the Redis server.
	*/
	RedisURL string
	/**
	* StreamName is the name of the Redis stream to interact with.
	*/
	StreamName string
	/**
	* ConsumerGroup is the name of the consumer group for stream processing.
	*/
	ConsumerGroup string
}
