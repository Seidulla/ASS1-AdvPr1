package main

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 3)
var log = logrus.New()

func main() {

	log.SetFormatter(&logrus.JSONFormatter{})
	logFile, err := os.OpenFile("application.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Error("Failed to open log file: ", err)
		return
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	r := mux.NewRouter()
	r.HandleFunc("/", limitHandler(mainPageHandler)).Methods("GET")
	r.HandleFunc("/json", limitHandler(handleJSONRequest)).Methods("POST")
	r.HandleFunc("/device", limitHandler(createDeviceHandler)).Methods("POST")
	r.HandleFunc("/device/{id}", limitHandler(getDeviceHandler)).Methods("GET")
	r.HandleFunc("/device/{id}", limitHandler(updateDeviceHandler)).Methods("PUT")
	r.HandleFunc("/device/{id}", limitHandler(deleteDeviceHandler)).Methods("DELETE")

	log.Info("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
}

func limitHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}
