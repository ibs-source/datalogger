/**
 * Package global provides functions and types for initializing and managing global configurations.
 */
package global

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// EmptyString is a constant for an empty string.
const EmptyString = ""

/**
 * GlobalConfiguration holds the global settings for the application.
 */
type GlobalConfiguration struct {
	DebugMode             int    // DebugMode controls the logging level.
	ConfigFilePath        string // ConfigFilePath is the path to the configuration file.
	DefaultMaxStorageLogs int64  // DefaultMaxStorageLogs is the default maximum number of storage logs per variable.
}

/**
 * Initializes and returns a new GlobalConfiguration.
 * It reads environment variables and command-line flags to set the configuration values.
 *
 * @return A pointer to the initialized GlobalConfiguration.
 */
func InitializeGlobalConfiguration() *GlobalConfiguration {
	global := &GlobalConfiguration{
		DebugMode:             GetEnvAsInt("DEBUG", 1),
		ConfigFilePath:        GetEnv("CONFIG_FILE", "config.json"),
		DefaultMaxStorageLogs: GetEnvAsInt64("MAX_LOG_STORAGE", 1000),
	}
	overrideGlobalConfigWithFlags(global)
	return global
}

/**
 * Overrides the GlobalConfiguration values with command-line flags.
 *
 * @param global A pointer to the GlobalConfiguration to be overridden.
 */
func overrideGlobalConfigWithFlags(global *GlobalConfiguration) {
	flag.IntVar(&global.DebugMode, "debug", global.DebugMode, "Debug mode (0=all logs, 1=info only, 2=errors only)")
	flag.StringVar(&global.ConfigFilePath, "config-file", global.ConfigFilePath, "Path to the configuration file")
	flag.Int64Var(&global.DefaultMaxStorageLogs, "max-log-storage", global.DefaultMaxStorageLogs, "Maximum number of storage logs per variable")
	flag.Parse()
}

/**
 * Returns the string representation of value if it's not empty; otherwise, returns defaultValue.
 * Supports various types including string, fmt.Stringer, int, uint, float, and bool.
 *
 * @param value        The value to be converted to a string.
 * @param defaultValue The default string to return if value is empty.
 * @return The string representation of value or defaultValue.
 */
func DefaultValueString(value interface{}, defaultValue string) string {
	switch option := value.(type) {
	case string:
		if option != EmptyString {
			return option
		}
	case fmt.Stringer:
		casted := option.String()
		if casted != EmptyString {
			return casted
		}
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", option)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", option)
	case float32, float64:
		return fmt.Sprintf("%f", option)
	case bool:
		return fmt.Sprintf("%t", option)
	default:
		return defaultValue
	}
	return defaultValue
}

/**
 * Returns the integer representation of value if possible; otherwise, returns defaultValue.
 * Supports int and string types.
 *
 * @param value        The value to be converted to an integer.
 * @param defaultValue The default integer to return if conversion fails.
 * @return The integer representation of value or defaultValue.
 */
func DefaultValueInt(value interface{}, defaultValue int) int {
	switch option := value.(type) {
	case int:
		return option
	case string:
		casted, err := strconv.Atoi(option)
		if err != nil {
			return defaultValue
		}
		return casted
	default:
		return defaultValue
	}
}

/**
 * Returns the 64-bit integer representation of value if possible; otherwise, it returns defaultValue.
 * It supports various integer types and strings.
 *
 * @param value        The value to be converted to int64.
 * @param defaultValue The default integer value to return if the conversion fails.
 * @return The 64-bit integer representation of value or defaultValue.
 */
func DefaultValueInt64(value interface{}, defaultValue int64) int64 {
	switch option := value.(type) {
	case int:
		return int64(option)
	case int8:
		return int64(option)
	case int16:
		return int64(option)
	case int32:
		return int64(option)
	case int64:
		return option
	case uint:
		if option > math.MaxInt64 {
			return defaultValue
		}
		return int64(option)
	case uint8:
		return int64(option)
	case uint16:
		return int64(option)
	case uint32:
		return int64(option)
	case uint64:
		if option > math.MaxInt64 {
			return defaultValue
		}
		return int64(option)
	case string:
		if parsed, err := strconv.ParseInt(option, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

/**
 * Retrieves the environment variable named by the key.
 * Returns the value or defaultValue if the variable does not exist or is empty.
 *
 * @param key          The name of the environment variable.
 * @param defaultValue The default value to return if the environment variable is not set.
 * @return The value of the environment variable or defaultValue.
 */
func GetEnv(key, defaultValue string) string {
	environment, exists := os.LookupEnv(key)
	if !exists || environment == EmptyString {
		return defaultValue
	}
	return environment
}

/**
 * Retrieves the environment variable named by name and converts it to an integer.
 * Returns the integer value or defaultValue if the variable does not exist or conversion fails.
 *
 * @param name         The name of the environment variable.
 * @param defaultValue The default integer value to return if conversion fails.
 * @return The integer value of the environment variable or defaultValue.
 */
func GetEnvAsInt(name string, defaultValue int) int {
	environment := GetEnv(name, EmptyString)
	return DefaultValueInt(environment, defaultValue)
}

/**
 * Retrieves the environment variable named by name and converts it to a 64-bit integer.
 * Returns the integer value or defaultValue if the variable does not exist or conversion fails.
 *
 * @param name         The name of the environment variable.
 * @param defaultValue The default 64-bit integer value to return if conversion fails.
 * @return The 64-bit integer value of the environment variable or defaultValue.
 */
func GetEnvAsInt64(name string, defaultValue int64) int64 {
	environment := GetEnv(name, EmptyString)
	return DefaultValueInt64(environment, defaultValue)
}

/**
 * Initializes and returns a new logrus.Logger based on the debug level.
 * Debug levels:
 * - 0: DebugLevel (all logs)
 * - 1: InfoLevel (info logs and above)
 * - 2: ErrorLevel (errors only)
 *
 * @param debug The debug level for the logger.
 * @return A pointer to the initialized logrus.Logger.
 */
func InitializeLogger(debug int) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	// Set the log level based on the debug parameter.
	switch debug {
	case 0:
		logger.SetLevel(logrus.DebugLevel)
	case 1:
		logger.SetLevel(logrus.InfoLevel)
	default:
		logger.SetLevel(logrus.ErrorLevel)
	}
	return logger
}
