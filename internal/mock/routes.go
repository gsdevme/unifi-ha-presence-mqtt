package mock

import "net/http"

// addRoutes is the single place mapping the UniFi controller URL surface to
// handlers, mirroring the two endpoints internal/unifi calls.
func addRoutes(mux *http.ServeMux, s *Server) {
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("GET /proxy/network/api/s/{site}/stat/sta", s.handleActiveClients)
}
