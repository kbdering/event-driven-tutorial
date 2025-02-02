package health
 
import (
	"net/http"
	"sync/atomic"
)
 
type HealthCheck struct {
	isReady    atomic.Bool
	isHealthy  atomic.Bool
	httpServer *http.Server
}
 
func NewHealthCheck() *HealthCheck {
	h := &HealthCheck{}
 
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.healthHandler)
	mux.HandleFunc("/ready", h.readyHandler)
 
	h.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
 
	return h
}
 
func (h *HealthCheck) Start() {
	go h.httpServer.ListenAndServe()
}
 
func (h *HealthCheck) Stop() error {
	return h.httpServer.Close()
}
 
func (h *HealthCheck) SetReady(ready bool) {
	h.isReady.Store(ready)
}
 
func (h *HealthCheck) SetHealthy(healthy bool) {
	h.isHealthy.Store(healthy)
}
 
func (h *HealthCheck) healthHandler(w http.ResponseWriter, r *http.Request) {
	if h.isHealthy.Load() {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}
 
func (h *HealthCheck) readyHandler(w http.ResponseWriter, r *http.Request) {
	if h.isReady.Load() {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}
