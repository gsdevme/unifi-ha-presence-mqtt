package mock

import "net/http"

// addRoutes is the single place mapping the UniFi Network Integration API URL
// surface to handlers, mirroring the two endpoints internal/unifi calls.
func addRoutes(mux *http.ServeMux, s *Server) {
	mux.HandleFunc("GET /proxy/network/integration/v1/sites", s.handleSites)
	mux.HandleFunc("GET /proxy/network/integration/v1/sites/{site}/clients", s.handleClients)
}
