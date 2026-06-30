package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

type Route struct {
	Prefix  string
	Target  string
}

var routes = []Route{
	{Prefix: "/auth/", Target: envOr("AUTH_SERVICE_URL", "http://localhost:8081")},
	{Prefix: "/notify/", Target: envOr("NOTIFY_SERVICE_URL", "http://localhost:8082")},
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func proxyHandler(target string) http.Handler {
	t, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(t)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error → %s: %v", target, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "upstream unavailable", "target": target})
	}
	return proxy
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    "ok",
			Service:   "api-gateway",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "1.0.0",
		})
	})

	mux.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routes)
	})

	for _, route := range routes {
		prefix := route.Prefix
		proxy := proxyHandler(route.Target)
		mux.Handle(prefix, http.StripPrefix(prefix[:len(prefix)-1], proxy))
	}

	port := envOr("PORT", "8080")
	handler := loggingMiddleware(corsMiddleware(mux))

	log.Printf("API Gateway v1.0.0 starting on :%s", port)
	log.Printf("Routes: %+v", routes)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("failed to start: %v", err)
	}
}
