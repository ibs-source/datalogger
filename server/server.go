/**
 * Package server provides the HTTP server implementation for the application,
 * including handlers for various endpoints and graceful shutdown mechanisms.
 */
 package server

 import (
	 "context"
	 "encoding/json"
	 "net/http"
	 "time"
 
	 "github.com/femogas/datalogger/app"
	 "github.com/femogas/datalogger/server/configuration"
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
	 srv.shutdownServer(server)
 }
 
 /**
  * shutdownServer gracefully shuts down the HTTP server with a timeout.
  *
  * @param server The HTTP server to shut down.
  */
 func (srv *Server) shutdownServer(server *http.Server) {
	 ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	 defer cancel()
	 err := server.Shutdown(ctxShutDown)
	 if err != nil {
		 srv.Logger.WithError(err).Error("Server Shutdown Failed")
		 return
	 }
	 srv.Logger.Info("Server gracefully stopped")
 }
 
 /**
  * handleUUIDMapping handles the /uuid-mapping endpoint and returns the UUID mapping as JSON.
  *
  * @param writer  The HTTP response writer.
  * @param request The HTTP request.
  */
 func (srv *Server) handleUUIDMapping(writer http.ResponseWriter, request *http.Request) {
	 // Read lock to safely access the UUID mapping.
	 srv.Main.UUIDMapper.RLock()
	 defer srv.Main.UUIDMapper.RUnlock()
	 srv.writeJSONResponse(writer, srv.Main.UUIDMapper.Mapping)
 }
 
 /**
  * handleStatus handles the /status endpoint and returns the connector status as JSON.
  *
  * @param writer  The HTTP response writer.
  * @param request The HTTP request.
  */
 func (srv *Server) handleStatus(writer http.ResponseWriter, request *http.Request) {
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
	 writer.WriteHeader(http.StatusOK)
	 _, err := writer.Write([]byte("Health ok!"))
	 if err != nil {
		 srv.Logger.WithError(err).Error("Failed to write healthz response")
		 return
	 }
 }
 