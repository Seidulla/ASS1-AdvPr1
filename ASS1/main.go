package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 3) // Rate limit of 1 request per second with a burst of 3 requests
var log = logrus.New()

func main() {
	// Set Logrus formatter
	log.SetFormatter(&logrus.JSONFormatter{})

	// Initialize the Gorilla mux router
	r := mux.NewRouter()

	// Handle main page requests
	r.HandleFunc("/", mainPageHandler)

	// Handle JSON requests
	r.HandleFunc("/json", handleJSONRequest).Methods("POST")

	// Handle device CRUD operations with rate limiting
	r.HandleFunc("/device", limitMiddleware(createDeviceHandler)).Methods("POST")
	r.HandleFunc("/device/{id}", limitMiddleware(getDeviceHandler)).Methods("GET")
	r.HandleFunc("/device/{id}", limitMiddleware(updateDeviceHandler)).Methods("PUT")
	r.HandleFunc("/device/{id}", limitMiddleware(deleteDeviceHandler)).Methods("DELETE")

	// Start the HTTP server
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
}

// Middleware function for rate limiting
func limitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}
