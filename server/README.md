# Server

This package provides the HTTP server implementation for the application,
including handlers for various endpoints and graceful shutdown mechanisms.

## `func NewServer(producer *producer.Main, logger *logrus.Logger) *Server`

Creates a new Server instance with the given producer and logger.

- **Parameters:**

  - `producer` — The main application context.
  - `logger` — The logger for logging messages.

- **Returns:**  
  A pointer to the initialized `Server`.

## `func (srv *Server) Start() error`

Initializes and starts the HTTP server.

- **Returns:**  
  An error if the server fails to start.
