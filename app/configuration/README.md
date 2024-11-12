# Global

| Parameter           | Type     | Description                                                            | Default       |
|---------------------|----------|------------------------------------------------------------------------|---------------|
| `DEBUG`             | `int`    | Enables or disables debug mode (1 to enable, 0 to disable).           | `1`           |
| `CONFIG_FILE`       | `string` | Specifies the path to the configuration file.                         | `config.json` |
| `MAX_STORAGE_LOG`   | `int`    | Defines the maximum number of logs that can be stored.                | `15000`       |
| `MAX_STORAGE_TTL`   | `int`    | Sets the time-to-live for stored logs, in seconds.                    | `3600`        |

## `func InitializeGlobalConfiguration() *GlobalConfiguration`

Initializes and returns a new GlobalConfiguration. It reads environment variables and command-line flags to set the configuration values.

 * **Returns:** A pointer to the initialized GlobalConfiguration.

## `func DefaultValueString(value interface`

Returns the string representation of value if it's not empty; otherwise, returns defaultValue. Supports various types including string, fmt.Stringer, int, uint, float, and bool.

 * **Parameters:**
   * `value` — The value to be converted to a string.
   * `defaultValue` — The default string to return if value is empty.
 * **Returns:** The string representation of value or defaultValue.

## `func DefaultValueInt(value interface`

Returns the integer representation of value if possible; otherwise, returns defaultValue. Supports int and string types.

 * **Parameters:**
   * `value` — The value to be converted to an integer.
   * `defaultValue` — The default integer to return if conversion fails.
 * **Returns:** The integer representation of value or defaultValue.

## `func GetEnv(key, defaultValue string) string`

Retrieves the environment variable named by the key. Returns the value or defaultValue if the variable does not exist or is empty.

 * **Parameters:**
   * `key` — The name of the environment variable.
   * `defaultValue` — The default value to return if the environment variable is not set.
 * **Returns:** The value of the environment variable or defaultValue.

## `func GetEnvAsInt(name string, defaultValue int) int`

Retrieves the environment variable named by name and converts it to an integer. Returns the integer value or defaultValue if the variable does not exist or conversion fails.

 * **Parameters:**
   * `name` — The name of the environment variable.
   * `defaultValue` — The default integer value to return if conversion fails.
 * **Returns:** The integer value of the environment variable or defaultValue.

## `func InitializeLogger(debug int) *logrus.Logger`

Initializes and returns a new logrus.Logger based on the debug level. Debug levels: - 0: DebugLevel (all logs) - 1: InfoLevel (info logs and above) - 2: ErrorLevel (errors only)

 * **Parameters:** `debug` — The debug level for the logger.
 * **Returns:** A pointer to the initialized logrus.Logger.