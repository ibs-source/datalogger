/**
 * Package server provides the HTTP server implementation for the application,
 * including handlers for various endpoints and graceful shutdown mechanisms.
 */
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ibs-source/datalogger/application"
	"github.com/ibs-source/datalogger/server/configuration"
	"github.com/sirupsen/logrus"
)

/**
 * Server represents the HTTP server with its configuration, main application context,
 * logger, and HTTP handler.
 */
type Server struct {
	Configuration *configuration.Server // Configuration holds server settings.
	Main          *producer.Main        // Main is the main application context.
	Logger        *logrus.Logger        // Logger for logging messages.
	Handler       http.Handler          // Handler is the HTTP handler.
	shutdownOnce  sync.Once             // Ensures shutdown happens only once
}

/**
 * NewServer creates a new Server instance with the given producer and logger.
 *
 * @param producer The main application context.
 * @param logger   The logger for logging messages.
 * @return A pointer to the initialized Server.
 */
func NewServer(producer *producer.Main, logger *logrus.Logger) *Server {
	config := &configuration.Server{
		Address: ":8080",
	}
	return &Server{
		Configuration: config,
		Main:          producer,
		Logger:        logger,
	}
}

/**
 * Start initializes and starts the HTTP server.
 *
 * @return An error if the server fails to start.
 */
func (srv *Server) Start() error {
	server := srv.createHTTPServer()
	// Start the server in a new goroutine.
	go srv.listenAndServe(server)
	// Listen for context cancellation to shutdown the server.
	go srv.shutdownOnContextDone(server)
	srv.Logger.Infof("Server started on %s", srv.Configuration.Address)
	return nil
}

/**
 * createHTTPServer sets up the HTTP server with the necessary handlers.
 *
 * @return A pointer to the configured http.Server.
 */
func (srv *Server) createHTTPServer() *http.Server {
	mux := http.NewServeMux()
	srv.registerHandlers(mux)
	return &http.Server{
		Addr:    srv.Configuration.Address,
		Handler: mux,
	}
}

/**
 * registerHandlers registers the HTTP handlers for the server endpoints.
 *
 * @param mux The HTTP request multiplexer.
 */
func (srv *Server) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/uuid-mapping", srv.handleUUIDMapping)
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/status", srv.handleStatus)
}

/**
 * listenAndServe starts the HTTP server and listens for incoming requests.
 *
 * @param server The HTTP server to start.
 */
func (srv *Server) listenAndServe(server *http.Server) {
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		// Server was closed gracefully.
		return
	}
	if err != nil {
		// Fatal error occurred, log and exit.
		srv.Logger.WithError(err).Fatal("Server failed")
	}
}

/**
 * shutdownOnContextDone waits for the application context to be done and shuts down the server.
 *
 * @param server The HTTP server to shut down.
 */
func (srv *Server) shutdownOnContextDone(server *http.Server) {
	<-srv.Main.Context.Done()
	// Use sync.Once to ensure shutdown happens only once
	srv.shutdownOnce.Do(func() {
		srv.shutdownServer(server)
	})
}

/**
 * shutdownServer gracefully shuts down the HTTP server with a timeout.
 * Includes safeguards to prevent indefinite blocking during shutdown.
 *
 * @param server The HTTP server to shut down.
 */
func (srv *Server) shutdownServer(server *http.Server) {
    srv.Logger.Info("Initiating server shutdown...")
    // Create a context with a reasonable timeout
    ctxShutDown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    // Set up a channel to track shutdown completion
    done := make(chan struct{})
    // Start shutdown operation in a separate goroutine
    go srv.shutdownServerWorker(server, ctxShutDown, done)
    
    // Wait for shutdown to complete or timeout
    select {
    case <-done:
        srv.Logger.Info("Server gracefully stopped")
    case <-time.After(15*time.Second): // Give a bit more time than the context timeout
        srv.Logger.Warn("Server shutdown timed out, forcing exit")
        // Force close connections
        if err := server.Close(); err != nil {
            srv.Logger.WithError(err).Error("Error during forced server close")
        }
    }
}
/**
 * validateMethod checks if the HTTP request uses the expected method.
 * If not, it returns an appropriate error response.
 *
 * @param writer      The HTTP response writer.
 * @param request     The HTTP request.
 * @param allowedMethod The HTTP method that is allowed (e.g., http.MethodGet).
 * @return bool       True if validation passes, false otherwise.
 */
func validateMethod(writer http.ResponseWriter, request *http.Request, allowedMethod string) bool {
    if request.Method != allowedMethod {
        http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
        return false
    }
    return true
}

/**
 * handleUUIDMapping handles the /uuid-mapping endpoint and returns the UUID mapping as JSON.
 *
 * @param writer  The HTTP response writer.
 * @param request The HTTP request.
 */
func (srv *Server) handleUUIDMapping(writer http.ResponseWriter, request *http.Request) {
    if !validateMethod(writer, request, http.MethodGet) {
        return
    }
    // Get the mapping
    mappingCopy := srv.Main.Redis.UUIDMapper.GetMappingCopy()
    srv.writeJSONResponse(writer, mappingCopy)
}

/**
 * handleStatus handles the /status endpoint and returns the connector status as JSON.
 *
 * @param writer  The HTTP response writer.
 * @param request The HTTP request.
 */
func (srv *Server) handleStatus(writer http.ResponseWriter, request *http.Request) {
    if !validateMethod(writer, request, http.MethodGet) {
        return
    }
    status := srv.Main.Connector.Status()
    srv.writeJSONResponse(writer, status)
}

/**
 * writeJSONResponse writes the given data as a JSON response.
 *
 * @param writer The HTTP response writer.
 * @param data   The data to encode and write as JSON.
 */
func (srv *Server) writeJSONResponse(writer http.ResponseWriter, data interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(writer).Encode(data)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

/**
 * handleHealthz handles the /healthz endpoint for health checks.
 *
 * @param writer  The HTTP response writer.
 * @param request The HTTP request.
 */
func (srv *Server) handleHealthz(writer http.ResponseWriter, request *http.Request) {
    if !validateMethod(writer, request, http.MethodGet) {
        return
    }
    writer.WriteHeader(http.StatusOK)
    _, err := writer.Write([]byte("Health ok!"))
    if err != nil {
        srv.Logger.WithError(err).Error("Failed to write healthz response")
        return
    }
}

/**
 * shutdownServerWorker performs the actual server shutdown and signals on the provided channel when complete.
 *
 * @param server     The HTTP server to shut down.
 * @param ctx        The context with timeout for the operation.
 * @param doneChan   The channel to signal when shutdown is complete.
 */
func (srv *Server) shutdownServerWorker(server *http.Server, ctx context.Context, doneChan chan<- struct{}) {
    err := server.Shutdown(ctx)
    if err != nil {
        srv.Logger.WithError(err).Error("Server Shutdown Failed")
    }
    close(doneChan)
}