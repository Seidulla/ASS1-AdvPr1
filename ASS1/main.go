package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var limiter = rate.NewLimiter(1, 10)
var log = logrus.New()

var cartStorage = struct {
	sync.RWMutex
	carts map[string][]Device
}{carts: make(map[string][]Device)}

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
	r.HandleFunc("/buy", limitHandler(buyHandler)).Methods("POST")
	r.HandleFunc("/json", limitHandler(handleJSONRequest)).Methods("POST")

	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/confirm", confirmHandler).Methods("GET")
	r.HandleFunc("/user", authMiddleware(userProfileHandler)).Methods("GET")
	r.HandleFunc("/admin", authMiddleware(adminMiddleware(adminProfileHandler))).Methods("GET")
	r.HandleFunc("/change-password", authMiddleware(changePasswordHandler)).Methods("POST")
	r.HandleFunc("/change-email", authMiddleware(changeEmailHandler)).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("GET")

	// Admin routes for device management
	r.HandleFunc("/device", authMiddleware(adminMiddleware(createDeviceHandler))).Methods("POST")
	r.HandleFunc("/device/{id}", authMiddleware(adminMiddleware(getDeviceHandler))).Methods("GET")
	r.HandleFunc("/device/{id}", authMiddleware(adminMiddleware(updateDeviceHandler))).Methods("POST", "PUT")
	r.HandleFunc("/device/{id}", authMiddleware(adminMiddleware(deleteDeviceHandler))).Methods("POST", "DELETE")

	r.HandleFunc("/admin/roles", authMiddleware(adminMiddleware(createRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/roles/update", authMiddleware(adminMiddleware(updateRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/roles/delete", authMiddleware(adminMiddleware(deleteRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/send-email", authMiddleware(adminMiddleware(sendEmailHandler))).Methods("POST")

	log.Info("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
}

func mainPageHandler(w http.ResponseWriter, r *http.Request) {
	devices, err := GetDevicesFromDB()
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}
	filter := r.URL.Query().Get("filter")
	sort := r.URL.Query().Get("sort")
	page := r.URL.Query().Get("page")

	limit := 10
	offset := 0

	if p, err := strconv.Atoi(page); err == nil && p > 1 {
		offset = (p - 1) * limit
	}

	log.WithFields(logrus.Fields{
		"action": "mainPageHandler",
		"method": r.Method,
		"path":   r.URL.Path,
	}).Info("Handling main page request")

	query := "SELECT id, type1, brand, model FROM electronic"
	if filter != "" {
		query += " WHERE brand LIKE '%" + filter + "%'"
	}
	if sort != "" {
		query += " ORDER BY " + sort
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	devices, err = GetDevicesFromDBWithPagination(query)
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}

	// Check if user is logged in
	isLoggedIn := false
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		isLoggedIn = true
	}

	data := struct {
		Devices    []Device
		IsLoggedIn bool
	}{
		Devices:    devices,
		IsLoggedIn: isLoggedIn,
	}

	tmpl, err := template.ParseFiles("pages/index.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
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

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		Value:  "",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func buyHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	deviceIDStr := r.FormValue("device_id")
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := GetDeviceByID(deviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	cartStorage.Lock()
	cartStorage.carts[userID] = append(cartStorage.carts[userID], device)
	cartStorage.Unlock()

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func GetDeviceByID(id int) (Device, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return Device{}, err
	}
	defer db.Close()

	var device Device
	query := "SELECT id, type1, brand, model FROM electronic WHERE id = ?"
	err = db.QueryRow(query, id).Scan(&device.ID, &device.Type1, &device.Brand, &device.Model)
	if err != nil {
		return Device{}, err
	}

	return device, nil
}
