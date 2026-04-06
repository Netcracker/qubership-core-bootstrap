// pkg/api/server.go
package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"ratelimit-service/pkg/controller"
	"ratelimit-service/pkg/ratelimit"
	"ratelimit-service/pkg/utils"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
)

type Server struct {
	redisClient      *ratelimit.RedisClient
	controller       *controller.ConfigMapController
	rateLimitManager *ratelimit.RateLimitManager
	router           *mux.Router
	apiKey           string
	ready            bool
    mu               sync.RWMutex
}

type ServerConfig struct {
	APIKey            string
	EnableAuth        bool
	RateLimitAPI      bool
	RequestsPerSecond int
}

func NewServer(redisClient *ratelimit.RedisClient, controller *controller.ConfigMapController, rateLimitManager *ratelimit.RateLimitManager) *Server {
	config := &ServerConfig{
		APIKey:            utils.GetEnv("API_KEY", ""),
		EnableAuth:        utils.GetEnv("ENABLE_API_AUTH", "false") == "true",
		RateLimitAPI:      utils.GetEnv("RATE_LIMIT_API", "false") == "true",
		RequestsPerSecond: 10,
	}

	s := &Server{
		redisClient:      redisClient,
		controller:       controller,
		rateLimitManager: rateLimitManager,
		router:           mux.NewRouter(),
		apiKey:           config.APIKey,
	}
	s.setupRoutes()
	return s
}

func (s *Server) Stop() {
	klog.Info("API server stopping...")

}

func (s *Server) setupRoutes() {
	// Monitoring API
	s.router.HandleFunc("/api/v1/users/{user_id}/limits", s.authenticate(s.getUserLimits)).Methods("GET")
	s.router.HandleFunc("/api/v1/users/violating", s.authenticate(s.getViolatingUsers)).Methods("GET")
	s.router.HandleFunc("/api/v1/statistics", s.authenticate(s.getStatistics)).Methods("GET")

	// Rate limit management endpoints
	s.router.HandleFunc("/api/v1/ratelimit/check", s.authenticate(s.checkRateLimit)).Methods("POST")
	s.router.HandleFunc("/api/v1/ratelimit/rules", s.authenticate(s.getRules)).Methods("GET")
	s.router.HandleFunc("/api/v1/ratelimit/rules", s.authenticate(s.addRule)).Methods("POST")
	s.router.HandleFunc("/api/v1/ratelimit/rules/{name}", s.authenticate(s.deleteRule)).Methods("DELETE")

	// Management endpoints
	s.router.HandleFunc("/api/v1/users/{user_id}/reset", s.authenticate(s.resetUserLimits)).Methods("POST")
	s.router.HandleFunc("/api/v1/config/reload", s.authenticate(s.reloadConfig)).Methods("POST")

	s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
	s.router.HandleFunc("/ready", s.readinessCheck).Methods("GET")
}

func (s *Server) getRules(w http.ResponseWriter, r *http.Request) {

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Rules endpoint - returns all rate limit rules",
	})
}

func (s *Server) addRule(w http.ResponseWriter, r *http.Request) {
	var rule struct {
		Name      string `json:"name"`
		Pattern   string `json:"pattern"`
		Limit     int    `json:"limit"`
		WindowSec int    `json:"window_sec"`
		Algorithm string `json:"algorithm"`
	}

	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newRule := &ratelimit.Rule{
		Name:      rule.Name,
		Pattern:   rule.Pattern,
		Limit:     rule.Limit,
		Window:    time.Duration(rule.WindowSec) * time.Second,
		Algorithm: ratelimit.Algorithm(rule.Algorithm),
	}

	s.rateLimitManager.AddRule(newRule)

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Rule " + rule.Name + " added",
	})
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	s.rateLimitManager.RemoveRule(name)

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Rule " + name + " deleted",
	})
}

func (s *Server) checkRateLimit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Components map[string]string `json:"components"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := s.rateLimitManager.CheckWithComponents(r.Context(), req.Components, "|")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (s *Server) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
			next(w, r)
			return
		}

		providedKey := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(s.apiKey)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) reloadConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	klog.Info("Manual config reload triggered via API")

	if err := s.controller.ReloadConfig(ctx); err != nil {
		klog.Errorf("Failed to reload config: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Config reload triggered successfully",
	})
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (s *Server) readinessCheck(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()
    ready := s.ready
    s.mu.RUnlock()
    
    if !ready {
        http.Error(w, "not ready", http.StatusServiceUnavailable)
        return
    }
    
    respondWithJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) getUserLimits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	info, err := s.redisClient.GetUserRateLimitInfo(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, info)
}

func (s *Server) getViolatingUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.redisClient.GetViolatingUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"violating_users": users,
		"count":           len(users),
		"timestamp":       time.Now().Unix(),
	})
}

func (s *Server) getStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := s.redisClient.GetAllStatistics(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func (s *Server) resetUserLimits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if err := s.redisClient.ResetUserRateLimit(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Rate limits reset for user: " + userID,
	})
}

func (s *Server) Run(addr string) error {
    s.mu.Lock()
    s.ready = true
    s.mu.Unlock()

	klog.Infof("API server listening on %s", addr)
	if s.apiKey != "" {
		klog.Info("API authentication enabled")
	} else {
		klog.Warning("API authentication disabled - set API_KEY environment variable")
	}
	return http.ListenAndServe(addr, s.router)
}

func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}
