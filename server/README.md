# Server

## `func NewServer(producer *producer.Main, logger *logrus.Logger) *Server`

NewServer creates a new Server instance with the given producer and logger.

 * **Parameters:**
   * `producer` — The main application context.
   * `logger` — The logger for logging messages.
 * **Returns:** A pointer to the initialized Server.

## `func (srv *Server) Start() error`

Start initializes and starts the HTTP server.

 * **Returns:** An error if the server fails to start.
