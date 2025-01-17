# Global

| Parameter         | Type     | Description                                                                                                                        | Default       |
| ----------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| `DEBUG`           | `int`    | Sets the logging level:<br> - `0`: DebugLevel (all logs)<br> - `1`: InfoLevel (info and above)<br> - `2`: ErrorLevel (errors only) | `1`           |
| `CONFIG_FILE`     | `string` | Specifies the path to the configuration file.                                                                                      | `config.json` |
| `MAX_LOG_STORAGE` | `int64`  | Defines the maximum number of storage logs per variable. This value is represented in the code as `DefaultMaxStorageLogs`.         | `1000`        |

---

## `func InitializeGlobalConfiguration() *GlobalConfiguration`

Initializes and returns a new `GlobalConfiguration`. It reads environment variables and command-line flags to set the configuration values.

- **Returns:** A pointer to the initialized `GlobalConfiguration`.

## `func overrideGlobalConfigWithFlags(global *GlobalConfiguration)`

Overrides the `GlobalConfiguration` values with command-line flags.

- **Parameters:**
  - `global` — A pointer to the `GlobalConfiguration` to be overridden.

## `func DefaultValueString(value interface{}, defaultValue string) string`

Returns the string representation of `value` if it's not empty; otherwise, returns `defaultValue`. Supports various types including `string`, `fmt.Stringer`, `int`, `uint`, `float`, and `bool`.

- **Parameters:**
  - `value` — The value to be converted to a string.
  - `defaultValue` — The default string to return if `value` is empty.
- **Returns:** The string representation of `value` or `defaultValue`.

## `func DefaultValueInt(value interface{}, defaultValue int) int`

Returns the integer representation of `value` if possible; otherwise, returns `defaultValue`. Supports `int` and `string` types.

- **Parameters:**
  - `value` — The value to be converted to an integer.
  - `defaultValue` — The default integer to return if conversion fails.
- **Returns:** The integer representation of `value` or `defaultValue`.

## `func DefaultValueInt64(value interface{}, defaultValue int64) int64`

Returns the 64-bit integer representation of `value` if possible; otherwise, returns `defaultValue`. Supports various integer types and strings.

- **Parameters:**
  - `value` — The value to be converted to a 64-bit integer.
  - `defaultValue` — The default 64-bit integer value to return if the conversion fails.
- **Returns:** The 64-bit integer representation of `value` or `defaultValue`.

## `func GetEnv(key, defaultValue string) string`

Retrieves the environment variable named by the `key`. Returns the value or `defaultValue` if the variable does not exist or is empty.

- **Parameters:**
  - `key` — The name of the environment variable.
  - `defaultValue` — The default value to return if the environment variable is not set.
- **Returns:** The value of the environment variable or `defaultValue`.

## `func GetEnvAsInt(name string, defaultValue int) int`

Retrieves the environment variable named by `name` and converts it to an integer. Returns the integer value or `defaultValue` if the variable does not exist or conversion fails.

- **Parameters:**
  - `name` — The name of the environment variable.
  - `defaultValue` — The default integer value to return if conversion fails.
- **Returns:** The integer value of the environment variable or `defaultValue`.

## `func GetEnvAsInt64(name string, defaultValue int64) int64`

Retrieves the environment variable named by `name` and converts it to a 64-bit integer. Returns the integer value or `defaultValue` if the variable does not exist or conversion fails.

- **Parameters:**
  - `name` — The name of the environment variable.
  - `defaultValue` — The default 64-bit integer value to return if conversion fails.
- **Returns:** The 64-bit integer value of the environment variable or `defaultValue`.

## `func InitializeLogger(debug int) *logrus.Logger`

Initializes and returns a new `logrus.Logger` based on the debug level.

Debug levels:

- `0`: DebugLevel (all logs)
- `1`: InfoLevel (info logs and above)
- `2`: ErrorLevel (errors only)

- **Parameters:**
  - `debug` — The debug level for the logger.
- **Returns:** A pointer to the initialized `logrus.Logger`.
