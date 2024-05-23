package main

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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
	r.Use(methodOverrideMiddleware)
	r.HandleFunc("/", limitHandler(mainPageHandler)).Methods("GET")
	r.HandleFunc("/json", limitHandler(handleJSONRequest)).Methods("POST")
	r.HandleFunc("/device", limitHandler(createDeviceHandler)).Methods("POST")
	r.HandleFunc("/device/{id}", limitHandler(getDeviceHandler)).Methods("GET")
	r.HandleFunc("/device/{id}", limitHandler(updateDeviceHandler)).Methods("PUT")
	r.HandleFunc("/device/{id}", limitHandler(deleteDeviceHandler)).Methods("DELETE")
	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/confirm", confirmHandler).Methods("GET")
	r.HandleFunc("/user", authMiddleware(userProfileHandler)).Methods("GET")
	r.HandleFunc("/admin", authMiddleware(adminMiddleware(adminProfileHandler))).Methods("GET")
	r.HandleFunc("/change-password", authMiddleware(changePasswordHandler)).Methods("POST")
	r.HandleFunc("/change-email", authMiddleware(changeEmailHandler)).Methods("POST")
	r.HandleFunc("/admin/roles", authMiddleware(adminMiddleware(createRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/roles/update", authMiddleware(adminMiddleware(updateRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/roles/delete", authMiddleware(adminMiddleware(deleteRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/send-email", authMiddleware(adminMiddleware(sendEmailHandler))).Methods("POST")

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
