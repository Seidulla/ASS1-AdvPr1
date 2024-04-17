package main

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 3)
var log = logrus.New()

func main() {
	log.SetFormatter(&logrus.JSONFormatter{})

	r := mux.NewRouter()

	r.HandleFunc("/", mainPageHandler)

	r.HandleFunc("/json", handleJSONRequest).Methods("POST")

	r.HandleFunc("/device", limitMiddleware(createDeviceHandler)).Methods("POST")
	r.HandleFunc("/device/{id}", limitMiddleware(getDeviceHandler)).Methods("GET")
	r.HandleFunc("/device/{id}", limitMiddleware(updateDeviceHandler)).Methods("PUT")
	r.HandleFunc("/device/{id}", limitMiddleware(deleteDeviceHandler)).Methods("DELETE")

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
}

func limitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}
